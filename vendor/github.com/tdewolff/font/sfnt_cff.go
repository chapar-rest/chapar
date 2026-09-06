package font

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
)

// TODO: add method to regenerate subrs optimally (useful after subset)
// TODO: use FDSelect for Font DICTs
// TODO: CFF has winding rule even-odd? CFF2 has winding rule nonzero

var ErrBadNumOperands = fmt.Errorf("bad number of operands for operator")

type cffTable struct {
	version int
	name    string
	top     *cffTopDICT
	//encoding    []uint8
	globalSubrs *cffINDEX
	charset     []string
	charStrings *cffINDEX
	fonts       *cffFontINDEX
}

func (sfnt *SFNT) parseCFF() error {
	b, ok := sfnt.Tables["CFF "]
	if !ok {
		return fmt.Errorf("CFF: missing table")
	}

	r := parse.NewBinaryReaderBytes(b)
	major := r.ReadUint8()
	minor := r.ReadUint8()
	if major != 1 || minor != 0 {
		return fmt.Errorf("CFF: bad version")
	}
	hdrSize := r.ReadUint8()
	if hdrSize != 4 {
		return fmt.Errorf("CFF: bad hdrSize")
	}
	offSize := r.ReadUint8() // offSize, not actually used
	if offSize == 0 || 4 < offSize {
		return fmt.Errorf("CFF: bad offSize")
	}

	nameINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: Name INDEX: %w", err)
	} else if len(nameINDEX.offset) != 2 {
		return fmt.Errorf("CFF: Name INDEX: bad count")
	}

	topINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: Top INDEX: %w", err)
	} else if topINDEX.Len() != nameINDEX.Len() {
		return fmt.Errorf("CFF: invalid lengths for Top INDEX or Name INDEX")
	} else if topINDEX.Len() != 1 {
		return fmt.Errorf("CFF: Top INDEX: only one font per file is supported")
	}

	stringINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: String INDEX: %w", err)
	}

	topDICT, err := parseTopDICT(topINDEX.Get(0), stringINDEX)
	if err != nil {
		return fmt.Errorf("CFF: Top DICT: %w", err)
	} else if topDICT.CharstringType != 2 {
		return fmt.Errorf("CFF: Type %d Charstring format not supported", topDICT.CharstringType)
	}

	//var encoding []uint8
	//r.Seek(uint32(topDICT.Encoding))
	//if format := r.ReadUint8(); format == 0 {
	//	numCodes := int(r.ReadUint8())
	//	encoding = make([]uint8, numCodes)
	//	for i := 0; i < numCodes; i++ {
	//		encoding[i] = r.ReadUint8()
	//	}
	//	fmt.Println(encoding)
	//} else if format == 1 {
	//	return fmt.Errorf("CFF: Encoding: unsupported format %v", format)
	//} else {
	//	return fmt.Errorf("CFF: Encoding: invalid format")
	//}

	globalSubrsINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: Global Subrs INDEX: %w", err)
	}

	r.Seek(int64(topDICT.CharStrings), 0)
	charStringsINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: CharStrings INDEX: %w", err)
	}

	var charset []string
	if topDICT.Charset < 3 {
		if topDICT.Charset != 0 {
			// 1 is Expert and 2 is ExpertSubset predefined glyph names
			return fmt.Errorf("CFF: Charset offset of %d not currently supported", topDICT.Charset)
		}
		charset = make([]string, charStringsINDEX.Len())
		copy(charset, cffStandardStrings)
	} else if 1 < charStringsINDEX.Len() {
		r.Seek(int64(topDICT.Charset), 0)
		if r.Len() == 0 {
			return fmt.Errorf("CFF: bad Charset")
		}

		numGlyphs := charStringsINDEX.Len()
		charset = make([]string, numGlyphs)
		charset[0] = ".notdef"

		format := r.ReadUint8()
		switch format {
		case 0:
			if r.Len() < 2*int64(numGlyphs-1) {
				return fmt.Errorf("CFF: bad Charset format 0")
			}
			for i := 1; i < numGlyphs; i++ {
				sid := r.ReadUint16()
				charset[i] = stringINDEX.GetSID(int(sid))
			}
		case 1, 2:
			for i := 1; i < numGlyphs; i++ {
				if r.Len() < 3 {
					return fmt.Errorf("CFF: bad Charset format %d", format)
				}
				first := int(r.ReadUint16())
				nLeft := 0
				if format == 1 {
					nLeft = int(r.ReadUint8())
				} else {
					nLeft = int(r.ReadUint16())
				}
				if numGlyphs < i+nLeft+1 {
					return fmt.Errorf("CFF: bad Charset format %d", format)
				}
				for j := 0; j < nLeft+1; j++ {
					charset[i+j] = stringINDEX.GetSID(first + j)
				}
				i += nLeft
			}
		default:
			return fmt.Errorf("CFF: unknown Charset format %d", format)
		}
	}

	sfnt.CFF = &cffTable{
		version: 1,
		name:    string(nameINDEX.Get(0)),
		top:     topDICT,
		//encoding:    encoding,
		globalSubrs: globalSubrsINDEX,
		charset:     charset,
		charStrings: charStringsINDEX,
	}

	if !topDICT.IsCID {
		if len(b) < topDICT.PrivateOffset || len(b)-topDICT.PrivateOffset < topDICT.PrivateLength {
			return fmt.Errorf("CFF: bad Private DICT offset")
		}
		privateDICT, err := parsePrivateDICT(b[topDICT.PrivateOffset:topDICT.PrivateOffset+topDICT.PrivateLength], false)
		if err != nil {
			return fmt.Errorf("CFF: Private DICT: %w", err)
		}

		localSubrsINDEX := &cffINDEX{}
		if privateDICT.Subrs != 0 {
			if len(b)-topDICT.PrivateOffset < privateDICT.Subrs {
				return fmt.Errorf("CFF: bad Local Subrs INDEX offset")
			}
			r.Seek(int64(topDICT.PrivateOffset+privateDICT.Subrs), 0)
			localSubrsINDEX, err = parseINDEX(r, false)
			if err != nil {
				return fmt.Errorf("CFF: Local Subrs INDEX: %w", err)
			}
		}
		sfnt.CFF.fonts = &cffFontINDEX{
			private:    []*cffPrivateDICT{privateDICT},
			localSubrs: []*cffINDEX{localSubrsINDEX},
			first:      []uint32{0, uint32(charStringsINDEX.Len())},
			fd:         []uint16{0},
		}
	} else {
		// CID font
		fonts, err := parseFontINDEX(b, topDICT.FDArray, topDICT.FDSelect, charStringsINDEX.Len(), false)
		if err != nil {
			return fmt.Errorf("CFF: %w", err)
		}
		sfnt.CFF.fonts = fonts
	}
	return nil
}

func (sfnt *SFNT) parseCFF2() error {
	return fmt.Errorf("CFF2: not supported")

	b, ok := sfnt.Tables["CFF2"]
	if !ok {
		return fmt.Errorf("CFF2: missing table")
	}

	r := parse.NewBinaryReaderBytes(b)
	major := r.ReadUint8()
	minor := r.ReadUint8()
	if major != 2 || minor != 0 {
		return fmt.Errorf("CFF2: bad version")
	}
	headerSize := r.ReadUint8()
	if headerSize != 5 {
		return fmt.Errorf("CFF2: bad headerSize")
	}
	topDictLength := r.ReadUint16()

	topDICT, err := parseTopDICT2(r.ReadBytes(int64(topDictLength)))
	if err != nil {
		return fmt.Errorf("CFF2: Top DICT: %w", err)
	}

	globalSubrsINDEX, err := parseINDEX(r, true)
	if err != nil {
		return fmt.Errorf("CFF2: Global Subrs INDEX: %w", err)
	}

	r.Seek(int64(topDICT.CharStrings), 0)
	charStringsINDEX, err := parseINDEX(r, true)
	if err != nil {
		return fmt.Errorf("CFF2: CharStrings INDEX: %w", err)
	}

	fonts, err := parseFontINDEX(b, topDICT.FDArray, topDICT.FDSelect, charStringsINDEX.Len(), true)
	if err != nil {
		return fmt.Errorf("CFF2: %w", err)
	}

	sfnt.CFF = &cffTable{
		version:     2,
		charStrings: charStringsINDEX,
		globalSubrs: globalSubrsINDEX,
		fonts:       fonts,
	}
	return nil
}

func (cff *cffTable) Version() int {
	return cff.version
}

func (cff *cffTable) TopDICT() *cffTopDICT {
	return cff.top
}

func (cff *cffTable) PrivateDICT(glyphID uint16) (*cffPrivateDICT, error) {
	privateDICT := cff.fonts.GetPrivate(glyphID)
	if privateDICT == nil {
		return nil, fmt.Errorf("bad glyph ID %v for Private DICT", glyphID)
	}
	return privateDICT, nil
}

func (cff *cffTable) GlyphName(glyphID uint16) (string, bool) {
	if int(glyphID) < len(cff.charset) {
		return cff.charset[glyphID], true
	}
	return "", false
}

func (cff *cffTable) SetGlyphName(glyphID uint16, name string) {
	if int(glyphID) < len(cff.charset) {
		cff.charset[glyphID] = name
	}
}

func (cff *cffTable) SetGlyphNames(names []string) {
	cff.charset = names
}

