package ui

import (
	"sort"
	"strings"

	"github.com/mirzakhany/yoga/highlight"
)

// Whole-line commands: duplicate, move, delete and toggle comment. Each works
// on the lines every selection touches, merged into blocks, and is one undo
// step however many selections there are.

// lineBlock is a run of whole lines [first, last] and the selections in it.
type lineBlock struct {
	first, last int
	owned       []int // indexes into the cursor list
}

// lineSpan is the lines a selection touches. A selection that ends at the
// start of a line does not touch that line.
func (e *Editor) lineSpan(c cursor) (first, last int) {
	lo, hi := c.lo(), c.hi()
	first, last = e.lineOf(lo), e.lineOf(hi)
	if hi > lo && last > first && hi == e.pt.LineStart(last) {
		last--
	}
	return first, last
}

// lineBlocks groups the selections' lines into blocks in document order.
// With adjacent, blocks that touch merge too (for moving lines).
func (e *Editor) lineBlocks(cs []cursor, adjacent bool) []lineBlock {
	var bs []lineBlock
	for i, c := range cs {
		f, l := e.lineSpan(c)
		bs = append(bs, lineBlock{first: f, last: l, owned: []int{i}})
	}
	sort.Slice(bs, func(i, j int) bool { return bs[i].first < bs[j].first })
	out := bs[:0]
	for _, b := range bs {
		if n := len(out); n > 0 {
			gap := 0
			if adjacent {
				gap = 1
			}
			if p := &out[n-1]; b.first <= p.last+gap {
				p.last = max(p.last, b.last)
				p.owned = append(p.owned, b.owned...)
				continue
			}
		}
		out = append(out, b)
	}
	return out
}

// lineHi is the offset just past line ln's newline, or the document end.
func (e *Editor) lineHi(ln int) int {
	if ln+1 < e.pt.LineCount() {
		return e.pt.LineStart(ln + 1)
	}
	return e.pt.Len()
}

// blockEdit replaces [lo, hi) with text. Selections the block owns move by
// remap (old offset to new); the others after the block shift with it.
type blockEdit struct {
	lo, hi int
	text   string
	owned  []int
	remap  func(off int) int
}

// applyBlockEdits applies edits (in document order, not overlapping) from the
// last to the first as one undo step, and places the selections.
func (e *Editor) applyBlockEdits(cs []cursor, edits []blockEdit) {
	if len(edits) == 0 {
		return
	}
	before := e.allCursors()
	g := e.group(func() {
		for i := len(edits) - 1; i >= 0; i-- {
			ed := edits[i]
			own := map[int]bool{}
			for _, k := range ed.owned {
				own[k] = true
			}
			d := len(ed.text) - (ed.hi - ed.lo)
			move := func(off int) int {
				if off < 0 {
					return off
				}
				if off >= ed.hi {
					return off + d
				}
				return off
			}
			for k := range cs {
				if own[k] {
					cs[k].caret = ed.remap(cs[k].caret)
					if cs[k].anchor >= 0 {
						cs[k].anchor = ed.remap(cs[k].anchor)
					}
				} else if cs[k].lo() >= ed.hi {
					cs[k].caret, cs[k].anchor = move(cs[k].caret), move(cs[k].anchor)
				}
			}
			e.applyEdit(ed.lo, ed.hi-ed.lo, ed.text, mergeNone)
		}
	})
	e.setCursors(cs)
	if g != 0 {
		e.stampGroup(g, before)
	}
}

// setCursors makes cs the selections, the first one primary.
func (e *Editor) setCursors(cs []cursor) {
	n := e.pt.Len()
	for k := range cs {
		cs[k].caret = clampInt(cs[k].caret, 0, n)
		if cs[k].anchor >= 0 {
			cs[k].anchor = clampInt(cs[k].anchor, 0, n)
		}
	}
	e.caret, e.selAnchor = cs[0].caret, cs[0].anchor
	e.extra = append(e.extra[:0], cs[1:]...)
	e.normalizeCursors()
	e.lastMerge = mergeNone
	e.goalSet = false
	e.ensureCaretVisible()
}

