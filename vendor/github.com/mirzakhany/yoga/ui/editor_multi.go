package ui

import (
	"bytes"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/input"
)

// Multiple selections. The editor's caret and selAnchor stay the primary
// selection, which everything single-caret (search, completion, hover,
// scrolling) follows; extra holds the others. An action with several
// selections runs once per selection, from the last in the document to the
// first, so an edit never moves a selection not yet visited, and the
// visited ones shift by the edit's size.

// cursor is one extra selection: a caret and its anchor, -1 for none.
type cursor struct{ caret, anchor int }

func (c cursor) lo() int {
	if c.anchor >= 0 && c.anchor < c.caret {
		return c.anchor
	}
	return c.caret
}

func (c cursor) hi() int {
	if c.anchor > c.caret {
		return c.anchor
	}
	return c.caret
}

// Cursors is how many selections there are, 1 or more.
func (e *Editor) Cursors() int { return 1 + len(e.extra) }

// Selections returns every selection as [start, end) ranges in document
// order. An empty range is a bare caret.
func (e *Editor) Selections() [][2]int {
	out := make([][2]int, 0, e.Cursors())
	for _, c := range e.allCursors() {
		out = append(out, [2]int{c.lo(), c.hi()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

// ClearCursors drops every selection but the primary one.
func (e *Editor) ClearCursors() { e.extra = nil }

// AddCursorAt adds a caret at byte offset off, keeping the others.
func (e *Editor) AddCursorAt(off int) {
	off = clampInt(off, 0, e.pt.Len())
	e.extra = append(e.extra, cursor{e.caret, e.selAnchor})
	e.caret, e.selAnchor = off, -1
	e.normalizeCursors()
	e.ensureCaretVisible()
}

// AddCursorLine adds a caret on the row above (dir -1) or below (dir 1)
// the outermost caret that way. Successive adds keep to the x of the caret
// they started from, so a column of carets survives short lines.
func (e *Editor) AddCursorLine(dir int) {
	all := e.allCursors()
	edge := all[0]
	for _, c := range all {
		if (dir < 0 && c.caret < edge.caret) || (dir > 0 && c.caret > edge.caret) {
			edge = c
		}
	}
	if len(e.extra) == 0 {
		e.addGoalX = e.xOf(e.caret)
	}
	off, ok := e.verticalFrom(edge.caret, dir, e.addGoalX)
	if !ok {
		return
	}
	e.AddCursorAt(off)
}

// xOf is the x of offset off within its row, as Up/Down measure it.
func (e *Editor) xOf(off int) float32 {
	if e.SoftWrap && len(e.rowPrefix) > 1 {
		e.freshWrap()
		return e.xInRow(off)
	}
	return e.caretXInLine(e.lineOf(off), off)
}

// verticalFrom is where Up (dir -1) or Down (dir 1) from off lands keeping to
// x, and whether it moved to another row. The editor's own caret and
// selection are left as they were.
func (e *Editor) verticalFrom(off, dir int, x float32) (int, bool) {
	caret, anchor, gx, gs, sx, sy := e.caret, e.selAnchor, e.goalX, e.goalSet, e.ScrollPx, e.ScrollX
	extra := e.extra
	e.extra = nil
	e.caret, e.selAnchor = off, -1
	e.goalX, e.goalSet = x, true
	from := e.rowAt(off)
	e.moveVertical(dir, false)
	to, moved := e.caret, e.rowAt(e.caret) != from
	e.caret, e.selAnchor, e.goalX, e.goalSet, e.ScrollPx, e.ScrollX = caret, anchor, gx, gs, sx, sy
	e.extra = extra
	return to, moved
}

// rowAt is the visual row of off: its line, or with soft wrap its wrapped row.
func (e *Editor) rowAt(off int) int {
	if e.SoftWrap && len(e.rowPrefix) > 1 {
		return e.rowOfByte(off)
	}
	return e.lineOf(off)
}

// boxSelect is a column selection being dragged with ⌥⇧.
type boxSelect struct {
	on       bool
	line     int     // where the drag started
	x        float32 // text x where it started
	lastLine int
	lastX    float32
}

// textX is the x of a point relative to the start of the text, scrolled.
func (e *Editor) textX(px float32) float32 {
	f := e.viewport.Frame
	return px - (f.X + e.gutterW + e.textPad - e.ScrollX)
}

func (e *Editor) startBoxSelect(px, py float32) {
	ln := e.lineOf(e.offsetAtPoint(px, py))
	e.box = boxSelect{on: true, line: ln, x: e.textX(px), lastLine: -1}
	e.updateBoxSelect(px, py)
}

// updateBoxSelect selects, on every line between the drag's start and the
// pointer, the columns between their two x positions. A line too short for
// the column gets a caret at its end.
func (e *Editor) updateBoxSelect(px, py float32) {
	b := &e.box
	ln, x := e.lineOf(e.offsetAtPoint(px, py)), e.textX(px)
	if ln == b.lastLine && x == b.lastX {
		return
	}
	b.lastLine, b.lastX = ln, x
	step := 1
	if ln < b.line {
		step = -1
	}
	var cs []cursor
	for l := ln; ; l -= step {
		anchor := e.offsetAtLineX(l, b.x)
		caret := e.offsetAtLineX(l, x)
		if anchor == caret {
			anchor = -1
		}
		cs = append(cs, cursor{caret: caret, anchor: anchor})
		if l == b.line {
			break
		}
	}
	e.caret, e.selAnchor = cs[0].caret, cs[0].anchor // the pointer's line is primary
	e.extra = append(e.extra[:0], cs[1:]...)
	e.normalizeCursors()
	e.dragging = false
	e.lastMerge = mergeNone
	e.goalSet = false
	e.blinkStart = time.Now()
}

// offsetAtLineX is the offset on line ln nearest text x.
func (e *Editor) offsetAtLineX(ln int, x float32) int {
	ls := e.pt.LineStart(ln)
	if e.SoftWrap || e.isHuge(ln) {
		view := e.lineView(ln)
		return ls + e.monoByteForX(view, x)
	}
	return ls + e.shapedLine(ln).ByteForX(x)
}

// AddNextOccurrence selects the word at the caret, or, with text selected,
// adds the next occurrence of it as another selection (⌘D).
func (e *Editor) AddNextOccurrence() {
	if !e.hasSelection() {
		lo, hi := e.wordRangeAt(e.caret)
		if lo < hi {
			e.selAnchor, e.caret = lo, hi
		}
		return
	}
	lo, hi := e.selRange()
	needle := e.pt.Bytes()[lo:hi]
	doc := e.pt.Bytes()
	// Search after the last selection of it, wrapping to the start.
	from := hi
	for _, c := range e.extra {
		if c.hi() > from {
			from = c.hi()
		}
	}
	taken := map[int]bool{lo: true}
	for _, c := range e.extra {
		taken[c.lo()] = true
	}
	for _, start := range []int{from, 0} {
		for at := start; at <= len(doc); {
			i := bytes.Index(doc[at:], needle)
			if i < 0 {
				break
			}
			pos := at + i
			if !taken[pos] {
				e.extra = append(e.extra, cursor{e.caret, e.selAnchor})
				e.selAnchor, e.caret = pos, pos+len(needle)
				e.normalizeCursors()
				e.ensureCaretVisible()
				return
			}
			at = pos + max(1, len(needle))
		}
	}
}

// SelectAllOccurrences selects every occurrence of the selection, or of the
// word at the caret (⌘⇧L).
func (e *Editor) SelectAllOccurrences() {
	if !e.hasSelection() {
		e.AddNextOccurrence()
		if !e.hasSelection() {
			return
		}
	}
	lo, hi := e.selRange()
	needle := e.pt.Bytes()[lo:hi]
	doc := e.pt.Bytes()
	e.extra = e.extra[:0]
	for at := 0; at <= len(doc); {
		i := bytes.Index(doc[at:], needle)
		if i < 0 {
			break
		}
		pos := at + i
		if pos != lo {
			e.extra = append(e.extra, cursor{pos + len(needle), pos})
		}
		at = pos + len(needle)
	}
	e.normalizeCursors()
}

func (e *Editor) allCursors() []cursor {
	return append([]cursor{{e.caret, e.selAnchor}}, e.extra...)
}

// normalizeCursors drops extra selections that repeat or overlap another,
// merging overlapping ranges into the one that comes first.
func (e *Editor) normalizeCursors() {
	if len(e.extra) == 0 {
		return
	}
	n := e.pt.Len()
	all := e.allCursors()
	for i := range all {
		all[i].caret = clampInt(all[i].caret, 0, n)
		if all[i].anchor >= 0 {
			all[i].anchor = clampInt(all[i].anchor, 0, n)
		}
	}
	primary := all[0]
	idx := make([]int, len(all))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return all[idx[a]].lo() < all[idx[b]].lo() })
	kept := []cursor{}
	keptPrimary := -1
	for _, i := range idx {
		c := all[i]
		if k := len(kept) - 1; k >= 0 && (c.lo() < kept[k].hi() || c.lo() == kept[k].lo()) {
			// Overlaps the previous: grow it to cover both.
			p := kept[k]
			lo, hi := min(p.lo(), c.lo()), max(p.hi(), c.hi())
			if lo == hi {
				kept[k] = cursor{lo, -1}
			} else {
				kept[k] = cursor{hi, lo}
			}
			if i == 0 {
				keptPrimary = k
			}
			continue
		}
		if i == 0 {
			keptPrimary = len(kept)
		}
		kept = append(kept, c)
	}
	if keptPrimary < 0 {
		keptPrimary = 0
		kept[0] = primary
	}
	e.caret, e.selAnchor = kept[keptPrimary].caret, kept[keptPrimary].anchor
	e.extra = e.extra[:0]
	for i, c := range kept {
		if i != keptPrimary {
			e.extra = append(e.extra, c)
		}
	}
}

// multi runs fn once per selection, with that selection as the primary one.
// edits says fn may change the text: the runs then form one undo step.
func (e *Editor) multi(edits bool, fn func()) {
	all := e.allCursors()
	order := make([]int, len(all))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return all[order[a]].lo() > all[order[b]].lo() })
	group := 0
	if edits && e.undoGroup == 0 {
		e.groupSeq++
		e.undoGroup = e.groupSeq
		group = e.undoGroup
		defer func() { e.undoGroup = 0 }()
	}
	e.inMulti = true
	defer func() { e.inMulti = false }()
	out := make([]cursor, len(all))
	var done []int
	for _, i := range order {
		e.caret, e.selAnchor = all[i].caret, all[i].anchor
		before := e.pt.Len()
		fn()
		out[i] = cursor{e.caret, e.selAnchor}
		if d := e.pt.Len() - before; d != 0 {
			for _, j := range done {
				out[j].caret += d
				if out[j].anchor >= 0 {
					out[j].anchor += d
				}
			}
		}
		done = append(done, i)
	}
	e.caret, e.selAnchor = out[0].caret, out[0].anchor
	e.extra = append(e.extra[:0], out[1:]...)
	e.normalizeCursors()
	if group != 0 {
		e.stampGroup(group, all)
	}
	e.ensureCaretVisible()
	if len(e.extra) > 0 {
		e.closeCompletion() // completion is for a single caret
	}
}