const (
	cffHstem      int32 = 1
	cffVstem      int32 = 3
	cffVmoveto    int32 = 4
	cffRlineto    int32 = 5
	cffHlineto    int32 = 6
	cffVlineto    int32 = 7
	cffRrcurveto  int32 = 8
	cffCallsubr   int32 = 10
	cffReturn     int32 = 11
	cffEscape     int32 = 12
	cffEndchar    int32 = 14
	cffHstemhm    int32 = 18
	cffHintmask   int32 = 19
	cffCntrmask   int32 = 20
	cffRmoveto    int32 = 21
	cffHmoveto    int32 = 22
	cffVstemhm    int32 = 23
	cffRcurveline int32 = 24
	cffRlinecurve int32 = 25
	cffVvcurveto  int32 = 26
	cffHhcurveto  int32 = 27
	cffShortint   int32 = 28
	cffCallgsubr  int32 = 29
	cffVhcurveto  int32 = 30
	cffHvcurveto  int32 = 31
	cffAnd        int32 = 256 + 3
	cffOr         int32 = 256 + 4
	cffNot        int32 = 256 + 5
	cffAbs        int32 = 256 + 9
	cffAdd        int32 = 256 + 10
	cffSub        int32 = 256 + 11
	cffDiv        int32 = 256 + 12
	cffNeg        int32 = 256 + 14
	cffEq         int32 = 256 + 15
	cffDrop       int32 = 256 + 18
	cffPut        int32 = 256 + 20
	cffGet        int32 = 256 + 21
	cffIfelse     int32 = 256 + 22
	cffRandom     int32 = 256 + 23
	cffMul        int32 = 256 + 24
	cffSqrt       int32 = 256 + 26
	cffDup        int32 = 256 + 27
	cffExch       int32 = 256 + 28
	cffIndex      int32 = 256 + 29
	cffRoll       int32 = 256 + 30
	cffHflex      int32 = 256 + 34
	cffFlex       int32 = 256 + 35
	cffHflex1     int32 = 256 + 36
	cffFlex1      int32 = 256 + 37

	cff2Vsindex int32 = 15
	cff2Blend   int32 = 16
)

func cffReadCharStringNumber(r *parse.BinaryReader, b0 int32) int32 {
	var v int32
	if b0 == 28 {
		v = int32(r.ReadInt16()) << 16
	} else if b0 < 247 {
		v = (b0 - 139) << 16
	} else if b0 < 251 {
		b1 := int32(r.ReadUint8())
		v = ((b0-247)*256 + b1 + 108) << 16
	} else if b0 < 255 {
		b1 := int32(r.ReadUint8())
		v = (-(b0-251)*256 - b1 - 108) << 16
	} else {
		v = r.ReadInt32() // least-significant 16 bits are fraction
	}
	return v
}

func (cff *cffTable) getSubroutine(glyphID uint16, b0 int32, stack int32) (int32, []byte, error) {
	typ := "local"
	var subrs *cffINDEX
	if b0 == cffCallsubr {
		subrs = cff.fonts.GetLocalSubrs(glyphID)
		if subrs == nil {
			return 0, nil, fmt.Errorf("%v subroutine: glyph's font doesn't have local subroutines", typ)
		}
	} else {
		typ = "global"
		subrs = cff.globalSubrs
	}

	// add bias
	n := subrs.Len()
	index := stack >> 16
	if n < 1240 {
		index += 107
	} else if n < 33900 {
		index += 1131
	} else {
		index += 32768
	}
	if index < 0 || math.MaxUint16 < index {
		return 0, nil, fmt.Errorf("%v subroutine: bad index %v", typ, index)
	}

	// get subroutine charString
	subr := subrs.Get(uint16(index))
	if subr == nil {
		return 0, nil, fmt.Errorf("%v subroutine: %v doesn't exist", typ, index)
	} else if 65535 < len(subr) {
		return 0, nil, fmt.Errorf("%v subroutine: %v too long", typ, index)
	}
	return index, subr, nil
}

func (cff *cffTable) parseCharString(glyphID uint16, cb func(*parse.BinaryReader, int32, []int32) error) error {
	table := "CFF"
	if cff.version == 2 {
		table = "CFF2"
	}

	charString := cff.charStrings.Get(glyphID)
	if charString == nil {
		return fmt.Errorf("%v: bad glyphID %v", table, glyphID)
	} else if 65535 < len(charString) {
		return fmt.Errorf("%v: charstring too long", table)
	}

	callStack := []*parse.BinaryReader{}
	r := parse.NewBinaryReaderBytes(charString)

	hints := 0
	stack := []int32{}
	beforeMoveto := true
	for {
		if cff.version == 2 && r.Len() == 0 && 0 < len(callStack) {
			// end of subroutine
			r = callStack[len(callStack)-1]
			callStack = callStack[:len(callStack)-1]
			continue
		} else if r.Len() == 0 {
			break
		}

		b0 := int32(r.ReadUint8())
		if 32 <= b0 || b0 == cffShortint {
			v := cffReadCharStringNumber(r, b0)
			if cff.version == 1 && 48 <= len(stack) || cff.version == 2 && 513 <= len(stack) {
				return fmt.Errorf("%v: too many operands for operator", table)
			}
			stack = append(stack, v)
		} else {
			if b0 == cffEscape {
				b0 = 256 + int32(r.ReadUint8())
			}

			if beforeMoveto && cff.version == 1 {
				if b0 == cffHstem || b0 == cffVstem || b0 == cffVmoveto || b0 == cffEndchar || cffHstemhm <= b0 && b0 <= cffVstemhm {
					// stack clearing operators, parse optional width
					hasWidth := len(stack)%2 == 1
					if b0 == cffHmoveto || b0 == cffVmoveto {
						hasWidth = !hasWidth
					}
					if hasWidth {
						stack = stack[1:]
					}
					if b0 == cffRmoveto || b0 == cffHmoveto || b0 == cffVmoveto {
						beforeMoveto = false
					}
				} else if b0 != cffCallsubr && b0 != cffCallgsubr && b0 != cffReturn {
					// return could be of main charstring or of subroutine
					return fmt.Errorf("%v: unexpected operator %d before moveto", table, b0)
				}
			} else if !beforeMoveto && cff.version == 1 && (b0 == cffHstem || b0 == cffVstem || b0 == cffHstemhm || b0 == cffVstemhm) {
				return fmt.Errorf("%v: unexpected operator %d after moveto", table, b0)
			}

			if err := cb(r, b0, stack); err != nil {
				return fmt.Errorf("%v: %v", table, err)
			}

			// handle hint and subroutine operators, clear stack for most operators
			switch b0 {
			case cffEndchar:
				if cff.version == 2 {
					return fmt.Errorf("CFF2: unsupported operator %d", b0)
				} else if len(stack) == 4 {
					return fmt.Errorf("CFF: unsupported endchar operands")
				} else if len(stack) != 0 {
					return fmt.Errorf("%v: %v", table, ErrBadNumOperands)
				}
				return nil
			case cffHstem, cffVstem, cffHstemhm, cffVstemhm:
				if len(stack) < 2 || len(stack)%2 != 0 {
					return fmt.Errorf("%v: %v", table, ErrBadNumOperands)
				}
				hints += len(stack) / 2
				if 96 < hints {
					return fmt.Errorf("%v: too many stem hints", table)
				}
				stack = stack[:0]
			case cffHintmask, cffCntrmask:
				if b0 == cffHintmask && 0 < len(stack) {
					// vstem or vstemhm
					if len(stack)%2 != 0 {
						return fmt.Errorf("%v: %v", table, ErrBadNumOperands)
					}
					hints += len(stack) / 2
					if 96 < hints {
						return fmt.Errorf("%v: too many stem hints", table)
					}
					stack = stack[:0]
				}
				r.ReadBytes(int64((hints + 7) / 8))
				stack = stack[:0]
			case cffCallsubr, cffCallgsubr:
				// callsubr and callgsubr
				if 10 < len(callStack) {
					return fmt.Errorf("%v: too many nested subroutines", table)
				} else if len(stack) == 0 {
					return fmt.Errorf("%v: %v", table, ErrBadNumOperands)
				}

				_, subr, err := cff.getSubroutine(glyphID, b0, stack[len(stack)-1])
				if err != nil {
					return fmt.Errorf("%v: %v", table, err)
				}
				stack = stack[:len(stack)-1]

				callStack = append(callStack, r)
				r = parse.NewBinaryReaderBytes(subr)
			case cffReturn:
				if cff.version == 2 {
					return fmt.Errorf("%v: unsupported operator %d", table, b0)
				} else if len(callStack) == 0 {
					return fmt.Errorf("%v: bad return", table)
				}
				r = callStack[len(callStack)-1]
				callStack = callStack[:len(callStack)-1]
			case cffRmoveto, cffHmoveto, cffVmoveto, cffRlineto, cffHlineto, cffVlineto, cffRrcurveto, cffHhcurveto, cffHvcurveto, cffRcurveline, cffRlinecurve, cffVhcurveto, cffVvcurveto, cffFlex, cffHflex, cffHflex1, cffFlex1:
				// path contruction operators
				stack = stack[:0]
			case cff2Blend:
				// blend
				if cff.version == 1 {
					return fmt.Errorf("CFF: unsupported operator %d", b0)
				}
				// TODO: blend
			case cff2Vsindex:
				// vsindex
				if cff.version == 1 {
					return fmt.Errorf("CFF: unsupported operator %d", b0)
				}
				// TODO: vsindex
			default:
				// TODO: arithmetic, storage, and conditional operators for CFF version 1?
				if 256 <= b0 {
					return fmt.Errorf("%v: unsupported operator 12 %d", table, b0-256)
				}
				return fmt.Errorf("%v: unsupported operator %d", table, b0)
			}
		}
	}

	if cff.version == 1 {
		return fmt.Errorf("CFF: charstring must end with endchar operator")
	}
	return nil
}

