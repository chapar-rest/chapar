package font

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	stdStrconv "strconv"

	"github.com/tdewolff/parse/v2/strconv"
)

type AFMCharMetrics struct {
	CharacterCode int
	Width         uint16
	BBox          [4]int16
	Name          string
}

type AFM struct {
	FontName   string
	FullName   string
	FamilyName string
	Weight     string
	FontBBox   [4]int16

	CapHeight          uint16
	XHeight            uint16
	Ascender           uint16
	Descender          uint16
	UnderlinePosition  uint16
	UnderlineThickness uint16
	ItalicAngle        float64
	CharWidth          [2]uint16
	IsFixedPitch       bool

	CharMetrics []AFMCharMetrics
	Ligatures   map[[2]uint16]uint16
	KernPairs   map[[2]uint16]int16

	names   map[string]uint16
	unicode map[rune]uint16
}

// NumGlyphs returns the number of glyphs the font contains.
func (afm *AFM) NumGlyphs() uint16 {
	return uint16(len(afm.CharMetrics))
}

// GlyphIndex returns the glyphID for a given rune. When the rune is not defined it returns 0.
func (afm *AFM) GlyphIndex(r rune) uint16 {
	return afm.unicode[r]
}

// GlyphName returns the name of the glyph. It returns an empty string when no name exists.
func (afm *AFM) GlyphName(glyphID uint16) string {
	if len(afm.CharMetrics) <= int(glyphID) {
		return ""
	}
	return afm.CharMetrics[glyphID].Name
}

// FindGlyphName returns the glyphID for a given glyph name. When the name is not defined it returns 0.
func (afm *AFM) FindGlyphName(name string) uint16 {
	return afm.names[name]
}

// GlyphAdvance returns the (horizontal) advance width of the glyph.
func (afm *AFM) GlyphAdvance(glyphID uint16) uint16 {
	if afm.IsFixedPitch {
		return afm.CharWidth[0]
	} else if len(afm.CharMetrics) <= int(glyphID) {
		return 0
	}
	return afm.CharMetrics[glyphID].Width
}

// GlyphBounds returns the bounding rectangle (xmin,ymin,xmax,ymax) of the glyph.
func (afm *AFM) GlyphBounds(glyphID uint16) (int16, int16, int16, int16) {
	if len(afm.CharMetrics) <= int(glyphID) {
		return 0, 0, 0, 0
	}
	bbox := afm.CharMetrics[glyphID].BBox
	return bbox[0], bbox[1], bbox[2], bbox[3]
}

// Kerning returns the kerning between two glyphs, i.e. the advance correction for glyph pairs.
func (afm *AFM) Kerning(left, right uint16) int16 {
	return afm.KernPairs[[2]uint16{left, right}]
}

func afmIsWhitespace(c byte) bool {
	return c == ' ' || c == '\t'
}

func afmSkipWhitespace(b []byte, i int) int {
	for i < len(b) && afmIsWhitespace(b[i]) {
		i++
	}
	return i
}

func afmNextValue(b []byte, i int) ([]byte, int) {
	start := i
	for i < len(b) && !afmIsWhitespace(b[i]) {
		i++
	}
	return b[start:i], i
}

func afmParseString(b []byte, i int) (string, int) {
	i = afmSkipWhitespace(b, i)
	var v []byte
	v, i = afmNextValue(b, i)
	return string(v), i
}

func afmParseInteger(b []byte, i int) (int, int) {
	i = afmSkipWhitespace(b, i)
	v, n := strconv.ParseInt(b[i:])
	return int(v), i + n
}

func afmParseNumber(b []byte, i int) (float64, int) {
	i = afmSkipWhitespace(b, i)
	v, n := strconv.ParseDecimal(b[i:])
	return v, i + n
}

type afmLigature struct {
	base      uint16
	successor string
	ligature  string
}

