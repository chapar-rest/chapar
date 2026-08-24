// Package fribidi is a Golang port of the C/C++
// Free Implementation of the Unicode Bidirectional Algorithm
// (https://github.com/fribidi/fribidi).
// It supports the main features required to do text layout.
// It depends on golang.org/x/text/unicode/bidi to fetch
// unicode properties of runes.
package fribidi

import (
	"fmt"

	"golang.org/x/text/unicode/bidi"
)

// controls whether we print debugMode info to stdout
const debugMode = false

// Level is the embedding level in a paragraph
type Level int8

// returns 0 or 1
func (lev Level) isRtl() Level { return lev & 1 }

func maxL(l1, l2 Level) Level {
	if l1 < l2 {
		return l2
	}
	return l1
}

const (
	/* The maximum embedding level value assigned by explicit marks */
	maxExplicitLevel = 125

	/* The maximum *number* of different resolved embedding levels: 0-126 */
	bidiMaxResolvedLevels = 127

	localBracketSize = 16

	/* The maximum *number* of nested brackets: 0-63 */
	maxNestedBracketPairs = 63
)

const (
	charLRM = 0x200E
	charRLM = 0x200F
	charLRE = 0x202A
	charRLE = 0x202B
	charPDF = 0x202C
	charLRO = 0x202D
	charRLO = 0x202E
	charLRI = 0x2066
	charRLI = 0x2067
	charFSI = 0x2068
	charPDI = 0x2069
)

/*
 * Define bit masks that bidi types are based on, each mask has
 * only one bit set.
 */
const (
	/* RTL mask better be the least significant bit. */
	maskRTL    = 0x00000001 /* Is right to left */
	maskARABIC = 0x00000002 /* Is arabic */

	/* Each char can be only one of the three following. */
	maskSTRONG   = 0x00000010 /* Is strong */
	maskWEAK     = 0x00000020 /* Is weak */
	maskNEUTRAL  = 0x00000040 /* Is neutral */
	maskSentinel = 0x00000080 /* Is sentinel */
	/* Sentinels are not valid chars, just identify the start/end of strings. */

	/* Each char can be only one of the six following. */
	maskLETTER    = 0x00000100 /* Is letter: L, R, AL */
	maskNUMBER    = 0x00000200 /* Is number: EN, AN */
	maskNUMSEPTER = 0x00000400 /* Is separator or terminator: ES, ET, CS */
	maskSPACE     = 0x00000800 /* Is space: BN, BS, SS, WS */
	maskEXPLICIT  = 0x00001000 /* Is explicit mark: LRE, RLE, LRO, RLO, PDF */
	maskISOLATE   = 0x00008000 /* Is isolate mark: LRI, RLI, FSI, PDI */

	/* Can be set only if maskSPACE is also set. */
	maskSEPARATOR = 0x00002000 /* Is text separator: BS, SS */
	/* Can be set only if maskEXPLICIT is also set. */
	maskOVERRIDE = 0x00004000 /* Is explicit override: LRO, RLO */
	maskFIRST    = 0x02000000 /* Whether direction is determined by first strong */

	/* The following exist to make types pairwise different, some of them can
	 * be removed but are here because of efficiency (make queries faster). */

	maskES = 0x00010000
	maskET = 0x00020000
	maskCS = 0x00040000

	maskNSM = 0x00080000
	maskBN  = 0x00100000

	maskBS = 0x00200000
	maskSS = 0x00400000
	maskWS = 0x00800000

	/* We reserve a single bit for user's private use: we will never use it. */
	maskPRIVATE = 0x01000000
)

type ParType = CharType

const (
	LTR  = maskSTRONG | maskLETTER
	RTL  = maskSTRONG | maskLETTER | maskRTL
	EN   = maskWEAK | maskNUMBER
	ON   = maskNEUTRAL
	WLTR = maskWEAK
	WRTL = maskWEAK | maskRTL
	PDF  = maskWEAK | maskEXPLICIT
	LRI  = maskNEUTRAL | maskISOLATE
	RLI  = maskNEUTRAL | maskISOLATE | maskRTL
	FSI  = maskNEUTRAL | maskISOLATE | maskFIRST
	BS   = maskNEUTRAL | maskSPACE | maskSEPARATOR | maskBS
	NSM  = maskWEAK | maskNSM
	AL   = maskSTRONG | maskLETTER | maskRTL | maskARABIC
	AN   = maskWEAK | maskNUMBER | maskARABIC
	CS   = maskWEAK | maskNUMSEPTER | maskCS
	ET   = maskWEAK | maskNUMSEPTER | maskET
	PDI  = maskNEUTRAL | maskWEAK | maskISOLATE // Pop Directional Isolate
	LRO  = maskSTRONG | maskEXPLICIT | maskOVERRIDE
	RLO  = maskSTRONG | maskEXPLICIT | maskOVERRIDE | maskRTL
	RLE  = maskSTRONG | maskEXPLICIT | maskRTL
	LRE  = maskSTRONG | maskEXPLICIT
	WS   = maskNEUTRAL | maskSPACE | maskWS
	ES   = maskWEAK | maskNUMSEPTER | maskES
	BN   = maskWEAK | maskSPACE | maskBN
	SS   = maskNEUTRAL | maskSPACE | maskSEPARATOR | maskSS
)

