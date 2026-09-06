package font

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"

	"github.com/andybalholm/brotli"

	"github.com/tdewolff/parse/v2"
)

// Specification:
// https://www.w3.org/TR/WOFF2/

// Validation tests:
// https://github.com/w3c/woff2-tests

// Other implementations:
// http://git.savannah.gnu.org/cgit/freetype/freetype2.git/tree/src/sfnt/sfwoff2.c
// https://github.com/google/woff2/tree/master/src
// https://github.com/fonttools/fonttools/blob/master/Lib/fontTools/ttLib/woff2.py

type woff2Table struct {
	tag              string
	origLength       uint32
	transformVersion int
	transformLength  uint32
	data             []byte
}

var woff2TableTags = []string{
	"cmap", "head", "hhea", "hmtx",
	"maxp", "name", "OS/2", "post",
	"cvt ", "fpgm", "glyf", "loca",
	"prep", "CFF ", "VORG", "EBDT",
	"EBLC", "gasp", "hdmx", "kern",
	"LTSH", "PCLT", "VDMX", "vhea",
	"vmtx", "BASE", "GDEF", "GPOS",
	"GSUB", "EBSC", "JSTF", "MATH",
	"CBDT", "CBLC", "COLR", "CPAL",
	"SVG ", "sbix", "acnt", "avar",
	"bdat", "bloc", "bsln", "cvar",
	"fdsc", "feat", "fmtx", "fvar",
	"gvar", "hsty", "just", "lcar",
	"mort", "morx", "opbd", "prop",
	"trak", "Zapf", "Silf", "Glat",
	"Gloc", "Feat", "Sill",
}

