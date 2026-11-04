package ui

import (
	"unicode"
	"unicode/utf8"
)

// Caret movement by word, line and document, and deleting by word. Each
// runs at every selection; the keymap binds them per platform.

// wordLeft is where moving one word left from off lands: past spaces, then
// past a run of word characters or of punctuation. A line start is a stop
// of its own, so the move never jumps over a line break and a word at once.
func (e *Editor) wordLeft(off int) int {
	b := e.pt.Bytes()
	if off <= 0 {
		return 0
	}
	if b[off-1] == '\n' {
		return off - 1
	}
	i := off
	for i > 0 {
		r, sz := utf8.DecodeLastRune(b[:i])
		if r == '\n' || !unicode.IsSpace(r) {
			break
		}
		i -= sz
	}
	if i == 0 || b[i-1] == '\n' {
		return i
	}
	r, _ := utf8.DecodeLastRune(b[:i])
	cls := classOf(r)
	for i > 0 {
		r, sz := utf8.DecodeLastRune(b[:i])
		if r == '\n' || classOf(r) != cls {
			break
		}
		i -= sz
	}
	return i
}

// wordRight is wordLeft's mirror: past spaces, then to the end of the word or
// punctuation run.
func (e *Editor) wordRight(off int) int {
	b := e.pt.Bytes()
	n := len(b)
	if off >= n {
		return n
	}
	if b[off] == '\n' {
		return off + 1
	}
	i := off
	for i < n {
		r, sz := utf8.DecodeRune(b[i:])
		if r == '\n' || !unicode.IsSpace(r) {
			break
		}
		i += sz
	}
	if i == n || b[i] == '\n' {
		return i
	}
	r, _ := utf8.DecodeRune(b[i:])
	cls := classOf(r)
	for i < n {
		r, sz := utf8.DecodeRune(b[i:])
		if r == '\n' || classOf(r) != cls {
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
