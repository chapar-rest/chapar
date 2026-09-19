package font

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"github.com/tdewolff/parse/v2"
)

var (
	KeepAllTables = []string{"all"}
	KeepMinTables = []string{"min"}
	KeepPDFTables = []string{"pdf"}
)

type SubsetOptions struct {
	Tables []string // names of tables to include, special values are KeepAllTables, KeepMinTables, or KeepPDFTables
}

// Subset trims an SFNT font to contain only the passed glyphIDs, thereby resulting in a significant size reduction. The glyphIDs will apear in the specified order in the file and their dependencies are added to the end.
func (sfnt *SFNT) Subset(glyphIDs []uint16, options SubsetOptions) (*SFNT, error) {
	if sfnt.IsCFF {
		if _, ok := sfnt.Tables["CFF2"]; ok {
			// TODO: support subsetting CFF2
			return nil, fmt.Errorf("subsetting CFF2 fonts not supported")
		}
	}

	// set up glyph mapping from original to subset
	glyphMap := make(map[uint16]uint16, len(glyphIDs))
	for subsetGlyphID, glyphID := range glyphIDs {
		glyphMap[glyphID] = uint16(subsetGlyphID)
	}

	if sfnt.IsTrueType {
		// add dependencies for composite glyphs add the end
		origLen := len(glyphIDs)
		for i := 0; i < origLen; i++ {
			if glyphIDs[i] == 0 {
				continue
			}
			deps, err := sfnt.Glyf.Dependencies(glyphIDs[i])
			if err != nil {
				return nil, err
			}
			for _, glyphID := range deps[1:] {
				if _, ok := glyphMap[glyphID]; !ok {
					glyphMap[glyphID] = uint16(len(glyphIDs))
					glyphIDs = append(glyphIDs, glyphID)
				}
			}
		}
	}
	if math.MaxUint16 < len(glyphIDs) {
		return nil, fmt.Errorf("too many glyphs for one font")
	}

	// specify tables to include
	var tags []string
	if len(options.Tables) == 1 && options.Tables[0] == "min" {
		tags = []string{"DSIG", "cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post"}
		if sfnt.IsTrueType {
			tags = append(tags, "glyf", "loca")
		} else if sfnt.IsCFF {
			if _, ok := sfnt.Tables["CFF2"]; ok {
				tags = append(tags, "CFF2")
			} else {
				tags = append(tags, "CFF ")
			}
		}
	} else if len(options.Tables) == 1 && options.Tables[0] == "pdf" {
		// good summary at http://www.4real.gr/technical-documents-ttf-subset.html
		if sfnt.IsTrueType {
			tags = append(tags, "DSIG", "glyf", "head", "hhea", "hmtx", "loca", "maxp")
			for _, tag := range []string{"cvt ", "fpgm", "prep"} {
				if _, ok := sfnt.Tables[tag]; ok {
					tags = append(tags, tag)
				}
			}
		} else if sfnt.IsCFF {
			// head and maxp tables are needed for almost all viewers
			// hhea and hmtx tables are needed for Evince (and possibly other viewers)
			// DSIG is recommended by Microsoft's FontValidator
			if _, ok := sfnt.Tables["CFF2"]; ok {
				tags = append(tags, "DSIG", "cmap", "CFF2", "head", "hhea", "hmtx", "maxp") // not strictly allowed
			} else {
				tags = append(tags, "DSIG", "cmap", "CFF ", "head", "hhea", "hmtx", "maxp")
			}
		}
	} else if len(options.Tables) == 1 && options.Tables[0] == "all" {
		for tag := range sfnt.Tables {
			tags = append(tags, tag)
		}
	} else {
		tags = options.Tables
	}
	sort.Strings(tags) // so that glyf is before loca

	// preliminary calculations
	indexToLocFormat := int16(1)                   // for head and loca
	glyfOffsets := make([]uint32, len(glyphIDs)+1) // for loca
	ulUnicodeRange := [4]uint32{0, 0, 0, 0}        // for OS/2
	advances := make([]uint16, len(glyphIDs))      // for hhea and hmtx
	for subsetGlyphID, glyphID := range glyphIDs {
		advances[subsetGlyphID] = sfnt.Hmtx.Advance(glyphID)
	}

	// https://learn.microsoft.com/en-us/typography/opentype/spec/recom#hmtx-table
	// It is recommended that for CFF fonts the numberOfHMetrics must be equal to the number
	// of glyphs, but emperically there is no difference.
	//if !sfnt.IsCFF {
	numberOfHMetrics := uint16(len(glyphIDs)) // for hhea and hmtx
	for 1 < numberOfHMetrics {
		if advances[numberOfHMetrics-1] != advances[numberOfHMetrics-2] {
			break
		}
		numberOfHMetrics--
	}
	//}

	// copy to new SFNT
	sfntOld := sfnt
	sfnt = &SFNT{
		Version:    sfntOld.Version,
		Length:     12, // increased below
		IsCFF:      sfntOld.IsCFF,
		IsTrueType: sfntOld.IsTrueType,
		Tables:     map[string][]byte{},
		Loca:       &locaTable{}, // for glyf
	}

	// copy and rewrite tables
	for _, tag := range tags {
		if tag == "maxp" {
			maxp := *sfntOld.Maxp
			sfnt.Maxp = &maxp
			sfnt.Maxp.NumGlyphs = uint16(len(glyphIDs))
			break
		}
	}
	for _, tag := range tags {
		table, ok := sfntOld.Tables[tag]
		if !ok {
			continue
		}

		switch tag {
		// TODO: GDEF, GPOS, GSUB
		// TODO; cvt, fpgm, prep
		case "cmap":
			rs := make([]rune, 0, len(glyphIDs))
			runeMap := make(map[rune]uint16, len(glyphIDs)) // for OS/2
			for subsetGlyphID, glyphID := range glyphIDs {
				rsGlyph := sfntOld.Cmap.ToUnicode(glyphID)
				rs = append(rs, rsGlyph...)
				for _, r := range rsGlyph {
					runeMap[r] = uint16(subsetGlyphID)
				}
			}
			ulUnicodeRange = os2UlUnicodeRange(rs)

			sfnt.Tables[tag] = cmapWrite(rs, runeMap)
			if err := sfnt.parseCmap(); err != nil {
				return nil, err
			}
		case "CFF ":
			cff := *sfntOld.CFF
			cff.charStrings = &cffINDEX{}
			for _, glyphID := range glyphIDs {
				if glyphID == 0 {
					// make .notdef empty
					cff.charStrings.Add([]byte{0x0E}) // endchar
					continue
				}

				charString := sfntOld.CFF.charStrings.Get(glyphID)
				if charString == nil {
					return nil, fmt.Errorf("CFF: bad charString for glyph %v", glyphID)
				}
				cff.charStrings.Add(charString) // copies data
			}

			if cff.charset != nil {
				charset := make([]string, len(glyphIDs))
				for i, glyphID := range glyphIDs {
					charset[i] = cff.charset[glyphID]
				}
				cff.charset = charset
			}

			// trim globalSubrs and localSubrs INDEX
			if err := cff.ReindexSubrs(); err != nil {
				return nil, err
			}

			b, err := cff.Write()
			if err != nil {
				return nil, err
			}
			sfnt.Tables[tag] = b
			sfnt.CFF = &cff
		case "glyf":
			w := parse.NewBinaryWriter([]byte{})
			for i, glyphID := range glyphIDs {
				if glyphID == 0 {
					// empty .notdef
					glyfOffsets[i+1] = uint32(w.Len())
					continue
				} else if sfntOld.Glyf.IsComposite(glyphID) {
					// composite glyphs, update glyphIDs and make sure not to write to b
					b := sfntOld.Glyf.Get(glyphID)
					start := uint32(w.Len())
					w.WriteBytes(b)

					offset := uint32(10)
					for {
						flags := binary.BigEndian.Uint16(b[offset:])
						subGlyphID := binary.BigEndian.Uint16(b[offset+2:])
						binary.BigEndian.PutUint16(w.Bytes()[start+offset+2:], glyphMap[subGlyphID])

						length, more := glyfCompositeLength(flags)
						if !more {
							break
						}
						offset += length
					}
				} else {
					// simple glyph
					contour, err := sfntOld.Glyf.Contour(glyphID)
					if err != nil {
						// bad glyf data or bug in Contour, write original glyph data
						w.WriteBytes(sfntOld.Glyf.Get(glyphID))
					} else if 0 < len(contour.EndPoints) { // not empty
						// optimize glyph data
						numberOfContours := int16(len(contour.EndPoints))
						w.WriteInt16(numberOfContours)
						w.WriteInt16(contour.XMin)
						w.WriteInt16(contour.YMin)
						w.WriteInt16(contour.XMax)
						w.WriteInt16(contour.YMax)
						for _, endPoint := range contour.EndPoints {
							w.WriteUint16(endPoint)
						}
						w.WriteUint16(uint16(len(contour.Instructions)))
						w.WriteBytes(contour.Instructions)

						repeats := 0
						xs := parse.NewBinaryWriter([]byte{})
						ys := parse.NewBinaryWriter([]byte{})
						numPoints := int(contour.EndPoints[numberOfContours-1]) + 1
						for i := 0; i < numPoints; i++ {
							dx := contour.XCoordinates[i]
							dy := contour.YCoordinates[i]
							if 0 < i {
								dx -= contour.XCoordinates[i-1]
								dy -= contour.YCoordinates[i-1]
							}

							var flag byte
							if dx == 0 {
								flag |= 0x10 // X_IS_SAME_OR_POSITIVE_X_SHORT_VECTOR
							} else if -256 < dx && dx < 256 {
								flag |= 0x02 // X_SHORT_VECTOR
								if 0 < dx {
									flag |= 0x10 // X_IS_SAME_OR_POSITIVE_X_SHORT_VECTOR
									xs.WriteInt8(int8(dx))
								} else {
									xs.WriteInt8(int8(-dx))
								}
							} else {
								xs.WriteInt16(dx)
							}

							if dy == 0 {
								flag |= 0x20 // Y_IS_SAME_OR_POSITIVE_Y_SHORT_VECTOR
							} else if -256 < dy && dy < 256 {
								flag |= 0x04 // Y_SHORT_VECTOR
								if 0 < dy {
									flag |= 0x20 // Y_IS_SAME_OR_POSITIVE_Y_SHORT_VECTOR
									ys.WriteByte(byte(dy))
								} else {
									ys.WriteByte(byte(-dy))
								}
							} else {
								ys.WriteInt16(dy)
							}

							if contour.OnCurve[i] {
								flag |= 0x01
							}
							if contour.OverlapSimple[i] {
								flag |= 0x40
							}

							// handle flag repeats
							if 0 < i && repeats < 255 && flag == w.Bytes()[w.Len()-1] {
								repeats++
							} else {
								if 1 < repeats {
									w.Bytes()[w.Len()-1] |= 0x08 // REPEAT_FLAG
									w.WriteByte(byte(repeats))
									repeats = 0
								} else if repeats == 1 {
									w.WriteByte(w.Bytes()[w.Len()-1])
									repeats = 0
								}
								w.WriteByte(flag)
							}
						}
						if 1 < repeats {
							w.Bytes()[w.Len()-1] |= 0x08 // REPEAT_FLAG
							w.WriteByte(byte(repeats))
						} else if repeats == 1 {
							w.WriteByte(w.Bytes()[w.Len()-1])
						}
						w.WriteBytes(xs.Bytes())
						w.WriteBytes(ys.Bytes())
					}
				}

				// padding to ensure glyph offsets are on even bytes for loca short format
				if w.Len()%2 == 1 {
					w.WriteByte(0)
				}
				glyfOffsets[i+1] = uint32(w.Len())
			}
			if w.Len() <= math.MaxUint16 {
				indexToLocFormat = 0 // short format
			}
			sfnt.Tables[tag] = w.Bytes()

			sfnt.Glyf = &glyfTable{
				data: w.Bytes(),
				loca: sfnt.Loca,
			}
		case "head":
			w := parse.NewBinaryWriter(make([]byte, 0, len(sfntOld.Tables["head"])))
			w.WriteBytes(table[:50])
			w.WriteInt16(indexToLocFormat) // indexToLocFormat
			w.WriteBytes(table[52:])
			sfnt.Tables[tag] = w.Bytes()

			head := *sfntOld.Head
			sfnt.Head = &head
			sfnt.Head.IndexToLocFormat = indexToLocFormat
		case "hhea":
			w := parse.NewBinaryWriter(make([]byte, 0, len(sfntOld.Tables["hhea"])))
			w.WriteBytes(table[:34])
			w.WriteUint16(numberOfHMetrics) // numberOfHMetrics
			w.WriteBytes(table[36:])
			sfnt.Tables[tag] = w.Bytes()

			hhea := *sfntOld.Hhea
			sfnt.Hhea = &hhea
			sfnt.Hhea.NumberOfHMetrics = numberOfHMetrics
		case "hmtx":
			sfnt.Hmtx = &hmtxTable{}
			sfnt.Hmtx.HMetrics = make([]hmtxLongHorMetric, numberOfHMetrics)
			sfnt.Hmtx.LeftSideBearings = make([]int16, len(glyphIDs)-int(numberOfHMetrics))

			n := 4*int(numberOfHMetrics) + 2*(len(glyphIDs)-int(numberOfHMetrics))
			w := parse.NewBinaryWriter(make([]byte, 0, n))
			for subsetGlyphID, glyphID := range glyphIDs {
				lsb := sfntOld.Hmtx.LeftSideBearing(glyphID)
				if subsetGlyphID < int(numberOfHMetrics) {
					sfnt.Hmtx.HMetrics[subsetGlyphID].AdvanceWidth = advances[subsetGlyphID]
					sfnt.Hmtx.HMetrics[subsetGlyphID].LeftSideBearing = lsb
					w.WriteUint16(advances[subsetGlyphID])
				} else {
					sfnt.Hmtx.LeftSideBearings[subsetGlyphID-int(numberOfHMetrics)] = lsb
				}
				w.WriteInt16(lsb)
			}
			sfnt.Tables[tag] = w.Bytes()
		case "kern":
			// handle kern table that could be removed
			kern := kernTable{
				Subtables: []kernFormat0{},
			}
			for _, subtable := range sfntOld.Kern.Subtables {
				pairs := []kernPair{}
				for l, lOrig := range glyphIDs {
					if lOrig == 0 {
						continue
					}
					for r, rOrig := range glyphIDs {
						if rOrig == 0 {
							continue
						}
						if value := subtable.Get(lOrig, rOrig); value != 0 {
							pairs = append(pairs, kernPair{
								Key:   uint32(l)<<16 + uint32(r),
								Value: value,
							})
						}
					}
				}
				if 0 < len(pairs) {
					kern.Subtables = append(kern.Subtables, kernFormat0{
						Coverage: subtable.Coverage,
						Pairs:    pairs,
					})
				}
			}
			if len(kern.Subtables) == 0 {
				continue
			}
			sfnt.Tables[tag] = kern.Write()
			sfnt.Kern = &kern
		case "loca":
			var w *parse.BinaryWriter
			if indexToLocFormat == 0 {
				// short format
				w = parse.NewBinaryWriter(make([]byte, 0, 2*len(glyfOffsets)))
				for _, offset := range glyfOffsets {
					w.WriteUint16(uint16(offset / 2))
				}
			} else {
				// long format
				w = parse.NewBinaryWriter(make([]byte, 0, 4*len(glyfOffsets)))
				for _, offset := range glyfOffsets {
					w.WriteUint32(offset)
				}
			}
			sfnt.Tables[tag] = w.Bytes()

			sfnt.Loca.Format = indexToLocFormat
			sfnt.Loca.data = w.Bytes()
		case "maxp":
			w := parse.NewBinaryWriter(make([]byte, 0, len(sfntOld.Tables["maxp"])))
			w.WriteBytes(table[:4])
			w.WriteUint16(uint16(len(glyphIDs))) // numGlyphs
			w.WriteBytes(table[6:])
			sfnt.Tables[tag] = w.Bytes()
		case "name":
			w := parse.NewBinaryWriter(make([]byte, 0, 6))
			w.WriteUint16(0) // version
			w.WriteUint16(0) // count
			w.WriteUint16(6) // storageOffset
			sfnt.Tables[tag] = w.Bytes()

			sfnt.Name = &nameTable{}
		case "OS/2":
			sfnt.OS2 = sfntOld.OS2
			sfnt.OS2.UlUnicodeRange1 = ulUnicodeRange[0]
			sfnt.OS2.UlUnicodeRange2 = ulUnicodeRange[1]
			sfnt.OS2.UlUnicodeRange3 = ulUnicodeRange[2]
			sfnt.OS2.UlUnicodeRange4 = ulUnicodeRange[3]

			w := parse.NewBinaryWriter(make([]byte, 0, len(sfntOld.Tables["OS/2"])))
			w.WriteBytes(table[:42])
			w.WriteUint32(sfnt.OS2.UlUnicodeRange1)
			w.WriteUint32(sfnt.OS2.UlUnicodeRange2)
			w.WriteUint32(sfnt.OS2.UlUnicodeRange3)
			w.WriteUint32(sfnt.OS2.UlUnicodeRange4)
			w.WriteBytes(table[58:])
			sfnt.Tables[tag] = w.Bytes()
		case "post":
			w := parse.NewBinaryWriter(make([]byte, 0, 32))
			w.WriteUint32(0x00030000) // version
			w.WriteBytes(table[4:32])
			sfnt.Tables[tag] = w.Bytes()

			sfnt.Post = &postTable{
				ItalicAngle:        sfntOld.Post.ItalicAngle,
				UnderlinePosition:  sfntOld.Post.UnderlinePosition,
				UnderlineThickness: sfntOld.Post.UnderlineThickness,
				IsFixedPitch:       sfntOld.Post.IsFixedPitch,
				MinMemType42:       sfntOld.Post.MinMemType42,
				MaxMemType42:       sfntOld.Post.MaxMemType42,
				MinMemType1:        sfntOld.Post.MinMemType1,
				MaxMemType1:        sfntOld.Post.MaxMemType1,
			}
		default:
			sfnt.Tables[tag] = table
		}

		sfnt.Length += uint32(16 + len(sfnt.Tables[tag]))        // 16 for the table record
		sfnt.Length += (4 - uint32(len(sfnt.Tables[tag]))&3) & 3 // padding
	}
	return sfnt, nil
}

