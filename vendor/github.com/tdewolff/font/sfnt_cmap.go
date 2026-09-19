package font

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/tdewolff/parse/v2"
)

type cmapFormat0 struct {
	GlyphIdArray [256]uint8

	unicodeMap map[uint16][]rune
	once       sync.Once
}

func (subtable *cmapFormat0) Get(r rune) (uint16, bool) {
	if r < 0 || 256 <= r {
		return 0, false
	}
	return uint16(subtable.GlyphIdArray[r]), true
}

func (subtable *cmapFormat0) ToUnicode(glyphID uint16) []rune {
	if 256 <= glyphID {
		return nil
	}
	subtable.once.Do(func() {
		subtable.unicodeMap = make(map[uint16][]rune, 256)
		for r, id := range subtable.GlyphIdArray {
			subtable.unicodeMap[uint16(id)] = append(subtable.unicodeMap[uint16(id)], rune(r))
		}
	})
	return subtable.unicodeMap[glyphID]
}

type cmapFormat4 struct {
	StartCode     []uint16
	EndCode       []uint16
	IdDelta       []int16
	IdRangeOffset []uint16
	GlyphIdArray  []uint16

	unicodeMap map[uint16][]rune
	once       sync.Once
}

func (subtable *cmapFormat4) Get(r rune) (uint16, bool) {
	if r < 0 || 65536 <= r {
		return 0, false
	}
	n := len(subtable.StartCode)
	for i := 0; i < n; i++ {
		if subtable.StartCode[i] <= uint16(r) && uint16(r) <= subtable.EndCode[i] {
			if subtable.IdRangeOffset[i] == 0 {
				// is modulo 65536 with the idDelta cast and addition overflow
				return uint16(subtable.IdDelta[i]) + uint16(r), true
			}
			// idRangeOffset/2  ->  offset value to index of words
			// r-startCode  ->  difference of rune with startCode
			// -(n-1)  ->  subtract offset from the current idRangeOffset item
			index := int(subtable.IdRangeOffset[i]/2) + int(uint16(r)-subtable.StartCode[i]) - (n - i)
			return subtable.GlyphIdArray[index], true // index is always valid
		}
	}
	return 0, false
}

func (subtable *cmapFormat4) ToUnicode(glyphID uint16) []rune {
	subtable.once.Do(func() {
		subtable.unicodeMap = map[uint16][]rune{}
		n := len(subtable.StartCode)
		for i := 0; i < n; i++ {
			for r := rune(subtable.StartCode[i]); r <= rune(subtable.EndCode[i]); r++ {
				var id uint16
				if subtable.IdRangeOffset[i] == 0 {
					// is modulo 65536 with the idDelta cast and addition overflow
					id = uint16(subtable.IdDelta[i]) + uint16(r)
				} else {
					// idRangeOffset/2  ->  offset value to index of words
					// r-startCode  ->  difference of rune with startCode
					// -(n-1)  ->  subtract offset from the current idRangeOffset item
					index := int(subtable.IdRangeOffset[i]/2) + int(uint16(r)-subtable.StartCode[i]) - (n - i)
					id = subtable.GlyphIdArray[index]
				}
				subtable.unicodeMap[id] = append(subtable.unicodeMap[id], r)
			}
		}
	})
	return subtable.unicodeMap[glyphID]
}

type cmapFormat6 struct {
	FirstCode    uint16
	GlyphIdArray []uint16

	unicodeMap map[uint16][]rune
	once       sync.Once
}

func (subtable *cmapFormat6) Get(r rune) (uint16, bool) {
	if r < int32(subtable.FirstCode) || uint32(len(subtable.GlyphIdArray)) <= uint32(r)-uint32(subtable.FirstCode) {
		return 0, false
	}
	return subtable.GlyphIdArray[uint32(r)-uint32(subtable.FirstCode)], true
}

func (subtable *cmapFormat6) ToUnicode(glyphID uint16) []rune {
	subtable.once.Do(func() {
		subtable.unicodeMap = make(map[uint16][]rune, len(subtable.GlyphIdArray))
		for i, id := range subtable.GlyphIdArray {
			r := rune(subtable.FirstCode) + rune(i)
			subtable.unicodeMap[id] = append(subtable.unicodeMap[id], r)
		}
	})
	return subtable.unicodeMap[glyphID]
}

