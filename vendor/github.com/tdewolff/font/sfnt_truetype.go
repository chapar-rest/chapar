package font

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/tdewolff/parse/v2"
)

////////////////////////////////////////////////////////////////

type glyfContour struct {
	GlyphID                uint16
	XMin, YMin, XMax, YMax int16
	EndPoints              []uint16
	Instructions           []byte
	OnCurve                []bool
	OverlapSimple          []bool
	XCoordinates           []int16
	YCoordinates           []int16
}

func (contour *glyfContour) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Glyph %v:\n", contour.GlyphID)
	fmt.Fprintf(&b, "  Contours: %v\n", len(contour.EndPoints))
	fmt.Fprintf(&b, "  XMin: %v\n", contour.XMin)
	fmt.Fprintf(&b, "  YMin: %v\n", contour.YMin)
	fmt.Fprintf(&b, "  XMax: %v\n", contour.XMax)
	fmt.Fprintf(&b, "  YMax: %v\n", contour.YMax)
	fmt.Fprintf(&b, "  EndPoints: %v\n", contour.EndPoints)
	fmt.Fprintf(&b, "  Instruction length: %v\n", len(contour.Instructions))
	if len(contour.EndPoints) == 0 {
		fmt.Fprintf(&b, "  Empty glyph\n")
	} else {
		fmt.Fprintf(&b, "  Coordinates:\n")
		for i := 0; i <= int(contour.EndPoints[len(contour.EndPoints)-1]); i++ {
			fmt.Fprintf(&b, "    ")
			if i < len(contour.XCoordinates) {
				fmt.Fprintf(&b, "%8v", contour.XCoordinates[i])
			} else {
				fmt.Fprintf(&b, "  ----  ")
			}
			if i < len(contour.YCoordinates) {
				fmt.Fprintf(&b, " %8v", contour.YCoordinates[i])
			} else {
				fmt.Fprintf(&b, "   ----  ")
			}
			if i < len(contour.OnCurve) {
				onCurve := "Off"
				if contour.OnCurve[i] {
					onCurve = "On"
				}
				fmt.Fprintf(&b, " %3v\n", onCurve)
			} else {
				fmt.Fprintf(&b, " ---\n")
			}
		}
	}
	return b.String()
}

type glyfTable struct {
	data []byte
	loca *locaTable
}

// Get returns the glyph data corresponding to the passed glyphID. It returns nil if the glyph doesn't exist.
func (glyf *glyfTable) Get(glyphID uint16) []byte {
	start, ok1 := glyf.loca.Get(glyphID)
	end, ok2 := glyf.loca.Get(glyphID + 1)
	if !ok1 || !ok2 {
		return nil
	}
	return glyf.data[start:end]
}

// IsComposite returns true if the glyph is a composite glyph
func (glyf *glyfTable) IsComposite(glyphID uint16) bool {
	b := glyf.Get(glyphID)
	if len(b) < 1 {
		return false
	}
	return b[0]&0x80 != 0 // sign bit is set on numberOfContours
}

// Dependencies returns all the glyph IDs that a composite glyph uses.
func (glyf *glyfTable) Dependencies(glyphID uint16) ([]uint16, error) {
	return glyf.dependencies(glyphID, 0)
}

func (glyf *glyfTable) dependencies(glyphID uint16, level int) ([]uint16, error) {
	deps := []uint16{glyphID}
	b := glyf.Get(glyphID)
	if b == nil {
		return nil, fmt.Errorf("glyf: bad glyphID %v", glyphID)
	} else if len(b) == 0 {
		return deps, nil
	}
	r := parse.NewBinaryReaderBytes(b)
	if r.Len() < 10 {
		return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
	}
	numberOfContours := r.ReadInt16()
	_ = r.ReadBytes(8)
	if numberOfContours < 0 {
		if 7 < level {
			return nil, fmt.Errorf("glyf: compound glyphs too deeply nested")
		}

		// composite glyph
		for {
			if r.Len() < 4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags := r.ReadUint16()
			subGlyphID := r.ReadUint16()
			subDeps, err := glyf.dependencies(subGlyphID, level+1)
			if err != nil {
				return nil, err
			}
			deps = append(deps, subDeps...)

			length, more := glyfCompositeLength(flags)
			if r.Len() < int64(length)-4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}
			_ = r.ReadBytes(int64(length) - 4)
			if !more {
				break
			}
		}
	}
	return deps, nil
}

