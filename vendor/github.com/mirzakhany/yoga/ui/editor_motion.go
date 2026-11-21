package ui

import (
	"unicode"
	"unicode/utf8"
)

// Caret movement by word, line and document, and deleting by word. Each
// runs at every selection; the keymap binds them per platform.

func (e *Editor) wordLeft(off int) int  { return wordLeftIn(e.pt.Bytes(), off) }
func (e *Editor) wordRight(off int) int { return wordRightIn(e.pt.Bytes(), off) }

// wordLeftIn is where moving one word left from off in b lands: past spaces,
// then past a run of word characters or of punctuation. A line start is a
// stop of its own, so the move never jumps over a line break and a word at
// once.
func wordLeftIn(b []byte, off int) int {
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
		if isBreak(r) || !unicode.IsSpace(r) {
			break
		}
		i -= sz
	}
	if i == 0 || isBreak(rune(b[i-1])) {
		return i
	}
	r, _ := utf8.DecodeLastRune(b[:i])
	cls := classOf(r)
	for i > 0 {
		r, sz := utf8.DecodeLastRune(b[:i])
		if isBreak(r) || classOf(r) != cls {
			break
		}
		i -= sz
	}
	return i
}

// isBreak reports whether r ends a line: '\n', or the '\r' of a "\r\n".
func isBreak(r rune) bool { return r == '\n' || r == '\r' }

// wordRightIn is wordLeftIn's mirror: past spaces, then to the end of the
// word or punctuation run.
func wordRightIn(b []byte, off int) int {
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
		if isBreak(r) || !unicode.IsSpace(r) {
			break
		}
		i += sz
	}
	if i == n || isBreak(rune(b[i])) {
		return i
	}
	r, _ := utf8.DecodeRune(b[i:])
	cls := classOf(r)
	for i < n {
		r, sz := utf8.DecodeRune(b[i:])
		if isBreak(r) || classOf(r) != cls {
			break
		}
		i += sz
	}
	return i
}

// lineStartSmart is where Home goes: the first non-blank character of the
// line, or column 0 when the caret is already there (or the line is blank).
// On a wrapped continuation row it goes to the row's start first.
func (e *Editor) lineStartSmart(off int) int {
	ln := e.lineOf(off)
	ls := e.pt.LineStart(ln)
	if e.SoftWrap && len(e.rowPrefix) > 1 {
		e.freshWrap()
		if _, rs, _ := e.wrapRowAbs(e.rowOfByte(off)); rs > ls && off != rs {
			return rs
		}
	}
	line := e.pt.Line(ln)
	ind := 0
	for ind < len(line) && (line[ind] == ' ' || line[ind] == '\t') {
		ind++
	}
	if ind == len(line) || off == ls+ind {
		return ls
	}
	return ls + ind
}

// lineEndOf is the end of off's line, before its newline.
func (e *Editor) lineEndOf(off int) int {
	ln := e.lineOf(off)
	return e.pt.LineStart(ln) + e.pt.LineLen(ln)
}

// moveAction moves every selection to where(caret), extending them with
// extend.
func moveAction(where func(e *Editor, off int) int, extend bool) func(e *Editor) bool {
	return func(e *Editor) bool {
		step := func() { e.moveTo(where(e, e.caret), extend) }
		if len(e.extra) > 0 && !e.inMulti {
			e.multi(false, step)
		} else {
			step()
		}
		return true
	}
}

// deleteAction deletes from each caret to where(caret), or the selection
// where there is one, as one undo step.
func deleteAction(where func(e *Editor, off int) int) func(e *Editor) bool {
	return func(e *Editor) bool {
		if e.readOnly {
			return true
		}
		step := func() {
			if e.deleteSelection() {
				return
			}
			to := where(e, e.caret)
			lo, hi := min(to, e.caret), max(to, e.caret)
			if lo < hi {
				e.applyEdit(lo, hi-lo, "", mergeNone)
			}
		}
		if len(e.extra) > 0 && !e.inMulti {
			e.multi(true, step)
		} else {
			step()
		}
		return true
	}
}

func init() {
	wl := func(e *Editor, off int) int { return e.wordLeft(off) }
	wr := func(e *Editor, off int) int { return e.wordRight(off) }
	ls := func(e *Editor, off int) int { return e.lineStartSmart(off) }
	le := func(e *Editor, off int) int { return e.lineEndOf(off) }
	ll := func(e *Editor, off int) int { return e.pt.LineStart(e.lineOf(off)) }
	fs := func(*Editor, int) int { return 0 }
	fe := func(e *Editor, _ int) int { return e.pt.Len() }
	for act, fn := range map[EditorAction]func(e *Editor) bool{
		ActionCursorWordLeft:        moveAction(wl, false),
		ActionCursorWordLeftSelect:  moveAction(wl, true),
		ActionCursorWordRight:       moveAction(wr, false),
		ActionCursorWordRightSelect: moveAction(wr, true),
		ActionCursorLineStart:       moveAction(ls, false),
		ActionCursorLineStartSelect: moveAction(ls, true),
		ActionCursorLineEnd:         moveAction(le, false),
		ActionCursorLineEndSelect:   moveAction(le, true),
		ActionCursorFileStart:       moveAction(fs, false),
		ActionCursorFileStartSelect: moveAction(fs, true),
		ActionCursorFileEnd:         moveAction(fe, false),
		ActionCursorFileEndSelect:   moveAction(fe, true),
		ActionDeleteWordLeft:        deleteAction(wl),
		ActionDeleteWordRight:       deleteAction(wr),
		ActionDeleteLineLeft:        deleteAction(ll),
	} {
		editorActions[act] = fn
	}
}