type cmapFormat12 struct {
	StartCharCode []uint32
	EndCharCode   []uint32
	StartGlyphID  []uint32

	unicodeMap map[uint16][]rune
	once       sync.Once
}

func (subtable *cmapFormat12) Get(r rune) (uint16, bool) {
	if r < 0 {
		return 0, false
	}
	for i := 0; i < len(subtable.StartCharCode); i++ {
		if subtable.StartCharCode[i] <= uint32(r) && uint32(r) <= subtable.EndCharCode[i] {
			return uint16((uint32(r) - subtable.StartCharCode[i]) + subtable.StartGlyphID[i]), true
		}
	}
	return 0, false
}

func (subtable *cmapFormat12) ToUnicode(glyphID uint16) []rune {
	subtable.once.Do(func() {
		subtable.unicodeMap = map[uint16][]rune{}
		for i := 0; i < len(subtable.StartCharCode); i++ {
			for r := subtable.StartCharCode[i]; r <= subtable.EndCharCode[i]; r++ {
				id := uint16((r - subtable.StartCharCode[i]) + subtable.StartGlyphID[i])
				subtable.unicodeMap[id] = append(subtable.unicodeMap[id], rune(r))
			}
		}
	})
	return subtable.unicodeMap[glyphID]
}

type cmapFormat14 struct {
	glyphIDMap map[rune]uint16

	unicodeMap map[uint16][]rune
	once       sync.Once
}

func (subtable *cmapFormat14) Get(r rune) (uint16, bool) {
	glyphID, ok := subtable.glyphIDMap[r]
	return glyphID, ok
}

func (subtable *cmapFormat14) ToUnicode(glyphID uint16) []rune {
	subtable.once.Do(func() {
		subtable.unicodeMap = make(map[uint16][]rune, len(subtable.glyphIDMap))
		for r, id := range subtable.glyphIDMap {
			subtable.unicodeMap[id] = append(subtable.unicodeMap[id], r)
		}
	})
	return subtable.unicodeMap[glyphID]
}