func glyfCompositeLength(flags uint16) (length uint32, more bool) {
	length = 4 + 2
	if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
		length += 2
	}
	if flags&0x0008 != 0 { // WE_HAVE_A_SCALE
		length += 2
	} else if flags&0x0040 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
		length += 4
	} else if flags&0x0080 != 0 { // WE_HAVE_A_TWO_BY_TWO
		length += 8
	}
	more = flags&0x0020 != 0 // MORE_COMPONENTS
	return
}

// Contour returns the contours of a glyph. It unpacks composite glyphs into their final shape.
func (glyf *glyfTable) Contour(glyphID uint16) (*glyfContour, error) {
	// TODO: cache output
	return glyf.contour(glyphID, 0)
}

func (glyf *glyfTable) contour(glyphID uint16, level int) (*glyfContour, error) {
	b := glyf.Get(glyphID)
	if b == nil {
		return nil, fmt.Errorf("glyf: bad glyphID %v", glyphID)
	} else if len(b) == 0 {
		return &glyfContour{GlyphID: glyphID}, nil
	}
	r := parse.NewBinaryReaderBytes(b)
	if r.Len() < 10 {
		return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
	}

	contour := &glyfContour{}
	contour.GlyphID = glyphID
	numberOfContours := r.ReadInt16()
	contour.XMin = r.ReadInt16()
	contour.YMin = r.ReadInt16()
	contour.XMax = r.ReadInt16()
	contour.YMax = r.ReadInt16()
	if 0 <= numberOfContours {
		// simple glyph
		if r.Len() < 2*int64(numberOfContours)+2 {
			return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
		}
		contour.EndPoints = make([]uint16, numberOfContours)
		for i := 0; i < int(numberOfContours); i++ {
			contour.EndPoints[i] = r.ReadUint16()
		}

		instructionLength := r.ReadUint16()
		if r.Len() < int64(instructionLength) {
			return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
		}
		contour.Instructions = r.ReadBytes(int64(instructionLength))

		numPoints := int(contour.EndPoints[numberOfContours-1]) + 1
		flags := make([]byte, numPoints)
		contour.OnCurve = make([]bool, numPoints)
		contour.OverlapSimple = make([]bool, numPoints)
		for i := 0; i < numPoints; i++ {
			if r.Len() < 1 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags[i] = r.ReadUint8()
			contour.OnCurve[i] = flags[i]&0x01 != 0
			contour.OverlapSimple[i] = flags[i]&0x40 != 0
			if flags[i]&0x08 != 0 { // REPEAT_FLAG
				repeats := r.ReadUint8()
				for j := 1; j <= int(repeats); j++ {
					flags[i+j] = flags[i]
					contour.OnCurve[i+j] = contour.OnCurve[i]
					contour.OverlapSimple[i+j] = contour.OverlapSimple[i]
				}
				i += int(repeats)
			}
		}

		var x int16
		contour.XCoordinates = make([]int16, numPoints)
		for i := 0; i < numPoints; i++ {
			xShortVector := flags[i]&0x02 != 0
			xIsSameOrPositiveXShortVector := flags[i]&0x10 != 0
			if xShortVector {
				if r.Len() < 1 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				if xIsSameOrPositiveXShortVector {
					x += int16(r.ReadUint8())
				} else {
					x -= int16(r.ReadUint8())
				}
			} else if !xIsSameOrPositiveXShortVector {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				x += r.ReadInt16()
			}
			contour.XCoordinates[i] = x
		}

		var y int16
		contour.YCoordinates = make([]int16, numPoints)
		for i := 0; i < numPoints; i++ {
			yShortVector := flags[i]&0x04 != 0
			yIsSameOrPositiveYShortVector := flags[i]&0x20 != 0
			if yShortVector {
				if r.Len() < 1 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				if yIsSameOrPositiveYShortVector {
					y += int16(r.ReadUint8())
				} else {
					y -= int16(r.ReadUint8())
				}
			} else if !yIsSameOrPositiveYShortVector {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				y += r.ReadInt16()
			}
			contour.YCoordinates[i] = y
		}
	} else {
		if 7 < level {
			return nil, fmt.Errorf("glyf: compound glyphs too deeply nested")
		}

		// composite glyph
		hasInstructions := false
		for {
			if r.Len() < 4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags := r.ReadUint16()
			subGlyphID := r.ReadUint16()
			if flags&0x0002 == 0 { // ARGS_ARE_XY_VALUES
				return nil, fmt.Errorf("glyf: composite glyph not supported")
			}
			var dx, dy int16
			if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
				if r.Len() < 4 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				dx = r.ReadInt16()
				dy = r.ReadInt16()
			} else {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				dx = int16(r.ReadInt8())
				dy = int16(r.ReadInt8())
			}
			var txx, txy, tyx, tyy int16
			if flags&0x0008 != 0 { // WE_HAVE_A_SCALE
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				tyy = txx
			} else if flags&0x0040 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
				if r.Len() < 4 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				tyy = r.ReadInt16()
			} else if flags&0x0080 != 0 { // WE_HAVE_A_TWO_BY_TWO
				if r.Len() < 8 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				txy = r.ReadInt16()
				tyx = r.ReadInt16()
				tyy = r.ReadInt16()
			}
			if flags&0x0100 != 0 {
				hasInstructions = true
			}

			subContour, err := glyf.contour(subGlyphID, level+1)
			if err != nil {
				return nil, err
			}

			var numPoints uint16
			if 0 < len(contour.EndPoints) {
				numPoints = contour.EndPoints[len(contour.EndPoints)-1] + 1
			}
			for i := 0; i < len(subContour.EndPoints); i++ {
				contour.EndPoints = append(contour.EndPoints, numPoints+subContour.EndPoints[i])
			}
			contour.OnCurve = append(contour.OnCurve, subContour.OnCurve...)
			contour.OverlapSimple = append(contour.OverlapSimple, subContour.OverlapSimple...)
			for i := 0; i < len(subContour.XCoordinates); i++ {
				x := subContour.XCoordinates[i]
				y := subContour.YCoordinates[i]
				if flags&0x00C8 != 0 { // has transformation
					const half = 1 << 13
					xt := int16((int64(x)*int64(txx)+half)>>14) + int16((int64(y)*int64(tyx)+half)>>14)
					yt := int16((int64(x)*int64(txy)+half)>>14) + int16((int64(y)*int64(tyy)+half)>>14)
					x, y = xt, yt
				}
				contour.XCoordinates = append(contour.XCoordinates, dx+x)
				contour.YCoordinates = append(contour.YCoordinates, dy+y)
			}
			if flags&0x0020 == 0 { // MORE_COMPONENTS
				break
			}
		}
		if hasInstructions {
			instructionLength := r.ReadUint16()
			if r.Len() < int64(instructionLength) {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}
			contour.Instructions = r.ReadBytes(int64(instructionLength))
		}
	}
	return contour, nil
}