// copyAll copies the selection, or with several selections each one's
// text, a line apiece.
func (e *Editor) copyAll(clip input.Clipboard) {
	if len(e.extra) == 0 {
		e.copy(clip)
		return
	}
	if clip != nil {
		clip.Set(e.selectionsText())
	}
}

// cutAll is copyAll that then deletes what it copied.
func (e *Editor) cutAll(clip input.Clipboard) {
	if len(e.extra) == 0 {
		e.cut(clip)
		return
	}
	if clip != nil {
		clip.Set(e.selectionsText())
	}
	e.multi(true, func() { e.deleteSelection() })
}

// pasteAll pastes at every selection. When the clipboard has as many lines as
// there are selections, each gets one line, in document order.
func (e *Editor) pasteAll(clip input.Clipboard) {
	if len(e.extra) == 0 || e.findFocused() {
		e.paste(clip)
		return
	}
	if clip == nil {
		return
	}
	text := clip.Get()
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != e.Cursors() {
		e.multi(true, func() { e.replaceSelection(text, mergeNone) })
		return
	}
	sels := e.Selections()
	e.multi(true, func() {
		lo, _ := e.selRange()
		for k, s := range sels {
			if s[0] == lo {
				e.replaceSelection(lines[k], mergeNone)
				return
			}
		}
	})
}