// ParseWOFF2 parses the WOFF2 font format and returns its contained SFNT font format (TTF or OTF). See https://www.w3.org/TR/WOFF2/
func ParseWOFF2(b []byte) ([]byte, error) {
	if len(b) < 48 {
		return nil, ErrInvalidFontData
	}

	r := parse.NewBinaryReaderBytes(b)
	signature := r.ReadString(4)
	if signature != "wOF2" {
		return nil, fmt.Errorf("bad signature")
	}
	flavor := r.ReadUint32()
	if uint32ToString(flavor) == "ttcf" {
		return nil, fmt.Errorf("collections are unsupported")
	}
	length := r.ReadUint32()              // length
	numTables := r.ReadUint16()           // numTables
	reserved := r.ReadUint16()            // reserved
	totalSfntSize := r.ReadUint32()       // totalSfntSize
	totalCompressedSize := r.ReadUint32() // totalCompressedSize
	_ = r.ReadUint16()                    // majorVersion
	_ = r.ReadUint16()                    // minorVersion
	_ = r.ReadUint32()                    // metaOffset
	_ = r.ReadUint32()                    // metaLength
	_ = r.ReadUint32()                    // metaOrigLength
	_ = r.ReadUint32()                    // privOffset
	_ = r.ReadUint32()                    // privLength
	if r.Err() == io.EOF {
		return nil, ErrInvalidFontData
	} else if length != uint32(len(b)) {
		return nil, fmt.Errorf("length in header must match file size")
	} else if numTables == 0 {
		return nil, fmt.Errorf("numTables in header must not be zero")
	} else if reserved != 0 {
		return nil, fmt.Errorf("reserved in header must be zero")
	}

	tags := []string{}
	tagTableIndex := map[string]int{}
	tables := []woff2Table{}
	var uncompressedSize uint32
	for i := 0; i < int(numTables); i++ {
		flags := r.ReadUint8()
		tagIndex := int(flags & 0x3F)
		transformVersion := int((flags & 0xC0) >> 6)

		var tag string
		if tagIndex == 63 {
			tag = uint32ToString(r.ReadUint32())
		} else {
			tag = woff2TableTags[tagIndex]
		}

		origLength, err := readUintBase128(r) // if EOF is encountered above
		if err != nil {
			return nil, err
		}

		var transformLength uint32
		if (tag == "glyf" || tag == "loca") && transformVersion == 0 || tag == "hmtx" && transformVersion == 1 {
			transformLength, err = readUintBase128(r)
			if err != nil || tag != "loca" && transformLength == 0 {
				return nil, fmt.Errorf("%s: transformLength must be set", tag)
			}
			if math.MaxUint32-uncompressedSize < transformLength {
				return nil, ErrInvalidFontData
			}
			uncompressedSize += transformLength
		} else if transformVersion == 0 || transformVersion == 3 && (tag == "glyf" || tag == "loca") {
			if math.MaxUint32-uncompressedSize < origLength {
				return nil, ErrInvalidFontData
			}
			uncompressedSize += origLength
		} else {
			return nil, fmt.Errorf("%s: invalid transformation", tag)
		}

		if tag == "loca" {
			iGlyf, hasGlyf := tagTableIndex["glyf"]
			if uint32ToString(flavor) == "ttcf" && (!hasGlyf || i-1 != iGlyf) {
				// TODO: should find last glyf table, map lookup probably doesn't work here
				return nil, fmt.Errorf("loca: must come directly after glyf table")
			} else if !hasGlyf {
				return nil, fmt.Errorf("loca: must come after glyf table")
			}
		}
		if _, ok := tagTableIndex[tag]; ok {
			return nil, fmt.Errorf("%s: table defined more than once", tag)
		}

		tags = append(tags, tag)
		tagTableIndex[tag] = len(tables)
		tables = append(tables, woff2Table{
			tag:              tag,
			origLength:       origLength,
			transformVersion: transformVersion,
			transformLength:  transformLength,
		})
	}

	iGlyf, hasGlyf := tagTableIndex["glyf"]
	iLoca, hasLoca := tagTableIndex["loca"]
	if hasGlyf != hasLoca || hasGlyf && tables[iGlyf].transformVersion != tables[iLoca].transformVersion {
		return nil, fmt.Errorf("glyf and loca tables must be both present and either be both transformed or untransformed")
	} else if hasLoca && tables[iLoca].transformLength != 0 {
		return nil, fmt.Errorf("loca: transformLength must be zero")
	}

	// TODO: (WOFF2) parse collection directory format

	// decompress font data using Brotli
	compData := r.ReadBytes(int64(totalCompressedSize))
	if err := r.Err(); err == io.EOF {
		return nil, ErrInvalidFontData
	} else if err != nil {
		return nil, err
	} else if MaxMemory < uncompressedSize {
		return nil, ErrExceedsMemory
	}
	rBrotli := brotli.NewReader(bytes.NewReader(compData)) // err is always nil
	dataBuf := bytes.NewBuffer(make([]byte, 0, uncompressedSize))
	if _, err := io.Copy(dataBuf, rBrotli); err != nil {
		return nil, err
	}
	data := dataBuf.Bytes()
	if uint32(len(data)) != uncompressedSize {
		return nil, fmt.Errorf("sum of table lengths must match decompressed font data size")
	}

	// read font data
	var offset uint32
	for i := range tables {
		if tables[i].tag == "loca" && tables[i].transformVersion == 0 {
			continue // will be reconstructed
		}

		n := tables[i].origLength
		if tables[i].transformLength != 0 {
			n = tables[i].transformLength
		}
		if uint32(len(data))-offset < n {
			return nil, ErrInvalidFontData
		}
		tables[i].data = data[offset : offset+n : offset+n]
		offset += n
	}

	// detransform font data tables
	if hasGlyf {
		if tables[iGlyf].transformVersion == 0 {
			var err error
			tables[iGlyf].data, tables[iLoca].data, err = reconstructGlyfLoca(tables[iGlyf].data, tables[iLoca].origLength)
			if err != nil {
				return nil, err
			}
			if tables[iLoca].origLength != uint32(len(tables[iLoca].data)) {
				return nil, fmt.Errorf("loca: invalid value for origLength")
			}
		} else {
			rGlyf := parse.NewBinaryReaderBytes(tables[iGlyf].data)
			_ = rGlyf.ReadUint32() // version
			numGlyphs := uint32(rGlyf.ReadUint16())
			indexFormat := rGlyf.ReadUint16()
			if rGlyf.Err() == io.EOF {
				return nil, ErrInvalidFontData
			}
			if indexFormat == 0 && tables[iLoca].origLength != (numGlyphs+1)*2 || indexFormat == 1 && tables[iLoca].origLength != (numGlyphs+1)*4 {
				return nil, fmt.Errorf("loca: invalid value for origLength")
			}
		}
	}

	if iHmtx, hasHmtx := tagTableIndex["hmtx"]; hasHmtx && tables[iHmtx].transformVersion == 1 {
		iHead, ok := tagTableIndex["head"]
		if !ok {
			return nil, fmt.Errorf("hmtx: head table must be defined in order to rebuild hmtx table")
		}
		if !hasGlyf {
			return nil, fmt.Errorf("hmtx: glyf table must be defined in order to rebuild hmtx table")
		}
		if !hasLoca {
			return nil, fmt.Errorf("hmtx: loca table must be defined in order to rebuild hmtx table")
		}
		iMaxp, ok := tagTableIndex["maxp"]
		if !ok {
			return nil, fmt.Errorf("hmtx: maxp table must be defined in order to rebuild hmtx table")
		}
		iHhea, ok := tagTableIndex["hhea"]
		if !ok {
			return nil, fmt.Errorf("hmtx: hhea table must be defined in order to rebuild hmtx table")
		}
		var err error
		tables[iHmtx].data, err = reconstructHmtx(tables[iHmtx].data, tables[iHead].data, tables[iGlyf].data, tables[iLoca].data, tables[iMaxp].data, tables[iHhea].data)
		if err != nil {
			return nil, err
		}
	}

	// set checkSumAdjustment to zero to enable calculation of table checksum and overal checksum
	// also clear 11th bit in flags field
	iHead, hasHead := tagTableIndex["head"]
	if !hasHead || len(tables[iHead].data) < 18 {
		return nil, fmt.Errorf("head: must be present")
	}
	binary.BigEndian.PutUint32(tables[iHead].data[8:], 0x00000000) // clear checkSumAdjustment
	if flags := binary.BigEndian.Uint16(tables[iHead].data[16:]); flags&0x0800 == 0 {
		return nil, fmt.Errorf("head: bit 11 in flags must be set")
	} else if _, hasDSIG := tagTableIndex["DSIG"]; hasDSIG {
		return nil, fmt.Errorf("DSIG: must be removed")
	}

	// find values for offset table
	var searchRange uint16 = 1
	var entrySelector uint16
	var rangeShift uint16
	for {
		if numTables < searchRange*2 {
			break
		}
		searchRange *= 2
		entrySelector++
	}
	searchRange *= 16
	rangeShift = numTables*16 - searchRange

	// write offset table
	if MaxMemory < totalSfntSize {
		return nil, ErrExceedsMemory
	}
	w := parse.NewBinaryWriter(make([]byte, 0, totalSfntSize)) // initial guess, will be bigger
	w.WriteUint32(flavor)
	w.WriteUint16(numTables)
	w.WriteUint16(searchRange)
	w.WriteUint16(entrySelector)
	w.WriteUint16(rangeShift)

	// write table record entries, sorted alphabetically
	sort.Strings(tags)
	sfntOffset := 12 + 16*uint32(numTables) // can never exceed uint32 as numTables is uint16
	for _, tag := range tags {
		i := tagTableIndex[tag]
		actualLength := uint32(len(tables[i].data))

		// add padding
		nPadding := (4 - actualLength&3) & 3
		if math.MaxUint32-actualLength < nPadding || math.MaxUint32-actualLength-nPadding < sfntOffset {
			// both actualLength and sfntOffset can overflow, check for both
			return nil, ErrInvalidFontData
		}
		for j := 0; j < int(nPadding); j++ {
			tables[i].data = append(tables[i].data, 0x00)
		}

		w.WriteUint32(binary.BigEndian.Uint32([]byte(tables[i].tag)))
		w.WriteUint32(calcChecksum(tables[i].data))
		w.WriteUint32(sfntOffset)
		w.WriteUint32(actualLength)
		sfntOffset += uint32(len(tables[i].data))
	}

	// write tables
	var iCheckSumAdjustment uint32
	for _, tag := range tags {
		if tag == "head" {
			iCheckSumAdjustment = uint32(w.Len()) + 8
		}
		table := tables[tagTableIndex[tag]]
		w.WriteBytes(table.data)
	}

	buf := w.Bytes()
	checkSumAdjustment := 0xB1B0AFBA - calcChecksum(buf)
	binary.BigEndian.PutUint32(buf[iCheckSumAdjustment:], checkSumAdjustment)
	return buf, nil
}