func (glyf *glyfTable) ToPath(p Pather, glyphID, ppem uint16, x, y, f float64, hinting Hinting) error {
	contour, err := glyf.Contour(glyphID)
	if err != nil {
		return err
	}

	var i uint16
	for _, endPoint := range contour.EndPoints {
		j := i
		first := true
		firstOff := false
		prevOff := false
		startX, startY := 0.0, 0.0
		for ; i <= endPoint; i++ {
			if first {
				if contour.OnCurve[i] {
					startX = float64(contour.XCoordinates[i])
					startY = float64(contour.YCoordinates[i])
					p.MoveTo(x+f*startX, y+f*startY)
					first = false
				} else if !prevOff {
					// first point is off
					firstOff = true
					prevOff = true
				} else {
					// first and second point are off
					startX = float64(contour.XCoordinates[i-1]+contour.XCoordinates[i]) / 2.0
					startY = float64(contour.YCoordinates[i-1]+contour.YCoordinates[i]) / 2.0
					p.MoveTo(x+f*startX, y+f*startY)
					first = false
				}
			} else if !prevOff {
				if contour.OnCurve[i] {
					p.LineTo(x+f*float64(contour.XCoordinates[i]), y+f*float64(contour.YCoordinates[i]))
				} else {
					prevOff = true
				}
			} else {
				if contour.OnCurve[i] {
					p.QuadTo(x+f*float64(contour.XCoordinates[i-1]), y+f*float64(contour.YCoordinates[i-1]), x+f*float64(contour.XCoordinates[i]), y+f*float64(contour.YCoordinates[i]))
					prevOff = false
				} else {
					midX := float64(contour.XCoordinates[i-1]+contour.XCoordinates[i]) / 2.0
					midY := float64(contour.YCoordinates[i-1]+contour.YCoordinates[i]) / 2.0
					p.QuadTo(x+f*float64(contour.XCoordinates[i-1]), y+f*float64(contour.YCoordinates[i-1]), x+f*midX, y+f*midY)
				}
			}
		}
		if firstOff {
			if prevOff {
				midX := float64(contour.XCoordinates[i-1]+contour.XCoordinates[j]) / 2.0
				midY := float64(contour.YCoordinates[i-1]+contour.YCoordinates[j]) / 2.0
				p.QuadTo(x+f*float64(contour.XCoordinates[i-1]), y+f*float64(contour.YCoordinates[i-1]), x+f*midX, y+f*midY)
				p.QuadTo(x+f*float64(contour.XCoordinates[j]), y+f*float64(contour.YCoordinates[j]), x+f*startX, y+f*startY)
			} else {
				p.QuadTo(x+f*float64(contour.XCoordinates[j]), y+f*float64(contour.YCoordinates[j]), x+f*startX, y+f*startY)
			}
		} else if prevOff {
			p.QuadTo(x+f*float64(contour.XCoordinates[i-1]), y+f*float64(contour.YCoordinates[i-1]), x+f*startX, y+f*startY)
		}
		p.Close()
	}
	return nil
}