/* Return the minimum level of the direction, 0 for FRIBIDI_TYPE_LTR and
   1 for FRIBIDI_TYPE_RTL and FRIBIDI_TYPE_AL. */
func dirToLevel(dir ParType) Level {
	if dir.IsRtl() {
		return 1
	}
	return 0
}

/* Return the bidi type corresponding to the direction of the level number,
   FRIBIDI_TYPE_LTR for evens and FRIBIDI_TYPE_RTL for odds. */
func levelToDir(lev Level) CharType {
	if lev.isRtl() != 0 {
		return RTL
	}
	return LTR
}

/* Override status of an explicit mark:
 * LRO,LRE->LTR, RLO,RLE->RTL, otherwise->ON. */
func explicitToOverrideDir(p CharType) CharType {
	if p.isOverride() {
		return levelToDir(dirToLevel(p))
	}
	return ON
}

type CharType uint32

func (p CharType) String() string {
	switch p {
	case LTR:
		return "LTR"
	case RTL:
		return "RTL"
	case EN:
		return "EN"
	case ON:
		return "ON"
	case WLTR:
		return "WLTR"
	case WRTL:
		return "WRTL"
	case PDF:
		return "PDF"
	case LRI:
		return "LRI"
	case RLI:
		return "RLI"
	case FSI:
		return "FSI"
	case BS:
		return "BS"
	case NSM:
		return "NSM"
	case AL:
		return "AL"
	case AN:
		return "AN"
	case CS:
		return "CS"
	case ET:
		return "ET"
	case PDI:
		return "PDI"
	case LRO:
		return "LRO"
	case RLO:
		return "RLO"
	case RLE:
		return "RLE"
	case LRE:
		return "LRE"
	case WS:
		return "WS"
	case ES:
		return "ES"
	case BN:
		return "BN"
	case SS:
		return "SS"
	default:
		return fmt.Sprintf("<unknown type: %d>", p)
	}
}

// IsStrong checks if `p` is string.
func (p CharType) IsStrong() bool { return p&maskSTRONG != 0 }

// IsRtl checks is `p` is right to left: RTL, AL, RLE, RLO ?
func (p CharType) IsRtl() bool { return p&maskRTL != 0 }

// isNeutral checks is `p` is neutral ?
func (p CharType) isNeutral() bool { return p&maskNEUTRAL != 0 }

// IsLetter checks is `p` is letter : L, R, AL ?
func (p CharType) IsLetter() bool { return p&maskLETTER != 0 }

// IsNumber checks is `p` is number : EN, AN ?
func (p CharType) IsNumber() bool { return p&maskNUMBER != 0 }

// IsNumber checks is `p` is number  separator or terminator: ES, ET, CS ?
func (p CharType) isNumberSeparatorOrTerminator() bool { return p&maskNUMSEPTER != 0 }

// isExplicit checks is `p` is explicit  mark: LRE, RLE, LRO, RLO, PDF ?
func (p CharType) isExplicit() bool { return p&maskEXPLICIT != 0 }

// IsIsolate checks is `p` is isolator
func (p CharType) IsIsolate() bool { return p&maskISOLATE != 0 }

// IsText checks is `p` is text  separator: BS, SS ?
func (p CharType) isSeparator() bool { return p&maskSEPARATOR != 0 }

// IsExplicit checks is `p` is explicit  override: LRO, RLO ?
func (p CharType) isOverride() bool { return p&maskOVERRIDE != 0 }

// IsES checks is `p` is eS  or CS: ES, CS ?
func (p CharType) isEsOrCs() bool { return p&(maskES|maskCS) != 0 }

// IsExplicit checks is `p` is explicit  or BN: LRE, RLE, LRO, RLO, PDF, BN ?
func (p CharType) isExplicitOrBn() bool { return p&(maskEXPLICIT|maskBN) != 0 }

// IsExplicit checks is `p` is explicit  or BN or NSM: LRE, RLE, LRO, RLO, PDF, BN, NSM ?
func (p CharType) isExplicitOrBnOrNsm() bool { return p&(maskEXPLICIT|maskBN|maskNSM) != 0 }