func signInt16(flag byte, pos uint) int16 {
	if flag&(1<<pos) != 0 {
		return 1 // positive if bit on position is set
	}
	return -1
}

// Remarkable! This code was written on a Sunday evening, and after fixing the compiler errors it worked flawlessly!
// Edit: oops, there was actually a subtle bug fixed in dx of flag < 120 of simple glyphs.
func reconstructGlyfLoca(b []byte, origLocaLength uint32) ([]byte, []byte, error) {
	r := parse.NewBinaryReaderBytes(b)
	_ = r.ReadUint16() // version
	optionFlags := r.ReadUint16()
	numGlyphs := r.ReadUint16()
	indexFormat := r.ReadUint16()
	nContourStreamSize := r.ReadUint32()
	nPointsStreamSize := r.ReadUint32()
	flagStreamSize := r.ReadUint32()
	glyphStreamSize := r.ReadUint32()
	compositeStreamSize := r.ReadUint32()
	bboxStreamSize := r.ReadUint32()
	instructionStreamSize := r.ReadUint32()
	if r.Err() == io.EOF || nContourStreamSize != 2*uint32(numGlyphs) {
		return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
	}

	bitmapSize := ((uint32(numGlyphs) + 31) >> 5) << 2
	nContourStream := parse.NewBinaryReaderBytes(r.ReadBytes(int64(nContourStreamSize)))
	nPointsStream := parse.NewBinaryReaderBytes(r.ReadBytes(int64(nPointsStreamSize)))
	flagStream := parse.NewBinaryReaderBytes(r.ReadBytes(int64(flagStreamSize)))
	glyphStream := parse.NewBinaryReaderBytes(r.ReadBytes(int64(glyphStreamSize)))
	compositeStream := parse.NewBinaryReaderBytes(r.ReadBytes(int64(compositeStreamSize)))
	bboxBitmap := parse.NewBitmapReader(r.ReadBytes(int64(bitmapSize)))
	bboxStream := parse.NewBinaryReaderBytes(r.ReadBytes(int64(bboxStreamSize - bitmapSize)))
	instructionStream := parse.NewBinaryReaderBytes(r.ReadBytes(int64(instructionStreamSize)))
	var overlapSimpleBitmap *parse.BitmapReader
	if optionFlags&0x0001 != 0 { // overlapSimpleBitmap present
		// overlapSimpleBitmap is a packed bit array of numGlyphs bits with no 4-byte padding
		overlapBitmapSize := (uint32(numGlyphs) + 7) >> 3
		overlapSimpleBitmap = parse.NewBitmapReader(r.ReadBytes(int64(overlapBitmapSize)))
	}
	if r.Err() == io.EOF {
		return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
	}

	locaLength := (uint32(numGlyphs) + 1) * 2
	if indexFormat != 0 {
		locaLength *= 2
	}
	if locaLength != origLocaLength {
		return nil, nil, fmt.Errorf("loca: origLength must match numGlyphs+1 entries")
	}

	w := parse.NewBinaryWriter([]byte{}) // size unknown
	loca := parse.NewBinaryWriter(make([]byte, 0, locaLength))
	for iGlyph := uint16(0); iGlyph < numGlyphs; iGlyph++ {
		if indexFormat == 0 {
			// short loca stores offset/2 as uint16, so glyf must fit in 0xFFFE*2 bytes,
			// longer tables must use indexFormat 1
			if int64(0xFFFE*2) < w.Len() {
				return nil, nil, fmt.Errorf("glyf: short loca exceeded: %w", ErrInvalidFontData)
			}
			loca.WriteUint16(uint16(w.Len() >> 1))
		} else {
			loca.WriteUint32(uint32(w.Len()))
		}

		explicitBbox := bboxBitmap.Read()       // EOF cannot occur
		nContours := nContourStream.ReadInt16() // EOF cannot occur
		if nContours == 0 {                     // empty glyph
			if explicitBbox {
				return nil, nil, fmt.Errorf("glyf: empty glyph cannot have bbox definition")
			}
			continue
		} else if 0 < nContours { // simple glyph
			var xMin, yMin, xMax, yMax int16
			if explicitBbox {
				xMin = bboxStream.ReadInt16()
				yMin = bboxStream.ReadInt16()
				xMax = bboxStream.ReadInt16()
				yMax = bboxStream.ReadInt16()
				if bboxStream.Err() == io.EOF {
					return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
				}
			}

			var nPoints uint16
			endPtsOfContours := make([]uint16, nContours)
			for iContour := int16(0); iContour < nContours; iContour++ {
				nPoint := read255Uint16(nPointsStream)
				if math.MaxUint16-nPoints < nPoint {
					return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
				}
				nPoints += nPoint
				// the first contour must contribute at least one point,
				// otherwise nPoints-1 below would underflow
				if nPoints == 0 {
					return nil, nil, fmt.Errorf("glyf: leading contour has no points: %w", ErrInvalidFontData)
				}
				endPtsOfContours[iContour] = nPoints - 1
			}
			if nPointsStream.Err() == io.EOF {
				return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
			}

			var x, y int16
			outlineFlags := make([]byte, 0, nPoints)
			xCoordinates := make([]int16, 0, nPoints)
			yCoordinates := make([]int16, 0, nPoints)
			for iPoint := uint16(0); iPoint < nPoints; iPoint++ {
				flag := flagStream.ReadUint8()
				onCurve := (flag & 0x80) == 0
				flag &= 0x7f

				// used for reference: https://github.com/fonttools/fonttools/blob/master/Lib/fontTools/ttLib/woff2.py
				// as well as: https://github.com/google/woff2/blob/master/src/woff2_dec.cc
				var dx, dy int16
				if flag < 10 {
					coord0 := int16(glyphStream.ReadUint8())
					dy = signInt16(flag, 0) * (int16(flag&0x0E)<<7 + coord0)
				} else if flag < 20 {
					coord0 := int16(glyphStream.ReadUint8())
					dx = signInt16(flag, 0) * (int16((flag-10)&0x0E)<<7 + coord0)
				} else if flag < 84 {
					coord0 := int16(glyphStream.ReadUint8())
					dx = signInt16(flag, 0) * (1 + int16((flag-20)&0x30) + coord0>>4)
					dy = signInt16(flag, 1) * (1 + int16((flag-20)&0x0C)<<2 + (coord0 & 0x0F))
				} else if flag < 120 {
					coord0 := int16(glyphStream.ReadUint8())
					coord1 := int16(glyphStream.ReadUint8())
					dx = signInt16(flag, 0) * (1 + int16((flag-84)/12)<<8 + coord0)
					dy = signInt16(flag, 1) * (1 + (int16((flag-84)%12)>>2)<<8 + coord1)
				} else if flag < 124 {
					coord0 := int16(glyphStream.ReadUint8())
					coord1 := int16(glyphStream.ReadUint8())
					coord2 := int16(glyphStream.ReadUint8())
					dx = signInt16(flag, 0) * (coord0<<4 + coord1>>4)
					dy = signInt16(flag, 1) * ((coord1&0x0F)<<8 + coord2)
				} else {
					coord0 := int16(glyphStream.ReadUint8())
					coord1 := int16(glyphStream.ReadUint8())
					coord2 := int16(glyphStream.ReadUint8())
					coord3 := int16(glyphStream.ReadUint8())
					dx = signInt16(flag, 0) * (coord0<<8 + coord1)
					dy = signInt16(flag, 1) * (coord2<<8 + coord3)
				}
				xCoordinates = append(xCoordinates, dx)
				yCoordinates = append(yCoordinates, dy)

				// keep bit 1-5 zero, that means x and y are two bytes long, this flag is not
				// repeated, and all coordinates are in the coordinate array even if they are the
				// same as the previous.
				var outlineFlag byte
				if onCurve {
					outlineFlag |= 0x01 // ON_CURVE_POINT
				}
				if overlapSimpleBitmap != nil && overlapSimpleBitmap.Read() {
					outlineFlag |= 0x40 // OVERLAP_SIMPLE
				}
				outlineFlags = append(outlineFlags, outlineFlag)

				// calculate bbox
				if !explicitBbox {
					if 0 < x && math.MaxInt16-x < dx || x < 0 && dx < math.MinInt16-x ||
						0 < y && math.MaxInt16-y < dy || y < 0 && dy < math.MinInt16-y {
						return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
					}
					x += dx
					y += dy
					if iPoint == 0 {
						xMin, xMax = x, x
						yMin, yMax = y, y
					} else {
						if x < xMin {
							xMin = x
						} else if xMax < x {
							xMax = x
						}
						if y < yMin {
							yMin = y
						} else if yMax < y {
							yMax = y
						}
					}
				}
			}
			if flagStream.Err() == io.EOF || glyphStream.Err() == io.EOF {
				return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
			}

			instructionLength := read255Uint16(glyphStream)
			instructions := instructionStream.ReadBytes(int64(instructionLength))
			if instructionStream.Err() == io.EOF {
				return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
			}

			// write simple glyph definition
			w.WriteInt16(nContours) // numberOfContours
			w.WriteInt16(xMin)
			w.WriteInt16(yMin)
			w.WriteInt16(xMax)
			w.WriteInt16(yMax)
			for _, endPtsOfContour := range endPtsOfContours {
				w.WriteUint16(endPtsOfContour)
			}
			w.WriteUint16(instructionLength)
			w.WriteBytes(instructions)

			// compact coordinate encoding: a delta of 0 uses *_IS_SAME (no byte),
			// a delta in [-255,255] uses *_SHORT_VECTOR plus the IS_POSITIVE bit
			// (1 byte), larger deltas fall back to a signed int16, otherwise
			// every coordinate would take two bytes and could push glyf past
			// the short-loca byte cap
			for i, dx := range xCoordinates {
				dy := yCoordinates[i]
				flag := outlineFlags[i]
				if dx == 0 {
					flag |= 0x10 // X_IS_SAME
				} else if -255 <= dx && dx <= 255 {
					flag |= 0x02 // X_SHORT_VECTOR
					if 0 < dx {
						flag |= 0x10 // positive
					}
				}
				if dy == 0 {
					flag |= 0x20 // Y_IS_SAME
				} else if -255 <= dy && dy <= 255 {
					flag |= 0x04 // Y_SHORT_VECTOR
					if 0 < dy {
						flag |= 0x20 // positive
					}
				}
				outlineFlags[i] = flag
			}
			for _, flag := range outlineFlags {
				w.WriteByte(flag)
			}
			for i, dx := range xCoordinates {
				flag := outlineFlags[i]
				if flag&0x02 != 0 {
					if dx < 0 {
						dx = -dx
					}
					w.WriteByte(byte(dx))
				} else if flag&0x10 == 0 {
					w.WriteInt16(dx)
				}
			}
			for i, dy := range yCoordinates {
				flag := outlineFlags[i]
				if flag&0x04 != 0 {
					if dy < 0 {
						dy = -dy
					}
					w.WriteByte(byte(dy))
				} else if flag&0x20 == 0 {
					w.WriteInt16(dy)
				}
			}
		} else { // composite glyph
			if !explicitBbox {
				return nil, nil, fmt.Errorf("glyf: composite glyph must have bbox definition")
			}

			xMin := bboxStream.ReadInt16()
			yMin := bboxStream.ReadInt16()
			xMax := bboxStream.ReadInt16()
			yMax := bboxStream.ReadInt16()
			if bboxStream.Err() == io.EOF {
				return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
			}

			// write composite glyph definition
			w.WriteInt16(nContours) // numberOfContours
			w.WriteInt16(xMin)
			w.WriteInt16(yMin)
			w.WriteInt16(xMax)
			w.WriteInt16(yMax)

			hasInstructions := false
			for {
				compositeFlag := compositeStream.ReadUint16()
				argsAreWords := (compositeFlag & 0x0001) != 0
				haveScale := (compositeFlag & 0x0008) != 0
				moreComponents := (compositeFlag & 0x0020) != 0
				haveXYScales := (compositeFlag & 0x0040) != 0
				have2by2 := (compositeFlag & 0x0080) != 0
				haveInstructions := (compositeFlag & 0x0100) != 0

				numBytes := 4 // 2 for glyphIndex and 2 for XY bytes
				if argsAreWords {
					numBytes += 2
				}
				if haveScale {
					numBytes += 2
				} else if haveXYScales {
					numBytes += 4
				} else if have2by2 {
					numBytes += 8
				}
				compositeBytes := compositeStream.ReadBytes(int64(numBytes))
				if compositeStream.Err() == io.EOF {
					return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
				}

				w.WriteUint16(compositeFlag)
				w.WriteBytes(compositeBytes)

				if haveInstructions {
					hasInstructions = true
				}
				if !moreComponents {
					break
				}
			}

			if hasInstructions {
				instructionLength := read255Uint16(glyphStream)
				instructions := instructionStream.ReadBytes(int64(instructionLength))
				if instructionStream.Err() == io.EOF {
					return nil, nil, fmt.Errorf("glyf: %w", ErrInvalidFontData)
				}
				w.WriteUint16(instructionLength)
				w.WriteBytes(instructions)
			}
		}

		// align glyphs to the loca offset granularity: 2 bytes for short, 4 for long
		align := int64(2)
		if indexFormat != 0 {
			align = 4
		}
		for w.Len()%align != 0 {
			w.WriteByte(0x00)
		}
	}

	// last entry in loca table
	if indexFormat == 0 {
		if int64(0xFFFE*2) < w.Len() {
			return nil, nil, fmt.Errorf("glyf: short loca exceeded: %w", ErrInvalidFontData)
		}
		loca.WriteUint16(uint16(w.Len() >> 1))
	} else {
		loca.WriteUint32(uint32(w.Len()))
	}
	return w.Bytes(), loca.Bytes(), nil
}