func (sfnt *SFNT) parseGlyf() error {
	if sfnt.Loca == nil {
		return fmt.Errorf("glyf: missing loca table")
	} else if sfnt.Maxp == nil {
		return fmt.Errorf("glyf: missing maxp table")
	}

	b, ok := sfnt.Tables["glyf"]
	if !ok {
		return fmt.Errorf("glyf: missing table")
	} else if length, _ := sfnt.Loca.Get(sfnt.Maxp.NumGlyphs); uint32(len(b)) < length {
		return fmt.Errorf("glyf: bad table")
	}

	sfnt.Glyf = &glyfTable{
		data: b,
		loca: sfnt.Loca,
	}
	return nil
}

////////////////////////////////////////////////////////////////

type locaTable struct {
	Format int16
	data   []byte
}

func (loca *locaTable) Get(glyphID uint16) (uint32, bool) {
	if loca.Format == 0 && int(glyphID)*2 < len(loca.data) {
		return 2 * uint32(binary.BigEndian.Uint16(loca.data[int(glyphID)*2:])), true
	} else if loca.Format == 1 && int(glyphID)*4 < len(loca.data) {
		return binary.BigEndian.Uint32(loca.data[int(glyphID)*4:]), true
	}
	return 0, false
}

func (sfnt *SFNT) parseLoca() error {
	if sfnt.Head == nil {
		return fmt.Errorf("loca: missing head table")
	}

	b, ok := sfnt.Tables["loca"]
	if !ok {
		return fmt.Errorf("loca: missing table")
	}

	sfnt.Loca = &locaTable{
		Format: sfnt.Head.IndexToLocFormat,
		data:   b,
	}
	//sfnt.Loca.Offsets = make([]uint32, sfnt.Maxp.NumGlyphs+1)
	//r := parse.NewBinaryReaderBytes(b)
	//if sfnt.Head.IndexToLocFormat == 0 {
	//	if uint32(len(b)) != 2*(uint32(sfnt.Maxp.NumGlyphs)+1) {
	//		return fmt.Errorf("loca: bad table")
	//	}
	//	for i := 0; i < int(sfnt.Maxp.NumGlyphs+1); i++ {
	//		sfnt.Loca.Offsets[i] = uint32(r.ReadUint16())
	//		if 0 < i && sfnt.Loca.Offsets[i] < sfnt.Loca.Offsets[i-1] {
	//			return fmt.Errorf("loca: bad offsets")
	//		}
	//	}
	//} else if sfnt.Head.IndexToLocFormat == 1 {
	//	if uint32(len(b)) != 4*(uint32(sfnt.Maxp.NumGlyphs)+1) {
	//		return fmt.Errorf("loca: bad table")
	//	}
	//	for i := 0; i < int(sfnt.Maxp.NumGlyphs+1); i++ {
	//		sfnt.Loca.Offsets[i] = r.ReadUint32()
	//		if 0 < i && sfnt.Loca.Offsets[i] < sfnt.Loca.Offsets[i-1] {
	//			return fmt.Errorf("loca: bad offsets")
	//		}
	//	}
	//}
	return nil
}

////////////////////////////////////////////////////////////////

type kernPair struct {
	Key   uint32
	Value int16
}

type kernFormat0 struct {
	Coverage [8]bool
	Pairs    []kernPair
}

func (subtable *kernFormat0) Get(l, r uint16) int16 {
	key := uint32(l)<<16 | uint32(r)
	lo, hi := 0, len(subtable.Pairs)
	for lo < hi {
		mid := (lo + hi) / 2 // can be rounded down if odd
		pair := subtable.Pairs[mid]
		if pair.Key < key {
			lo = mid + 1
		} else if key < pair.Key {
			hi = mid
		} else {
			return pair.Value
		}
	}
	return 0
}

type kernTable struct {
	Subtables []kernFormat0
}