// IsExplicit checks is `p` is explicit  or BN or NSM: LRE, RLE, LRO, RLO, PDF, BN, NSM ?
func (p CharType) isExplicitOrIsolateOrBnOrNsm() bool {
	return p&(maskEXPLICIT|maskISOLATE|maskBN|maskNSM) != 0
}

// IsExplicit checks is `p` is explicit  or BN or WS: LRE, RLE, LRO, RLO, PDF, BN, WS ?
func (p CharType) isExplicitOrBnOrWs() bool { return p&(maskEXPLICIT|maskBN|maskWS) != 0 }

// IsExplicit checks is `p` is explicit  or separator or BN or WS: LRE, RLE, LRO, RLO, PDF, BS, SS, BN, WS ?
func (p CharType) isExplicitOrSeparatorOrBnOrWs() bool {
	return p&(maskEXPLICIT|maskSEPARATOR|maskBN|maskWS) != 0
}

// the following are used outside the pacakge

// IsArabic checks if `p` is arabic: AL, AN?
func (p CharType) IsArabic() bool { return p&maskARABIC != 0 }

// IsWeak checks if `p` is weak
func (p CharType) IsWeak() bool { return p&maskWEAK != 0 }

//  Define some conversions.

/* Change numbers to RTL: EN,AN -> RTL. */
func (p CharType) changeNumberToRTL() CharType {
	if p.IsNumber() {
		return RTL
	}
	return p
}

// convert from golang enums to frididi types
func newCharType(class bidi.Class) CharType {
	switch class {
	case bidi.L: // LeftToRight
		return LTR
	case bidi.R: // RightToLeft
		return RTL
	case bidi.EN: // EuropeanNumber
		return EN
	case bidi.ES: // EuropeanSeparator
		return ES
	case bidi.ET: // EuropeanTerminator
		return ET
	case bidi.AN: // ArabicNumber
		return AN
	case bidi.CS: // CommonSeparator
		return CS
	case bidi.B: // ParagraphSeparator
		return BS
	case bidi.S: // SegmentSeparator
		return SS
	case bidi.WS: // WhiteSpace
		return WS
	case bidi.ON: // OtherNeutral
		return ON
	case bidi.BN: // BoundaryNeutral
		return BN
	case bidi.NSM: // NonspacingMark
		return NSM
	case bidi.AL: // ArabicLetter
		return AL
	case bidi.LRO: // LeftToRightOverride
		return LRO
	case bidi.RLO: // RightToLeftOverride
		return RLO
	case bidi.LRE: // LeftToRightEmbedding
		return LRE
	case bidi.RLE: // RightToLeftEmbedding
		return RLE
	case bidi.PDF: // PopDirectionalFormat
		return PDF
	case bidi.LRI: // LeftToRightIsolate
		return LRI
	case bidi.RLI: // RightToLeftIsolate
		return RLI
	case bidi.FSI: // FirstStrongIsolate
		return FSI
	case bidi.PDI: // PopDirectionalIsolate
		return PDI
	default:
		return LTR
	}
}

// GetBidiType returns the bidi type of a character as defined in Table 3.7
// Bidirectional Character Types of the Unicode Bidirectional Algorithm
// available at http://www.unicode.org/reports/tr9/#Bidirectional_Character_Types, using
// data provided by golang.org/x/text/unicode/bidi
func GetBidiType(ch rune) CharType {
	props, _ := bidi.LookupRune(ch)
	return newCharType(props.Class())
}

func getBidiTypes(str []rune) []CharType {
	out := make([]CharType, len(str))
	for i, r := range str {
		out[i] = GetBidiType(r)
	}
	return out
}

// Options is a flag to customize fribidi behaviour.
// The flags beginning with Shape affects the `Shape` function.
type Options int

// Define option flags that various functions use.
const (
	// Whether non-spacing marks for right-to-left parts of the text should be reordered to come after
	// their base characters in the visual string or not.
	// Most rendering engines expect this behavior, but console-based systems for example do not like it.
	// It is on in DefaultFlags.
	ReorderNSM Options = 1 << 1

	ShapeMirroring Options = 1      // in DefaultFlags, do mirroring
	ShapeArabPres  Options = 1 << 8 // in DefaultFlags, shape Arabic characters to their presentation form glyphs
	ShapeArabLiga  Options = 1 << 9 // in DefaultFlags, form mandatory Arabic ligatures

	// Perform additional Arabic shaping suitable for text rendered on
	// grid terminals with no mark rendering capabilities.
	// NOT SUPPORTED YET
	ShapeArabConsole Options = 1 << 10

	removeSpecials Options = 1 << 18

	// And their combinations

	baseDefault = ShapeMirroring | ReorderNSM | removeSpecials

	// recommended in any environment that doesn't have
	// other means for doing Arabic shaping.
	arabic = ShapeArabPres | ShapeArabLiga

	DefaultFlags = baseDefault | arabic
)