func reconstructHmtx(b, head, glyf, loca, maxp, hhea []byte) ([]byte, error) {
	// get indexFormat
	rHead := parse.NewBinaryReaderBytes(head)
	_ = rHead.ReadBytes(50) // skip
	indexFormat := rHead.ReadInt16()
	if rHead.Err() == io.EOF {
		return nil, ErrInvalidFontData
	}

	// get numGlyphs
	rMaxp := parse.NewBinaryReaderBytes(maxp)
	_ = rMaxp.ReadUint32() // version
	numGlyphs := rMaxp.ReadUint16()
	if rMaxp.Err() == io.EOF {
		return nil, ErrInvalidFontData
	}

	// get numHMetrics
	rHhea := parse.NewBinaryReaderBytes(hhea)
	_ = rHhea.ReadBytes(34) // skip all but the last header field
	numHMetrics := rHhea.ReadUint16()
	if rHhea.Err() == io.EOF {
		return nil, ErrInvalidFontData
	} else if numHMetrics < 1 {
		return nil, fmt.Errorf("hmtx: must have at least one entry")
	} else if numGlyphs < numHMetrics {
		return nil, fmt.Errorf("hmtx: more entries than glyphs in glyf")
	}

	// check loca table
	locaLength := (uint32(numGlyphs) + 1) * 2
	if indexFormat != 0 {
		locaLength *= 2
	}
	if locaLength != uint32(len(loca)) {
		return nil, ErrInvalidFontData
	}
	rLoca := parse.NewBinaryReaderBytes(loca)

	r := parse.NewBinaryReaderBytes(b)
	flags := r.ReadUint8() // flags
	reconstructProportional := flags&0x01 != 0
	reconstructMonospaced := flags&0x02 != 0
	if flags&0xFC != 0 {
		return nil, fmt.Errorf("hmtx: reserved bits in flags must not be set")
	} else if !reconstructProportional && !reconstructMonospaced {
		return nil, fmt.Errorf("hmtx: must reconstruct at least one left side bearing array")
	}

	n := 1 + uint32(numHMetrics)*2
	if !reconstructProportional {
		n += uint32(numHMetrics) * 2
	} else if !reconstructMonospaced {
		n += (uint32(numGlyphs) - uint32(numHMetrics)) * 2
	}
	if n != uint32(len(b)) {
		return nil, ErrInvalidFontData
	}

	advanceWidths := make([]uint16, numHMetrics)
	lsbs := make([]int16, numGlyphs)
	for iHMetric := uint16(0); iHMetric < numHMetrics; iHMetric++ {
		advanceWidths[iHMetric] = r.ReadUint16()
	}
	if !reconstructProportional {
		for iHMetric := uint16(0); iHMetric < numHMetrics; iHMetric++ {
			lsbs[iHMetric] = r.ReadInt16()
		}
	}
	if !reconstructMonospaced {
		for iLeftSideBearing := numHMetrics; iLeftSideBearing < numGlyphs; iLeftSideBearing++ {
			lsbs[iLeftSideBearing] = r.ReadInt16()
		}
	}

	// extract xMin values from glyf table using loca indices
	rGlyf := parse.NewBinaryReaderBytes(glyf)
	iGlyphMin := uint16(0)
	iGlyphMax := numGlyphs
	if !reconstructProportional {
		iGlyphMin = numHMetrics
		if indexFormat != 0 {
			_ = rLoca.ReadBytes(4 * int64(iGlyphMin))
		} else {
			_ = rLoca.ReadBytes(2 * int64(iGlyphMin))
		}
	} else if !reconstructMonospaced {
		iGlyphMax = numHMetrics
	}
	var offset, offsetNext uint32
	if indexFormat != 0 {
		offset = rLoca.ReadUint32()
	} else {
		offset = uint32(rLoca.ReadUint16()) << 1
	}
	for iGlyph := iGlyphMin; iGlyph < iGlyphMax; iGlyph++ {
		if indexFormat != 0 {
			offsetNext = rLoca.ReadUint32()
		} else {
			offsetNext = uint32(rLoca.ReadUint16()) << 1
		}

		if offsetNext == offset {
			lsbs[iGlyph] = 0
		} else {
			rGlyf.Seek(int64(offset), 0)
			_ = rGlyf.ReadInt16() // numContours
			xMin := rGlyf.ReadInt16()
			if rGlyf.Err() == io.EOF {
				return nil, ErrInvalidFontData
			}
			lsbs[iGlyph] = xMin
		}
		offset = offsetNext
	}

	w := parse.NewBinaryWriter(make([]byte, 0, 2*numGlyphs+2*numHMetrics))
	for iHMetric := uint16(0); iHMetric < numHMetrics; iHMetric++ {
		w.WriteUint16(advanceWidths[iHMetric])
		w.WriteInt16(lsbs[iHMetric])
	}
	for iLeftSideBearing := numHMetrics; iLeftSideBearing < numGlyphs; iLeftSideBearing++ {
		w.WriteInt16(lsbs[iLeftSideBearing])
	}
	return w.Bytes(), nil
}

