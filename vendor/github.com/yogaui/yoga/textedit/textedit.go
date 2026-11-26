// Package textedit holds the editing rules shared by Yoga's text widgets:
// word boundaries, multi-click timing and how keystrokes group into undo
// steps. ui.TextInput uses it, and so can any text widget built outside the
// ui package, so that every text field in an app moves, selects and undoes
// the same way.
package textedit

import (
	"time"
	"unicode"
	"unicode/utf8"
)

// DoubleClickInterval is the longest gap between presses that still counts as
// part of the same multi-click sequence; MultiClickSlop bounds how far, in
// pixels, the pointer may move between them.
const (
	DoubleClickInterval = 400 * time.Millisecond
	MultiClickSlop      = 4
)

// Class groups runes for word boundaries.
type Class int

const (
	ClassSpace Class = iota // whitespace
	ClassWord               // letters, digits and '_'
	ClassOther              // punctuation and symbols
)

// ClassOf returns the class of r.
func ClassOf(r rune) Class {
	switch {
	case unicode.IsSpace(r):
		return ClassSpace
	case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
		return ClassWord
	default:
		return ClassOther
	}
}

// IsBreak reports whether r ends a line: '\n', or the '\r' of a "\r\n".
func IsBreak(r rune) bool { return r == '\n' || r == '\r' }

// WordLeft is where moving one word left from off in b lands: past spaces,
// then past a run of word characters or of punctuation. A line start is a
// stop of its own, so the move never jumps over a line break and a word at
// once.
func WordLeft(b []byte, off int) int {
	if off <= 0 {
		return 0
	}
	if b[off-1] == '\n' {
		if off >= 2 && b[off-2] == '\r' {
			return off - 2
		}
		return off - 1
	}
	i := off
	for i > 0 {
		r, sz := utf8.DecodeLastRune(b[:i])
		if IsBreak(r) || !unicode.IsSpace(r) {
			break
		}
		i -= sz
	}
	if i == 0 || IsBreak(rune(b[i-1])) {
		return i
	}
	r, _ := utf8.DecodeLastRune(b[:i])
	cls := ClassOf(r)
	for i > 0 {
		r, sz := utf8.DecodeLastRune(b[:i])
		if IsBreak(r) || ClassOf(r) != cls {
			break
		}
		i -= sz
	}
	return i
}

// WordRight is WordLeft's mirror: past spaces, then to the end of the word
// or punctuation run.
func WordRight(b []byte, off int) int {
	n := len(b)
	if off >= n {
		return n
	}
	if b[off] == '\n' {
		return off + 1
	}
	if b[off] == '\r' && off+1 < n && b[off+1] == '\n' {
		return off + 2
	}
	i := off
	for i < n {
		r, sz := utf8.DecodeRune(b[i:])
		if IsBreak(r) || !unicode.IsSpace(r) {
			break
		}
		i += sz
	}
	if i == n || IsBreak(rune(b[i])) {
		return i
	}
	r, _ := utf8.DecodeRune(b[i:])
	cls := ClassOf(r)
	for i < n {
		r, sz := utf8.DecodeRune(b[i:])
		if IsBreak(r) || ClassOf(r) != cls {
			break
		}
		i += sz
	}
	return i
}

// UndoMergeGap is the longest pause between keystrokes that still merges them
// into one undo step.
const UndoMergeGap = time.Second

// Merge says whether an edit may merge into the previous undo step, and with
// which kind of neighbour.
type Merge uint8

const (
	MergeNone      Merge = iota // always a step of its own (paste, cut, newline, …)
	MergeType                   // typed text, extending forward
	MergeBackspace              // single-cluster delete before the caret
	MergeDelete                 // single-cluster delete after the caret
)

// StartsWord reports whether typing next right after prev begins a new word,
// which starts a new undo step so undo walks back one word at a time.
func StartsWord(prev, next string) bool {
	p, _ := utf8.DecodeLastRuneInString(prev)
	n, _ := utf8.DecodeRuneInString(next)
	return unicode.IsSpace(p) && !unicode.IsSpace(n)
}