func (f Options) adjust(mask Options, cond bool) Options {
	if !cond {
		mask = 0
	}
	return (f & ^mask) | mask
}

// Visual is the visual output as specified by the Unicode Bidirectional Algorithm
type Visual struct {
	Str             []rune  // visual string
	VisualToLogical []int   // mapping from visual string back to the logical string indexes
	EmbeddingLevels []Level // list of embedding levels
}

// LogicalToVisual reverts `VisualToLogical`,
// return the mapping from logical to visual string indexes
func (v Visual) LogicalToVisual() []int {
	out := make([]int, len(v.VisualToLogical))
	for i, vToL := range v.VisualToLogical {
		out[vToL] = i
	}
	return out
}

// LogicalToVisual converts the logical input string to the visual output
// strings as specified by the Unicode Bidirectional Algorithm.  As a side
// effect it also generates mapping lists between the two strings, and the
// list of embedding levels as defined by the algorithm.
//
// Note that this function handles one-line paragraphs. For multi-
// paragraph texts it is necessary to first split the text into
// separate paragraphs and then carry over the resolved `paragraphBaseDir`
// between the subsequent invocations.
//
// The maximum level found plus one is also returned.
func LogicalToVisual(flags Options, str []rune, paragraphBaseDir *ParType /* requested and resolved paragraph base direction */) (Visual, Level) {
	bidiTypes := getBidiTypes(str)

	bracketTypes := getBracketTypes(str, bidiTypes)

	embeddingLevels, maxLevel := GetParEmbeddingLevels(bidiTypes, bracketTypes, paragraphBaseDir)

	/* Set up the ordering array to identity order */
	positionsVToL := make([]int, len(str))
	for i := range positionsVToL {
		positionsVToL[i] = i
	}

	visualStr := append([]rune{}, str...) // copy

	/* Arabic joining */
	arProps := getJoiningTypes(str, bidiTypes)
	joinArabic(bidiTypes, embeddingLevels, arProps)
	Shape(flags, embeddingLevels, arProps, visualStr)
	/* line breaking goes here, but we assume one line in this function */

	/* and this should be called once per line, but again, we assume one
	* line in this deprecated function */
	ReorderLine(flags, bidiTypes, len(str), 0, *paragraphBaseDir,
		embeddingLevels, visualStr, positionsVToL)

	return Visual{Str: visualStr, VisualToLogical: positionsVToL, EmbeddingLevels: embeddingLevels}, maxLevel
}

// RemoveBidiMarks removes the bidi and boundary-neutral marks out of an string
// and the accompanying lists.  It implements rule X9 of the Unicode
// Bidirectional Algorithm available at
// http://www.unicode.org/reports/tr9/#X9, with the exception that it removes
// U+200E LEFT-TO-RIGHT MARK and U+200F RIGHT-TO-LEFT MARK too.
//
// If any of the input lists are empty, the list is skipped.  If str is the
// visual string, then positions_to_this is positions_L_to_V and
// position_from_this_list is positions_V_to_L;  if str is the logical
// string, the other way. Moreover, the position maps should be filled with
// valid entries.
//
// A position map pointing to a removed character is filled with \(mi1. By the
// way, you should not use embedding_levels if str is visual string.
//
// For best results this function should be run on a whole paragraph, not
// lines; but feel free to do otherwise if you know what you are doing.
//
// The input slice is mutated and resliced to its new length, then returned
func RemoveBidiMarks(str []rune, positionsToThis, positionFromThis []int, embeddingLevels []Level) []rune {
	/* If to_this is not NULL, we must have from_this as well. If it is
	not given by the caller, we have to make a private instance of it. */
	if len(positionsToThis) != 0 && len(positionFromThis) == 0 {
		positionFromThis = make([]int, len(str))
		for i, to := range positionsToThis {
			positionFromThis[to] = i
		}
	}

	hasTo := len(positionsToThis) != 0
	hasLevels := len(embeddingLevels) != 0
	hasFrom := len(positionFromThis) != 0

	var j int
	for i, r := range str {
		if bType := GetBidiType(r); !bType.isExplicitOrBn() && !bType.IsIsolate() &&
			r != charLRM && r != charRLM {
			str[j] = r
			if hasLevels {
				embeddingLevels[j] = embeddingLevels[i]
			}
			if hasFrom {
				positionFromThis[j] = positionFromThis[i]
			}
			j++
		}
	}

	/* Convert the from_this list to to_this */
	if hasTo {
		for i := range positionsToThis {
			positionsToThis[i] = -1
		}
		for i, from := range positionFromThis {
			positionsToThis[from] = i
		}
	}
	return str[0:j]
}