func readUintBase128(r *parse.BinaryReader) (uint32, error) {
	// see https://www.w3.org/TR/WOFF2/#DataTypes
	var accum uint32
	for i := 0; i < 5; i++ {
		dataByte := r.ReadUint8()
		if r.Err() == io.EOF {
			return 0, ErrInvalidFontData
		}
		if i == 0 && dataByte == 0x80 {
			return 0, fmt.Errorf("readUintBase128: must not start with leading zeros")
		}
		if (accum & 0xFE000000) != 0 {
			return 0, fmt.Errorf("readUintBase128: overflow")
		}
		accum = (accum << 7) | uint32(dataByte&0x7F)
		if (dataByte & 0x80) == 0 {
			return accum, nil
		}
	}
	return 0, fmt.Errorf("readUintBase128: exceeds 5 bytes")
}

func read255Uint16(r *parse.BinaryReader) uint16 {
	// see https://www.w3.org/TR/WOFF2/#DataTypes
	code := r.ReadUint8()
	if code == 253 {
		return r.ReadUint16()
	} else if code == 255 {
		return uint16(r.ReadUint8()) + 253
	} else if code == 254 {
		return uint16(r.ReadUint8()) + 253*2
	} else {
		return uint16(code)
	}
}

func (sfnt *SFNT) WriteWOFF2() ([]byte, error) {
	tags := make([]string, 0, len(sfnt.Tables))
	for tag := range sfnt.Tables {
		if tag == "DSIG" {
			continue // exclude DSIG table
		}
		tags = append(tags, tag)
	}
	// TODO: (WOFF2) loca must follow glyf for TTC
	sort.Strings(tags)

	w := parse.NewBinaryWriter(make([]byte, 0, sfnt.Length*6/10)) // estimated size
	w.WriteString("wOF2")                                         // signature
	w.WriteString(sfnt.Version)                                   // flavor
	w.WriteUint32(0)                                              // length (set later)
	w.WriteUint16(uint16(len(tags)))                              // numTables
	w.WriteUint16(0)                                              // reserved
	w.WriteUint32(sfnt.Length)                                    // totalSfntSize
	w.WriteUint32(0)                                              // totalCompressedSize (set later)
	w.WriteUint16(1)                                              // majorVersion
	w.WriteUint16(0)                                              // minorVersion
	w.WriteUint32(0)                                              // metaOffset
	w.WriteUint32(0)                                              // metaLength
	w.WriteUint32(0)                                              // metaOrigLength
	w.WriteUint32(0)                                              // privOffset
	w.WriteUint32(0)                                              // privLength

	var glyf, hmtx []byte
	_, hasGlyf := sfnt.Tables["glyf"]
	_, hasLoca := sfnt.Tables["loca"]
	_, hasHmtx := sfnt.Tables["hmtx"]
	if hasGlyf && hasLoca {
		var xMins []int16
		glyf, xMins = transformGlyf(sfnt.NumGlyphs(), sfnt.Glyf, sfnt.Loca)
		if glyf != nil && hasHmtx {
			hmtx = transformHmtx(sfnt.Hmtx, xMins)
		}
	}
	for _, tag := range tags {
		tagIndex := -1
		for index, woff2Tag := range woff2TableTags {
			if woff2Tag == tag {
				tagIndex = index
				break
			}
		}

		transformVersion := 0
		if glyf == nil && (tag == "glyf" || tag == "loca") {
			transformVersion = 3
		} else if hmtx != nil && tag == "hmtx" {
			transformVersion = 1
		}
		w.WriteUint8(byte(transformVersion)<<6 | byte(tagIndex)&0x3F) // flags
		if tagIndex == -1 {
			w.WriteString(tag) // tag
		}
		writeUintBase128(w, uint32(len(sfnt.Tables[tag])))
		if glyf != nil && tag == "glyf" {
			writeUintBase128(w, uint32(len(glyf)))
		} else if glyf != nil && tag == "loca" {
			writeUintBase128(w, 0)
		} else if hmtx != nil && tag == "hmtx" {
			writeUintBase128(w, uint32(len(hmtx)))
		}
	}

	if sfnt.Version == "ttcf" {
		// TODO: (WOFF2) support TTC
		w.WriteUint32(0)     // version
		write255Uint16(w, 0) // numFonts
	}

	headerLength := w.Len()
	wBrotli := brotli.NewWriter(w)
	for _, tag := range tags {
		table := sfnt.Tables[tag]
		if tag == "head" {
			head := make([]byte, len(table))
			copy(head, table)
			flags := binary.BigEndian.Uint16(head[16:])
			flags |= 0x0800 // set bit 11, font is compressed
			binary.BigEndian.PutUint16(head[16:], flags)
			table = head
		} else if glyf != nil && tag == "glyf" {
			table = glyf
		} else if glyf != nil && tag == "loca" {
			continue
		} else if hmtx != nil && tag == "hmtx" {
			table = hmtx
		}
		if _, err := wBrotli.Write(table); err != nil {
			return nil, err
		}
	}
	if err := wBrotli.Close(); err != nil {
		return nil, err
	}

	// pad to 4-byte boundary
	// apparently not in the specification, but required by at least Firefox
	totalCompressedSize := w.Len() - headerLength // should not include null bytes (see https://github.com/fontforge/fontforge/issues/5101#issuecomment-1414201810)
	padding := (4 - w.Len()&3) & 3
	for i := 0; i < int(padding); i++ {
		w.WriteByte(0)
	}

	b := w.Bytes()
	binary.BigEndian.PutUint32(b[8:], uint32(len(b)))               // length
	binary.BigEndian.PutUint32(b[20:], uint32(totalCompressedSize)) // totalCompressedSize
	return b, nil
}