func (cff *cffTable) ToPath(p Pather, glyphID, ppem uint16, x0, y0, f float64, hinting Hinting) error {
	// x,y are raised to most-significant 16 bits and treat less-significant bits as fraction
	var x, y int32
	f /= float64(1 << 16) // correct back

	err := cff.parseCharString(glyphID, func(_ *parse.BinaryReader, b0 int32, stack []int32) error {
		switch b0 {
		case cffRmoveto:
			if len(stack) != 2 {
				return ErrBadNumOperands
			}
			x += stack[0]
			y += stack[1]
			p.Close()
			p.MoveTo(x0+f*float64(x), y0+f*float64(y))
		case cffHmoveto:
			if len(stack) != 1 {
				return ErrBadNumOperands
			}
			x += stack[0]
			p.Close()
			p.MoveTo(x0+f*float64(x), y0+f*float64(y))
		case cffVmoveto:
			if len(stack) != 1 {
				return ErrBadNumOperands
			}
			y += stack[0]
			p.Close()
			p.MoveTo(x0+f*float64(x), y0+f*float64(y))
		case cffRlineto:
			if len(stack) == 0 || len(stack)%2 != 0 {
				return ErrBadNumOperands
			}
			for i := 0; i < len(stack); i += 2 {
				x += stack[i+0]
				y += stack[i+1]
				p.LineTo(x0+f*float64(x), y0+f*float64(y))
			}
		case cffHlineto, cffVlineto:
			if len(stack) == 0 {
				return ErrBadNumOperands
			}
			vertical := b0 == cffVlineto
			for i := 0; i < len(stack); i++ {
				if !vertical {
					x += stack[i]
				} else {
					y += stack[i]
				}
				p.LineTo(x0+f*float64(x), y0+f*float64(y))
				vertical = !vertical
			}
		case cffRrcurveto:
			if len(stack) == 0 || len(stack)%6 != 0 {
				return ErrBadNumOperands
			}
			for i := 0; i < len(stack); i += 6 {
				x += stack[i+0]
				y += stack[i+1]
				cpx1, cpy1 := x, y
				x += stack[i+2]
				y += stack[i+3]
				cpx2, cpy2 := x, y
				x += stack[i+4]
				y += stack[i+5]
				p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
			}
		case cffHhcurveto, cffVvcurveto:
			if len(stack) < 4 || len(stack)%4 != 0 && (len(stack)-1)%4 != 0 {
				return ErrBadNumOperands
			}
			vertical := b0 == cffVvcurveto
			i := 0
			if len(stack)%4 == 1 {
				if !vertical {
					y += stack[0]
				} else {
					x += stack[0]
				}
				i++
			}
			for ; i < len(stack); i += 4 {
				if !vertical {
					x += stack[i+0]
				} else {
					y += stack[i+0]
				}
				cpx1, cpy1 := x, y
				x += stack[i+1]
				y += stack[i+2]
				cpx2, cpy2 := x, y
				if !vertical {
					x += stack[i+3]
				} else {
					y += stack[i+3]
				}
				p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
			}
		case cffHvcurveto, cffVhcurveto:
			if len(stack) < 4 || len(stack)%4 != 0 && (len(stack)-1)%4 != 0 {
				return ErrBadNumOperands
			}
			vertical := b0 == cffVhcurveto
			for i := 0; i < len(stack); i += 4 {
				if !vertical {
					x += stack[i+0]
				} else {
					y += stack[i+0]
				}
				cpx1, cpy1 := x, y
				x += stack[i+1]
				y += stack[i+2]
				cpx2, cpy2 := x, y
				if !vertical {
					y += stack[i+3]
				} else {
					x += stack[i+3]
				}
				if i+5 == len(stack) {
					if !vertical {
						x += stack[i+4]
					} else {
						y += stack[i+4]
					}
					i++
				}
				p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
				vertical = !vertical
			}
		case cffRcurveline:
			if len(stack) < 2 || (len(stack)-2)%6 != 0 {
				return ErrBadNumOperands
			}
			i := 0
			for ; i < len(stack)-2; i += 6 {
				x += stack[i+0]
				y += stack[i+1]
				cpx1, cpy1 := x, y
				x += stack[i+2]
				y += stack[i+3]
				cpx2, cpy2 := x, y
				x += stack[i+4]
				y += stack[i+5]
				p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
			}
			x += stack[i+0]
			y += stack[i+1]
			p.LineTo(x0+f*float64(x), y0+f*float64(y))
		case cffRlinecurve:
			if len(stack) < 6 || (len(stack)-6)%2 != 0 {
				return ErrBadNumOperands
			}
			i := 0
			for ; i < len(stack)-6; i += 2 {
				x += stack[i+0]
				y += stack[i+1]
				p.LineTo(x0+f*float64(x), y0+f*float64(y))
			}
			x += stack[i+0]
			y += stack[i+1]
			cpx1, cpy1 := x, y
			x += stack[i+2]
			y += stack[i+3]
			cpx2, cpy2 := x, y
			x += stack[i+4]
			y += stack[i+5]
			p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
		case cffFlex:
			if len(stack) != 13 {
				return ErrBadNumOperands
			}
			// always use cubic Béziers
			for i := 0; i < 12; i += 6 {
				x += stack[i+0]
				y += stack[i+1]
				cpx1, cpy1 := x, y
				x += stack[i+2]
				y += stack[i+3]
				cpx2, cpy2 := x, y
				x += stack[i+4]
				y += stack[i+5]
				p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
			}
		case cffHflex:
			// hflex
			if len(stack) != 7 {
				return ErrBadNumOperands
			}
			// always use cubic Béziers
			y1 := y
			x += stack[0]
			cpx1, cpy1 := x, y
			x += stack[1]
			y += stack[2]
			cpx2, cpy2 := x, y
			x += stack[3]
			p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))

			x += stack[4]
			cpx1, cpy1 = x, y
			x += stack[5]
			y = y1
			cpx2, cpy2 = x, y
			x += stack[6]
			p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
		case cffHflex1:
			if len(stack) != 9 {
				return ErrBadNumOperands
			}
			// always use cubic Béziers
			y1 := y
			x += stack[0]
			y += stack[1]
			cpx1, cpy1 := x, y
			x += stack[2]
			y += stack[3]
			cpx2, cpy2 := x, y
			x += stack[4]
			p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))

			x += stack[5]
			cpx1, cpy1 = x, y
			x += stack[6]
			y += stack[7]
			cpx2, cpy2 = x, y
			x += stack[8]
			y = y1
			p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
		case cffFlex1:
			if len(stack) != 11 {
				return ErrBadNumOperands
			}
			// always use cubic Béziers
			x1, y1 := x, y
			x += stack[0]
			y += stack[1]
			cpx1, cpy1 := x, y
			x += stack[2]
			y += stack[3]
			cpx2, cpy2 := x, y
			x += stack[4]
			y += stack[5]
			p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))

			x += stack[6]
			y += stack[7]
			cpx1, cpy1 = x, y
			x += stack[8]
			y += stack[9]
			cpx2, cpy2 = x, y
			dx, dy := x-x1, y-y1
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}
			if dy < dx {
				x += stack[10]
				y = y1
			} else {
				x = x1
				y += stack[10]
			}
			p.CubeTo(x0+f*float64(cpx1), y0+f*float64(cpy1), x0+f*float64(cpx2), y0+f*float64(cpy2), x0+f*float64(x), y0+f*float64(y))
		}
		return nil
	})
	if err != nil {
		return err
	}

	p.Close()
	return nil
}

func cffCharStringSubrsBias(n int) int {
	bias := 32768
	if n < 1240 {
		bias = 107
	} else if n < 33900 {
		bias = 1131
	}
	return bias
}

func cffNumberSize(i int) int {
	if -107 <= i && i <= 107 {
		return 1
	} else if -1131 <= i && i <= -108 || 108 <= i && i <= 1131 {
		return 2
	} else if -32767 <= i && i <= 32767 {
		return 3
	}
	return 5
}

type cffSubrIndexChange struct {
	start, end uint32
	index      int32
}