// DuplicateLines copies the lines each selection touches below them (dir 1)
// or above them (dir -1), and moves the selections onto the copy.
func (e *Editor) DuplicateLines(dir int) {
	if e.readOnly {
		return
	}
	cs := e.allCursors()
	doc := e.pt.Bytes()
	var edits []blockEdit
	for _, b := range e.lineBlocks(cs, false) {
		lo, hi := e.pt.LineStart(b.first), e.lineHi(b.last)
		block := string(doc[lo:hi])
		nl := ""
		if !strings.HasSuffix(block, "\n") {
			nl = "\n" // the last line has no newline to copy
		}
		shift := 0
		if dir > 0 {
			shift = len(block) + len(nl)
		}
		edits = append(edits, blockEdit{
			lo: lo, hi: hi, text: block + nl + block, owned: b.owned,
			remap: func(off int) int { return off + shift },
		})
	}
	e.applyBlockEdits(cs, edits)
}

// MoveLines moves the lines each selection touches up (dir -1) or down
// (dir 1) by one line, keeping the selections on them.
func (e *Editor) MoveLines(dir int) {
	if e.readOnly {
		return
	}
	cs := e.allCursors()
	doc := e.pt.Bytes()
	lc := e.pt.LineCount()
	var edits []blockEdit
	for _, b := range e.lineBlocks(cs, true) {
		if (dir < 0 && b.first == 0) || (dir > 0 && b.last+1 >= lc) {
			continue
		}
		lo, hi := e.pt.LineStart(b.first), e.lineHi(b.last)
		block := string(doc[lo:hi])
		if dir > 0 {
			nhi := e.lineHi(b.last + 1)
			next := string(doc[hi:nhi])
			text, shift := next+block, len(next)
			if !strings.HasSuffix(next, "\n") {
				// The next line is the last and has no newline: it takes
				// the block's.
				text = next + "\n" + strings.TrimSuffix(block, "\n")
				shift = len(next) + 1
			}
			edits = append(edits, blockEdit{lo: lo, hi: nhi, text: text, owned: b.owned,
				remap: func(off int) int { return off + shift }})
			continue
		}
		plo := e.pt.LineStart(b.first - 1)
		prev := string(doc[plo:lo])
		text := block + prev
		if !strings.HasSuffix(block, "\n") {
			text = block + "\n" + strings.TrimSuffix(prev, "\n")
		}
		shift := len(prev)
		edits = append(edits, blockEdit{lo: plo, hi: hi, text: text, owned: b.owned,
			remap: func(off int) int { return off - shift }})
	}
	e.applyBlockEdits(cs, edits)
}

// DeleteLines deletes the lines each selection touches. The caret keeps its
// column on the line that takes their place.
func (e *Editor) DeleteLines() {
	if e.readOnly {
		return
	}
	cs := e.allCursors()
	lc := e.pt.LineCount()
	var edits []blockEdit
	for _, b := range e.lineBlocks(cs, false) {
		lo, hi := e.pt.LineStart(b.first), e.lineHi(b.last)
		// The line that takes their place: the next one, or for the last
		// lines the one before, whose newline goes with them.
		landStart, landLen := lo, 0
		switch {
		case b.last+1 < lc:
			landLen = e.pt.LineLen(b.last + 1)
		case b.first > 0:
			lo--
			landStart, landLen = e.pt.LineStart(b.first-1), e.pt.LineLen(b.first-1)
		}
		edits = append(edits, blockEdit{lo: lo, hi: hi, owned: b.owned,
			remap: func(off int) int {
				col := off - e.pt.LineStart(e.lineOf(off))
				return landStart + min(col, landLen)
			}})
	}
	e.applyBlockEdits(cs, edits)
}

// SetLineComment sets the comment syntax toggle comment uses for this
// editor; by default it follows the file's extension (highlight.CommentFor).
func (e *Editor) SetLineComment(c highlight.Comment) { e.comment, e.commentSet = c, true }

func (e *Editor) commentSyntax() highlight.Comment {
	if e.commentSet {
		return e.comment
	}
	return highlight.CommentFor(e.Path)
}