func (kern *kernTable) Get(l, r uint16) (k int16) {
	for _, subtable := range kern.Subtables {
		if !subtable.Coverage[1] { // kerning values
			k += subtable.Get(l, r)
		} else if min := subtable.Get(l, r); k < min { // minimum values (usually last subtable)
			k = min // TODO: test minimal kerning
		}
	}
	return
}

func (sfnt *SFNT) parseKern() error {
	// TODO: lazy parse
	b, ok := sfnt.Tables["kern"]
	if !ok {
		return fmt.Errorf("kern: missing table")
	} else if len(b) < 4 {
		return fmt.Errorf("kern: bad table")
	}

	r := parse.NewBinaryReaderBytes(b)
	majorVersion := r.ReadUint16()
	if majorVersion != 0 && majorVersion != 1 {
		return fmt.Errorf("kern: bad version %d", majorVersion)
	}

	var nTables uint32
	if majorVersion == 0 {
		nTables = uint32(r.ReadUint16())
	} else if majorVersion == 1 {
		minorVersion := r.ReadUint16()
		if minorVersion != 0 {
			return fmt.Errorf("kern: bad minor version %d", minorVersion)
		}
		nTables = r.ReadUint32()
	}

	sfnt.Kern = &kernTable{}
	for j := 0; j < int(nTables); j++ {
		if r.Len() < 6 {
			return fmt.Errorf("kern: bad subtable %d", j)
		}

		subtable := kernFormat0{}
		startPos := r.Pos()
		subtableVersion := r.ReadUint16()
		if subtableVersion != 0 {
			// TODO: supported other kern subtable versions
			continue
		}
		length := r.ReadUint16()
		format := r.ReadUint8()
		subtable.Coverage = Uint8ToFlags(r.ReadUint8())
		if format != 0 {
			// TODO: supported other kern subtable formats
			continue
		} else if r.Len() < 8 {
			return fmt.Errorf("kern: bad subtable %d", j)
		}
		nPairs := r.ReadUint16()
		_ = r.ReadUint16() // searchRange
		_ = r.ReadUint16() // entrySelector
		_ = r.ReadUint16() // rangeShift
		if int64(length) < 14+6*int64(nPairs) || r.Len() < int64(length) {
			if j+1 == int(nTables) {
				// for some fonts the subtable's length exceeds what can fit in a uint16
				// we allow only the last subtable to exceed as long as it stays within the table
				pairsLength := 6 * int64(nPairs)
				pairsLength &= 0xFFFF
				if int64(length) != 14+pairsLength || r.Len() < pairsLength {
					return fmt.Errorf("kern: bad length for subtable %d", j)
				}
			} else {
				return fmt.Errorf("kern: bad length for subtable %d", j)
			}
		}

		sorted := true
		subtable.Pairs = make([]kernPair, nPairs)
		for i := 0; i < int(nPairs); i++ {
			subtable.Pairs[i].Key = r.ReadUint32()
			subtable.Pairs[i].Value = r.ReadInt16()
			if 0 < i && subtable.Pairs[i].Key <= subtable.Pairs[i-1].Key {
				sorted = false
			}
		}
		if !sorted {
			// some fonts haven't sorted the pairs, allow those subtables and sort them here
			sort.SliceStable(subtable.Pairs, func(i, j int) bool {
				return subtable.Pairs[i].Key < subtable.Pairs[j].Key
			})
		}

		// read unread bytes if length is bigger
		_ = r.ReadBytes(int64(length) - (r.Pos() - startPos))
		sfnt.Kern.Subtables = append(sfnt.Kern.Subtables, subtable)
	}
	return nil
}

func (kern *kernTable) Write() []byte {
	w := parse.NewBinaryWriter([]byte{})
	w.WriteUint16(0)                           // version
	w.WriteUint16(uint16(len(kern.Subtables))) // nTables
	for _, subtable := range kern.Subtables {
		w.WriteUint16(0)                                     // version
		w.WriteUint16(6 + 8 + 6*uint16(len(subtable.Pairs))) // length
		w.WriteUint8(0)                                      // format
		w.WriteUint8(flagsToUint8(subtable.Coverage))        // coverage

		nPairs := uint16(len(subtable.Pairs))
		entrySelector := uint16(math.Log2(float64(nPairs)))
		searchRange := uint16(1 << entrySelector * 6)
		w.WriteUint16(nPairs)
		w.WriteUint16(searchRange)
		w.WriteUint16(entrySelector)
		w.WriteUint16(nPairs*6 - searchRange)
		for _, pair := range subtable.Pairs {
			w.WriteUint32(pair.Key)
			w.WriteInt16(pair.Value)
		}
	}
	return w.Bytes()
}
