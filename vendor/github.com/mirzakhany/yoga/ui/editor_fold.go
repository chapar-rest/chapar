package ui

import (
	"bytes"
	"sort"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Code folding. A fold keeps its first line on screen, followed by a "…"
// placeholder and the block's closing text, and hides the lines after it.
// Hidden lines take no visual rows: with soft wrap their rowPrefix entry
// adds nothing, and without it foldState.rows maps lines to rows the same
// way. The text itself is untouched, so selections, copy, search and undo
// still see every line.

// fold is a collapsed block: line start stays shown; start+1..end are hidden.
type fold struct {
	start, end int
	tail       string // closing text shown after the placeholder, such as "}"
}

type foldState struct {
	off    bool     // WithoutFolding
	folds  []fold   // collapsed blocks, by start line
	hidden [][2]int // merged hidden line ranges, inclusive
	rows   []int    // wrap off, with folds: rows[ln] is line ln's visual row; len LineCount+1

	cache    map[int]foldRange // foldable blocks by start line, for cacheRev
	cacheRev uint64

	hoverIn bool       // the pointer is over the editor, so chevrons show
	pills   []foldPill // placeholder hit boxes from the last paint
	anim    foldAnim
}

// foldRange is the block a line starts, if any.
type foldRange struct {
	end  int
	tail string
	ok   bool
}

type foldPill struct {
	r    render.Rect
	line int
}

// foldAnim slides the rows after a fold that just opened or closed.
type foldAnim struct {
	on     bool
	row    int // visual row of the fold's first line
	rows   int // rows hidden or shown
	expand bool
	start  time.Time
}

const (
	foldGutterW     = float32(16)
	foldAnimTime    = 170 * time.Millisecond
	foldAnimMaxRows = 400 // larger blocks open and close at once
	maxFoldScan     = 512 << 10
	foldTailMax     = 24
)

// WithoutFolding turns code folding off for an editor: no chevrons, and the
// fold commands do nothing.
func WithoutFolding() EditorOption { return func(e *Editor) { e.fold.off = true } }

// Folded reports whether line (0-based) starts a collapsed block.
func (e *Editor) Folded(line int) bool { return e.foldAt(line) >= 0 }

// Folds returns the collapsed blocks as [first, last] line pairs, the first
// line being the one that stays shown.
func (e *Editor) Folds() [][2]int {
	out := make([][2]int, len(e.fold.folds))
	for i, f := range e.fold.folds {
		out[i] = [2]int{f.start, f.end}
	}
	return out
}

// FoldRange returns the last line of the block line starts, if it starts
// one that can fold.
func (e *Editor) FoldRange(line int) (end int, ok bool) {
	r := e.foldRangeAt(line)
	return r.end, r.ok
}

// Fold collapses the block line starts and reports whether there was one.
func (e *Editor) Fold(line int) bool {
	if e.fold.off || e.Folded(line) {
		return false
	}
	r := e.foldRangeAt(line)
	if !r.ok {
		return false
	}
	before := e.VisualRowCount()
	e.fold.folds = append(e.fold.folds, fold{start: line, end: r.end, tail: r.tail})
	e.foldsChanged()
	e.moveCursorsOutOfFolds()
	e.startFoldAnim(line, before-e.VisualRowCount(), false)
	return true
}

// Unfold opens the collapsed block line starts and reports whether there was
// one.
func (e *Editor) Unfold(line int) bool {
	i := e.foldAt(line)
	if i < 0 {
		return false
	}
	before := e.VisualRowCount()
	e.fold.folds = append(e.fold.folds[:i], e.fold.folds[i+1:]...)
	e.foldsChanged()
	e.startFoldAnim(line, e.VisualRowCount()-before, true)
	return true
}

// ToggleFold folds or unfolds the block line starts.
func (e *Editor) ToggleFold(line int) bool {
	if e.Unfold(line) {
		return true
	}
	return e.Fold(line)
}

// FoldAll collapses every block that can fold, nested ones included, so
// unfolding an outer block shows its inner ones still folded.
func (e *Editor) FoldAll() {
	if e.fold.off {
		return
	}
	have := map[int]bool{}
	for _, f := range e.fold.folds {
		have[f.start] = true
	}
	for ln, lc := 0, e.pt.LineCount(); ln < lc; ln++ {
		if have[ln] {
			continue
		}
		if r := e.foldRangeAt(ln); r.ok {
			e.fold.folds = append(e.fold.folds, fold{start: ln, end: r.end, tail: r.tail})
		}
	}
	e.foldsChanged()
	e.moveCursorsOutOfFolds()
}

// UnfoldAll opens every collapsed block.
func (e *Editor) UnfoldAll() {
	if len(e.fold.folds) == 0 {
		return
	}
	e.fold.folds = e.fold.folds[:0]
	e.foldsChanged()
}

// foldAtCursor folds the innermost block around the caret's line: the one
// the line starts, or the nearest enclosing one.
func (e *Editor) foldAtCursor() {
	ln := e.lineOf(e.caret)
	if !e.Folded(ln) && e.Fold(ln) {
		return
	}
	for up := ln - 1; up >= 0 && ln-up < 5000; up-- {
		if e.Folded(up) {
			continue
		}
		if r := e.foldRangeAt(up); r.ok && r.end >= ln {
			e.Fold(up)
			return
		}
	}
}

// unfoldAtCursor opens the fold on the caret's line, or the innermost one
// around it.
func (e *Editor) unfoldAtCursor() {
	ln := e.lineOf(e.caret)
	if e.Unfold(ln) {
		return
	}
	best := -1
	for _, f := range e.fold.folds {
		if f.start <= ln && ln <= f.end && (best < 0 || f.start > best) {
			best = f.start
		}
	}
	if best >= 0 {
		e.Unfold(best)
	}
}

func (e *Editor) foldAt(line int) int {
	for i, f := range e.fold.folds {
		if f.start == line {
			return i
		}
	}
	return -1
}

// foldsChanged rebuilds the hidden ranges and row maps after folds change.
func (e *Editor) foldsChanged() {
	fs := &e.fold
	sort.Slice(fs.folds, func(i, j int) bool { return fs.folds[i].start < fs.folds[j].start })
	fs.hidden = fs.hidden[:0]
	for _, f := range fs.folds {
		lo, hi := f.start+1, f.end
		if lo > hi {
			continue
		}
		if n := len(fs.hidden); n > 0 && lo <= fs.hidden[n-1][1]+1 {
			fs.hidden[n-1][1] = max(fs.hidden[n-1][1], hi)
			continue
		}
		fs.hidden = append(fs.hidden, [2]int{lo, hi})
	}
	e.rebuildFoldRows()
	if e.SoftWrap && len(e.lineRows) == e.pt.LineCount() {
		e.rebuildRowPrefix()
	}
	e.contentSizeDirty = true
	e.blinkStart = time.Now()
}

// rebuildFoldRows maps lines to visual rows for wrap-off layout.
func (e *Editor) rebuildFoldRows() {
	fs := &e.fold
	if len(fs.hidden) == 0 {
		fs.rows = nil
		return
	}
	lc := e.pt.LineCount()
	if cap(fs.rows) >= lc+1 {
		fs.rows = fs.rows[:lc+1]
	} else {
		fs.rows = make([]int, lc+1)
	}
	h := 0
	for ln := 0; ln < lc; ln++ {
		for h < len(fs.hidden) && fs.hidden[h][1] < ln {
			h++
		}
		add := 1
		if h < len(fs.hidden) && fs.hidden[h][0] <= ln {
			add = 0
		}
		fs.rows[ln+1] = fs.rows[ln] + add
	}
}

// lineHidden reports whether line ln is inside a collapsed block.
func (e *Editor) lineHidden(ln int) bool {
	hs := e.fold.hidden
	if len(hs) == 0 {
		return false
	}
	i := sort.Search(len(hs), func(i int) bool { return hs[i][1] >= ln })
	return i < len(hs) && hs[i][0] <= ln
}

// shownRows is how many visual rows line ln takes with soft wrap.
func (e *Editor) shownRows(ln int) int {
	if e.lineHidden(ln) {
		return 0
	}
	return len(e.lineRows[ln])
}

// Wrap-off row mapping. Without folds a line is its own row.

func (e *Editor) foldRowsFresh() []int {
	if e.fold.rows != nil && len(e.fold.rows) != e.pt.LineCount()+1 {
		e.rebuildFoldRows()
	}
	return e.fold.rows
}

// lineRow is line ln's visual row; a hidden line maps to its fold's row.
func (e *Editor) lineRow(ln int) int {
	rows := e.foldRowsFresh()
	if rows == nil {
		return ln
	}
	ln = clampInt(ln, 0, len(rows)-2)
	if e.lineHidden(ln) {
		return max(rows[ln]-1, 0)
	}
	return rows[ln]
}

// rowLine is the line shown on visual row r.
func (e *Editor) rowLine(r int) int {
	rows := e.foldRowsFresh()
	if rows == nil {
		return r
	}
	lo, hi := 0, len(rows)-2
	for lo < hi {
		mid := (lo + hi) / 2
		if rows[mid+1] <= r {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// lineRowCount is how many visual rows the lines take with wrap off.
func (e *Editor) lineRowCount() int {
	rows := e.foldRowsFresh()
	if rows == nil {
		return e.pt.LineCount()
	}
	return rows[len(rows)-1]
}

// foldsAfterEdit keeps folds on their lines across an edit: folds below it
// shift by the lines it added or removed, and a fold the edit reaches into
// opens. An edit within a fold's first line alone leaves it.
func (e *Editor) foldsAfterEdit(ed highlight.Edit) {
	fs := &e.fold
	if len(fs.folds) == 0 {
		return
	}
	d := ed.NewEnd.Row - ed.OldEnd.Row
	out := fs.folds[:0]
	for _, f := range fs.folds {
		switch {
		case ed.Start.Row > f.end:
		case ed.OldEnd.Row < f.start:
			f.start += d
			f.end += d
		case ed.Start.Row == f.start && ed.OldEnd.Row == f.start && ed.NewEnd.Row == f.start:
		default:
			continue // the edit reached into the block
		}
		out = append(out, f)
	}
	fs.folds = out
	e.foldsChanged()
}

// revealLine opens every fold that hides line ln.
func (e *Editor) revealLine(ln int) {
	if !e.lineHidden(ln) {
		return
	}
	out := e.fold.folds[:0]
	for _, f := range e.fold.folds {
		if f.start < ln && ln <= f.end {
			continue
		}
		out = append(out, f)
	}
	e.fold.folds = out
	e.foldsChanged()
}

// skipFolded moves an offset that landed on a hidden line out of it: past the
// block when moving forward, to the end of its first line when moving back.
func (e *Editor) skipFolded(off int, forward bool) int {
	if len(e.fold.hidden) == 0 {
		return off
	}
	ln := e.lineOf(off)
	if !e.lineHidden(ln) {
		return off
	}
	for _, h := range e.fold.hidden {
		if h[0] <= ln && ln <= h[1] {
			if forward {
				if h[1]+1 < e.pt.LineCount() {
					return e.pt.LineStart(h[1] + 1)
				}
				return e.pt.Len()
			}
			return e.lineHi(h[0]-1) - 1
		}
	}
	return off
}

// moveCursorsOutOfFolds puts carets a fold just hid at the end of its first
// line.
func (e *Editor) moveCursorsOutOfFolds() {
	if len(e.fold.hidden) == 0 {
		return
	}
	moved := false
	cs := e.allCursors()
	for k := range cs {
		if ln := e.lineOf(cs[k].caret); e.lineHidden(ln) {
			cs[k].caret, cs[k].anchor = e.skipFolded(cs[k].caret, false), -1
			moved = true
		}
	}
	if moved {
		e.setCursors(cs)
	}
}

// foldRangeAt finds the block line ln starts, cached until the next edit.
func (e *Editor) foldRangeAt(ln int) foldRange {
	fs := &e.fold
	if fs.off || ln < 0 || ln >= e.pt.LineCount() {
		return foldRange{}
	}
	if fs.cache == nil || fs.cacheRev != e.revision {
		fs.cache, fs.cacheRev = map[int]foldRange{}, e.revision
	}
	if r, ok := fs.cache[ln]; ok {
		return r
	}
	r := e.computeFoldRange(ln)
	fs.cache[ln] = r
	return r
}

// computeFoldRange finds the block line ln starts: an opening bracket left
// open at the line's end, a block comment opened there, or else the lines
// after it indented deeper.
func (e *Editor) computeFoldRange(ln int) foldRange {
	syn := e.commentSyntax()
	line := e.pt.Line(ln)
	if strings.TrimSpace(string(line)) == "" {
		return foldRange{}
	}
	ls := e.pt.LineStart(ln)
	if syn.Open != "<!--" {
		var open []int
		st := e.foldScan(ls, ls+len(line), syn, func(i int, c byte) bool {
			switch c {
			case '{', '[', '(':
				open = append(open, i)
			default:
				if len(open) > 0 {
					open = open[:len(open)-1]
				}
			}
			return true
		})
		if len(open) > 0 {
			if r, ok := e.bracketFold(ln, open[len(open)-1], syn); ok {
				return r
			}
		}
		if st.blockComment && syn.Close != "" {
			doc := e.pt.Bytes()
			end := bytes.Index(doc[ls+len(line):], []byte(syn.Close))
			if end >= 0 {
				if m := e.lineOf(ls + len(line) + end); m > ln {
					return foldRange{end: m, tail: syn.Close, ok: true}
				}
			}
		}
	}
	return e.indentFold(ln)
}

// bracketFold is the block from the bracket at p, on line ln, to its match.
// When the closing bracket starts its line, that line folds too and its
// text shows after the placeholder ("{ … }"); otherwise the fold stops above
// it.
func (e *Editor) bracketFold(ln, p int, syn highlight.Comment) (foldRange, bool) {
	depth := 0
	q := -1
	e.foldScan(p, min(p+maxFoldScan, e.pt.Len()), syn, func(i int, c byte) bool {
		switch c {
		case '{', '[', '(':
			depth++
		default:
			depth--
			if depth == 0 {
				q = i
				return false
			}
		}
		return true
	})
	if q < 0 {
		return foldRange{}, false
	}
	m := e.lineOf(q)
	if m <= ln {
		return foldRange{}, false
	}
	mStart := e.pt.LineStart(m)
	closing := string(e.pt.Line(m))
	before := strings.TrimSpace(closing[:q-mStart])
	rest := strings.TrimSpace(closing[q-mStart:])
	if before == "" && !strings.ContainsAny(rest[1:], "{[(") && len(rest) <= foldTailMax {
		return foldRange{end: m, tail: rest, ok: true}, true
	}
	if m-1 > ln {
		return foldRange{end: m - 1, ok: true}, true
	}
	return foldRange{}, false
}

// indentFold is the run of lines after ln indented deeper than it, blank
// lines inside it included and trailing ones left out.
func (e *Editor) indentFold(ln int) foldRange {
	tabW := max(e.tabW, 1)
	ind := indentDepth(e.lineView(ln), tabW)
	end := -1
	for k, lc := ln+1, e.pt.LineCount(); k < lc && k-ln < 100000; k++ {
		l := e.lineView(k)
		if len(bytes.TrimSpace(l)) == 0 {
			continue
		}
		if indentDepth(l, tabW) <= ind {
			break
		}
		end = k
	}
	if end <= ln {
		return foldRange{}
	}
	return foldRange{end: end, ok: true}
}

// foldScanState is where a scan stopped: inside a block comment, say.
type foldScanState struct{ blockComment bool }

// foldScan calls fn for every bracket in doc[from:to] outside strings and
// comments, until fn returns false.
func (e *Editor) foldScan(from, to int, syn highlight.Comment, fn func(i int, c byte) bool) foldScanState {
	doc := e.pt.Bytes()
	to = min(to, len(doc))
	lineC := syn.Line
	if lineC == "" && syn.Open == "" {
		lineC = "//"
	}
	blockO, blockC := syn.Open, syn.Close
	if syn.Line == "" && syn.Open == "" {
		blockO, blockC = "/*", "*/"
	}
	for i := from; i < to; i++ {
		c := doc[i]
		switch {
		case lineC != "" && c == lineC[0] && bytes.HasPrefix(doc[i:], []byte(lineC)):
			for i < to && doc[i] != '\n' {
				i++
			}
		case blockO != "" && c == blockO[0] && bytes.HasPrefix(doc[i:], []byte(blockO)):
			end := bytes.Index(doc[i+len(blockO):to], []byte(blockC))
			if end < 0 {
				return foldScanState{blockComment: true}
			}
			i += len(blockO) + end + len(blockC) - 1
		case c == '"' || c == '\'' || c == '`':
			j := i + 1
			for j < to && doc[j] != c && (c == '`' || doc[j] != '\n') {
				if doc[j] == '\\' && c != '`' {
					j++
				}
				j++
			}
			if j >= to || doc[j] != c {
				// Unterminated on this line: an apostrophe, say. Not a string.
				continue
			}
			i = j
		case c == '{' || c == '[' || c == '(' || c == '}' || c == ']' || c == ')':
			if !fn(i, c) {
				return foldScanState{}
			}
		}
	}
	return foldScanState{}
}

// Animation.

func (e *Editor) startFoldAnim(line, rows int, expand bool) {
	if rows <= 0 || rows > foldAnimMaxRows || theme.ReducedMotion() {
		e.fold.anim.on = false
		return
	}
	e.fold.anim = foldAnim{on: true, row: e.rowOfByte(e.pt.LineStart(line)), rows: rows, expand: expand, start: time.Now()}
}

// foldEase is how far the running animation is, 0 to 1.
func (e *Editor) foldEase() (float32, bool) {
	a := &e.fold.anim
	if !a.on {
		return 1, false
	}
	t := float32(time.Since(a.start)) / float32(foldAnimTime)
	if t >= 1 {
		a.on = false
		return 1, false
	}
	return easeOutCubic(t), true
}

// foldDY is how far visual row r is drawn from its place while a fold
// animates: rows below a closing fold slide up into place, and rows below an
// opening one slide down over the lines it reveals.
func (e *Editor) foldDY(r int) float32 {
	t, on := e.foldEase()
	if !on {
		return 0
	}
	a := e.fold.anim
	h := float32(a.rows) * e.lineH * (1 - t)
	switch {
	case !a.expand && r > a.row:
		return h
	case a.expand && r > a.row+a.rows:
		return -h
	}
	return 0
}

// foldCovers reports whether row r slides over other rows and so paints its
// own background.
func (e *Editor) foldCovers(r int) bool {
	a := e.fold.anim
	return a.on && a.expand && r > a.row+a.rows
}

// rowY is the top of visual row r on screen.
func (e *Editor) rowY(r int) float32 {
	return e.viewport.Frame.Y + float32(r)*e.lineH - e.ScrollPx + e.foldDY(r)
}

// Painting.

// paintFoldPlaceholder draws "…" and the block's closing text after the
// first line of a collapsed block.
func (e *Editor) paintFoldPlaceholder(dl *render.DrawList, engine *shape.Engine, pr paintRow, x0, y, vpad float32) {
	if len(e.fold.folds) == 0 || pr.rowEnd != pr.lineEnd {
		return
	}
	i := e.foldAt(pr.line)
	if i < 0 {
		return
	}
	f := e.fold.folds[i]
	th := theme.Current()
	end := e.pt.LineStart(pr.line) + pr.lineEnd
	x := x0 + e.xInRow(end) + e.cellW*0.5
	dotsW, _ := engine.MeasureMono("…")
	pill := render.Rect{X: x, Y: y + 2, W: dotsW + 2*e.cellW*0.6, H: e.lineH - 4}
	bg := alpha(th.Foreground, 0.12)
	if lo, hi := e.selRange(); lo < e.lineHi(f.end) && hi > end {
		bg = th.Selection // the selection runs through the hidden lines
	}
	if t, on := e.foldEase(); on && !e.fold.anim.expand {
		bg.A *= t
	}
	dl.AddRoundedRect(pill, theme.Current().Radius.Medium, bg)
	engine.DrawStringTopMono(dl, "…", pill.X+e.cellW*0.6, y+vpad, th.ForegroundMuted)
	if f.tail != "" {
		engine.DrawStringTopMono(dl, f.tail, pill.X+pill.W+e.cellW*0.5, y+vpad, th.Foreground)
	}
	e.fold.pills = append(e.fold.pills, foldPill{r: pill, line: pr.line})
}

// paintFoldChevron draws the gutter chevron for the line pr starts: always
// for a collapsed block, and for one that could fold while the pointer is
// over the editor.
func (e *Editor) paintFoldChevron(dl *render.DrawList, pr paintRow, gutter render.Rect, y float32) {
	if e.fold.off || !pr.first {
		return
	}
	folded := e.Folded(pr.line)
	if !folded && (!e.fold.hoverIn || !e.foldRangeAt(pr.line).ok) {
		return
	}
	th := theme.Current()
	icon := icons.ChevronDown
	col := th.ForegroundSubtle
	if folded {
		icon, col = icons.ChevronRight, th.ForegroundMuted
	}
	sz := min(foldGutterW-4, e.lineH-4)
	r := render.Rect{X: gutter.X + gutter.W - foldGutterW + (foldGutterW-sz)/2, Y: y + (e.lineH-sz)/2, W: sz, H: sz}
	frameIcons().Draw(dl, icon, r, col)
}

// foldMouse handles clicks on chevrons and placeholders, and tracks whether
// the pointer is over the editor. It reports whether it took the event.
func (e *Editor) foldMouse(m *input.Mouse) bool {
	if e.fold.off {
		return false
	}
	f := e.viewport.Frame
	if in := f.Contains(m.X, m.Y); in != e.fold.hoverIn {
		e.fold.hoverIn = in
	}
	if !m.Pressed || m.Consumed || !f.Contains(m.X, m.Y) {
		return false
	}
	for _, p := range e.fold.pills {
		if p.r.Contains(m.X, m.Y) {
			e.Unfold(p.line)
			m.Consumed = true
			return true
		}
	}
	if e.gutterW > 0 && m.X >= f.X+e.gutterW-foldGutterW && m.X < f.X+e.gutterW {
		ln := e.lineOf(e.offsetAtPoint(m.X, m.Y))
		if e.ToggleFold(ln) {
			m.Consumed = true
			return true
		}
	}
	return false
}

func init() {
	for act, fn := range map[EditorAction]func(e *Editor) bool{
		ActionFold:      func(e *Editor) bool { e.foldAtCursor(); return true },
		ActionUnfold:    func(e *Editor) bool { e.unfoldAtCursor(); return true },
		ActionFoldAll:   func(e *Editor) bool { e.FoldAll(); return true },
		ActionUnfoldAll: func(e *Editor) bool { e.UnfoldAll(); return true },
	} {
		editorActions[act] = fn
	}
}