// ToggleComment comments out the lines the selections touch, or uncomments
// them when every non-blank one is already commented. Line comments go at
// the block's smallest indent; languages without them wrap each line in a
// block comment.
func (e *Editor) ToggleComment() {
	if e.readOnly {
		return
	}
	syn := e.commentSyntax()
	open, close := syn.Line, ""
	if open == "" {
		open, close = syn.Open, syn.Close
	}
	if open == "" {
		return
	}
	cs := e.allCursors()
	blocks := e.lineBlocks(cs, false)
	lineText := func(ln int) string { return string(e.pt.Line(ln)) }
	indentOf := func(s string) int { return len(s) - len(strings.TrimLeft(s, " \t")) }
	commented := func(s string) bool {
		t := strings.TrimLeft(s, " \t")
		return strings.HasPrefix(t, open) && (close == "" || strings.HasSuffix(strings.TrimRight(t, " \t"), close))
	}
	// Uncomment only when every non-blank line of every block is commented.
	uncomment, any := true, false
	for _, b := range blocks {
		for ln := b.first; ln <= b.last; ln++ {
			s := lineText(ln)
			if strings.TrimSpace(s) == "" {
				continue
			}
			any = true
			if !commented(s) {
				uncomment = false
			}
		}
	}
	if !any {
		return
	}
	var edits []blockEdit
	for _, b := range blocks {
		minInd := -1
		for ln := b.first; ln <= b.last; ln++ {
			if s := lineText(ln); strings.TrimSpace(s) != "" && (minInd < 0 || indentOf(s) < minInd) {
				minInd = indentOf(s)
			}
		}
		if minInd < 0 {
			continue
		}
		lo := e.pt.LineStart(b.first)
		hi := lo
		var out strings.Builder
		// For each line: where it starts in the old and new text, and the
		// column where text was added (+) or removed (-), to move carets.
		type lineMap struct{ oldStart, newStart, at, added int }
		var maps []lineMap
		for ln := b.first; ln <= b.last; ln++ {
			s := lineText(ln)
			ls := e.pt.LineStart(ln)
			m := lineMap{oldStart: ls, newStart: lo + out.Len()}
			switch {
			case strings.TrimSpace(s) == "":
				out.WriteString(s)
			case uncomment:
				ind := indentOf(s)
				t := s[ind+len(open):]
				removed := len(open)
				if strings.HasPrefix(t, " ") {
					t, removed = t[1:], removed+1
				}
				if close != "" {
					tr := strings.TrimRight(t, " \t")
					trail := t[len(tr):]
					tr = strings.TrimSuffix(tr, close)
					tr = strings.TrimSuffix(tr, " ")
					t = tr + trail
				}
				out.WriteString(s[:ind] + t)
				m.at, m.added = ind, -removed
			default:
				line := s[:minInd] + open + " " + s[minInd:]
				if close != "" {
					line += " " + close
				}
				out.WriteString(line)
				m.at, m.added = minInd, len(open)+1
			}
			maps = append(maps, m)
			hi = ls + len(s)
			if ln < b.last {
				out.WriteByte('\n')
				hi++
			}
		}
		edits = append(edits, blockEdit{lo: lo, hi: hi, text: out.String(), owned: b.owned,
			remap: func(off int) int {
				i := sort.Search(len(maps), func(i int) bool { return maps[i].oldStart > off }) - 1
				m := maps[max(i, 0)]
				col := off - m.oldStart
				switch {
				case m.added > 0 && col >= m.at:
					col += m.added
				case m.added < 0 && col > m.at:
					col = max(m.at, col+m.added)
				}
				return m.newStart + col
			}})
	}
	e.applyBlockEdits(cs, edits)
}

func init() {
	for act, fn := range map[EditorAction]func(e *Editor) bool{
		ActionDuplicateLinesDown: func(e *Editor) bool { e.DuplicateLines(1); return true },
		ActionDuplicateLinesUp:   func(e *Editor) bool { e.DuplicateLines(-1); return true },
		ActionMoveLinesDown:      func(e *Editor) bool { e.MoveLines(1); return true },
		ActionMoveLinesUp:        func(e *Editor) bool { e.MoveLines(-1); return true },
		ActionDeleteLines:        func(e *Editor) bool { e.DeleteLines(); return true },
		ActionToggleComment:      func(e *Editor) bool { e.ToggleComment(); return true },
	} {
		editorActions[act] = fn
	}
}
