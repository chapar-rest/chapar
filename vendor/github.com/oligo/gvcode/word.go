package gvcode

import (
	"slices"
	"strings"
	"unicode"
)

const (
	// defaultWordSeperators defines a default set of word seperators. It is
	// used when no custom word seperators are set.
	defaultWordSeperators = "`~!@#$%^&*()-=+[{]}\\|;:'\",.<>/?"
)

// isWordSeperator check r to see if it is a word seperator. A word seperator
// set the boundary when navigating by words, or deleting by words.
// TODO: does it make sence to use unicode space definition here?
func (e *textView) isWordSeperator(r rune) bool {
	seperators := e.WordSeperators

	if e.WordSeperators == "" {
		seperators = defaultWordSeperators
	}

	return strings.ContainsRune(seperators, r) || unicode.IsSpace(r)

}

// MoveWord moves the caret to the next few words in the specified direction.
// Positive is forward, negative is backward.
// The final caret position will be aligned to a grapheme cluster boundary.
func (e *textView) MoveWords(distance int, selAct selectionAction) {
	// split the distance information into constituent parts to be
	// used independently.
	words, direction := distance, 1
	if distance < 0 {
		words, direction = distance*-1, -1
	}
	// atEnd if caret is at either side of the buffer.
	caret := e.closestToRune(e.caret.start)
	atEnd := func() bool {
		return caret.Runes == 0 || caret.Runes == e.Len()
	}
	// next returns the appropriate rune given the direction.
	next := func() (r rune) {
		if direction < 0 {
			r, _ = e.src.ReadRuneAt(caret.Runes - 1)
		} else {
			r, _ = e.src.ReadRuneAt(caret.Runes)
		}
		return r
	}
	for ii := 0; ii < words; ii++ {
		for r := next(); e.isWordSeperator(r) && !atEnd(); r = next() {
			e.MoveCaret(direction, 0)
			caret = e.closestToRune(e.caret.start)
		}
		e.MoveCaret(direction, 0)
		caret = e.closestToRune(e.caret.start)
		for r := next(); !e.isWordSeperator(r) && !atEnd(); r = next() {
			e.MoveCaret(direction, 0)
			caret = e.closestToRune(e.caret.start)
		}
	}
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

// ReadWord tries to read one word nearby the caret, returning the word if there's one,
// and the offset of the caret in the word.
//
// The word boundary is checked using the same conditions as MoveWords.
func (e *textView) ReadWord(bySpace bool) (string, int) {
	caret := e.closestToRune(e.caret.start)
	buf := make([]rune, 0)

	seperator := func(r rune) bool {
		if bySpace {
			return unicode.IsSpace(r)
		}
		return e.isWordSeperator(r)
	}

	scanWord := func(direction int, caretOff int) int {
		var prev = caretOff
		for {
			if caretOff < 0 || caretOff > e.Len() {
				break
			}

			r, err := e.src.ReadRuneAt(caretOff)
			if seperator(r) || err != nil {
				break
			}

			if direction < 0 {
				buf = slices.Insert(buf, 0, r)
				caretOff--
			} else {
				buf = append(buf, r)
				caretOff++
			}
		}

		return prev - caretOff
	}

	moves := scanWord(-1, caret.Runes-1)
	scanWord(1, caret.Runes)

	return string(buf), moves
}
