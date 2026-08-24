package fribidi

import (
	"unicode"

	ucd "github.com/benoitkugler/textlayout/unicodedata"
)

type JoiningType uint8

const (
	U JoiningType = 0                                   /* Un-joining, e.g. Full Stop */
	R JoiningType = joinsRight | arabShapes             /* Right-joining, e.g. Arabic Letter Dal */
	D JoiningType = joinsRight | joinsLeft | arabShapes /* Dual-joining, e.g. Arabic Letter Ain */
	C JoiningType = joinsRight | joinsLeft              /* Join-Causing, e.g. Tatweel, ZWJ */
	L JoiningType = joinsLeft | arabShapes              /* Left-joining, i.e. fictional */
	T JoiningType = transparent | arabShapes            /* Transparent, e.g. Arabic Fatha */
	G JoiningType = ignored                             /* Ignored, e.g. LRE, RLE, ZWNBSP */
)

// Define bit masks that joining types are based on
const (
	joinsRight  = 1 << iota // may join to right
	joinsLeft               // may join to left
	arabShapes              // may Arabic shape
	transparent             // is transparent
	ignored                 // is ignored
	ligatured               // is ligatured
)

/* iGnored */
func (p JoiningType) isG() bool {
	return ignored == p&(transparent|ignored)
}

/* Is skipped in joining: T, G? */
func (p JoiningType) isJoinSkipped() bool {
	return p&(transparent|ignored) != 0
}

/* May shape: R, D, L, T? */
func (p JoiningType) isArabShapes() bool {
	return p&arabShapes != 0
}

// the return verifies 0 <= s < 4
func (p JoiningType) joinShape() uint8 {
	return uint8(p & (joinsRight | joinsLeft))
}

func getJoiningType(ch rune, bidi CharType) JoiningType {
	if jt, ok := ucd.ArabicJoinings[ch]; ok {
		switch jt {
		case ucd.U:
			return U
		case ucd.R, ucd.Alaph, ucd.DalathRish:
			return R
		case ucd.D:
			return D
		case ucd.C:
			return C
		case ucd.T:
			return T
		case ucd.L:
			return L
		case ucd.G:
			return G
		}
	}
	// (general transparents) Those that are not explicitly listed and that are of General Category
	// Mn, Me, or Cf have joining type T.
	if unicode.In(ch, unicode.Mn, unicode.Me, unicode.Cf) {
		return T
	}
	// ignored bidi types
	switch bidi {
	case BN, LRE, RLE, LRO, RLO, PDF, LRI, RLI, FSI, PDI:
		return G
	default:
		// all others not explicitly listed have joining type U.
		return U
	}
}

func getJoiningTypes(str []rune, bidiTypes []CharType) []JoiningType {
	out := make([]JoiningType, len(str))
	for i, r := range str {
		out[i] = getJoiningType(r, bidiTypes[i])
	}
	return out
}

// joinArabic does the Arabic joining algorithm.  Means, given Arabic
// joining types of the characters in ar_props, this
// function modifies (in place) this properties to grasp the effect of neighboring
// characters. You probably need this information later to do Arabic shaping.
//
// This function implements rules R1 to R7 inclusive (all rules) of the Arabic
// Cursive Joining algorithm of the Unicode standard as available at
// http://www.unicode.org/versions/Unicode4.0.0/ch08.pdf#G7462.  It also
// interacts correctly with the bidirection algorithm as defined in Section
// 3.5 Shaping of the Unicode Bidirectional Algorithm available at
// http://www.unicode.org/reports/tr9/#Shaping.
func joinArabic(bidiTypes []CharType, embeddingLevels []Level, arProps []JoiningType) {
	/* The joining algorithm turned out very very dirty :(.  That's what happens
	 * when you follow the standard which has never been implemented closely
	 * before.
	 */

	/* 8.2 Arabic - Cursive Joining */
	var (
		saved                         = 0
		savedLevel              Level = levelSentinel
		savedShapes                   = false
		savedJoinsFollowingMask JoiningType
		joins                   = false
	)
	for i := range arProps {
		if !arProps[i].isG() {
			disjoin := false
			shapes := arProps[i].isArabShapes()

			//  FRIBIDI_CONSISTENT_LEVEL
			var level Level = levelSentinel
			if !bidiTypes[i].isExplicitOrBn() {
				level = embeddingLevels[i]
			}

			if levelMatch := savedLevel == level || savedLevel == levelSentinel || level == levelSentinel; joins && !levelMatch {
				disjoin = true
				joins = false
			}
			if !arProps[i].isJoinSkipped() {
				var joinsPrecedingMask JoiningType = joinsLeft
				if level.isRtl() != 0 {
					joinsPrecedingMask = joinsRight
				}

				if !joins {
					if shapes {
						arProps[i] &= ^joinsPrecedingMask // unset bits
					}
				} else if arProps[i]&joinsPrecedingMask == 0 { // ! test bits
					disjoin = true
				} else {
					/* This is a FriBidi extension:  we set joining properties
					 * for skipped characters in between, so we can put NSMs on tatweel
					 * later if we want.  Useful on console for example.
					 */
					for j := saved + 1; j < i; j++ {
						arProps[j] |= joinsPrecedingMask | savedJoinsFollowingMask
					}
				}
			}

			if disjoin && savedShapes {
				arProps[saved] &= ^savedJoinsFollowingMask // unset bits
			}

			if !arProps[i].isJoinSkipped() {
				saved = i
				savedLevel = level
				savedShapes = shapes
				// FRIBIDI_JOINS_FOLLOWING_MASK(level)
				if level.isRtl() != 0 {
					savedJoinsFollowingMask = joinsLeft
				} else {
					savedJoinsFollowingMask = joinsRight
				}
				joins = arProps[i]&savedJoinsFollowingMask != 0
			}
		}
	}
	if joins && savedShapes {
		arProps[saved] &= ^savedJoinsFollowingMask
	}
}
