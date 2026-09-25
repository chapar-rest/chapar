package ui

import (
	"time"
	"unicode"
	"unicode/utf8"
)

// undoMergeGap is the longest pause between keystrokes that still merges them
// into one undo step.
const undoMergeGap = time.Second

// editMerge says whether an edit may merge into the previous undo step, and
// with which kind of neighbour.
type editMerge uint8

const (
	mergeNone      editMerge = iota // always a step of its own (paste, cut, newline, …)
	mergeType                       // typed text, extending forward
	mergeBackspace                  // single-cluster delete before the caret
	mergeDelete                     // single-cluster delete after the caret
)

// startsWord reports whether typing next right after prev begins a new word,
// which starts a new undo step so undo walks back one word at a time.
func startsWord(prev, next string) bool {
	p, _ := utf8.DecodeLastRuneInString(prev)
	n, _ := utf8.DecodeRuneInString(next)
	return unicode.IsSpace(p) && !unicode.IsSpace(n)
}