func cmapWriteFormat4(w *parse.BinaryWriter, rs []rune, runeMap map[rune]uint16) {
	// rs is sorted
	data := cmapFormat4{}
	addSegment := func(firstCode, lastCode rune, glyphIDs []uint16, contiguous bool) {
		data.EndCode = append(data.EndCode, uint16(lastCode))
		data.StartCode = append(data.StartCode, uint16(firstCode))
		if contiguous {
			// use idDelta
			firstGlyph := glyphIDs[0]
			delta := int(firstGlyph) - int(firstCode)
			if math.MaxInt16 < delta {
				delta -= 65536
			} else if delta < math.MinInt16 {
				delta += 65536
			}
			data.IdDelta = append(data.IdDelta, int16(delta))
			data.IdRangeOffset = append(data.IdRangeOffset, 0)
		} else {
			// use idRangeOffset
			// set the value of IdRangeOffset to the offset in GlyphIdArray, updated below
			data.IdDelta = append(data.IdDelta, 0)
			data.IdRangeOffset = append(data.IdRangeOffset, uint16(len(data.GlyphIdArray)))
			data.GlyphIdArray = append(data.GlyphIdArray, glyphIDs...)
		}
	}

	i0 := 0
	glyphIDs := []uint16{runeMap[rs[0]]}
	for i := 1; i <= len(rs); i++ {
		if i < len(rs) && rs[i] == rs[i-1] {
			continue
		} else if i == len(rs) || rs[i-1]+1 != rs[i] {
			// Find subsets of glyphIDs that are contiguous for at least 9 glyphs in a row.
			// Track index before which is already written as segment (j0) and track index
			// before which glyph indices are not contiguous (jc). Write a new segments whenever we
			// encounter a non-contiguous glyph ID (and writing a separate segment for the
			// contiguous glyph IDs beforehand is worthwhile) or end-of-array. We have a stretch
			// of non-contiguous glyph IDs followed by contiguous, one may be empty.
			j0, jc := 0, 0
			for j := 1; j <= len(glyphIDs); j++ {
				if j == len(glyphIDs) || glyphIDs[j-1]+1 != glyphIDs[j] && 8 < j-jc {
					if 8 < j-jc && j0 != jc {
						addSegment(rs[i0+j0], rs[i0+(jc-1)], glyphIDs[j0:jc], false)
						addSegment(rs[i0+jc], rs[i0+(j-1)], glyphIDs[jc:j], true)
						j0, jc = j, j
					} else if j == len(glyphIDs) {
						addSegment(rs[i0+j0], rs[i0+(j-1)], glyphIDs[j0:j], j0 == jc)
					}
					if j == len(glyphIDs) {
						break
					}
				} else if glyphIDs[j-1]+1 != glyphIDs[j] {
					jc = j
				}
			}
			if i == len(rs) {
				break
			}
			glyphIDs = glyphIDs[:0]
			i0 = i
		}
		glyphIDs = append(glyphIDs, runeMap[rs[i]])
	}
	if rs[len(rs)-1] != 0xFFFF {
		addSegment(0xFFFF, 0xFFFF, []uint16{0}, true) // map to .notdef
	}

	start := w.Len()
	w.WriteUint16(4) // format
	w.WriteUint16(0) // length (set later)
	w.WriteUint16(0) // language

	segCount := uint16(len(data.StartCode))
	searchRange := uint16(math.Exp2(math.Floor(math.Log2(float64(segCount)))))
	entrySelector := uint16(math.Log2(float64(searchRange)))
	w.WriteUint16(segCount * 2)                 // segCountX2
	w.WriteUint16(searchRange * 2)              // searchRange
	w.WriteUint16(entrySelector)                // entrySelector
	w.WriteUint16((segCount - searchRange) * 2) // rangeShift

	for _, endCode := range data.EndCode {
		w.WriteUint16(endCode)
	}
	w.WriteUint16(0) // reservedPad
	for _, startCode := range data.StartCode {
		w.WriteUint16(startCode)
	}
	for _, idDelta := range data.IdDelta {
		w.WriteInt16(idDelta)
	}
	for i, idRangeOffset := range data.IdRangeOffset {
		if data.IdDelta[i] != 0 {
			w.WriteUint16(0)
		} else {
			glyphIdArrayStart := uint16(len(data.IdRangeOffset) - i)
			w.WriteUint16((glyphIdArrayStart + idRangeOffset) * 2) // times 2 since entries are 16 bit
		}
	}
	for _, glyphID := range data.GlyphIdArray {
		w.WriteUint16(glyphID)
	}

	// TODO: subtable may exceed the range of the length field (16 bits, or 65536 bytes long)
	binary.BigEndian.PutUint16(w.Bytes()[start+2:], uint16(w.Len()-start)) // set length
}

func cmapWriteFormat12(w *parse.BinaryWriter, rs []rune, runeMap map[rune]uint16) {
	start := w.Len()
	w.WriteUint16(12) // format
	w.WriteUint16(0)  // reserved
	w.WriteUint32(0)  // length (set later)
	w.WriteUint32(0)  // language
	w.WriteUint32(0)  // numGroups (set later)

	numGroups := uint32(1)
	startCharCode := uint32(rs[0])
	startGlyphID := uint32(runeMap[rs[0]])
	n := uint32(1)
	for i := 1; i < len(rs); i++ {
		r := rs[i]
		subsetGlyphID := runeMap[r]
		if r == rs[i-1] {
			continue
		} else if uint32(r) == startCharCode+n && uint32(subsetGlyphID) == startGlyphID+n {
			n++
		} else {
			w.WriteUint32(uint32(startCharCode))         // startCharCode
			w.WriteUint32(uint32(startCharCode + n - 1)) // endCharCode
			w.WriteUint32(uint32(startGlyphID))          // startGlyphID
			numGroups++
			startCharCode = uint32(r)
			startGlyphID = uint32(subsetGlyphID)
			n = 1
		}
	}
	w.WriteUint32(uint32(startCharCode))         // startCharCode
	w.WriteUint32(uint32(startCharCode + n - 1)) // endCharCode
	w.WriteUint32(uint32(startGlyphID))          // startGlyphID

	binary.BigEndian.PutUint32(w.Bytes()[start+4:], uint32(w.Len()-start)) // set length
	binary.BigEndian.PutUint32(w.Bytes()[start+12:], numGroups)            // set numGroups
}