// multiKey handles a key with several selections and reports whether it did.
func (e *Editor) multiKey(ev input.KeyEvent) bool {
	switch ev.Key {
	case input.KeyEscape:
		e.extra = nil
		return true
	case input.KeyEnter, input.KeyTab, input.KeyBackspace, input.KeyDelete:
		e.multi(true, func() { e.HandleKeys([]input.KeyEvent{ev}) })
		return true
	case input.KeyLeft, input.KeyRight, input.KeyUp, input.KeyDown, input.KeyHome, input.KeyEnd:
		e.multi(false, func() { e.HandleKeys([]input.KeyEvent{ev}) })
		return true
	}
	return false
}

// selectionsText is the selected text of every selection, one per line.
func (e *Editor) selectionsText() string {
	doc := e.pt.Bytes()
	var parts []string
	for _, s := range e.Selections() {
		if s[0] < s[1] {
			parts = append(parts, string(doc[s[0]:s[1]]))
		}
	}
	return strings.Join(parts, "\n")
}

func clampInt(v, lo, hi int) int {
	return max(lo, min(v, hi))
}

func (e *Editor) extraCarets() []int {
	out := make([]int, len(e.extra))
	for i, c := range e.extra {
		out[i] = c.caret
	}
	return out
}

// CaretStyle is the shape the caret is drawn in.
type CaretStyle int