func (afm *AFM) parseCharMetrics(scanner *bufio.Scanner, j *int, n int) error {
	var ligatures []afmLigature
	afm.CharMetrics = make([]AFMCharMetrics, n)
	afm.names = make(map[string]uint16, n)
	afm.unicode = make(map[rune]uint16, n)
	for i := 0; i < n; i++ {
		*j++
		if !scanner.Scan() {
			return fmt.Errorf("afm: invalid char metrics at line %v", *j)
		}
		line := scanner.Bytes()

		var name string
		charMetrics := &afm.CharMetrics[i]
		val, pos := afmNextValue(line, 0)
		switch string(val) {
		case "C":
			charMetrics.CharacterCode, pos = afmParseInteger(line, pos)
		case "CH":
			pos = afmSkipWhitespace(line, pos)
			val, pos = afmNextValue(line, pos)
			if len(val) < 3 || val[0] != '<' || val[len(val)-1] != '>' {
				return fmt.Errorf("afm: invalid char metrics at line %v: expected <hex>", *j)
			} else if v, err := stdStrconv.ParseInt(string(val[1:len(val)-1]), 16, 0); err != nil {
				return fmt.Errorf("afm: invalid char metrics at line %v: %v", *j, err)
			} else {
				charMetrics.CharacterCode = int(v)
			}
		default:
			return fmt.Errorf("afm: invalid char metrics at line %v: unexpected %v", *j, string(val))
		}

		for pos < len(line) {
			pos = afmSkipWhitespace(line, pos)
			val, pos = afmNextValue(line, pos)
			pos = afmSkipWhitespace(line, pos)
			if pos == len(line) {
				continue
			} else if string(val) != ";" {
				return fmt.Errorf("afm: invalid char metrics at line %v: unexpected %v", *j, string(val))
			}

			var f float64
			val, pos = afmNextValue(line, pos)
			switch string(val) {
			// TODO: W0X, W1X, W0Y, W1Y, W, W0, W1, VV, L
			case "WX", "WY":
				f, pos = afmParseNumber(line, pos)
				charMetrics.Width = uint16(f + 0.5)
			case "N":
				name, pos = afmParseString(line, pos)
			case "B":
				for k := 0; k < 4; k++ {
					f, pos = afmParseNumber(line, pos)
					charMetrics.BBox[k] = int16(f + 0.5)
				}
			case "L":
				var successor, ligature string
				successor, pos = afmParseString(line, pos)
				ligature, pos = afmParseString(line, pos)
				ligatures = append(ligatures, afmLigature{uint16(i), successor, ligature})
			default:
				return fmt.Errorf("afm: invalid char metrics at line %v: unexpected %v", *j, string(val))
			}
		}
		if name == "" {
			return fmt.Errorf("afm: invalid char metrics at line %v: character name not specified", *j)
		} else if r, ok := glyphList[name]; !ok {
			return fmt.Errorf("afm: invalid char metrics at line %v: unknown character name: %v", *j, name)
		} else {
			afm.names[name] = uint16(i)
			afm.unicode[r] = uint16(i)
		}
	}
	*j++
	if !scanner.Scan() {
		return fmt.Errorf("afm: invalid char metrics at line %v", *j)
	} else if val, _ := afmNextValue(scanner.Bytes(), 0); string(val) != "EndCharMetrics" {
		return fmt.Errorf("afm: invalid char metrics at line %v: unexpected %v", *j, string(val))
	}

	afm.Ligatures = make(map[[2]uint16]uint16, len(ligatures))
	for _, lig := range ligatures {
		if successor, ok := afm.names[lig.successor]; !ok {
			return fmt.Errorf("afm: invalid char metrics at line %v: unknown character name: %v", *j, lig.successor)
		} else if ligature, ok := afm.names[lig.ligature]; !ok {
			return fmt.Errorf("afm: invalid char metrics at line %v: unknown character name: %v", *j, lig.ligature)
		} else {
			afm.Ligatures[[2]uint16{lig.base, successor}] = ligature
		}
	}
	return nil
}

func (afm *AFM) parseKernData(scanner *bufio.Scanner, j *int) error {
	for scanner.Scan() {
		*j++
		line := scanner.Bytes()
		key, pos := afmNextValue(line, 0)
		switch string(key) {
		case "StartTrackKern":
			n, _ := afmParseInteger(line, pos)
			afm.parseTrackKern(scanner, j, n)
		case "StartKernPairs", "StartKernPairs0":
			n, _ := afmParseInteger(line, pos)
			afm.parseKernPairs(scanner, j, n, 0)
		case "StartKernPairs1":
			n, _ := afmParseInteger(line, pos)
			afm.parseKernPairs(scanner, j, n, 1)
		case "EndKernData":
			return nil
		}
	}
	return fmt.Errorf("afm: invalid char metrics at line %v", *j)
}