func transformGlyf(numGlyphs uint16, glyf *glyfTable, loca *locaTable) ([]byte, []int16) {
	bitmapSize := ((uint32(numGlyphs) + 31) >> 5) << 2
	nContourStream := parse.NewBinaryWriter([]byte{})
	nPointsStream := parse.NewBinaryWriter([]byte{})
	flagStream := parse.NewBinaryWriter([]byte{})
	glyphStream := parse.NewBinaryWriter([]byte{})
	compositeStream := parse.NewBinaryWriter([]byte{})
	bboxBitmapStream := parse.NewBitmapWriter(make([]byte, bitmapSize))
	bboxStream := parse.NewBinaryWriter([]byte{})
	instructionStream := parse.NewBinaryWriter([]byte{})
	overlapSimpleStream := parse.NewBitmapWriter(make([]byte, bitmapSize))

	var optionFlags uint16
	xMins := make([]int16, numGlyphs)
	for glyphID := 0; glyphID < int(numGlyphs); glyphID++ {
		// composite glyphs have already been reconstructed as simple glyphs
		bboxEqual := false
		hasOverlap := false
		var xMin, yMin, xMax, yMax int16
		if !glyf.IsComposite(uint16(glyphID)) {
			// simple glyph
			contour, err := glyf.Contour(uint16(glyphID))
			if err != nil {
				return nil, nil
			} else if len(contour.EndPoints) == 0 {
				// empty glyph
				nContourStream.WriteInt16(0)
				bboxBitmapStream.Write(false)
				continue
			}
			xMins[glyphID] = contour.XMin

			// nContour and nPoints streams
			nContourStream.WriteInt16(int16(len(contour.EndPoints)))
			for i, endPoint := range contour.EndPoints {
				if 0 < i {
					endPoint -= contour.EndPoints[i-1]
				} else {
					endPoint++
				}
				write255Uint16(nPointsStream, endPoint)
			}

			// glyph, flag, and overlapSimple streams
			for i := range contour.XCoordinates {
				dx, dy := contour.XCoordinates[i], contour.YCoordinates[i]
				if 0 < i {
					dx -= contour.XCoordinates[i-1]
					dy -= contour.YCoordinates[i-1]
				}
				dxSign, dySign := byte(1), byte(1)
				if dx < 0 {
					dxSign = 0
					dx = -dx
				}
				if dy < 0 {
					dySign = 0
					dy = -dy
				}

				var flag byte
				if dx == 0 && dy < 1280 {
					// also dx==0 and dy==0
					delta := dy >> 8
					flag = byte(delta<<1) + dySign
					glyphStream.WriteByte(byte(dy - (delta << 8)))
				} else if dx < 1280 && dy == 0 {
					delta := dx >> 8
					flag = 10 + byte(delta<<1) + dxSign
					glyphStream.WriteByte(byte(dx - (delta << 8)))
				} else if dx < 65 && dy < 65 {
					deltax := (dx - 1) >> 4
					deltay := (dy - 1) >> 4
					flag = 20 + byte(deltax<<4) + byte(deltay<<2) + (dySign << 1) + dxSign
					glyphStream.WriteByte(byte(dx-1-(deltax<<4))<<4 | byte(dy-1-(deltay<<4)))
				} else if dx < 769 && dy < 769 {
					deltax := (dx - 1) >> 8
					deltay := (dy - 1) >> 8
					flag = 84 + byte(deltax<<2)*3 + byte(deltay<<2) + (dySign << 1) + dxSign
					glyphStream.WriteByte(byte(dx - 1 - (deltax << 8)))
					glyphStream.WriteByte(byte(dy - 1 - (deltay << 8)))
				} else if dx < 4096 && dy < 4096 {
					flag = 120 + (dySign << 1) + dxSign
					glyphStream.WriteByte(byte(dx & 0x0FF0 >> 4))
					glyphStream.WriteByte(byte(dx&0x000F)<<4 | byte(dy&0x0F00>>8))
					glyphStream.WriteByte(byte(dy & 0x00FF))
				} else {
					flag = 124 + (dySign << 1) + dxSign
					glyphStream.WriteInt16(dx)
					glyphStream.WriteInt16(dy)
				}
				if dxSign == 0 {
					dx = -dx
				}
				if dySign == 0 {
					dy = -dy
				}

				if !contour.OnCurve[i] {
					flag |= 0x80
				}
				flagStream.WriteByte(flag)
				if contour.OverlapSimple[i] {
					hasOverlap = true
					optionFlags |= 0x01
				}
			}

			// bbox streams
			xMin, xMax = contour.XCoordinates[0], contour.XCoordinates[0]
			yMin, yMax = contour.YCoordinates[0], contour.YCoordinates[0]
			for _, x := range contour.XCoordinates[1:] {
				if x < xMin {
					xMin = x
				}
				if xMax < x {
					xMax = x
				}
			}
			if xMin == contour.XMin && xMax == contour.XMax {
				for _, y := range contour.YCoordinates[1:] {
					if y < yMin {
						yMin = y
					}
					if yMax < y {
						yMax = y
					}
				}
				if yMin == contour.YMin && yMax == contour.YMax {
					bboxEqual = true
				} else {
					xMin, xMax = contour.XMin, contour.XMax
					yMin, yMax = contour.YMin, contour.YMax
				}
			}

			// instruction stream
			write255Uint16(glyphStream, uint16(len(contour.Instructions)))
			instructionStream.WriteBytes(contour.Instructions)
		} else {
			// composite glyph
			r := parse.NewBinaryReaderBytes(glyf.Get(uint16(glyphID)))
			_ = r.ReadInt16() // numberOfContours
			xMin = r.ReadInt16()
			yMin = r.ReadInt16()
			xMax = r.ReadInt16()
			yMax = r.ReadInt16()

			nContourStream.WriteInt16(-1)

			hasInstructions := false
			for {
				flags := r.ReadUint16()
				length, more := glyfCompositeLength(flags)
				if flags&0x0100 != 0 {
					hasInstructions = true
				}

				compositeStream.WriteUint16(flags)
				compositeStream.WriteBytes(r.ReadBytes(int64(length) - 2))
				if !more {
					break
				}
			}
			if hasInstructions {
				instructionLength := r.ReadUint16()
				write255Uint16(glyphStream, instructionLength)
				glyphStream.WriteBytes(r.ReadBytes(int64(instructionLength)))
			}
		}

		bboxBitmapStream.Write(!bboxEqual)
		if !bboxEqual {
			bboxStream.WriteInt16(xMin)
			bboxStream.WriteInt16(yMin)
			bboxStream.WriteInt16(xMax)
			bboxStream.WriteInt16(yMax)
		}

		overlapSimpleStream.Write(hasOverlap)
	}

	n := int64(36)
	n += nContourStream.Len() + nPointsStream.Len()
	n += flagStream.Len() + glyphStream.Len() + compositeStream.Len()
	n += bboxBitmapStream.Len() + bboxStream.Len() + instructionStream.Len()
	if optionFlags&0x01 != 0 {
		n += overlapSimpleStream.Len()
	}
	w := parse.NewBinaryWriter(make([]byte, 0, n))
	w.WriteUint16(0) // reserved
	w.WriteUint16(optionFlags)
	w.WriteUint16(numGlyphs)
	w.WriteUint16(uint16(loca.Format))
	w.WriteUint32(uint32(nContourStream.Len()))
	w.WriteUint32(uint32(nPointsStream.Len()))
	w.WriteUint32(uint32(flagStream.Len()))
	w.WriteUint32(uint32(glyphStream.Len()))
	w.WriteUint32(uint32(compositeStream.Len()))
	w.WriteUint32(uint32(bboxBitmapStream.Len() + bboxStream.Len()))
	w.WriteUint32(uint32(instructionStream.Len()))
	w.WriteBytes(nContourStream.Bytes())
	w.WriteBytes(nPointsStream.Bytes())
	w.WriteBytes(flagStream.Bytes())
	w.WriteBytes(glyphStream.Bytes())
	w.WriteBytes(compositeStream.Bytes())
	w.WriteBytes(bboxBitmapStream.Bytes())
	w.WriteBytes(bboxStream.Bytes())
	w.WriteBytes(instructionStream.Bytes())
	if optionFlags&0x01 != 0 {
		w.WriteBytes(overlapSimpleStream.Bytes())
	}
	return w.Bytes(), xMins
}