func cmapWrite(rs []rune, runeMap map[rune]uint16) []byte {
	if len(rs) == 0 {
		return []byte{0x00, 0x00, 0x00, 0x00}
	}

	sort.Slice(rs, func(i, j int) bool { return rs[i] < rs[j] })

	// TODO: it may be more efficient to have several tables of different formats
	w := parse.NewBinaryWriter([]byte{})
	w.WriteUint16(0) // version
	w.WriteUint16(2) // numTables, we specify 2 encodings for the same subtable

	var maxRune rune
	for _, r := range rs {
		if maxRune < r {
			maxRune = r
		}
	}
	if maxRune <= 0xffff {
		w.WriteUint16(0)  // platformID
		w.WriteUint16(3)  // encodingID
		w.WriteUint32(20) // subtableOffset
		w.WriteUint16(3)  // platformID
		w.WriteUint16(1)  // encodingID
		w.WriteUint32(20) // subtableOffset
		cmapWriteFormat4(w, rs, runeMap)
	} else {
		w.WriteUint16(0)  // platformID
		w.WriteUint16(4)  // encodingID
		w.WriteUint32(20) // subtableOffset
		w.WriteUint16(3)  // platformID
		w.WriteUint16(10) // encodingID
		w.WriteUint32(20) // subtableOffset
		cmapWriteFormat12(w, rs, runeMap)
	}
	return w.Bytes()
}

type cmapEncodingRecord struct {
	PlatformID uint16
	EncodingID uint16
	Format     uint16
	Subtable   uint16
}

type cmapSubtable interface {
	Get(rune) (uint16, bool)
	ToUnicode(uint16) []rune
}

type cmapTable struct {
	EncodingRecords []cmapEncodingRecord
	Subtables       []cmapSubtable
}

// Get returns the glyph ID for the corresponding rune. It looks for each subtable in the order in which they appear and returns the first match, or 0 when no match is found.
func (cmap *cmapTable) Get(r rune) uint16 {
	for _, subtable := range cmap.Subtables {
		if glyphID, ok := subtable.Get(r); ok {
			return glyphID
		}
	}
	return 0
}

// ToUnicode returns all runes for the corresponding glyph ID. It looks for each subtable in the order in which they appear and returns referenced runes for the glyph ID, or nil if none.
func (cmap *cmapTable) ToUnicode(glyphID uint16) []rune {
	var rs []rune
	for _, subtable := range cmap.Subtables {
	AppendLoop:
		for _, r := range subtable.ToUnicode(glyphID) {
			for _, r2 := range rs {
				if r2 == r {
					continue AppendLoop
				}
			}
			rs = append(rs, r)
		}
	}
	return rs
}