// updateSubrs changes all indices to local and global subroutines given the mappings for both
func (cff *cffTable) updateSubrs(localSubrsMap, globalSubrsMap map[int32]int32, localSubrs, globalSubrs *cffINDEX, localSubrsLen, globalSubrsLen int) error {
	if 1 < len(cff.fonts.localSubrs) {
		return fmt.Errorf("only single-font CFFs are supported")
	} else if len(localSubrsMap) == 0 && len(globalSubrsMap) == 0 {
		return nil
	}

	oldLocalSubrsLen := 0
	if 0 < len(cff.fonts.localSubrs) {
		oldLocalSubrsLen = cff.fonts.localSubrs[0].Len()
	}
	oldGlobalSubrsLen := cff.globalSubrs.Len()
	oldLocalSubrsBias := int32(cffCharStringSubrsBias(oldLocalSubrsLen))
	oldGlobalSubrsBias := int32(cffCharStringSubrsBias(oldGlobalSubrsLen))

	localSubrsHandled := map[int32]bool{}  // old indices
	globalSubrsHandled := map[int32]bool{} // old indices
	localSubrsBias := int32(cffCharStringSubrsBias(localSubrsLen))
	globalSubrsBias := int32(cffCharStringSubrsBias(globalSubrsLen))

	indexChanges := map[*cffINDEX][]cffSubrIndexChange{}
	var indexStack []*cffINDEX
	var offsetStack []uint32

	numGlyphs := uint16(cff.charStrings.Len())
	for glyphID := uint16(0); glyphID < numGlyphs; glyphID++ {
		// Change subroutine indices in the BinaryReader, this will make parseCharString use the
		// new index to pick the right subroutine. localSubrs and globalSubrs must thus already
		// have the new order/content.
		skipDepth := 0
		indexStack = append(indexStack[:0], cff.charStrings)
		offsetStack = append(offsetStack[:0], cff.charStrings.offset[glyphID])
		err := cff.parseCharString(glyphID, func(r *parse.BinaryReader, b0 int32, stack []int32) error {
			if b0 == cffCallsubr || b0 == cffCallgsubr {
				if len(stack) == 0 {
					return ErrBadNumOperands
				}

				num := stack[len(stack)-1]
				oldIndex, _, err := cff.getSubroutine(glyphID, b0, num)
				if err != nil {
					return err
				}

				mapped := false
				var oldBias int32
				var newIndex, newBias int32
				if b0 == cffCallsubr {
					if v, ok := localSubrsMap[oldIndex]; ok {
						oldBias = oldLocalSubrsBias
						newIndex = v
						newBias = localSubrsBias
						mapped = true
					}
				} else {
					if v, ok := globalSubrsMap[oldIndex]; ok {
						oldBias = oldGlobalSubrsBias
						newIndex = v
						newBias = globalSubrsBias
						mapped = true
					}
				}

				if skipDepth == 0 && mapped && oldIndex != newIndex {
					lenNumber := uint32(cffNumberSize(int(oldIndex - oldBias)))
					posNumber := uint32(r.Pos()) - 1 - lenNumber // -1 as we're past the operator

					index := indexStack[len(indexStack)-1]
					offset := offsetStack[len(offsetStack)-1]
					indexChanges[index] = append(indexChanges[index], cffSubrIndexChange{
						start: offset + posNumber,
						end:   offset + posNumber + lenNumber,
						index: newIndex - newBias,
					})
				}

				// only update subroutines once
				if 0 < skipDepth {
					skipDepth++
					return nil
				} else if b0 == cffCallsubr {
					if localSubrsHandled[oldIndex] {
						skipDepth++
						return nil
					} else {
						localSubrsHandled[oldIndex] = true
					}
				} else {
					if globalSubrsHandled[oldIndex] {
						skipDepth++
						return nil
					} else {
						globalSubrsHandled[oldIndex] = true
					}
				}

				var index *cffINDEX
				if b0 == cffCallsubr {
					index = localSubrs
				} else {
					index = globalSubrs
				}
				indexStack = append(indexStack, index)
				offsetStack = append(offsetStack, index.offset[newIndex])
			} else if b0 == cffReturn {
				if 0 < skipDepth {
					skipDepth--
					return nil
				}
				indexStack = indexStack[:len(indexStack)-1]
				offsetStack = offsetStack[:len(offsetStack)-1]
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	for index, changes := range indexChanges {
		sort.Slice(changes, func(i, j int) bool {
			return changes[i].start < changes[j].start
		})

		k := 1                                   // index into index.offset
		offset := uint32(0)                      // index into index.data
		diff := int32(0)                         // total number of bytes grown
		data := make([]byte, 0, len(index.data)) // destination, usually same size or shrinks
		for _, ch := range changes {
			// update index.offset upto the current change
			for k < len(index.offset) && index.offset[k] <= ch.start {
				index.offset[k] = uint32(int32(index.offset[k]) + diff)
				k++
			}

			// move bytes before current change
			data = append(data, index.data[offset:ch.start]...)

			// write new number
			n := len(data)
			if -107 <= ch.index && ch.index <= 107 {
				data = append(data, uint8(ch.index+139))
			} else if 108 <= ch.index && ch.index <= 1131 {
				ch.index -= 108
				data = append(data, uint8(ch.index/256+247))
				data = append(data, uint8(ch.index%256))
			} else if -1131 <= ch.index && ch.index <= -108 {
				ch.index = -ch.index - 108
				data = append(data, uint8(ch.index/256+251))
				data = append(data, uint8(ch.index%256))
			} else if -32768 <= ch.index && ch.index <= 32767 {
				data = append(data, 28, uint8(ch.index>>8), uint8(ch.index))
			} else {
				return fmt.Errorf("subroutine index outside valid range")
			}

			diff += int32(len(data)-n) - int32(ch.end-ch.start)
			offset = ch.end
		}

		// update index.offset upto the current change
		if diff != 0 {
			for k < len(index.offset) {
				index.offset[k] = uint32(int32(index.offset[k]) + diff)
				k++
			}
		}

		// move bytes before current change
		data = append(data, index.data[offset:]...)
		index.data = data
	}

	// set new Subrs INDEX, charStrings already modified
	cff.globalSubrs = globalSubrs

	// copy pointers in fonts
	fonts := *cff.fonts
	cff.fonts = &fonts
	cff.fonts.localSubrs = []*cffINDEX{localSubrs}
	return nil
}

// reindex subroutines in the order in which they appear and rearrange the global and local subroutines INDEX
func (cff *cffTable) ReindexSubrs() error {
	if 1 < len(cff.fonts.localSubrs) {
		return fmt.Errorf("only single-font CFFs are supported")
	}

	// find used subroutines
	skipDepth := 0
	numGlyphs := uint16(cff.charStrings.Len())
	localSubrsIndices := []int32{}
	localSubrsData := map[int32][]byte{}
	localSubrsCount := map[int32]int{}
	globalSubrsIndices := []int32{}
	globalSubrsData := map[int32][]byte{}
	globalSubrsCount := map[int32]int{}
	for glyphID := uint16(0); glyphID < numGlyphs; glyphID++ {
		err := cff.parseCharString(glyphID, func(_ *parse.BinaryReader, b0 int32, stack []int32) error {
			if b0 == cffCallsubr || b0 == cffCallgsubr {
				if 0 < skipDepth {
					// don't process subroutines twice
					skipDepth++
					return nil
				} else if len(stack) == 0 {
					return ErrBadNumOperands
				}

				index, subr, err := cff.getSubroutine(glyphID, b0, stack[len(stack)-1])
				if err != nil {
					return err
				}
				if b0 == cffCallsubr {
					if _, ok := localSubrsCount[index]; !ok {
						localSubrsIndices = append(localSubrsIndices, index)
						localSubrsData[index] = subr
					}
					localSubrsCount[index] = localSubrsCount[index] + 1
				} else {
					if _, ok := globalSubrsCount[index]; !ok {
						globalSubrsIndices = append(globalSubrsIndices, index)
						globalSubrsData[index] = subr
					}
					globalSubrsCount[index] = globalSubrsCount[index] + 1
				}
			} else if b0 == cffReturn {
				if 0 < skipDepth {
					skipDepth--
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	// sort descending
	sort.SliceStable(localSubrsIndices, func(i, j int) bool {
		return localSubrsCount[localSubrsIndices[j]] < localSubrsCount[localSubrsIndices[i]]
	})
	sort.SliceStable(globalSubrsIndices, func(i, j int) bool {
		return globalSubrsCount[globalSubrsIndices[j]] < globalSubrsCount[globalSubrsIndices[i]]
	})

	// construct subroutine index mapping
	localSubrs := &cffINDEX{}          // new INDEX
	globalSubrs := &cffINDEX{}         // new INDEX
	var localSubrsMap map[int32]int32  // old to new index
	var globalSubrsMap map[int32]int32 // old to new index
	if 0 < len(localSubrsIndices) {
		localSubrsMap = map[int32]int32{}
	}
	if 0 < len(globalSubrsIndices) {
		globalSubrsMap = map[int32]int32{}
	}
	for _, oldIndex := range localSubrsIndices {
		newIndex := localSubrs.Add(localSubrsData[oldIndex]) // copies data
		localSubrsMap[oldIndex] = int32(newIndex)
	}
	for _, oldIndex := range globalSubrsIndices {
		newIndex := globalSubrs.Add(globalSubrsData[oldIndex]) // copies data
		globalSubrsMap[oldIndex] = int32(newIndex)
	}

	// update subrs indices in charStrings for all glyphs and their subroutines
	return cff.updateSubrs(localSubrsMap, globalSubrsMap, localSubrs, globalSubrs, localSubrs.Len(), globalSubrs.Len())
}

type cffINDEX struct {
	offset []uint32
	data   []byte
}

func (t *cffINDEX) Len() int {
	if len(t.offset) == 0 {
		return 0
	}
	return len(t.offset) - 1
}

func (t *cffINDEX) Copy() *cffINDEX {
	offset := make([]uint32, len(t.offset))
	data := make([]byte, len(t.data))
	copy(offset, t.offset)
	copy(data, t.data)
	return &cffINDEX{
		offset: offset,
		data:   data,
	}
}

func (t *cffINDEX) Get(i uint16) []byte {
	if int(i) < t.Len() {
		return t.data[t.offset[i]:t.offset[i+1]:t.offset[i+1]]
	}
	return nil
}

func (t *cffINDEX) GetSID(sid int) string {
	// only for String INDEX
	if sid < len(cffStandardStrings) {
		return cffStandardStrings[sid]
	}
	sid -= len(cffStandardStrings)
	if math.MaxUint16 < sid {
		return ""
	}
	if b := t.Get(uint16(sid)); b != nil {
		return string(b)
	}
	return ""
}

func (t *cffINDEX) Add(data []byte) int {
	if len(t.offset) == 0 {
		t.offset = append(t.offset, 0)
	}
	t.data = append(t.data, data...)
	t.offset = append(t.offset, uint32(len(t.data)))
	return len(t.offset) - 2
}

func (t *cffINDEX) AddSID(data []byte) int {
	for i, s := range cffStandardStrings {
		if bytes.Equal(data, []byte(s)) {
			return i
		}
	}
	for i := 0; i+1 < len(t.offset); i++ {
		if bytes.Equal(data, t.data[t.offset[i]:t.offset[i+1]]) {
			return i
		}
	}
	// warning: this may become 65000 or larger which is not strictly allowed
	return t.Add(data) + len(cffStandardStrings)
}

func (t *cffINDEX) Extend(o *cffINDEX) {
	if o == nil || len(o.offset) < 2 {
		return
	} else if len(t.offset) < 2 {
		t.offset = o.offset
		t.data = o.data
		return
	}

	offset := make([]uint32, len(t.offset)+len(o.offset)-1)
	copy(offset, t.offset)
	for i := 0; i+1 < len(o.offset); i++ {
		offset[len(t.offset)+i] = uint32(len(t.data)) + o.offset[i+1]
	}
	t.offset = offset
	t.data = append(t.data, o.data...)
}

func (t *cffINDEX) String() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "INDEX: offsets=%v len=%v\n", t.offset, len(t.data))
	for i := 0; i+1 < len(t.offset); i++ {
		fmt.Fprintf(sb, "%v: %v\n", i, t.data[t.offset[i]:t.offset[i+1]])
	}
	return sb.String()
}

func parseINDEX(r *parse.BinaryReader, isCFF2 bool) (*cffINDEX, error) {
	t := &cffINDEX{}
	var count uint32
	if !isCFF2 {
		count = uint32(r.ReadUint16())
	} else {
		count = r.ReadUint32()
	}
	if count == 0 {
		// empty
		return t, nil
	} else if 1e6 < count {
		return nil, fmt.Errorf("too big")
	}

	offSize := r.ReadUint8()
	if offSize == 0 || 4 < offSize {
		return nil, fmt.Errorf("bad offSize")
	}
	if r.Len() < int64(offSize)*(int64(count)+1) {
		return nil, fmt.Errorf("bad data")
	}

	t.offset = make([]uint32, count+1)
	if offSize == 1 {
		for i := uint32(0); i < count+1; i++ {
			t.offset[i] = uint32(r.ReadUint8()) - 1
		}
	} else if offSize == 2 {
		for i := uint32(0); i < count+1; i++ {
			t.offset[i] = uint32(r.ReadUint16()) - 1
		}
	} else if offSize == 3 {
		for i := uint32(0); i < count+1; i++ {
			t.offset[i] = uint32(r.ReadUint16())<<8 + uint32(r.ReadUint8()) - 1
		}
	} else {
		for i := uint32(0); i < count+1; i++ {
			t.offset[i] = r.ReadUint32() - 1
		}
	}
	if r.Len() < int64(t.offset[count]) {
		return nil, fmt.Errorf("bad data")
	}
	t.data = r.ReadBytes(int64(t.offset[count]))
	return t, nil
}

func cffINDEXOffSize(n int) int {
	if n <= math.MaxUint8 {
		return 1
	} else if n <= math.MaxUint16 {
		return 2
	} else if n <= (2<<24 - 1) {
		return 3
	}
	return 4
}

func (t *cffINDEX) Write() ([]byte, error) {
	if math.MaxUint16 < len(t.offset)-1 {
		return nil, fmt.Errorf("too many indices")
	} else if len(t.data) == 0 {
		return []byte{0, 0}, nil // zero count
	} else if len(t.offset) == 0 || t.offset[0] != 0 || int(t.offset[len(t.offset)-1]) != len(t.data) {
		return nil, fmt.Errorf("bad offsets")
	}

	offSize := cffINDEXOffSize(len(t.data) + 1)
	n := 3 + len(t.data) + offSize*len(t.offset)
	w := parse.NewBinaryWriter(make([]byte, 0, n))
	w.WriteUint16(uint16(len(t.offset) - 1))
	w.WriteUint8(uint8(offSize))
	if offSize == 1 {
		for _, offset := range t.offset {
			w.WriteUint8(uint8(offset) + 1)
		}
	} else if offSize == 2 {
		for _, offset := range t.offset {
			w.WriteUint16(uint16(offset) + 1)
		}
	} else if offSize == 3 {
		for _, offset := range t.offset {
			w.WriteUint8(uint8((offset + 1) >> 16))
			w.WriteUint16(uint16((offset + 1) & 0x00FF))
		}
	} else if offSize == 4 {
		for _, offset := range t.offset {
			w.WriteUint32(offset + 1)
		}
	}
	w.WriteBytes(t.data)
	return w.Bytes(), nil
}

type cffTopDICT struct {
	IsSynthetic bool
	IsCID       bool

	Version            string
	Notice             string
	Copyright          string
	FullName           string
	FamilyName         string
	Weight             string
	IsFixedPitch       bool
	ItalicAngle        float64
	UnderlinePosition  float64
	UnderlineThickness float64
	PaintType          int
	CharstringType     int
	FontMatrix         [6]float64
	UniqueID           int
	FontBBox           [4]float64
	StrokeWidth        float64
	XUID               []int
	Charset            int
	Encoding           int
	CharStrings        int
	PrivateOffset      int
	PrivateLength      int
	SyntheticBase      int
	PostScript         string
	BaseFontName       string
	BaseFontBlend      []int
	ROS1               string
	ROS2               string
	ROS3               int
	CIDFontVersion     int
	CIDFontRevision    int
	CIDFontType        int
	CIDCount           int
	UIDBase            int
	FDArray            int
	FDSelect           int
	FontName           string
	Vstore             int // CFF2
}

func parseTopDICT(b []byte, stringINDEX *cffINDEX) (*cffTopDICT, error) {
	dict := &cffTopDICT{
		UnderlinePosition:  -100,
		UnderlineThickness: 50,
		CharstringType:     2,
		FontMatrix:         [6]float64{0.001, 0.0, 0.0, 0.001, 0.0, 0.0},
		CIDCount:           8720,
	}
	err := parseDICT(b, false, func(b0 int, is []int, fs []float64) bool {
		switch b0 {
		case 0:
			dict.Version = stringINDEX.GetSID(is[0])
		case 1:
			dict.Notice = stringINDEX.GetSID(is[0])
		case 256 + 0:
			dict.Copyright = stringINDEX.GetSID(is[0])
		case 2:
			dict.FullName = stringINDEX.GetSID(is[0])
		case 3:
			dict.FamilyName = stringINDEX.GetSID(is[0])
		case 4:
			dict.Weight = stringINDEX.GetSID(is[0])
		case 256 + 1:
			dict.IsFixedPitch = is[0] != 0
		case 256 + 2:
			dict.ItalicAngle = fs[0]
		case 256 + 3:
			dict.UnderlinePosition = fs[0]
		case 256 + 4:
			dict.UnderlineThickness = fs[0]
		case 256 + 5:
			dict.PaintType = is[0]
		case 256 + 6:
			dict.CharstringType = is[0]
		case 256 + 7:
			copy(dict.FontMatrix[:], fs)
		case 13:
			dict.UniqueID = is[0]
		case 5:
			copy(dict.FontBBox[:], fs)
		case 256 + 8:
			dict.StrokeWidth = fs[0]
		case 14:
			dict.XUID = append([]int{}, is...)
		case 15:
			dict.Charset = is[0]
		case 16:
			dict.Encoding = is[0]
		case 17:
			dict.CharStrings = is[0]
		case 18:
			dict.PrivateOffset = is[1]
			dict.PrivateLength = is[0]
		case 256 + 20:
			dict.IsSynthetic = true
			dict.SyntheticBase = is[0]
		case 256 + 21:
			dict.PostScript = stringINDEX.GetSID(is[0])
		case 256 + 22:
			dict.BaseFontName = stringINDEX.GetSID(is[0])
		case 256 + 23:
			dict.BaseFontBlend = append([]int{}, is...)
		case 256 + 30:
			// TODO: it is unclear how the ROS operator influences the GIDs/CIDs
			dict.IsCID = true
			dict.ROS1 = stringINDEX.GetSID(is[0])
			dict.ROS2 = stringINDEX.GetSID(is[1])
			dict.ROS3 = is[2]
		case 256 + 31:
			dict.CIDFontVersion = is[0]
		case 256 + 32:
			dict.CIDFontRevision = is[0]
		case 256 + 33:
			dict.CIDFontType = is[0]
		case 256 + 34:
			dict.CIDCount = is[0]
		case 256 + 35:
			dict.UIDBase = is[0]
		case 256 + 36:
			dict.FDArray = is[0]
		case 256 + 37:
			dict.FDSelect = is[0]
		case 256 + 38:
			dict.FontName = stringINDEX.GetSID(is[0])
		default:
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (t *cffTopDICT) Write(strings *cffINDEX) ([]byte, error) {
	// TODO: some values have no default and may need to be written always
	w := parse.NewBinaryWriter([]byte{})
	if t.Version != "" {
		writeDICTEntry(w, 0, strings.AddSID([]byte(t.Version)))
	}
	if t.Notice != "" {
		writeDICTEntry(w, 1, strings.AddSID([]byte(t.Notice)))
	}
	if t.Copyright != "" {
		writeDICTEntry(w, 256+0, strings.AddSID([]byte(t.Copyright)))
	}
	if t.FullName != "" {
		writeDICTEntry(w, 2, strings.AddSID([]byte(t.FullName)))
	}
	if t.FamilyName != "" {
		writeDICTEntry(w, 3, strings.AddSID([]byte(t.FamilyName)))
	}
	if t.Weight != "" {
		writeDICTEntry(w, 4, strings.AddSID([]byte(t.Weight)))
	}
	if t.IsFixedPitch {
		writeDICTEntry(w, 256+1, 1)
	}
	if t.ItalicAngle != 0.0 {
		writeDICTEntry(w, 256+2, t.ItalicAngle)
	}
	if t.UnderlinePosition != -100.0 {
		writeDICTEntry(w, 256+3, t.UnderlinePosition)
	}
	if t.UnderlineThickness != 50.0 {
		writeDICTEntry(w, 256+4, t.UnderlineThickness)
	}
	if t.PaintType != 0 {
		writeDICTEntry(w, 256+5, t.PaintType)
	}
	if t.CharstringType != 2 {
		return nil, fmt.Errorf("CharstringType must be 2")
	}
	if t.FontMatrix != [6]float64{0.001, 0, 0, 0.001, 0, 0} {
		writeDICTEntry(w, 256+7, t.FontMatrix[0], t.FontMatrix[1], t.FontMatrix[2], t.FontMatrix[3], t.FontMatrix[4], t.FontMatrix[5])
	}
	if t.UniqueID != 0 {
		writeDICTEntry(w, 13, t.UniqueID)
	}
	if t.FontBBox != [4]float64{0, 0, 0, 0} {
		writeDICTEntry(w, 5, t.FontBBox[0], t.FontBBox[1], t.FontBBox[2], t.FontBBox[3])
	}
	if t.StrokeWidth != 0.0 {
		writeDICTEntry(w, 256+8, t.StrokeWidth)
	}
	if 0 < len(t.XUID) {
		writeDICTEntry(w, 14, t.XUID)
	}
	// not supported
	//if t.Charset != 0 {
	//	writeDICTEntry(w, 15, t.Charset)
	//}
	//if t.Encoding != 0 {
	//	writeDICTEntry(w, 16, t.Encoding)
	//}
	// set later
	//if t.CharStrings != 0 {
	//	writeDICTEntry(w, 17, t.CharStrings)
	//}
	//if t.Private != 0 {
	//	writeDICTEntry(w, 18, t.Private)
	//}
	if t.SyntheticBase != 0 {
		writeDICTEntry(w, 256+20, t.SyntheticBase)
	}
	if t.PostScript != "" {
		writeDICTEntry(w, 256+21, strings.AddSID([]byte(t.PostScript)))
	}
	if t.BaseFontName != "" {
		writeDICTEntry(w, 256+22, strings.AddSID([]byte(t.BaseFontName)))
	}
	if 0 < len(t.BaseFontBlend) {
		writeDICTEntry(w, 256+23, t.BaseFontBlend)
	}
	return w.Bytes(), nil
}

type cffFontDICT struct {
	PrivateOffset int
	PrivateLength int
}

func parseFontDICT(b []byte, isCFF2 bool) (*cffFontDICT, error) {
	dict := &cffFontDICT{}
	return dict, parseDICT(b, isCFF2, func(b0 int, is []int, fs []float64) bool {
		switch b0 {
		case 18:
			dict.PrivateOffset = is[1]
			dict.PrivateLength = is[0]
		case 256 + 7:
			// FontMatrix
		case 256 + 38:
			// FontName
		default:
			return false
		}
		return true
	})
}

type cffPrivateDICT struct {
	BlueValues        []float64
	OtherBlues        []float64
	FamilyBlues       []float64
	FamilyOtherBlues  []float64
	BlueScale         float64
	BlueShift         float64
	BlueFuzz          float64
	StdHW             float64
	StdVW             float64
	StemSnapH         []float64
	StemSnapV         []float64
	ForceBold         bool
	LanguageGroup     int
	ExpansionFactor   float64
	InitialRandomSeed int
	Subrs             int
	DefaultWidthX     float64
	NominalWidthX     float64

	// CFF2
	Vsindex int
	Blend   []float64
}

func parsePrivateDICT(b []byte, isCFF2 bool) (*cffPrivateDICT, error) {
	dict := &cffPrivateDICT{
		BlueScale:       0.039625,
		BlueShift:       7.0,
		BlueFuzz:        1.0,
		ExpansionFactor: 0.06,
	}

	err := parseDICT(b, isCFF2, func(b0 int, is []int, fs []float64) bool {
		switch b0 {
		case 6:
			dict.BlueValues = append([]float64{}, fs...)
		case 7:
			dict.OtherBlues = append([]float64{}, fs...)
		case 8:
			dict.FamilyBlues = append([]float64{}, fs...)
		case 9:
			dict.FamilyOtherBlues = append([]float64{}, fs...)
		case 256 + 9:
			dict.BlueScale = fs[0]
		case 256 + 10:
			dict.BlueShift = fs[0]
		case 256 + 11:
			dict.BlueFuzz = fs[0]
		case 10:
			dict.StdHW = fs[0]
		case 11:
			dict.StdVW = fs[0]
		case 256 + 12:
			dict.StemSnapH = append([]float64{}, fs...)
		case 256 + 13:
			dict.StemSnapV = append([]float64{}, fs...)
		case 256 + 14:
			dict.ForceBold = is[0] != 0
		case 256 + 17:
			dict.LanguageGroup = is[0]
		case 256 + 18:
			dict.ExpansionFactor = fs[0]
		case 256 + 19:
			dict.InitialRandomSeed = is[0]
		case 19:
			dict.Subrs = is[0]
		case 20:
			dict.DefaultWidthX = fs[0]
		case 21:
			dict.NominalWidthX = fs[0]
		case 22:
			dict.Vsindex = is[0]
		case 23:
			dict.Blend = append([]float64{}, fs...)
		default:
			return false
		}
		return true
	})
	return dict, err
}

func (t *cffPrivateDICT) Write() ([]byte, error) {
	// TODO: some values have no default and may need to be written always
	w := parse.NewBinaryWriter([]byte{})
	if 0 < len(t.BlueValues) {
		writeDICTEntry(w, 6, t.BlueValues)
	}
	if 0 < len(t.OtherBlues) {
		writeDICTEntry(w, 7, t.OtherBlues)
	}
	if 0 < len(t.FamilyBlues) {
		writeDICTEntry(w, 8, t.FamilyBlues)
	}
	if 0 < len(t.FamilyOtherBlues) {
		writeDICTEntry(w, 9, t.FamilyOtherBlues)
	}
	if t.BlueScale != 0.039625 {
		writeDICTEntry(w, 256+9, t.BlueScale)
	}
	if t.BlueShift != 7.0 {
		writeDICTEntry(w, 256+10, t.BlueShift)
	}
	if t.BlueFuzz != 1.0 {
		writeDICTEntry(w, 256+11, t.BlueFuzz)
	}
	if t.StdHW != 0.0 {
		writeDICTEntry(w, 10, t.StdHW)
	}
	if t.StdVW != 0.0 {
		writeDICTEntry(w, 11, t.StdVW)
	}
	if len(t.StemSnapH) != 0 {
		writeDICTEntry(w, 256+12, t.StemSnapH)
	}
	if len(t.StemSnapV) != 0 {
		writeDICTEntry(w, 256+13, t.StemSnapV)
	}
	if t.ForceBold {
		writeDICTEntry(w, 256+14, t.ForceBold)
	}
	if t.LanguageGroup != 0 {
		writeDICTEntry(w, 256+17, t.LanguageGroup)
	}
	if t.ExpansionFactor != 0.06 {
		writeDICTEntry(w, 256+18, t.ExpansionFactor)
	}
	if t.InitialRandomSeed != 0 {
		writeDICTEntry(w, 256+19, t.InitialRandomSeed)
	}
	//if t.Subrs != 0 {
	//	return nil, fmt.Errorf("Local Subrs not supported")
	//}
	if t.DefaultWidthX != 0 {
		writeDICTEntry(w, 20, t.DefaultWidthX)
	}
	if t.NominalWidthX != 0 {
		writeDICTEntry(w, 21, t.NominalWidthX)
	}
	return w.Bytes(), nil
}

func parseTopDICT2(b []byte) (*cffTopDICT, error) {
	dict := &cffTopDICT{
		FontMatrix: [6]float64{0.001, 0.0, 0.0, 0.001, 0.0, 0.0},
	}
	return dict, parseDICT(b, true, func(b0 int, is []int, fs []float64) bool {
		switch b0 {
		case 256 + 7:
			copy(dict.FontMatrix[:], fs)
		case 17:
			dict.CharStrings = is[0]
		case 256 + 36:
			dict.FDArray = is[0]
		case 256 + 37:
			dict.FDSelect = is[0]
		case 24:
			dict.Vstore = is[0]
		default:
			return false
		}
		return true
	})
}

func parseDICT(b []byte, isCFF2 bool, callback func(b0 int, is []int, fs []float64) bool) error {
	opSize := map[int]int{
		256 + 7:  6,
		5:        4,
		14:       -1,
		18:       2,
		256 + 23: -1,
		256 + 30: 3,
		6:        -1,
		7:        -1,
		8:        -1,
		9:        -1,
		256 + 12: -1,
		256 + 13: -1,
	}

	r := parse.NewBinaryReaderBytes(b)
	ints := []int{}
	reals := []float64{}
	for 0 < r.Len() {
		b0 := int(r.ReadUint8())
		if b0 < 22 {
			// operator
			if b0 == 12 {
				b0 = 256 + int(r.ReadUint8())
			}

			size := 1
			if s, ok := opSize[b0]; ok {
				if s == -1 {
					size = len(ints)
				} else {
					size = s
				}
			}
			if len(ints) < size {
				return fmt.Errorf("too few operands for operator")
			}

			is := ints[len(ints)-size:]
			fs := reals[len(reals)-size:]
			ints = ints[:len(ints)-size]
			reals = reals[:len(reals)-size]

			if ok := callback(b0, is, fs); !ok {
				return fmt.Errorf("bad operator")
			}
		} else if 22 <= b0 && b0 < 28 || b0 == 31 || b0 == 255 {
			// reserved
		} else {
			if !isCFF2 && 48 <= len(ints) || isCFF2 && 513 <= len(ints) {
				return fmt.Errorf("too many operands for operator")
			}
			i, f := parseDICTNumber(b0, r)
			if math.IsNaN(f) {
				f = float64(i)
			} else {
				i = int(math.Round(f))
			}
			ints = append(ints, i)
			reals = append(reals, f)
		}
	}
	return nil
}

func parseDICTNumber(b0 int, r *parse.BinaryReader) (int, float64) {
	if b0 < 28 {
		// operator
		return 0, math.NaN()
	} else if b0 == 28 {
		return int(r.ReadInt16()), math.NaN()
	} else if b0 == 29 {
		return int(r.ReadInt32()), math.NaN()
	} else if b0 == 30 {
		num := []byte{}
		for {
			b := r.ReadUint8()
			for i := 0; i < 2; i++ {
				switch b >> 4 {
				case 0x0A:
					num = append(num, '.')
				case 0x0B:
					num = append(num, 'E')
				case 0x0C:
					num = append(num, 'E', '-')
				case 0x0D:
					// reserved
				case 0x0E:
					num = append(num, '-')
				case 0x0F:
					f, err := strconv.ParseFloat(string(num), 32)
					if err != nil {
						return 0, math.NaN()
					}
					return 0, f
				default:
					num = append(num, '0'+byte(b>>4))
				}
				b = b << 4
			}
		}
	} else if b0 == 31 {
		// reserved
		return 0, math.NaN()
	} else if b0 < 247 {
		return b0 - 139, math.NaN()
	} else if b0 < 251 {
		b1 := int(r.ReadUint8())
		return (b0-247)*256 + b1 + 108, math.NaN()
	} else if b0 < 255 {
		b1 := int(r.ReadUint8())
		return -(b0-251)*256 - b1 - 108, math.NaN()
	}
	// reserved
	return 0, math.NaN()
}

// returns: integer, float, isFrac, ok
func cffDICTNumber(val any) (int, float64, bool, bool) {
	switch v := val.(type) {
	case int:
		return v, float64(v), false, true
	case float64:
		var i int
		isFrac := false
		if integer, frac := math.Modf(v); frac == 0.0 {
			i = int(integer)
		} else {
			isFrac = true
		}
		return i, v, isFrac, true
	default:
		return 0, 0.0, false, false
	}
}

func cffDICTFloat(f float64) ([]byte, int) {
	floatNibbles := strconv.AppendFloat([]byte{}, f, 'G', 6, 64)
	if 1 < len(floatNibbles) && floatNibbles[0] == '0' && floatNibbles[1] == '.' {
		floatNibbles = floatNibbles[1:]
	}
	n := int(math.Ceil(float64(len(floatNibbles)+1)/2.0) + 0.5) // includes end nibbles
	return floatNibbles, 1 + n                                  // key and value
}

func cffDICTIntegerSize(i int) int {
	return cffNumberSize(i)
}

func writeDICTEntry(w *parse.BinaryWriter, op int, vals ...any) error {
	if len(vals) == 1 {
		switch vs := vals[0].(type) {
		case []int:
			vals = make([]any, 0, len(vs))
			for _, v := range vs {
				vals = append(vals, v)
			}
		case []float64:
			vals = make([]any, 0, len(vs))
			for _, v := range vs {
				vals = append(vals, v)
			}
		}
	}
	if 48 < len(vals) {
		return fmt.Errorf("too many operands")
	}

	allowFloat := false
	if 5 <= op && op <= 11 || op == 20 || op == 21 || 255+2 <= op && op <= 255+4 || 255+7 <= op && op <= 255+13 || op == 255+18 || op == 255+23 {
		// strictly most operators allow a number and thus an integer or real operand but in
		// practice these are offsets (and thus integers) and some parsers expect them to be
		// integers (even though a float may be shorter for e.g. 32790 (3 bytes) instead of
		// 4 bytes for an integer
		allowFloat = true
	}

	for _, val := range vals {
		i, f, isFrac, ok := cffDICTNumber(val)
		if !ok {
			return fmt.Errorf("unknown operand type: %T", val)
		}

		floatNibbles, nFloat := cffDICTFloat(f)
		if allowFloat && (isFrac || nFloat < cffDICTIntegerSize(i)) {
			n := 0
			var b uint8
			w.WriteUint8(30)
			for i := 0; i < len(floatNibbles); i++ {
				b <<= 4
				switch floatNibbles[i] {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					b |= uint8(floatNibbles[i] - '0')
				case '.':
					b |= 0x0A
				case 'E':
					if i+1 < len(floatNibbles) && floatNibbles[i+1] == '-' {
						b |= 0x0C
						i++
					} else {
						b |= 0x0B
					}
				case '-':
					b |= 0x0E
				}
				n++
				if n%2 == 0 {
					w.WriteUint8(b)
					b = 0
				}
			}
			if n%2 == 1 {
				b <<= 4
				b |= 0x0F
				w.WriteUint8(b)
			} else {
				w.WriteUint8(0xFF)
			}
		} else if isFrac {
			return fmt.Errorf("unexpected real operand for operator %v: %v", op, val)
		} else if -107 <= i && i <= 107 {
			w.WriteUint8(uint8(i + 139))
		} else if 108 <= i && i <= 1131 {
			i -= 108
			w.WriteUint8(uint8(i/256 + 247))
			w.WriteUint8(uint8(i % 256))
		} else if -1131 <= i && i <= -108 {
			i = -i - 108
			w.WriteUint8(uint8(i/256 + 251))
			w.WriteUint8(uint8(i % 256))
		} else if -32768 <= i && i <= 32767 {
			w.WriteUint8(28)
			w.WriteUint16(uint16(i))
		} else {
			w.WriteUint8(29)
			w.WriteUint32(uint32(i))
		}
	}

	if 256+256 <= op || op == 12 {
		return fmt.Errorf("bad operator: %v", op)
	} else if 256 <= op {
		w.WriteUint8(0x0C)
		op -= 256
	}
	w.WriteUint8(uint8(op))
	return nil
}

type cffFontINDEX struct {
	private    []*cffPrivateDICT
	localSubrs []*cffINDEX

	fds   []uint8 // fds or the other two are used
	first []uint32
	fd    []uint16
}

func (t *cffFontINDEX) Index(glyphID uint32) (uint16, bool) {
	if t.fds != nil {
		if len(t.fds) <= int(glyphID) {
			return 0, false
		}
		return uint16(t.fds[glyphID]), true
	} else if len(t.first) == 0 || t.first[len(t.first)-1] <= glyphID {
		return 0, false
	}

	i := 0
	for t.first[i+1] <= glyphID {
		i++
	}
	if len(t.fd) < i {
		return 0, false
	}
	return t.fd[i], true
}

// GetPrivate returns the Private DICT for the glyph's font, or nil if the font has none.
func (t *cffFontINDEX) GetPrivate(glyphID uint16) *cffPrivateDICT {
	i, ok := t.Index(uint32(glyphID))
	if !ok {
		return nil
	}
	return t.private[i]
}

// GetLocalSubrs returns the LocalSubrs INDEX for the glyph's font, or nil if the font has none.
func (t *cffFontINDEX) GetLocalSubrs(glyphID uint16) *cffINDEX {
	i, ok := t.Index(uint32(glyphID))
	if !ok {
		return nil
	}
	return t.localSubrs[i]
}

func parseFontINDEX(b []byte, fdArray, fdSelect, nGlyphs int, isCFF2 bool) (*cffFontINDEX, error) {
	if len(b) < fdArray {
		return nil, fmt.Errorf("bad Font INDEX offset")
	}

	r := parse.NewBinaryReaderBytes(b)
	r.Seek(int64(fdArray), 0)
	fontINDEX, err := parseINDEX(r, false)
	if err != nil {
		return nil, fmt.Errorf("Font INDEX: %w", err)
	}

	fonts := &cffFontINDEX{}
	fonts.private = make([]*cffPrivateDICT, fontINDEX.Len())
	fonts.localSubrs = make([]*cffINDEX, fontINDEX.Len())
	for i := 0; i < fontINDEX.Len(); i++ {
		fontDICT, err := parseFontDICT(fontINDEX.Get(uint16(i)), isCFF2)
		if err != nil {
			return nil, fmt.Errorf("Font DICT: %w", err)
		}
		if len(b) < fontDICT.PrivateOffset || len(b)-fontDICT.PrivateOffset < fontDICT.PrivateLength {
			return nil, fmt.Errorf("Font DICT: bad Private DICT offset")
		}
		privateDICT, err := parsePrivateDICT(b[fontDICT.PrivateOffset:fontDICT.PrivateOffset+fontDICT.PrivateLength], isCFF2)
		if err != nil {
			return nil, fmt.Errorf("Private DICT: %w", err)
		}
		fonts.private[i] = privateDICT

		if privateDICT.Subrs != 0 {
			if len(b)-fontDICT.PrivateOffset < privateDICT.Subrs {
				return nil, fmt.Errorf("bad Local Subrs INDEX offset")
			}
			r.Seek(int64(fontDICT.PrivateOffset+privateDICT.Subrs), 0)
			fonts.localSubrs[i], err = parseINDEX(r, isCFF2)
			if err != nil {
				return nil, fmt.Errorf("Local Subrs INDEX: %w", err)
			}
		} else if isCFF2 {
			return nil, fmt.Errorf("Private DICT must have Local Subrs INDEX offset")
		}
	}

	r.Seek(int64(fdSelect), 0)
	format := r.ReadUint8()
	if format == 0 {
		fonts.fds = make([]uint8, nGlyphs)
		for i := 0; i < nGlyphs; i++ {
			fonts.fds[i] = r.ReadUint8()
		}
	} else if format == 3 {
		nRanges := r.ReadUint16()
		fonts.first = make([]uint32, nRanges+1)
		fonts.fd = make([]uint16, nRanges)
		for i := 0; i < int(nRanges); i++ {
			fonts.first[i] = uint32(r.ReadUint16())
			fonts.fd[i] = uint16(r.ReadUint8())
		}
		fonts.first[nRanges] = uint32(r.ReadUint16())
	} else if isCFF2 && format == 4 {
		nRanges := r.ReadUint32()
		fonts.first = make([]uint32, nRanges+1)
		fonts.fd = make([]uint16, nRanges)
		for i := 0; i < int(nRanges); i++ {
			fonts.first[i] = r.ReadUint32()
			fonts.fd[i] = r.ReadUint16()
		}
		fonts.first[nRanges] = r.ReadUint32()
	} else {
		return nil, fmt.Errorf("FDSelect: bad format")
	}
	return fonts, nil
}

func (cff *cffTable) Write() ([]byte, error) {
	if cff.version != 1 {
		return nil, fmt.Errorf("unsupported version: %d", cff.version)
	}

	if 1 < len(cff.fonts.private) || 1 < len(cff.fonts.localSubrs) {
		return nil, fmt.Errorf("must contain only one font")
	}

	w := parse.NewBinaryWriter([]byte{})
	w.WriteUint8(1) // major version
	w.WriteUint8(0) // minor version
	w.WriteUint8(4) // hdrSize
	w.WriteUint8(1) // offSize, not used

	name := &cffINDEX{}
	name.Add([]byte(cff.name))
	nameINDEX, err := name.Write()
	if err != nil {
		return nil, fmt.Errorf("Name INDEX: %v", err)
	}
	w.WriteBytes(nameINDEX)

	strings := &cffINDEX{}
	if cff.top == nil {
		cff.top = &cffTopDICT{}
	}
	topDICT, err := cff.top.Write(strings)
	if err != nil {
		return nil, fmt.Errorf("Top DICT: %v", err)
	}

	var charset *parse.BinaryWriter
	numGlyphs := cff.charStrings.Len()
	if cff.charset != nil {
		// TODO: reorder entries in charString sequentially to make this smaller
		if len(cff.charset) != numGlyphs {
			return nil, fmt.Errorf("charset length must match number of glyphs")
		}

		sids := make([]int, numGlyphs-1)
		for i, name := range cff.charset[1:numGlyphs] {
			sids[i] = strings.AddSID([]byte(name))
		}

		// count number of ranges and max values of nLeft
		maxNLeft := 0
		lastStart := 0
		ranges := [][2]uint16{}
		for i, sid := range sids {
			if i != 0 && sid != sids[i-1]+1 {
				nLeft := i - lastStart - 1
				if maxNLeft < nLeft {
					maxNLeft = nLeft
				}
				ranges = append(ranges, [2]uint16{uint16(sids[lastStart]), uint16(nLeft)})
				lastStart = i
			}
		}
		if 0 < len(sids) {
			nLeft := len(sids) - lastStart - 1
			if maxNLeft < nLeft {
				maxNLeft = nLeft
			}
			ranges = append(ranges, [2]uint16{uint16(sids[lastStart]), uint16(nLeft)})
		}

		nLeftSize := 1
		if 256 <= maxNLeft {
			nLeftSize = 2
		}

		// write charset data for either format 0 or format 1/2, whichever is smaller
		charset = parse.NewBinaryWriter([]byte{})
		if 2*(numGlyphs-1) <= len(ranges)*(2+nLeftSize) {
			// format 0
			charset.WriteUint8(0)
			for _, sid := range sids {
				charset.WriteUint16(uint16(sid))
			}
		} else if nLeftSize == 1 {
			// format 1
			charset.WriteUint8(1)
			for _, ran := range ranges {
				charset.WriteUint16(ran[0])
				charset.WriteUint8(uint8(ran[1]))
			}
		} else {
			// format 2
			charset.WriteUint8(2)
			for _, ran := range ranges {
				charset.WriteUint16(ran[0])
				charset.WriteUint16(ran[1])
			}
		}
	} else if 229 < numGlyphs {
		return nil, fmt.Errorf("charset must be set explicitly for more than 229 glyphs")
	}

	stringINDEX, err := strings.Write()
	if err != nil {
		return nil, fmt.Errorf("String INDEX: %v", err)
	}

	if cff.globalSubrs == nil {
		cff.globalSubrs = &cffINDEX{}
	}
	globalSubrsINDEX, err := cff.globalSubrs.Write()
	if err != nil {
		return nil, fmt.Errorf("Global Subrs INDEX: %v", err)
	}

	if cff.charStrings == nil {
		cff.charStrings = &cffINDEX{}
	}
	charStringsINDEX, err := cff.charStrings.Write()
	if err != nil {
		return nil, fmt.Errorf("CharStrings INDEX: %v", err)
	}

	var privateDICT []byte
	var localSubrsINDEX []byte
	localSubrsOffset := 0
	if cff.fonts == nil {
		cff.fonts = &cffFontINDEX{}
	}
	if len(cff.fonts.private) != 0 {
		privateDICT, err = cff.fonts.private[0].Write()
		if err != nil {
			return nil, fmt.Errorf("Private DICT: %v", err)
		}

		if len(cff.fonts.localSubrs) != 0 {
			localSubrsINDEX, err = cff.fonts.localSubrs[0].Write()
			if err != nil {
				return nil, fmt.Errorf("Local Subrs INDEX: %v", err)
			}

			// write offset to Private DICT
			localSubrsOffset = len(privateDICT) + 1                  // key
			localSubrsOffset += cffDICTIntegerSize(localSubrsOffset) // val
			wPrivate := parse.NewBinaryWriter(privateDICT)
			writeDICTEntry(wPrivate, 19, localSubrsOffset)
			privateDICT = wPrivate.Bytes()
		}
	}

	// The offset values in Top DICT may push down the offsets themselves. Calculate the size
	// of Top DICT and thus the positions of each offset without the offset values in Top DICT
	// and then iteratively recalculate the offsets when added to Top DICT
	lenTopDICT := len(topDICT)
	if charset != nil {
		lenTopDICT += 1 // key
	}
	lenTopDICT += 1 // charStrings key
	if privateDICT != nil {
		lenTopDICT += 1 + cffDICTIntegerSize(len(privateDICT)) // key and size
	}
	lenTopDICTINDEXOffSize := cffINDEXOffSize(lenTopDICT)
	lenTopDICTINDEX := 3 + 2*lenTopDICTINDEXOffSize + lenTopDICT

	offset := int(w.Len()) + lenTopDICTINDEX + len(stringINDEX) // no overflow
	if math.MaxInt32-offset < len(globalSubrsINDEX) {
		return nil, fmt.Errorf("size too large")
	}
	offset += len(globalSubrsINDEX)

	charsetOffset := offset
	charsetLength := 0
	if charset != nil {
		charsetLength = int(charset.Len())
		if math.MaxInt32-charsetOffset < charsetLength {
			return nil, fmt.Errorf("size too large")
		}
	}
	charStringsOffset := charsetOffset + charsetLength
	if math.MaxInt32-charStringsOffset < len(charStringsINDEX) {
		return nil, fmt.Errorf("size too large")
	}
	privateOffset := charStringsOffset + len(charStringsINDEX)

	// correct for offset calculated above (grow Top DICT)
	correct, prevCorrect := 0, -1
	for correct != prevCorrect {
		prevCorrect = correct
		correct = 0
		if charset != nil {
			correct += cffDICTIntegerSize(charsetOffset + correct) // integer length in DICT
		}
		correct += cffDICTIntegerSize(charStringsOffset + correct) // integer length in DICT
		if privateDICT != nil {
			correct += cffDICTIntegerSize(privateOffset + correct) // integer length in DICT
		}
		correct += cffINDEXOffSize(lenTopDICT+correct) - lenTopDICTINDEXOffSize // offSize in INDEX
	}
	charsetOffset += correct
	charStringsOffset += correct
	privateOffset += correct

	// write offsets to Top DICT
	wTop := parse.NewBinaryWriter(topDICT)
	if charset != nil {
		writeDICTEntry(wTop, 15, charsetOffset)
	}
	writeDICTEntry(wTop, 17, charStringsOffset)
	if privateDICT != nil {
		writeDICTEntry(wTop, 18, len(privateDICT), privateOffset)
	}
	topDICT = wTop.Bytes()

	// create Top DICT INDEX
	top := &cffINDEX{}
	top.Add(topDICT)
	topDICTINDEX, err := top.Write()
	if err != nil {
		return nil, fmt.Errorf("Top DICT INDEX: %v", err)
	}

	// write out all data
	w.WriteBytes(topDICTINDEX)
	w.WriteBytes(stringINDEX)
	w.WriteBytes(globalSubrsINDEX)
	if charset != nil {
		w.WriteBytes(charset.Bytes())
	}
	w.WriteBytes(charStringsINDEX)
	if privateDICT != nil {
		w.WriteBytes(privateDICT)
	}
	w.WriteBytes(localSubrsINDEX)
	return w.Bytes(), nil
}