func (sfnt *SFNT) SetGlyphNames(names []string) error {
	if sfnt.IsCFF {
		same := false
		if len(sfnt.CFF.charset) == len(names) {
			same = true
			for i, name := range sfnt.CFF.charset {
				if name != names[i] {
					same = false
					break
				}
			}
		}
		if same {
			return nil
		}

		sfnt.CFF.charset = names
		b, err := sfnt.CFF.Write()
		if err != nil {
			return err
		}
		sfnt.Tables["CFF "] = b
		return nil
	}

	if _, ok := sfnt.Tables["post"]; !ok {
		return fmt.Errorf("post table doesn't exist")
	}

	sfnt.Post.NumGlyphs = sfnt.NumGlyphs()
	sfnt.Post.glyphNameIndex = make([]uint16, sfnt.NumGlyphs())
	sfnt.Post.stringData = [][]byte{}
	sfnt.Post.nameMap = nil

	lastIndex := uint16(258)
	if int(sfnt.NumGlyphs()) < len(names) {
		names = names[:sfnt.NumGlyphs()]
	}
	for glyphID, name := range names {
		var index uint16
		if glyphID2, ok := sfnt.Post.Find(name); ok {
			index = sfnt.Post.glyphNameIndex[glyphID2]
		} else if math.MaxUint16 < len(sfnt.Post.stringData)-258 {
			return fmt.Errorf("invalid post table: stringData has too many entries")
		} else {
			sfnt.Post.stringData = append(sfnt.Post.stringData, []byte(name))
			sfnt.Post.nameMap[name] = uint16(glyphID)
			index = lastIndex
			lastIndex++
		}
		sfnt.Post.glyphNameIndex[uint16(glyphID)] = index
	}

	b, err := sfnt.Post.Write()
	if err != nil {
		return err
	}

	sfnt.Tables["post"] = b
	return nil
}
