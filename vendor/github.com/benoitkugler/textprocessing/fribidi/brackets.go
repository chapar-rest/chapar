package fribidi

import (
	"golang.org/x/text/unicode/bidi"
)

// BracketType is a rune value with its MSB is used to indicate an opening bracket
type BracketType uint32

func (bt BracketType) isOpen() bool {
	return bt&bracketOpenMask > 0
}

func (bt BracketType) id() BracketType {
	return bt & bracketIDMask
}

const (
	NoBracket       BracketType = 0
	bracketOpenMask BracketType = 1 << 31
	bracketIDMask               = ^bracketOpenMask
)

// GetBracket finds the bracketed equivalent of a character as defined in
// the file BidiBrackets.txt of the Unicode Character Database available at
// http://www.unicode.org/Public/UNIDATA/BidiBrackets.txt.
//
// If the input character is declared as a brackets character in the
// Unicode standard and has a bracketed equivalent, the matching bracketed
// character is returned, with its high bit set.
// Otherwise zero is returned.
func GetBracket(ch rune) BracketType {
	props, _ := bidi.LookupRune(ch)
	if !props.IsBracket() {
		return NoBracket
	}
	pair := BracketType(bracketsTable[ch])
	pair &= bracketIDMask
	if props.IsOpeningBracket() {
		pair |= bracketOpenMask
	}
	return pair
}

// getBracketTypes finds the bracketed characters of an string of characters.
// `bidiTypes` is not needed strictly speaking, but is used as an optimization.
// see GetBracket for details.
func getBracketTypes(str []rune, bidiTypes []CharType) []BracketType {
	out := make([]BracketType, len(str))
	for i, r := range str {
		// Note the optimization that a bracket is always of type neutral
		if bidiTypes[i] == ON {
			out[i] = GetBracket(r)
		}
		// else -> zero
	}
	return out
}