const (
	CaretBar       CaretStyle = iota // a thin bar before the character
	CaretBlock                       // a block over the character
	CaretUnderline                   // a line under the character
)

// Replace replaces bytes [lo, hi) with text, as one undoable edit, and puts
// the caret after the new text. It does nothing in a read-only editor.
func (e *Editor) Replace(lo, hi int, text string) {
	lo, hi = e.clampOffset(lo), e.clampOffset(hi)
	if hi < lo {
		lo, hi = hi, lo
	}
	e.extra = nil
	e.applyEdit(lo, hi-lo, text, mergeNone)
	e.ensureCaretVisible()
}

// UndoGroup runs fn, and the edits it makes undo and redo as one step, which
// also puts back the selections from before and after it.
func (e *Editor) UndoGroup(fn func()) {
	before := e.allCursors()
	if g := e.group(fn); g != 0 {
		e.stampGroup(g, before)
	}
}

// group runs fn as one undo group and returns its id, or 0 when it ran
// inside a group already open.
func (e *Editor) group(fn func()) int {
	if e.undoGroup != 0 {
		fn()
		return 0
	}
	e.groupSeq++
	e.undoGroup = e.groupSeq
	defer func() { e.undoGroup = 0 }()
	fn()
	return e.groupSeq
}

// stampGroup records the selections before and after undo group g on its
// first and last edit.
func (e *Editor) stampGroup(g int, before []cursor) {
	n := len(e.undo)
	if n == 0 || e.undo[n-1].group != g {
		return
	}
	i := n - 1
	for i > 0 && e.undo[i-1].group == g {
		i--
	}
	e.undo[i].selBefore = slices.Clone(before)
	e.undo[n-1].selAfter = e.allCursors()
}