func (sfnt *SFNT) parseCmap() error {
	if sfnt.Maxp == nil {
		return fmt.Errorf("cmap: missing maxp table")
	}

	b, ok := sfnt.Tables["cmap"]
	if !ok {
		return fmt.Errorf("cmap: missing table")
	} else if len(b) < 4 {
		return fmt.Errorf("cmap: bad table")
	}

	sfnt.Cmap = &cmapTable{}
	r := parse.NewBinaryReaderBytes(b)
	if r.ReadUint16() != 0 {
		return fmt.Errorf("cmap: bad version")
	}
	numTables := r.ReadUint16()
	if uint32(len(b)) < 4+8*uint32(numTables) {
		return fmt.Errorf("cmap: bad table")
	}

	// find and extract subtables and make sure they don't overlap each other
	offsets, lengths := []uint32{0}, []uint32{4 + 8*uint32(numTables)}
	for j := 0; j < int(numTables); j++ {
		platformID := r.ReadUint16()
		encodingID := r.ReadUint16()
		subtableID := -1

		offset := r.ReadUint32()
		if uint32(len(b))-8 < offset { // to extract the subtable format and length
			return fmt.Errorf("cmap: bad subtable %d", j)
		}
		for i := 0; i < len(offsets); i++ {
			if offsets[i] < offset && offset < lengths[i] {
				return fmt.Errorf("cmap: bad subtable %d", j)
			}
		}

		// extract subtable length
		rs := parse.NewBinaryReaderBytes(b[offset:])
		format := rs.ReadUint16()
		var length uint32
		if format == 0 || format == 2 || format == 4 || format == 6 {
			length = uint32(rs.ReadUint16())
		} else if format == 8 || format == 10 || format == 12 || format == 13 {
			_ = rs.ReadUint16() // reserved
			length = rs.ReadUint32()
		} else if format == 14 {
			length = rs.ReadUint32()
		} else {
			return fmt.Errorf("cmap: bad format %d for subtable %d", format, j)
		}
		if length < 8 || math.MaxUint32-offset < length {
			return fmt.Errorf("cmap: bad subtable %d", j)
		}
		for i := 0; i < len(offsets); i++ {
			if offset == offsets[i] && length == lengths[i] {
				subtableID = int(i)
				break
			} else if offset <= offsets[i] && offsets[i] < offset+length {
				return fmt.Errorf("cmap: bad subtable %d", j)
			}
		}
		rs = parse.NewBinaryReaderBytes(b[offset+uint32(rs.Pos()) : offset+length])

		if subtableID == -1 {
			subtableID = len(sfnt.Cmap.Subtables)
			offsets = append(offsets, offset)
			lengths = append(lengths, length)

			switch format {
			case 0:
				if rs.Len() < 258 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				_ = rs.ReadUint16() // languageID

				subtable := &cmapFormat0{}
				copy(subtable.GlyphIdArray[:], rs.ReadBytes(256))
				for _, glyphID := range subtable.GlyphIdArray {
					if sfnt.Maxp.NumGlyphs <= uint16(glyphID) {
						return fmt.Errorf("cmap: bad glyphID in subtable %d", j)
					}
				}
				sfnt.Cmap.Subtables = append(sfnt.Cmap.Subtables, subtable)
			case 4:
				if rs.Len() < 10 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				_ = rs.ReadUint16() // languageID

				segCount := rs.ReadUint16()
				if segCount%2 != 0 || segCount == 0 {
					return fmt.Errorf("cmap: bad segCount in subtable %d", j)
				}
				segCount /= 2
				if MaxCmapSegments < segCount {
					return fmt.Errorf("cmap: too many segments in subtable %d", j)
				}
				_ = rs.ReadUint16() // searchRange
				_ = rs.ReadUint16() // entrySelector
				_ = rs.ReadUint16() // rangeShift

				subtable := &cmapFormat4{}
				if rs.Len() < 2+8*int64(segCount) {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				subtable.EndCode = make([]uint16, segCount)
				for i := 0; i < int(segCount); i++ {
					endCode := rs.ReadUint16()
					if 0 < i && endCode <= subtable.EndCode[i-1] {
						return fmt.Errorf("cmap: bad endCode in subtable %d", j)
					}
					subtable.EndCode[i] = endCode
				}
				_ = rs.ReadUint16() // reservedPad
				subtable.StartCode = make([]uint16, segCount)
				for i := 0; i < int(segCount); i++ {
					startCode := rs.ReadUint16()
					if subtable.EndCode[i] < startCode || 0 < i && startCode <= subtable.EndCode[i-1] {
						return fmt.Errorf("cmap: bad startCode in subtable %d", j)
					}
					subtable.StartCode[i] = startCode
				}
				if subtable.StartCode[segCount-1] != 0xFFFF || subtable.EndCode[segCount-1] != 0xFFFF {
					return fmt.Errorf("cmap: bad last startCode or endCode in subtable %d", j)
				}

				subtable.IdDelta = make([]int16, segCount)
				for i := 0; i < int(segCount-1); i++ {
					subtable.IdDelta[i] = rs.ReadInt16()
				}
				_ = rs.ReadUint16() // last value may be invalid
				subtable.IdDelta[segCount-1] = 1

				glyphIdArrayLength := int(rs.Len() - 2*int64(segCount))
				if glyphIdArrayLength%2 != 0 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				glyphIdArrayLength /= 2

				subtable.IdRangeOffset = make([]uint16, segCount)
				for i := 0; i < int(segCount-1); i++ {
					idRangeOffset := rs.ReadUint16()
					if idRangeOffset%2 != 0 {
						return fmt.Errorf("cmap: bad idRangeOffset in subtable %d", j)
					} else if idRangeOffset != 0 {
						index := int(idRangeOffset/2) + int(subtable.EndCode[i]-subtable.StartCode[i]) - (int(segCount) - i)
						if index < 0 || glyphIdArrayLength <= index {
							return fmt.Errorf("cmap: bad idRangeOffset in subtable %d", j)
						}
					}
					subtable.IdRangeOffset[i] = idRangeOffset
				}
				_ = rs.ReadUint16() // last value may be invalid
				subtable.IdRangeOffset[segCount-1] = 0

				subtable.GlyphIdArray = make([]uint16, glyphIdArrayLength)
				for i := 0; i < int(glyphIdArrayLength); i++ {
					glyphID := rs.ReadUint16()
					if sfnt.Maxp.NumGlyphs <= glyphID {
						return fmt.Errorf("cmap: bad glyphID in subtable %d", j)
					}
					subtable.GlyphIdArray[i] = glyphID
				}
				sfnt.Cmap.Subtables = append(sfnt.Cmap.Subtables, subtable)
			case 6:
				if rs.Len() < 6 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				_ = rs.ReadUint16() // language

				subtable := &cmapFormat6{}
				subtable.FirstCode = rs.ReadUint16()
				entryCount := rs.ReadUint16()
				if rs.Len() < 2*int64(entryCount) {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				subtable.GlyphIdArray = make([]uint16, entryCount)
				for i := 0; i < int(entryCount); i++ {
					subtable.GlyphIdArray[i] = rs.ReadUint16()
				}
				sfnt.Cmap.Subtables = append(sfnt.Cmap.Subtables, subtable)
			case 12:
				if rs.Len() < 8 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				_ = rs.ReadUint32() // language
				numGroups := rs.ReadUint32()
				if MaxCmapSegments < numGroups {
					return fmt.Errorf("cmap: too many segments in subtable %d", j)
				} else if rs.Len() < 12*int64(numGroups) {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}

				subtable := &cmapFormat12{}
				subtable.StartCharCode = make([]uint32, numGroups)
				subtable.EndCharCode = make([]uint32, numGroups)
				subtable.StartGlyphID = make([]uint32, numGroups)
				for i := 0; i < int(numGroups); i++ {
					startCharCode := rs.ReadUint32()
					endCharCode := rs.ReadUint32()
					startGlyphID := rs.ReadUint32()
					if endCharCode < startCharCode || 0 < i && startCharCode <= subtable.EndCharCode[i-1] {
						return fmt.Errorf("cmap: bad character code range in subtable %d", j)
					} else if uint32(sfnt.Maxp.NumGlyphs) <= endCharCode-startCharCode || uint32(sfnt.Maxp.NumGlyphs)-(endCharCode-startCharCode) <= startGlyphID {
						return fmt.Errorf("cmap: bad glyphID in subtable %d", j)
					}
					subtable.StartCharCode[i] = startCharCode
					subtable.EndCharCode[i] = endCharCode
					subtable.StartGlyphID[i] = startGlyphID
				}
				sfnt.Cmap.Subtables = append(sfnt.Cmap.Subtables, subtable)
			case 14:
				if platformID != 0 || encodingID != 5 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				// TODO: implement cmap subtable format 14
				//numVarSelectorRecords := rs.ReadUint32()
				//for i := 0; i < int(numVarSelectorRecords); i++ {
				//	varSelector := rs.ReadUint24()
				//	_ = rs.ReadUint32() // defaultUVSOffset
				//	nonDefaultUVSOffset := rs.ReadUint32()
				//	if nonDefaultUVSOffset != 0 {
				//		if length < nonDefaultUVSOffset+4 {
				//			return fmt.Errorf("cmap: bad subtable %d", j)
				//		}
				//		pos := rs.Pos()
				//		rs.Seek(nonDefaultUVSOffset)
				//		numUVSMappings := rs.ReadUint32()
				//		if length < nonDefaultUVSOffset+4+numUVSMappings*4 {
				//			return fmt.Errorf("cmap: bad subtable %d", j)
				//		}
				//		for j := 0; j < int(numUVSMappings); j++ {
				//			unicodeValue := rs.ReadUint24()
				//			glyphID := rs.ReadUint16()
				//			_ = varSelector
				//			_ = unicodeValue
				//			_ = glyphID
				//		}
				//		rs.Seek(pos)
				//	}
				//}
			default:
				return fmt.Errorf("cmap: unsupported subtable format %d", format)
			}
		}
		sfnt.Cmap.EncodingRecords = append(sfnt.Cmap.EncodingRecords, cmapEncodingRecord{
			PlatformID: platformID,
			EncodingID: encodingID,
			Format:     format,
			Subtable:   uint16(subtableID),
		})
	}
	return nil
}