func transformHmtx(hmtx *hmtxTable, xMins []int16) []byte {
	if len(xMins) != len(hmtx.HMetrics)+len(hmtx.LeftSideBearings) {
		return nil
	}

	omitLSBs, omitLeftSideBearings := true, true
	for i, hmetrics := range hmtx.HMetrics {
		if hmetrics.LeftSideBearing != xMins[i] {
			omitLSBs = false
			break
		}
	}
	for i, leftSideBearing := range hmtx.LeftSideBearings {
		if leftSideBearing != xMins[+len(hmtx.HMetrics)+i] {
			omitLeftSideBearings = false
			break
		}
	}
	if !omitLSBs && !omitLeftSideBearings {
		return nil
	}

	var flags byte
	n := 1 + len(hmtx.HMetrics)*2
	if !omitLSBs {
		n += len(hmtx.HMetrics) * 2
	} else {
		flags |= 0x01
	}
	if !omitLeftSideBearings {
		n += len(hmtx.LeftSideBearings) * 2
	} else {
		flags |= 0x02
	}

	w := parse.NewBinaryWriter(make([]byte, 0, n))
	w.WriteUint8(flags)
	for _, hmetrics := range hmtx.HMetrics {
		w.WriteUint16(hmetrics.AdvanceWidth)
	}
	if !omitLSBs {
		for _, hmetrics := range hmtx.HMetrics {
			w.WriteInt16(hmetrics.LeftSideBearing)
		}
	}
	if !omitLeftSideBearings {
		for _, leftSideBearing := range hmtx.LeftSideBearings {
			w.WriteInt16(leftSideBearing)
		}
	}
	return w.Bytes()
}

func writeUintBase128(w *parse.BinaryWriter, accum uint32) {
	// see https://www.w3.org/TR/WOFF2/#DataTypes
	if accum == 0 {
		w.WriteByte(0)
	}
	written := false
	for i := 4; 0 <= i; i-- {
		mask := uint32(0x7F) << (i * 7)
		if v := accum & mask; written || v != 0 {
			v >>= i * 7
			if i != 0 {
				v |= 0x80
			}
			w.WriteByte(byte(v))
			written = true
		}
	}
}

func write255Uint16(w *parse.BinaryWriter, val uint16) {
	// see https://www.w3.org/TR/WOFF2/#DataTypes
	if val < 253 {
		w.WriteByte(byte(val))
	} else if val < 256+253 {
		w.WriteByte(255)
		w.WriteByte(byte(val - 253))
	} else if val < 256+253*2 {
		w.WriteByte(254)
		w.WriteByte(byte(val - 253*2))
	} else {
		w.WriteByte(253)
		w.WriteUint16(val)
	}
}