func (afm *AFM) parseTrackKern(scanner *bufio.Scanner, j *int, n int) error {
	for i := 0; i < n; i++ {
		*j++
		if !scanner.Scan() {
			return fmt.Errorf("afm: invalid char metrics at line %v", *j)
		}
		// TODO
	}
	*j++
	if !scanner.Scan() {
		return fmt.Errorf("afm: invalid char metrics at line %v", *j)
	} else if val, _ := afmNextValue(scanner.Bytes(), 0); string(val) != "EndTrackKern" {
		return fmt.Errorf("afm: invalid char metrics at line %v: unexpected %v", *j, string(val))
	}
	return nil
}

func (afm *AFM) parseKernPairs(scanner *bufio.Scanner, j *int, n, writingDirection int) error {
	if afm.KernPairs == nil {
		afm.KernPairs = make(map[[2]uint16]int16, n)
	}
	for i := 0; i < n; i++ {
		*j++
		if !scanner.Scan() {
			return fmt.Errorf("afm: invalid char metrics at line %v", *j)
		} else if writingDirection != 0 {
			// TODO: support writingDirection != 0
			continue
		}
		line := scanner.Bytes()

		key, pos := afmNextValue(line, 0)
		switch string(key) {
		// TODO: KP, KPH, KPY
		case "KPX":
			var left, right string
			var kern float64
			left, pos = afmParseString(line, pos)
			right, pos = afmParseString(line, pos)
			kern, pos = afmParseNumber(line, pos)
			if l, ok := afm.names[left]; !ok {
				return fmt.Errorf("afm: invalid char metrics at line %v: unknown character name: %v", *j, left)
			} else if r, ok := afm.names[right]; !ok {
				return fmt.Errorf("afm: invalid char metrics at line %v: unknown character name: %v", *j, right)
			} else {
				afm.KernPairs[[2]uint16{l, r}] = int16(math.Round(kern))
			}
		}
	}
	*j++
	if !scanner.Scan() {
		return fmt.Errorf("afm: invalid char metrics at line %v", *j)
	} else if val, _ := afmNextValue(scanner.Bytes(), 0); string(val) != "EndKernPairs" {
		return fmt.Errorf("afm: invalid char metrics at line %v: unexpected %v", *j, string(val))
	}
	return nil
}

func ParseAFM(b []byte) (*AFM, error) {
	scanner := bufio.NewScanner(bytes.NewReader(b))
	if !scanner.Scan() {
		return nil, fmt.Errorf("invalid AFM file")
	} else if key, _ := afmNextValue(scanner.Bytes(), 0); string(key) != "StartFontMetrics" {
		return nil, fmt.Errorf("invalid AFM file")
	}

	j := 1 // line number
	afm := &AFM{}
Scanner:
	for scanner.Scan() {
		j++
		var f float64
		line := scanner.Bytes()
		key, pos := afmNextValue(line, 0)
		switch string(key) {
		case "FontName":
			afm.FontName, _ = afmParseString(line, pos)
		case "FullName":
			afm.FullName, _ = afmParseString(line, pos)
		case "FamilyName":
			afm.FamilyName, _ = afmParseString(line, pos)
		case "Weight":
			afm.Weight, _ = afmParseString(line, pos)
		case "FontBBox":
			for k := 0; k < 4; k++ {
				f, pos = afmParseNumber(line, pos)
				afm.FontBBox[k] = int16(math.Round(f))
			}
		case "CapHeight":
			f, _ = afmParseNumber(line, pos)
			afm.CapHeight = uint16(f + 0.5)
		case "XHeight":
			f, _ = afmParseNumber(line, pos)
			afm.XHeight = uint16(f + 0.5)
		case "Ascender":
			f, _ = afmParseNumber(line, pos)
			afm.Ascender = uint16(f + 0.5)
		case "Descender":
			f, _ = afmParseNumber(line, pos)
			afm.Descender = uint16(f + 0.5)
		case "UnderlinePosition":
			f, _ = afmParseNumber(line, pos)
			afm.UnderlinePosition = uint16(f + 0.5)
		case "UnderlineThickness":
			f, _ = afmParseNumber(line, pos)
			afm.UnderlineThickness = uint16(f + 0.5)
		case "ItalicAngle":
			afm.ItalicAngle, _ = afmParseNumber(line, pos)
		case "StartCharMetrics":
			n, _ := afmParseInteger(line, pos)
			if err := afm.parseCharMetrics(scanner, &j, n); err != nil {
				return nil, err
			}
		case "StartKernData":
			if err := afm.parseKernData(scanner, &j); err != nil {
				return nil, err
			}
		case "EndFontMetrics":
			break Scanner
		default:
			// no-op
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return afm, nil
}
