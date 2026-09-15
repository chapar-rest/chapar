package ui

import (
	"bytes"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/lsp"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/text"
	"github.com/mirzakhany/yoga/theme"
)

// wrapRow is one visual row of a logical line: a byte range relative to the
// line's start (end excludes the newline).
type wrapRow struct {
	start, end int32
}

// paintRow is one visual row to paint: either a whole logical line (wrap
// off, or a huge line's visible window) or one wrapped segment of a line.
type paintRow struct {
	line     int     // logical line index
	rowStart int     // byte offset within the line where painting starts
	rowEnd   int     // byte offset within the line where painting ends
	lineEnd  int     // byte offset within the line of the full line end
	visual   int     // visual row index (drives y; == line when wrap is off)
	first    bool    // first row of the logical line (gutter number, guides)
	xoff     float32 // x of the painted slice relative to the line's text origin
}

// bracketClose maps an opening bracket to its matching closing bracket.
var bracketClose = map[rune]rune{
	'(': ')',
	'{': '}',
	'[': ']',
}

// bracketOpen maps a closing bracket to its matching opening bracket.
var bracketOpen = map[rune]rune{
	')': '(',
	'}': '{',
	']': '[',
}

// searchState holds the state for the find/replace overlay.
type searchState struct {
	open         bool
	focused      bool // true when the panel has keyboard focus (vs. the editor text area)
	replaceMode  bool
	query        string
	replace      string
	focusField   int // 0=search input, 1=replace input
	queryCaret   int // byte offset of caret in query
	replaceCaret int // byte offset of caret in replace
	matches      []matchRange
	current      int // index of active match (-1 = none)
}

// matchRange is a [lo, hi) byte range in the document.
type matchRange struct{ lo, hi int }

// indentDepth returns the column depth of leading whitespace in s,
// counting spaces as 1 and tabs as advancing to the next tabW stop.
func indentDepth(s []byte, tabW int) int {
	n := 0
	for _, ch := range string(s) {
		if ch == ' ' {
			n++
		} else if ch == '\t' {
			n = ((n / tabW) + 1) * tabW
		} else {
			break
		}
	}
	return n
}

// Editor is a virtualized, syntax-highlighted, *editable* code view.
type Editor struct {
	host *layout.Element
	pt   *text.PieceTable
	hl   highlight.Highlighter

	Path string // file path, empty for an unsaved scratch buffer

	// ScrollPx / ScrollX are scroll positions in pixels (updated by scrollbars).
	// ContentHeight / ContentWidth are total document extents for scrollbar thumbs.
	ScrollPx, ScrollX float32
	ContentHeight     float32
	ContentWidth      float32

	viewport *layout.Element // clipped text area (paint + mouse)
	vbar     *Scrollbar
	hbar     *Scrollbar

	// Editing state.
	caret      int // byte offset of the caret
	selAnchor  int // byte offset where a selection began, or -1 for none
	dragging   bool
	blinkStart time.Time

	// Multi-click tracking for word (double) and line (triple) selection.
	lastClickTime time.Time
	lastClickX    float32
	lastClickY    float32
	clickCount    int
	modified      bool
	focused       bool

	// parseUntil bounds a short window after an edit during which the runtime
	// should poll for highlight results quickly.
	parseUntil time.Time

	// Undo/redo.
	undo        []editOp
	redo        []editOp
	canCoalesce bool

	// tokens is the latest syntax highlight result.
	tokens []highlight.Token

	// lsp connects this document to a language server for diagnostics, hover,
	// and completion. It is never nil (a no-op handle stands in when no server
	// is available), so call sites need no guards.
	lsp   lsp.Doc
	lspUI lspState

	contentSizeDirty bool

	gutterW float32
	lineH   float32
	textPad float32
	goalX   float32

	tabW    int     // tab width in columns, mirrors engine config
	fontGen uint64  // last-seen engine font generation
	cellW   float32 // mono cell advance width, mirrors fontGen

	// Soft wrap: when true, long logical lines are laid out into visual rows
	// that fit the viewport width. lineRows holds each line's row segments as
	// line-relative byte ranges, so an edit only invalidates the touched
	// lines; rowPrefix maps global visual row indexes to lines. Rebuilt fully
	// on resize/font changes, incrementally on edits.
	SoftWrap                 bool
	WrapWords                bool
	lineRows                 [][]wrapRow
	rowPrefix                []int   // rowPrefix[i] = visual rows before logical line i (len = LineCount+1)
	wrapCols                 int     // columns per visual row used when wrapping
	lastWrapW                float32 // last client width the wrap table was built for
	wrapFull                 bool    // next sync must rebuild every line
	wrapDirtyLo, wrapDirtyHi int     // line range touched by pending edits
	wrapDirtySet             bool

	search searchState

	paintRows []paintRow // scratch reused across paints

	lspOverlay *layout.Element
}

// editOp records a replacement for undo/redo.
type editOp struct {
	pos         int
	deleted     string
	inserted    string
	caretBefore int
	caretAfter  int
}

const editorBarSize = 14 // scrollbar thickness (vertical and horizontal)

// EditorDefaults are applied to every new Editor before any options. Apps
// embedding the editor can set these once at startup instead of configuring
// each editor individually.
var EditorDefaults = struct {
	SoftWrap  bool // wrap long lines at the viewport width
	WrapWords bool // prefer breaking wrapped rows at word boundaries
}{WrapWords: true}

// EditorOption configures an Editor at construction time.
type EditorOption func(*Editor)

// WithSoftWrap sets the initial soft-wrap state.
func WithSoftWrap(v bool) EditorOption { return func(e *Editor) { e.SoftWrap = v } }

// WithWrapWords toggles word-boundary wrapping of wrapped rows.
func WithWrapWords(v bool) EditorOption { return func(e *Editor) { e.WrapWords = v } }

// NewEditor creates a scratch editor over initial content with the given highlighter.
func NewEditor(initial []byte, hl highlight.Highlighter, opts ...EditorOption) *Editor {
	return newEditor("", initial, hl, opts...)
}

// NewEditorFor creates an editor for a real file.
func NewEditorFor(path string, content []byte, opts ...EditorOption) *Editor {
	return newEditor(path, content, highlight.ForPath(path), opts...)
}

func newEditor(path string, content []byte, hl highlight.Highlighter, opts ...EditorOption) *Editor {
	engine := frameText()
	m := engine.MetricsMono()
	cellW, _ := engine.MeasureMono("n")
	e := &Editor{
		pt:               text.New(content),
		hl:               hl,
		Path:             path,
		selAnchor:        -1,
		blinkStart:       time.Now(),
		gutterW:          52,
		lineH:            m.LineHeight,
		textPad:          8,
		tabW:             engine.TabWidth(),
		fontGen:          engine.FontGen(),
		cellW:            cellW,
		WrapWords:        EditorDefaults.WrapWords,
		SoftWrap:         EditorDefaults.SoftWrap,
		wrapFull:         true,
		contentSizeDirty: true,
	}
	for _, opt := range opts {
		opt(e)
	}
	e.viewport = layout.New(layout.Box().FlexGrow(1))
	e.viewport.Clip = true
	e.viewport.Paint = e.paint
	e.viewport.OnMouse = e.onMouse

	e.vbar = NewScrollbar(&e.ScrollPx, &e.ContentHeight, editorBarSize)
	e.hbar = NewScrollbarAxis(Horizontal, &e.ScrollX, &e.ContentWidth, editorBarSize)

	e.host = layout.New(layout.Box().FlexGrow(1), e.viewport, e.vbar.host, e.hbar.host)
	e.hl.Update(e.pt.Bytes())
	e.markParsePending()
	e.lsp = lspManager.Open(path, content)
	e.lspUI.init()
	return e
}

// markParsePending opens a brief fast-poll window.
func (e *Editor) markParsePending() {
	if _, noop := e.hl.(highlight.Noop); noop {
		return
	}
	e.parseUntil = time.Now().Add(1500 * time.Millisecond)
}

// Close releases the highlighter's worker, the language-server document, and
// native resources.
func (e *Editor) Close() {
	e.hl.Close()
	if e.lsp != nil {
		e.lsp.Close()
	}
}

// VisualRowCount returns the number of visual rows the document currently
// lays out as: the wrapped row count when soft wrap is on, otherwise the
// logical line count.
func (e *Editor) VisualRowCount() int {
	if e.SoftWrap && len(e.rowPrefix) > 0 {
		return e.rowPrefix[len(e.rowPrefix)-1]
	}
	return e.pt.LineCount()
}

// Bytes returns the current document content.
func (e *Editor) Bytes() []byte { return e.pt.Bytes() }

// Modified reports whether there are unsaved changes.
func (e *Editor) Modified() bool { return e.modified }

// MarkSaved clears the modified flag.
func (e *Editor) MarkSaved() { e.modified = false }

// Update polls the highlighter, recomputes scroll extents, and drives scrollbars.
func (e *Editor) Update(m *input.Mouse) {
	if gen := frameText().FontGen(); gen != e.fontGen {
		e.fontGen = gen
		e.lineH = frameText().MetricsMono().LineHeight
		e.tabW = frameText().TabWidth()
		w, _ := frameText().MeasureMono("n")
		e.cellW = w
		e.wrapFull = true
		e.contentSizeDirty = true
	}
	if e.SoftWrap {
		// Wrapping depends on the viewport width; relayout when it changes.
		if _, cw, _, _ := e.scrollMetrics(); cw != e.lastWrapW {
			e.lastWrapW = cw
			e.contentSizeDirty = true
			// Do not set wrapFull: syncWrap already rebuilds when wrapCols
			// actually changes. A few pixels of scrollbar width must not
			// re-wrap hundreds of thousands of lines at the same column count.
		}
	}
	if toks, ok := e.hl.Poll(); ok {
		e.tokens = toks
		e.parseUntil = time.Time{}
	}
	e.lspUpdate(m)
	if e.contentSizeDirty {
		e.recomputeContentSize()
	}
	if m != nil {
		_, _, vShow, hShow := e.scrollMetrics()
		e.syncScrollbarLayout(vShow, hShow)
		vp := e.contentViewport()
		e.vbar.Update(m, vp)
		e.hbar.Update(m, vp)
	}
	e.clampScroll()
}

const editorLargeDocLines = 500

// maxShapedLine bounds how many bytes of a line the editor ever shapes in one
// piece. Files like minified JSON/XML can hold megabytes on a single line;
// shaping such a line takes minutes and gigabytes, so lines longer than this
// are shaped one visible window at a time and use mono-cell arithmetic for
// caret/hit-test geometry instead of the shaped line.
const maxShapedLine = 8192

// SetSoftWrap toggles soft wrapping of long lines at the viewport width.
func (e *Editor) SetSoftWrap(v bool) {
	if e.SoftWrap == v {
		return
	}
	e.SoftWrap = v
	if !v {
		e.lineRows = nil
		e.rowPrefix = nil
		e.ScrollX = 0
	}
	e.wrapFull = true
	e.contentSizeDirty = true
	if _, clientH, _, _ := e.scrollMetrics(); clientH > 0 {
		e.clampScroll()
		e.ensureCaretVisible()
	}
}

// SetWrapWords toggles breaking wrapped rows at word boundaries (falling back
// to hard cell breaks for words longer than a row).
func (e *Editor) SetWrapWords(v bool) {
	if e.WrapWords == v {
		return
	}
	e.WrapWords = v
	e.wrapFull = true
	e.contentSizeDirty = true
}

func (e *Editor) recomputeContentSize() {
	if e.SoftWrap {
		e.syncWrap()
		e.ContentHeight = float32(e.rowPrefix[len(e.rowPrefix)-1]) * e.lineH
		// The viewport fits the wrapped rows, so there is never horizontal
		// scrolling in wrap mode.
		e.ContentWidth = e.gutterW + e.textPad + float32(e.wrapCols)*e.cellW
		if cw := e.contentViewport().W; cw > 0 && e.ContentWidth > cw {
			e.ContentWidth = cw
		}
		e.contentSizeDirty = false
		return
	}
	engine := frameText()
	e.ContentHeight = float32(e.pt.LineCount()) * e.lineH
	lc := e.pt.LineCount()
	var maxTextW float32
	if lc > editorLargeDocLines {
		longest, byteLen := e.pt.LongestLine()
		cellW, _ := engine.MeasureMono("n")
		maxTextW = f32max(float32(byteLen)*cellW, e.lineTextWidth(longest))
	} else {
		for ln := 0; ln < lc; ln++ {
			if w := e.lineTextWidth(ln); w > maxTextW {
				maxTextW = w
			}
		}
	}
	e.ContentWidth = e.gutterW + e.textPad + maxTextW
	e.contentSizeDirty = false
}

// wrapTargetCols returns the row width in cells for the current viewport.
func (e *Editor) wrapTargetCols() int {
	textW := e.textClientW(e.contentViewport().W)
	cols := 80 // fallback until the viewport has been laid out
	if e.cellW > 0 && textW >= 8*e.cellW {
		cols = int(textW / e.cellW)
	}
	if cols < 1 {
		cols = 1
	}
	return cols
}

// syncWrap brings the wrap tables up to date with the document and viewport.
// Pure text edits re-wrap only the touched lines (line-relative row offsets
// stay valid for untouched lines); resizes, font changes, and line-count
// changes rebuild fully, with a consistency check that falls back to a full
// rebuild if the incremental bookkeeping is ever inconsistent.
func (e *Editor) syncWrap() {
	cols := e.wrapTargetCols()
	lc := e.pt.LineCount()
	if e.wrapFull || cols != e.wrapCols || len(e.lineRows) != lc {
		e.wrapCols = cols
		e.rebuildWrapFull()
		return
	}
	e.wrapCols = cols
	if !e.wrapDirtySet {
		return
	}
	lo, hi := e.wrapDirtyLo, e.wrapDirtyHi
	e.rebuildWrapRange(lo, hi)
}

// rebuildWrapFull re-wraps every logical line.
func (e *Editor) rebuildWrapFull() {
	lc := e.pt.LineCount()
	e.lineRows = make([][]wrapRow, lc)
	for ln := 0; ln < lc; ln++ {
		e.lineRows[ln] = e.wrapLineRows(e.lineView(ln))
	}
	e.rebuildRowPrefix()
	e.wrapFull = false
	e.wrapDirtySet = false
}

// rebuildWrapRange re-wraps lines [lo, hi]. Rows are line-relative, so lines
// outside the range keep their segments; any line-count change (adds, joins,
// undo/redo) is handled by resizing lineRows first.
func (e *Editor) rebuildWrapRange(lo, hi int) {
	lc := e.pt.LineCount()
	if lc == 0 {
		e.lineRows = nil
		e.rebuildRowPrefix()
		return
	}
	if lo < 0 {
		lo = 0
	}
	if lo >= lc {
		lo = lc - 1
	}
	if hi < lo {
		hi = lo
	}
	if hi >= lc {
		hi = lc - 1
	}
	// Resize to match the document: append empties or truncate, then fill the
	// dirty range. Lines beyond the range were not re-wrapped, so for undo/redo
	// (which can shift everything after the edit) the syncWrap caller widens
	// the range; the consistency check in syncWrap catches anything else.
	for len(e.lineRows) < lc {
		e.lineRows = append(e.lineRows, nil)
	}
	if len(e.lineRows) > lc {
		e.lineRows = e.lineRows[:lc]
	}
	for ln := lo; ln <= hi; ln++ {
		e.lineRows[ln] = e.wrapLineRows(e.lineView(ln))
	}
	e.rebuildRowPrefix()
	e.wrapDirtySet = false
}

// rebuildRowPrefix recomputes the cumulative visual-row counts per line.
func (e *Editor) rebuildRowPrefix() {
	lc := len(e.lineRows)
	if cap(e.rowPrefix) >= lc+1 {
		e.rowPrefix = e.rowPrefix[:lc+1]
	} else {
		e.rowPrefix = make([]int, lc+1)
	}
	e.rowPrefix[0] = 0
	for i := 0; i < lc; i++ {
		e.rowPrefix[i+1] = e.rowPrefix[i] + len(e.lineRows[i])
	}
}

// wrapLineRows splits view into visual rows of at most cols mono cells each
// (tab stops included; no font shaping). With WrapWords on, rows break at the
// last word boundary that fits, falling back to a hard cell break for runs
// longer than a row.
func (e *Editor) wrapLineRows(view []byte) []wrapRow {
	cols := e.wrapCols
	rows := make([]wrapRow, 0, len(view)/cols+2)
	rowStart := 0
	for rowStart < len(view) {
		col := 0
		i := rowStart
		lastBreak := -1
		for i < len(view) {
			adv := 1
			if view[i] == '\t' {
				adv = e.tabW - col%e.tabW
			}
			if col+adv > cols {
				break
			}
			col += adv
			i++
			if e.WrapWords && breakAfter(view, i) {
				lastBreak = i
			}
		}
		if i >= len(view) { // the rest of the line fits in this row
			rows = append(rows, wrapRow{int32(rowStart), int32(len(view))})
			break
		}
		// Overflow at byte i. If a word straddles the boundary, break before it
		// (the trailing space stays on the ending row); if the overflow byte is
		// itself a space, swallow the space run into the ending row — trailing
		// spaces are invisible when clipped past the row width.
		if e.WrapWords && lastBreak > rowStart && view[i] != ' ' {
			i = lastBreak
		} else if view[i] == ' ' {
			for i < len(view) && view[i] == ' ' {
				i++
			}
		}
		if i <= rowStart { // a single cell wider than a row
			i = rowStart + 1
		}
		if i < len(view) && !utf8.RuneStart(view[i]) { // never split a rune
			if b := monoRuneBoundary(view, i); b > rowStart {
				i = b
			}
		}
		rows = append(rows, wrapRow{int32(rowStart), int32(i)})
		rowStart = i
	}
	if len(rows) == 0 { // empty line: one empty row
		rows = append(rows, wrapRow{0, int32(len(view))})
	}
	return rows
}

// breakAfter reports whether position p in view is a word-wrap opportunity:
// a space ends at p-1, so the break keeps trailing whitespace on the row that
// is ending. Runs with no whitespace (minified XML, long words) never break
// here and fall back to hard cell breaks.
func breakAfter(view []byte, p int) bool {
	if p <= 0 || p >= len(view) {
		return false
	}
	return view[p-1] == ' ' || view[p-1] == '\t'
}

// wrapRowAbs returns the logical line and absolute byte range of visual row r.
func (e *Editor) wrapRowAbs(r int) (line, absStart, absEnd int) {
	line = e.rowOfVisual(r)
	ls := e.pt.LineStart(line)
	row := e.lineRows[line][r-e.rowPrefix[line]]
	return line, ls + int(row.start), ls + int(row.end)
}

// rowOfVisual returns the logical line containing global visual row r.
func (e *Editor) rowOfVisual(r int) int {
	lo, hi := 0, len(e.rowPrefix)-2
	for lo < hi {
		mid := (lo + hi) / 2 // lower midpoint: both branches shrink the range
		if e.rowPrefix[mid+1] <= r {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func (e *Editor) shapedLine(ln int) shape.Line {
	return frameText().LineMono(e.pt.Line(ln))
}

// isHuge reports whether line ln exceeds the windowed-shaping threshold.
func (e *Editor) isHuge(ln int) bool {
	return e.pt.LineLen(ln) > maxShapedLine
}

// rowOfByte returns the visual row index containing byte offset off. With soft
// wrap off there is one row per line, so this is the logical line index.
func (e *Editor) rowOfByte(off int) int {
	if !e.SoftWrap || len(e.rowPrefix) < 2 {
		return e.lineOf(off)
	}
	if off < 0 {
		off = 0
	}
	if n := e.pt.Len(); off > n {
		off = n
	}
	ln := e.lineOf(off)
	rel := off - e.pt.LineStart(ln)
	rows := e.lineRows[ln]
	lo, hi := 0, len(rows)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if int(rows[mid].start) <= rel {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return e.rowPrefix[ln] + lo
}

// xInRow returns the x of byte offset off relative to its visual row's left
// edge (the row's left edge in wrap mode, the line's left edge otherwise).
func (e *Editor) xInRow(off int) float32 {
	if !e.SoftWrap || len(e.rowPrefix) < 2 {
		return e.caretXInLine(e.lineOf(off), off)
	}
	ln, absStart, _ := e.wrapRowAbs(e.rowOfByte(off))
	ls := e.pt.LineStart(ln)
	view := e.lineView(ln)
	rel := off - ls
	start := absStart - ls
	if rel < start {
		rel = start
	}
	return e.monoPrefixWidth(view[start:], rel-start)
}

// lineView returns a zero-copy view of line ln's bytes.
func (e *Editor) lineView(ln int) []byte {
	return e.pt.LineSlice(ln, 0, e.pt.LineLen(ln))
}

// monoPrefixWidth returns the x width of view[:off] in mono cells, expanding
// tabs to tab stops (the column of a byte equals its index for non-tab bytes).
func (e *Editor) monoPrefixWidth(view []byte, off int) float32 {
	if off > len(view) {
		off = len(view)
	}
	cells := 0
	for i := 0; i < off; i++ {
		if view[i] == '\t' {
			cells += e.tabW - cells%e.tabW
		} else {
			cells++
		}
	}
	return float32(cells) * e.cellW
}

// monoLineWidth returns the full rendered width of a mono line via a byte
// scan (no shaping), tab-aware.
func (e *Editor) monoLineWidth(view []byte) float32 {
	return e.monoPrefixWidth(view, len(view))
}

// monoByteForX maps an x offset (relative to the line's text origin) to a byte
// offset within view using mono-cell arithmetic, tab-aware.
func (e *Editor) monoByteForX(view []byte, x float32) int {
	if x <= 0 || e.cellW <= 0 {
		return 0
	}
	cells := int(x/e.cellW + 0.5)
	col := 0
	for i := 0; i < len(view); i++ {
		adv := 1
		if view[i] == '\t' {
			adv = e.tabW - col%e.tabW
		}
		if col+adv > cells {
			// Snap to the nearer side of the cell the click landed in.
			if cells-col >= adv/2+adv%2 {
				return i + 1
			}
			return i
		}
		col += adv
	}
	return len(view)
}

// monoRuneBoundary returns the nearest rune boundary at or before off.
func monoRuneBoundary(view []byte, off int) int {
	if off >= len(view) {
		return len(view)
	}
	if off < 0 {
		return 0
	}
	for off > 0 && !utf8.RuneStart(view[off]) {
		off--
	}
	return off
}

// visibleWindow returns the byte window [lo, hi) of txt that covers the
// horizontal viewport (with margin), for windowed shaping of huge lines.
func (e *Editor) visibleWindow(txt []byte) (lo, hi int) {
	if e.cellW <= 0 {
		return 0, len(txt)
	}
	textW := e.textClientW(e.contentViewport().W)
	lo = int(e.ScrollX/e.cellW) - 64
	hi = int((e.ScrollX+textW)/e.cellW) + 128
	if lo < 0 {
		lo = 0
	}
	// Every byte advances at least one cell, so taking hi cells worth of bytes
	// always reaches the right edge even in tab-heavy lines.
	if hi > len(txt) {
		hi = len(txt)
	}
	if hi < lo {
		hi = lo
	}
	lo = monoRuneBoundary(txt, lo)
	hi = monoRuneBoundary(txt, hi)
	return lo, hi
}

// lineTextWidth returns the rendered width of a line; huge lines are measured
// with a mono-cell byte scan instead of shaping.
func (e *Editor) lineTextWidth(line int) float32 {
	if e.isHuge(line) {
		return e.monoLineWidth(e.lineView(line))
	}
	return e.shapedLine(line).Width
}

func (e *Editor) scrollMetrics() (clientW, clientH float32, vShow, hShow bool) {
	f := e.viewport.Frame
	clientW, clientH = f.W, f.H
	vShow = e.ContentHeight > clientH
	if vShow {
		clientW = f.W - editorBarSize
	}
	hShow = e.ContentWidth > clientW
	if hShow {
		clientH = f.H - editorBarSize
	}
	if e.ContentHeight > clientH {
		vShow = true
		clientW = f.W - editorBarSize
		hShow = e.ContentWidth > clientW
		if hShow {
			clientH = f.H - editorBarSize
		}
	}
	return clientW, clientH, vShow, hShow
}

func (e *Editor) contentViewport() render.Rect {
	f := e.viewport.Frame
	cw, ch, _, _ := e.scrollMetrics()
	return render.Rect{X: f.X, Y: f.Y, W: cw, H: ch}
}

func (e *Editor) syncScrollbarLayout(vShow, hShow bool) {
	if vShow {
		bottom := float32(0)
		if hShow {
			bottom = editorBarSize
		}
		e.vbar.host.Style = layout.Box().W(editorBarSize).AbsTop(0).AbsRight(0).AbsBottom(bottom)
		e.vbar.host.ReapplyStyle()
	}
	if hShow {
		right := float32(0)
		if vShow {
			right = editorBarSize
		}
		e.hbar.host.Style = layout.Box().H(editorBarSize).AbsLeft(0).AbsRight(right).AbsBottom(0)
		e.hbar.host.ReapplyStyle()
	}
}

func (e *Editor) textClientW(clientW float32) float32 {
	w := clientW - e.gutterW - e.textPad
	if w < 0 {
		return 0
	}
	return w
}

func (e *Editor) maxScrollX(clientW float32) float32 {
	textExtent := e.ContentWidth - e.gutterW - e.textPad
	return f32max(0, textExtent-e.textClientW(clientW))
}

func (e *Editor) clampScroll() {
	clientW, clientH, _, _ := e.scrollMetrics()
	e.ScrollPx = clampf(e.ScrollPx, 0, f32max(0, e.ContentHeight-clientH))
	e.ScrollX = clampf(e.ScrollX, 0, e.maxScrollX(clientW))
}

func (e *Editor) overScrollbar(m *input.Mouse) bool {
	_, _, vShow, hShow := e.scrollMetrics()
	if vShow && e.vbar.host.Frame.Contains(m.X, m.Y) {
		return true
	}
	if hShow && e.hbar.host.Frame.Contains(m.X, m.Y) {
		return true
	}
	return false
}

// Focus grants keyboard focus to the editor.
func (e *Editor) Focus() {
	e.focused = true
	e.blinkStart = time.Now()
}

// Blur removes keyboard focus from the editor.
func (e *Editor) Blur() { e.focused = false }

// Focused reports whether the editor has keyboard focus.
func (e *Editor) Focused() bool { return e.focused }

// CapturesTab reports that plain Tab should indent rather than move focus.
func (e *Editor) CapturesTab() bool { return true }

// FocusOnClick reports that clicking the editor should grant focus.
func (e *Editor) FocusOnClick() bool { return true }

// FocusEl returns the element used for click-to-focus hit testing.
func (e *Editor) FocusEl() *layout.Element { return e.host }

// Layout is the new ui.View entry point: it registers the editor with the
// frame's focus scope and self-schedules caret-blink / parse / hover repaints
// via the context, replacing the app-level AnimationWait wiring.
func (e *Editor) Layout(c *Ctx) *layout.Element {
	c.Focus().Add(e)
	e.Update(c.Mouse())
	if d, ok := e.AnimationWait(); ok {
		c.Animate(d)
	}
	if e.lspOverlay == nil {
		e.lspOverlay = layout.New(layout.Box())
		e.lspOverlay.Overlay = true
		e.lspOverlay.Paint = func(dl *render.DrawList, eng *shape.Engine) {
			e.PaintLSPOverlay(dl, eng)
		}
		e.lspOverlay.OnMouse = func(el *layout.Element, m *input.Mouse) {
			e.LSPOverlayMouse(el, m)
		}
	}
	c.Overlay(e.lspOverlay)
	return e.host
}

// Contains reports whether (x, y) falls inside the editor's last laid-out frame.
func (e *Editor) Contains(x, y float32) bool {
	return e.host != nil && e.host.Frame.Contains(x, y)
}

// AnimationWait implements the runtime's optional animation hook.
func (e *Editor) AnimationWait() (time.Duration, bool) {
	hoverWake, hoverPending := e.lspHoverWake()
	if !e.focused && !time.Now().Before(e.parseUntil) && !hoverPending {
		return 0, false
	}
	const half = 500 * time.Millisecond
	since := time.Since(e.blinkStart) % (2 * half)
	wait := half - (since % half)
	if e.dragging {
		if fast := 16 * time.Millisecond; wait > fast {
			wait = fast
		}
	}
	if time.Now().Before(e.parseUntil) {
		// Fast enough to pick up highlighter results; not a 60fps pin.
		if poll := 100 * time.Millisecond; wait > poll {
			wait = poll
		}
	}
	if hoverPending && hoverWake < wait {
		wait = hoverWake
	}
	if wait < 16*time.Millisecond && !e.dragging {
		wait = 16 * time.Millisecond
	}
	return wait, true
}

// ---------------------------------------------------------------------------
// Editing core
// ---------------------------------------------------------------------------

func (e *Editor) applyEdit(pos, delLen int, ins string, coalesceTyping bool) int {
	b := e.pt.Bytes()
	if pos < 0 {
		pos = 0
	}
	if pos+delLen > len(b) {
		delLen = len(b) - pos
	}
	deleted := string(b[pos : pos+delLen])
	startPt := e.pointOf(pos)
	oldEndPt := e.pointOf(pos + delLen)

	op := editOp{
		pos:         pos,
		deleted:     deleted,
		inserted:    ins,
		caretBefore: e.caret,
		caretAfter:  pos + len(ins),
	}

	if delLen > 0 {
		e.pt.Delete(pos, delLen)
	}
	if len(ins) > 0 {
		e.pt.Insert(pos, []byte(ins))
	}

	merged := false
	if coalesceTyping && e.canCoalesce && delLen == 0 && len(e.undo) > 0 && !strings.Contains(ins, "\n") {
		last := &e.undo[len(e.undo)-1]
		if last.deleted == "" && last.pos+len(last.inserted) == pos {
			last.inserted += ins
			last.caretAfter = op.caretAfter
			merged = true
		}
	}
	if !merged {
		e.undo = append(e.undo, op)
	}
	e.canCoalesce = coalesceTyping && delLen == 0 && !strings.Contains(ins, "\n")

	e.redo = e.redo[:0]
	e.modified = true
	e.contentSizeDirty = true
	e.selAnchor = -1
	e.caret = op.caretAfter
	e.afterMutation(highlight.Edit{
		StartByte: pos, OldEndByte: pos + delLen, NewEndByte: pos + len(ins),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(pos + len(ins)),
	})
	return e.caret
}

func (e *Editor) afterMutation(edit highlight.Edit) {
	e.hl.UpdateEdit(e.pt.Bytes(), edit)
	e.contentSizeDirty = true // Undo/Redo don't set this themselves
	e.markParsePending()
	e.blinkStart = time.Now()
	e.ensureCaretVisible()
	if e.search.open {
		e.runSearch()
	}
	e.lspDidChange()
	e.markWrapDirty(edit)
}

// markWrapDirty records which logical lines an edit touched so the next wrap
// sync re-wraps only those. Any edit that adds or removes lines (or anything
// else that shifts line numbering) falls back to a full rebuild.
func (e *Editor) markWrapDirty(edit highlight.Edit) {
	if !e.SoftWrap {
		return
	}
	if edit.NewEnd.Row != edit.OldEnd.Row {
		e.wrapFull = true
		return
	}
	lo, hi := edit.Start.Row, edit.NewEnd.Row
	if hi < lo {
		hi = lo
	}
	if e.wrapDirtySet {
		if lo < e.wrapDirtyLo {
			e.wrapDirtyLo = lo
		}
		if hi > e.wrapDirtyHi {
			e.wrapDirtyHi = hi
		}
	} else {
		e.wrapDirtyLo, e.wrapDirtyHi = lo, hi
		e.wrapDirtySet = true
	}
}

func (e *Editor) selRange() (int, int) {
	if e.selAnchor < 0 || e.selAnchor == e.caret {
		return e.caret, e.caret
	}
	if e.selAnchor < e.caret {
		return e.selAnchor, e.caret
	}
	return e.caret, e.selAnchor
}

func (e *Editor) hasSelection() bool {
	return e.selAnchor >= 0 && e.selAnchor != e.caret
}

func (e *Editor) replaceSelection(s string, coalesceTyping bool) {
	lo, hi := e.selRange()
	e.applyEdit(lo, hi-lo, s, coalesceTyping)
}

func (e *Editor) deleteSelection() bool {
	if !e.hasSelection() {
		return false
	}
	lo, hi := e.selRange()
	e.applyEdit(lo, hi-lo, "", false)
	return true
}

// Undo reverts the most recent edit.
func (e *Editor) Undo() {
	if len(e.undo) == 0 {
		return
	}
	op := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	startPt := e.pointOf(op.pos)
	oldEndPt := e.pointOf(op.pos + len(op.inserted))
	if len(op.inserted) > 0 {
		e.pt.Delete(op.pos, len(op.inserted))
	}
	if len(op.deleted) > 0 {
		e.pt.Insert(op.pos, []byte(op.deleted))
	}
	e.redo = append(e.redo, op)
	e.caret = op.caretBefore
	e.selAnchor = -1
	e.canCoalesce = false
	e.modified = true
	e.afterMutation(highlight.Edit{
		StartByte: op.pos, OldEndByte: op.pos + len(op.inserted), NewEndByte: op.pos + len(op.deleted),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(op.pos + len(op.deleted)),
	})
}

// Redo re-applies the most recently undone edit.
func (e *Editor) Redo() {
	if len(e.redo) == 0 {
		return
	}
	op := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	startPt := e.pointOf(op.pos)
	oldEndPt := e.pointOf(op.pos + len(op.deleted))
	if len(op.deleted) > 0 {
		e.pt.Delete(op.pos, len(op.deleted))
	}
	if len(op.inserted) > 0 {
		e.pt.Insert(op.pos, []byte(op.inserted))
	}
	e.undo = append(e.undo, op)
	e.caret = op.caretAfter
	e.selAnchor = -1
	e.canCoalesce = false
	e.modified = true
	e.afterMutation(highlight.Edit{
		StartByte: op.pos, OldEndByte: op.pos + len(op.deleted), NewEndByte: op.pos + len(op.inserted),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(op.pos + len(op.inserted)),
	})
}

// ---------------------------------------------------------------------------
// Input handling
// ---------------------------------------------------------------------------

// HandleText inserts text-producing characters.
func (e *Editor) HandleText(runes []rune) {
	if len(runes) == 0 {
		return
	}
	if e.search.open && e.search.focused {
		e.searchHandleText(runes)
		return
	}
	if len(runes) == 1 {
		r := runes[0]
		// Auto-close opening brackets.
		if closing, ok := bracketClose[r]; ok {
			if e.hasSelection() {
				lo, hi := e.selRange()
				selected := string(e.pt.Bytes()[lo:hi])
				e.applyEdit(lo, hi-lo, string(r)+selected+string(closing), false)
			} else {
				pos := e.caret
				e.applyEdit(pos, 0, string(r)+string(closing), false)
				e.caret = pos + utf8.RuneLen(r)
				e.ensureCaretVisible()
			}
			return
		}
		// Skip over auto-inserted closing bracket.
		if _, isClose := bracketOpen[r]; isClose {
			b := e.pt.Bytes()
			if e.caret < len(b) {
				next, sz := utf8.DecodeRune(b[e.caret:])
				if next == r {
					e.moveTo(e.caret+sz, false)
					return
				}
			}
		}
	}
	e.replaceSelection(string(runes), true)
	e.lspAfterType(runes)
}

// HandleKeys processes navigation, editing, and shortcut keys for one frame.
func (e *Editor) HandleKeys(keys []input.KeyEvent) {
	clip := frameClipboard()
	for _, ev := range keys {
		shift := ev.Mods.Has(input.ModShift)

		// Search bar consumes keys only while it is focused.
		if e.search.open && e.search.focused {
			e.searchHandleKey(ev)
			continue
		}

		// The completion popup, when open, captures navigation/accept keys.
		if e.lspUI.compOpen && e.handleCompletionKey(ev) {
			continue
		}

		if ev.Mods.Primary() {
			switch ev.Key {
			case input.KeySpace:
				e.requestCompletion()
			case input.KeyA:
				e.selAnchor = 0
				e.caret = e.pt.Len()
			case input.KeyC:
				e.copy(clip)
			case input.KeyX:
				e.cut(clip)
			case input.KeyV:
				e.paste(clip)
			case input.KeyZ:
				if shift {
					e.Redo()
				} else {
					e.Undo()
				}
			case input.KeyF:
				e.openSearch(false)
			case input.KeyH:
				e.openSearch(true)
			case input.KeyW:
				if ev.Mods.Has(input.ModAlt) {
					e.SetSoftWrap(!e.SoftWrap)
				}
			}
			continue
		}
		switch ev.Key {
		case input.KeyEscape:
			e.closeSearch()
		case input.KeyEnter:
			e.replaceSelection("\n", false)
		case input.KeyTab:
			e.replaceSelection("\t", false)
		case input.KeyBackspace:
			e.backspace()
		case input.KeyDelete:
			e.deleteForward()
		case input.KeyLeft:
			e.moveCluster(-1, shift)
		case input.KeyRight:
			e.moveCluster(1, shift)
		case input.KeyUp:
			e.moveVertical(-1, shift)
		case input.KeyDown:
			e.moveVertical(1, shift)
		case input.KeyHome:
			ln := e.lineOf(e.caret)
			e.moveTo(e.pt.LineStart(ln), shift)
		case input.KeyEnd:
			ln := e.lineOf(e.caret)
			e.moveTo(e.pt.LineStart(ln)+len(e.pt.Line(ln)), shift)
		}
	}
}

func (e *Editor) backspace() {
	if e.deleteSelection() {
		return
	}
	if e.caret == 0 {
		return
	}
	prev := e.prevRune(e.caret)
	e.applyEdit(prev, e.caret-prev, "", false)
}

func (e *Editor) deleteForward() {
	if e.deleteSelection() {
		return
	}
	next := e.nextRune(e.caret)
	if next == e.caret {
		return
	}
	e.applyEdit(e.caret, next-e.caret, "", false)
}

func (e *Editor) copy(clip input.Clipboard) {
	if clip == nil || !e.hasSelection() {
		return
	}
	lo, hi := e.selRange()
	clip.Set(string(e.pt.Bytes()[lo:hi]))
}

func (e *Editor) cut(clip input.Clipboard) {
	if clip == nil || !e.hasSelection() {
		return
	}
	lo, hi := e.selRange()
	clip.Set(string(e.pt.Bytes()[lo:hi]))
	e.applyEdit(lo, hi-lo, "", false)
}

func (e *Editor) paste(clip input.Clipboard) {
	if clip == nil {
		return
	}
	s := clip.Get()
	if s == "" {
		return
	}
	e.replaceSelection(s, false)
}

func (e *Editor) moveTo(off int, extend bool) {
	if off < 0 {
		off = 0
	}
	if n := e.pt.Len(); off > n {
		off = n
	}
	if extend {
		if e.selAnchor < 0 {
			e.selAnchor = e.caret
		}
	} else {
		e.selAnchor = -1
	}
	e.caret = off
	e.canCoalesce = false
	e.blinkStart = time.Now()
	e.ensureCaretVisible()
}

func (e *Editor) moveCluster(dir int, extend bool) {
	ln := e.lineOf(e.caret)
	ls := e.pt.LineStart(ln)
	off := e.caret - ls
	view := e.lineView(ln)
	var next int
	var gx float32
	switch {
	case e.SoftWrap:
		// Rune-step horizontally; the row's left edge anchors the goal x.
		next = monoRuneStep(view, off, dir)
		rowStart := e.rowOfByte(ls + next)
		_, absStart, _ := e.wrapRowAbs(rowStart)
		gx = e.monoPrefixWidth(view, next) - e.monoPrefixWidth(view, absStart-ls)
	case e.isHuge(ln):
		if off < 0 {
			off = 0
		}
		if off > len(view) {
			off = len(view)
		}
		if dir < 0 {
			if off > 0 {
				_, sz := utf8.DecodeLastRune(view[:off])
				next = off - sz
			}
		} else {
			if off < len(view) {
				_, sz := utf8.DecodeRune(view[off:])
				next = off + sz
			} else {
				next = off
			}
		}
		gx = e.monoPrefixWidth(view, next)
	default:
		line := e.shapedLine(ln)
		if dir < 0 {
			next = line.PrevCluster(off)
		} else {
			next = line.NextCluster(off, len(e.pt.Line(ln)))
		}
		gx = line.XForByte(next)
	}
	e.goalX = gx
	e.moveTo(ls+next, extend)
}

// monoRuneStep moves off by one rune within view, clamped to the ends.
func monoRuneStep(view []byte, off, dir int) int {
	if dir < 0 {
		if off <= 0 {
			return 0
		}
		_, sz := utf8.DecodeLastRune(view[:off])
		return off - sz
	}
	if off >= len(view) {
		return len(view)
	}
	_, sz := utf8.DecodeRune(view[off:])
	return off + sz
}

func (e *Editor) moveVertical(dir int, extend bool) {
	if e.SoftWrap && len(e.rowPrefix) > 1 {
		// Up/Down move one visual row, preserving the goal x within the row.
		r := e.rowOfByte(e.caret)
		if e.goalX == 0 {
			e.goalX = e.xInRow(e.caret)
		}
		r += dir
		if r < 0 {
			r = 0
		}
		if total := e.rowPrefix[len(e.rowPrefix)-1]; r >= total {
			r = total - 1
		}
		ln, absStart, absEnd := e.wrapRowAbs(r)
		ls := e.pt.LineStart(ln)
		view := e.pt.LineSlice(ln, absStart-ls, absEnd-ls)
		off := e.monoByteForX(view, e.goalX)
		e.moveTo(absStart+off, extend)
		return
	}
	line, _ := e.lineColOf(e.caret)
	if e.goalX == 0 {
		e.goalX = e.caretXInLine(e.lineOf(e.caret), e.caret)
	}
	line += dir
	if line < 0 {
		line = 0
	}
	if lc := e.pt.LineCount(); line >= lc {
		line = lc - 1
	}
	ls := e.pt.LineStart(line)
	var off int
	if e.isHuge(line) {
		off = e.monoByteForX(e.lineView(line), e.goalX)
	} else {
		off = e.shapedLine(line).ByteForX(e.goalX)
	}
	e.moveTo(ls+off, extend)
}

// ---------------------------------------------------------------------------
// Offset / line-column geometry
// ---------------------------------------------------------------------------

func (e *Editor) prevRune(off int) int {
	if off <= 0 {
		return 0
	}
	b := e.pt.Bytes()
	_, sz := utf8.DecodeLastRune(b[:off])
	return off - sz
}

func (e *Editor) nextRune(off int) int {
	b := e.pt.Bytes()
	if off >= len(b) {
		return len(b)
	}
	_, sz := utf8.DecodeRune(b[off:])
	return off + sz
}

func (e *Editor) pointOf(off int) highlight.Pt {
	line := e.lineOf(off)
	return highlight.Pt{Row: line, Col: off - e.pt.LineStart(line)}
}

func (e *Editor) lineOf(off int) int {
	lo, hi := 0, e.pt.LineCount()-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if e.pt.LineStart(mid) <= off {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

func (e *Editor) lineColOf(off int) (line, col int) {
	line = e.lineOf(off)
	ls := e.pt.LineStart(line)
	if off < ls {
		off = ls
	}
	return line, utf8.RuneCount(e.pt.Bytes()[ls:off])
}

func (e *Editor) offsetOf(line, col int) int {
	if line < 0 {
		line = 0
	}
	if lc := e.pt.LineCount(); line >= lc {
		line = lc - 1
	}
	ls := e.pt.LineStart(line)
	txt := e.pt.Line(line)
	n := 0
	for i := range txt {
		if n == col {
			return ls + i
		}
		n++
	}
	return ls + len(txt)
}

// ---------------------------------------------------------------------------
// Mouse: click to place caret, drag to select
// ---------------------------------------------------------------------------

// searchBarRect returns the bounding box of the search overlay panel.
func (e *Editor) searchBarRect() render.Rect {
	f := e.viewport.Frame
	rows := 1
	if e.search.replaceMode {
		rows = 2
	}
	_, _, vShow, _ := e.scrollMetrics()
	vbarW := float32(0)
	if vShow {
		vbarW = editorBarSize
	}
	return render.Rect{
		X: f.X + f.W - searchBarW - 6 - vbarW,
		Y: f.Y + 6,
		W: searchBarW,
		H: searchRowH * float32(rows),
	}
}

func (e *Editor) onMouse(el *layout.Element, m *input.Mouse) {
	if el.Frame.Contains(m.X, m.Y) && (m.ScrollY != 0 || m.ScrollX != 0) {
		e.vbar.ApplyWheel(m, el.Frame)
		e.hbar.ApplyWheel(m, el.Frame)
		e.clampScroll()
	}
	if e.overScrollbar(m) {
		return
	}

	// Search panel interaction.
	if e.search.open && m.Pressed {
		bar := e.searchBarRect()
		if bar.Contains(m.X, m.Y) {
			// Close button occupies the rightmost 20 px of the first row.
			if m.X >= bar.X+bar.W-20 && m.Y <= bar.Y+searchRowH {
				e.closeSearch()
			} else {
				e.search.focused = true
			}
			m.Consumed = true
			return
		}
		// Click outside panel: hand focus back to the editor text area.
		e.search.focused = false
	}

	f := el.Frame
	if m.Pressed && f.Contains(m.X, m.Y) {
		off := e.offsetAtPoint(m.X, m.Y)
		if m.Consumed {
			return
		}
		now := time.Now()
		if now.Sub(e.lastClickTime) < doubleClickInterval &&
			absF(m.X-e.lastClickX) <= multiClickSlop &&
			absF(m.Y-e.lastClickY) <= multiClickSlop {
			e.clickCount++
			if e.clickCount > 3 {
				e.clickCount = 1
			}
		} else {
			e.clickCount = 1
		}
		e.lastClickTime = now
		e.lastClickX, e.lastClickY = m.X, m.Y

		switch e.clickCount {
		case 2: // word selection
			lo, hi := e.wordRangeAt(off)
			e.selAnchor = lo
			e.caret = hi
			e.dragging = false
		case 3: // whole-line selection
			lo, hi := e.lineRangeAt(off)
			e.selAnchor = lo
			e.caret = hi
			e.dragging = false
		default: // single click: place caret and start a drag selection
			e.caret = off
			e.selAnchor = off
			e.dragging = true
		}
		e.canCoalesce = false
		e.blinkStart = time.Now()
		m.Consumed = true
	}
	if e.dragging && m.Down {
		e.caret = e.offsetAtPoint(m.X, m.Y)
	}
	if !m.Down {
		e.dragging = false
	}
}

func (e *Editor) caretXInLine(ln, caret int) float32 {
	ls := e.pt.LineStart(ln)
	if e.isHuge(ln) {
		return e.monoPrefixWidth(e.lineView(ln), caret-ls)
	}
	return e.shapedLine(ln).XForByte(caret - ls)
}

func (e *Editor) offsetAtPoint(px, py float32) int {
	f := e.viewport.Frame
	if e.SoftWrap && len(e.rowPrefix) > 1 {
		total := e.rowPrefix[len(e.rowPrefix)-1]
		row := int((py - f.Y + e.ScrollPx) / e.lineH)
		if row < 0 {
			row = 0
		}
		if row >= total {
			row = total - 1
		}
		x0 := f.X + e.gutterW + e.textPad - e.ScrollX
		ln, absStart, absEnd := e.wrapRowAbs(row)
		ls := e.pt.LineStart(ln)
		view := e.pt.LineSlice(ln, absStart-ls, absEnd-ls)
		off := e.monoByteForX(view, px-x0)
		e.goalX = e.monoPrefixWidth(view, off)
		return absStart + off
	}
	line := int((py - f.Y + e.ScrollPx) / e.lineH)
	if line < 0 {
		line = 0
	}
	if lc := e.pt.LineCount(); line >= lc {
		line = lc - 1
	}
	ls := e.pt.LineStart(line)
	x0 := f.X + e.gutterW + e.textPad - e.ScrollX
	var off int
	if e.isHuge(line) {
		view := e.lineView(line)
		off = e.monoByteForX(view, px-x0)
		e.goalX = e.monoPrefixWidth(view, off)
	} else {
		ln := e.shapedLine(line)
		off = ln.ByteForX(px - x0)
		e.goalX = ln.XForByte(off)
	}
	return ls + off
}

// doubleClickInterval is the maximum gap between presses that still counts as
// part of the same multi-click sequence; multiClickSlop bounds cursor movement.
const (
	doubleClickInterval = 400 * time.Millisecond
	multiClickSlop      = 4 // pixels
)

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// charClass groups runes for word-boundary expansion.
type charClass int

const (
	classSpace charClass = iota
	classWord
	classOther
)

func classOf(r rune) charClass {
	switch {
	case unicode.IsSpace(r):
		return classSpace
	case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
		return classWord
	default:
		return classOther
	}
}

// wordRangeAt returns the byte range of the "word" surrounding off: a run of
// like-classed runes (identifier characters, whitespace, or punctuation). When
// off sits just past a word (a common double-click landing spot), it expands the
// word to the left.
func (e *Editor) wordRangeAt(off int) (int, int) {
	b := e.pt.Bytes()
	n := len(b)
	if n == 0 {
		return 0, 0
	}
	if off > n {
		off = n
	}

	// Determine the class to expand over. Prefer the rune at off; if that is not
	// a word char but the preceding rune is, snap onto the preceding word.
	cls := classSpace
	if off < n {
		r, _ := utf8.DecodeRune(b[off:])
		cls = classOf(r)
	}
	if cls != classWord && off > 0 {
		pr, _ := utf8.DecodeLastRune(b[:off])
		if classOf(pr) == classWord {
			off = e.prevRune(off)
			cls = classWord
		}
	}
	if off >= n {
		off = e.prevRune(n)
	}

	lo := off
	for lo > 0 {
		pr, sz := utf8.DecodeLastRune(b[:lo])
		if classOf(pr) != cls {
			break
		}
		lo -= sz
	}
	hi := off
	for hi < n {
		r, sz := utf8.DecodeRune(b[hi:])
		if classOf(r) != cls {
			break
		}
		hi += sz
	}
	return lo, hi
}

// lineRangeAt returns the byte range of the whole line containing off, including
// its trailing newline when present so the selection spans the line break.
func (e *Editor) lineRangeAt(off int) (int, int) {
	line := e.lineOf(off)
	lo := e.pt.LineStart(line)
	if line+1 < e.pt.LineCount() {
		return lo, e.pt.LineStart(line + 1)
	}
	return lo, e.pt.Len()
}

func (e *Editor) ensureCaretVisible() {
	_, clientH, _, _ := e.scrollMetrics()
	if clientH <= 0 {
		return
	}
	row := e.rowOfByte(e.caret)
	top := float32(row) * e.lineH
	if top < e.ScrollPx {
		e.ScrollPx = top
	} else if top+e.lineH > e.ScrollPx+clientH {
		e.ScrollPx = top + e.lineH - clientH
	}
	if e.ScrollPx < 0 {
		e.ScrollPx = 0
	}
	if e.SoftWrap {
		e.ScrollX = 0 // rows always fit the viewport
		return
	}

	clientW, _, _, _ := e.scrollMetrics()
	textW := e.textClientW(clientW)
	if textW <= 0 {
		return
	}
	relX := e.caretXInLine(row, e.caret)
	if relX < e.ScrollX {
		e.ScrollX = relX
	} else if relX+2 > e.ScrollX+textW {
		e.ScrollX = relX + 2 - textW
	}
	if e.ScrollX < 0 {
		e.ScrollX = 0
	}
}

// ---------------------------------------------------------------------------
// Painting
// ---------------------------------------------------------------------------

func (e *Editor) colorAt(byteOffset int) highlight.ColorClass {
	c, _ := e.colorAtHint(byteOffset, 0)
	return c
}

// colorAtHint is colorAt with a token-index cursor. Glyphs are painted in
// increasing byte order, so a 460k-token file is an amortized O(1) walk
// instead of a cache-miss binary search per glyph.
func (e *Editor) colorAtHint(byteOffset, hint int) (highlight.ColorClass, int) {
	tok := e.tokens
	n := len(tok)
	if n == 0 {
		return highlight.ClassDefault, 0
	}
	i := hint
	if i < 0 {
		i = 0
	}
	if i > n {
		i = n
	}
	for i > 0 && tok[i-1].Start > byteOffset {
		i--
	}
	for i < n && tok[i].End <= byteOffset {
		i++
	}
	if i < n && byteOffset >= tok[i].Start && byteOffset < tok[i].End {
		return tok[i].Class, i
	}
	return highlight.ClassDefault, i
}

func tokenIndexAt(tok []highlight.Token, off int) int {
	lo, hi := 0, len(tok)
	for lo < hi {
		mid := (lo + hi) / 2
		if tok[mid].End <= off {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func (e *Editor) paint(dl *render.DrawList, _ *shape.Engine) {
	engine := frameText()
	th := theme.Current()
	f := e.viewport.Frame
	vp := e.contentViewport()
	gutter := render.Rect{X: f.X, Y: f.Y, W: e.gutterW, H: vp.H}
	textArea := render.Rect{X: f.X + e.gutterW, Y: vp.Y, W: vp.W - e.gutterW, H: vp.H}
	m := engine.MetricsMono()
	cellW, _ := engine.MeasureMono("n")
	// The line box (e.lineH) may be taller than the text itself when a
	// line-height multiplier is configured; center the natural text block
	// (ascent+descent) within the box so rows look vertically centered.
	textH := m.Ascent + m.Descent
	vpad := (e.lineH - textH) / 2

	dl.AddRect(f, th.Background)

	// paintRow is one visual row to paint: either a whole logical line (wrap
	// off, or a huge line's visible window) or one wrapped segment of a line.
	rows := e.paintRows[:0]
	first := int(e.ScrollPx / e.lineH)
	if first < 0 {
		first = 0
	}
	visibleCount := int(vp.H/e.lineH) + 2
	if e.SoftWrap && len(e.rowPrefix) > 1 {
		total := e.rowPrefix[len(e.rowPrefix)-1]
		first := minInt(first, total-1) // guard against stale scroll vs tables
		rLast := minInt(first+visibleCount, total)
		ln := e.rowOfVisual(first)
		ri := first - e.rowPrefix[ln]
		for r := first; r < rLast; r++ {
			for ln < len(e.lineRows) && ri >= len(e.lineRows[ln]) {
				ln++
				ri = 0
			}
			if ln >= len(e.lineRows) {
				break // tables stale; next Update rebuilds them
			}
			row := e.lineRows[ln][ri]
			rows = append(rows, paintRow{
				line: ln, rowStart: int(row.start), rowEnd: int(row.end),
				lineEnd: e.pt.LineLen(ln), visual: r, first: row.start == 0,
			})
			ri++
		}
	} else {
		lc := e.pt.LineCount()
		rLast := minInt(first+visibleCount, lc)
		for ln := first; ln < rLast; ln++ {
			ll := e.pt.LineLen(ln)
			pr := paintRow{line: ln, rowStart: 0, rowEnd: ll, lineEnd: ll, visual: ln, first: true}
			// Windowed shaping: huge lines only paint their visible byte window.
			if ll > maxShapedLine {
				pr.rowStart, pr.rowEnd = e.visibleWindow(e.lineView(ln))
				// The window sits at the scrolled-to position within the line.
				pr.xoff = e.monoPrefixWidth(e.lineView(ln), pr.rowStart)
			}
			rows = append(rows, pr)
		}
	}
	e.paintRows = rows

	selLo, selHi := e.selRange()
	hasSel := selLo != selHi
	x0 := f.X + e.gutterW + e.textPad - e.ScrollX

	// Current line highlight — drawn before text clips so it spans full width.
	if e.focused {
		r := e.rowOfByte(e.caret)
		lineY := f.Y + float32(r)*e.lineH - e.ScrollPx
		if lineY >= vp.Y-e.lineH && lineY < vp.Y+vp.H {
			lineHL := render.Color{
				R: th.Foreground.R, G: th.Foreground.G, B: th.Foreground.B, A: 0.05,
			}
			dl.PushClip(vp)
			dl.AddRect(render.Rect{X: vp.X, Y: lineY, W: vp.W, H: e.lineH}, lineHL)
			dl.PopClip()
		}
	}

	// Bracket match positions (computed once, used inside the loop).
	bracketA, bracketB, hasBrackets := e.bracketMatchOffset()
	var doc []byte
	if hasBrackets {
		doc = e.pt.Bytes()
	}

	dl.PushClip(vp)
	dl.PushClip(textArea)
	tokHint := 0
	if len(rows) > 0 && len(e.tokens) > 0 {
		tokHint = tokenIndexAt(e.tokens, e.pt.LineStart(rows[0].line)+rows[0].rowStart)
	}
	for _, pr := range rows {
		y := f.Y + float32(pr.visual)*e.lineH - e.ScrollPx
		if y >= vp.Y+vp.H {
			break
		}
		ls := e.pt.LineStart(pr.line)
		txt := e.lineView(pr.line)
		lineEnd := ls + pr.lineEnd

		// The painted slice is small in every mode: whole line, wrapped row, or
		// a huge line's visible window. Wrapped rows always start at the text
		// origin; only a non-wrapped huge line's window is offset within it.
		sline := frameText().LineMono(string(txt[pr.rowStart:pr.rowEnd]))
		drawX := x0 + pr.xoff
		if !e.SoftWrap && pr.rowStart == 0 {
			if w := e.gutterW + e.textPad + sline.Width; w > e.ContentWidth {
				e.ContentWidth = w
			}
		}

		// Block guide lines at each tab-stop within the line's indentation.
		if pr.first {
			if depth := indentDepth(txt, e.tabW); depth > 0 {
				guideCol := render.Color{
					R: th.Border.R, G: th.Border.G, B: th.Border.B, A: 0.98,
				}
				for col := e.tabW; col < depth; col += e.tabW {
					gx := x0 + float32(col)*cellW
					dl.AddRect(render.Rect{X: gx, Y: y, W: 1, H: e.lineH}, guideCol)
				}
			}
		}

		// Search match highlights.
		if e.search.open {
			matchCol := render.Color{R: 0.85, G: 0.72, B: 0.18, A: 0.35}
			activeCol := render.Color{R: 1.0, G: 0.60, B: 0.10, A: 0.60}
			for i, mr := range e.search.matches {
				if mr.hi <= ls || mr.lo >= lineEnd {
					continue
				}
				a := maxInt(mr.lo, ls)
				b := minInt(mr.hi, lineEnd)
				col := matchCol
				if i == e.search.current {
					col = activeCol
				}
				for _, rect := range e.windowRects(sline, a-ls, b-ls, pr.rowStart, pr.rowEnd, y, e.lineH) {
					dl.AddRect(render.Rect{X: drawX + rect[0], Y: rect[1], W: rect[2], H: rect[3]}, col)
				}
			}
		}

		// Selection highlights.
		if hasSel && selLo <= lineEnd && selHi >= ls {
			a := maxInt(selLo, ls)
			b := minInt(selHi, lineEnd)
			for _, rect := range e.windowRects(sline, a-ls, b-ls, pr.rowStart, pr.rowEnd, y, e.lineH) {
				dl.AddRect(render.Rect{X: drawX + rect[0], Y: rect[1], W: rect[2], H: rect[3]}, th.Selection)
			}
		}

		// Bracket match highlights.
		if hasBrackets {
			bracketHL := render.Color{R: 0.35, G: 0.90, B: 0.55, A: 0.40}
			for _, bpos := range [2]int{bracketA, bracketB} {
				if bpos >= ls+pr.rowStart && bpos < ls+pr.rowEnd {
					_, bsz := utf8.DecodeRune(doc[bpos:])
					for _, rect := range e.windowRects(sline, bpos-ls, bpos-ls+bsz, pr.rowStart, pr.rowEnd, y, e.lineH) {
						dl.AddRect(render.Rect{X: drawX + rect[0], Y: rect[1], W: rect[2], H: rect[3]}, bracketHL)
					}
				}
			}
		}

		topY := y + vpad
		tintBase := ls + pr.rowStart
		engine.DrawLineGlyphs(dl, sline, drawX, topY, func(byteOff int) render.Color {
			col, h := e.colorAtHint(tintBase+byteOff, tokHint)
			tokHint = h
			return th.SyntaxColor(col)
		})

		e.paintDiagnosticsLine(dl, ls, lineEnd, sline, drawX, pr.rowStart, pr.rowEnd, y)
	}
	if e.focused && e.caretVisible() {
		r := e.rowOfByte(e.caret)
		cx := x0 + e.xInRow(e.caret)
		cy := f.Y + float32(r)*e.lineH - e.ScrollPx
		dl.AddRect(render.Rect{X: cx, Y: cy + vpad, W: 2, H: textH}, th.Accent)
	}
	dl.PopClip()
	dl.PopClip()

	dl.AddRect(gutter, th.ChromeMuted)
	// 1px divider between gutter and text area.
	dl.AddRect(render.Rect{X: gutter.X + gutter.W - 1, Y: gutter.Y, W: 1, H: vp.H}, th.Border)
	dl.PushClip(render.Rect{X: gutter.X, Y: gutter.Y, W: gutter.W, H: vp.H})
	for _, pr := range rows {
		y := f.Y + float32(pr.visual)*e.lineH - e.ScrollPx
		if y >= vp.Y+vp.H {
			break
		}
		if !pr.first {
			continue // continuation rows carry no gutter number
		}
		num := strconv.Itoa(pr.line + 1)
		numW, _ := engine.MeasureMono(num)
		engine.DrawStringTopMono(dl, num, f.X+e.gutterW-numW-8, y+vpad, th.ForegroundSubtle)
	}
	dl.PopClip()

	// Search/replace overlay (drawn last, on top of everything).
	e.paintSearchBar(dl, engine)
}

// caretVisible returns the blink phase.
func (e *Editor) caretVisible() bool {
	if !e.focused {
		return false
	}
	const period = 1000 * time.Millisecond
	since := time.Since(e.blinkStart) % period
	return since < period/2
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// windowRects returns selection rectangles for the line-relative byte range
// [a, b), clamped to the shaped window [winLo, winHi) and translated into
// window-relative coordinates. For non-windowed lines winLo is 0 and winHi is
// the line length, so this is a plain SelectionRects call.
func (e *Editor) windowRects(sline shape.Line, a, b, winLo, winHi int, y, lineH float32) [][4]float32 {
	na := maxInt(a, winLo)
	nb := minInt(b, winHi)
	if nb <= na {
		return nil
	}
	return sline.SelectionRects(na-winLo, nb-winLo, y, lineH)
}

// ---------------------------------------------------------------------------
// Bracket matching
// ---------------------------------------------------------------------------

// bracketMatchOffset returns the byte positions of a matched bracket pair
// adjacent to the caret. Returns (posA, posB, true) when a match is found.
func (e *Editor) bracketMatchOffset() (int, int, bool) {
	if !e.focused {
		return 0, 0, false
	}
	b := e.pt.Bytes()
	// Check the character to the left of the caret first.
	if e.caret > 0 {
		r, sz := utf8.DecodeLastRune(b[:e.caret])
		if closing, ok := bracketClose[r]; ok {
			if pos := e.findMatchForward(e.caret, r, closing); pos >= 0 {
				return e.caret - sz, pos, true
			}
		} else if opening, ok := bracketOpen[r]; ok {
			if pos := e.findMatchBackward(e.caret-sz, r, opening); pos >= 0 {
				return e.caret - sz, pos, true
			}
		}
	}
	// Then check the character at (right of) the caret.
	if e.caret < len(b) {
		r, sz := utf8.DecodeRune(b[e.caret:])
		if closing, ok := bracketClose[r]; ok {
			if pos := e.findMatchForward(e.caret+sz, r, closing); pos >= 0 {
				return e.caret, pos, true
			}
		} else if opening, ok := bracketOpen[r]; ok {
			if pos := e.findMatchBackward(e.caret, r, opening); pos >= 0 {
				return e.caret, pos, true
			}
		}
	}
	return 0, 0, false
}

// findMatchForward scans forward from `from`, counting nesting of same/other
// brackets, and returns the byte position where depth reaches 0.
// `same` increases depth (it's the bracket we already have), `other` decreases.
// maxBracketScan bounds how far a match search may walk. Files of hundreds of
// thousands of lines would otherwise spend ~12ms per paint scanning for a pair.
const maxBracketScan = 64 * 1024

func (e *Editor) findMatchForward(from int, same, other rune) int {
	b := e.pt.Bytes()
	limit := from + maxBracketScan
	if limit > len(b) {
		limit = len(b)
	}
	depth := 1
	for i := from; i < limit; {
		r, sz := utf8.DecodeRune(b[i:])
		if r == same {
			depth++
		} else if r == other {
			depth--
			if depth == 0 {
				return i
			}
		}
		i += sz
	}
	return -1
}

// findMatchBackward scans backward from `from`, counting nesting of same/other
// brackets, and returns the byte position where depth reaches 0.
// `same` increases depth, `other` decreases.
func (e *Editor) findMatchBackward(from int, same, other rune) int {
	b := e.pt.Bytes()[:from]
	start := 0
	if len(b) > maxBracketScan {
		start = len(b) - maxBracketScan
	}
	depth := 1
	for i := len(b); i > start; {
		r, sz := utf8.DecodeLastRune(b[:i])
		if r == same {
			depth++
		} else if r == other {
			depth--
			if depth == 0 {
				return i - sz
			}
		}
		i -= sz
	}
	return -1
}

// ---------------------------------------------------------------------------
// Search / replace
// ---------------------------------------------------------------------------

func (e *Editor) openSearch(replaceMode bool) {
	e.search.open = true
	e.search.focused = true
	e.search.replaceMode = replaceMode
	e.search.focusField = 0
	if e.hasSelection() {
		lo, hi := e.selRange()
		e.search.query = string(e.pt.Bytes()[lo:hi])
		e.search.queryCaret = len(e.search.query)
	}
	e.runSearch()
}

func (e *Editor) closeSearch() {
	e.search.open = false
	e.search.focused = false
	e.search.matches = e.search.matches[:0]
}

// runSearch updates e.search.matches for the current query.
func (e *Editor) runSearch() {
	e.search.matches = e.search.matches[:0]
	if e.search.query == "" {
		e.search.current = -1
		return
	}
	b := e.pt.Bytes()
	q := []byte(e.search.query)
	for start := 0; start+len(q) <= len(b); {
		idx := bytes.Index(b[start:], q)
		if idx < 0 {
			break
		}
		pos := start + idx
		e.search.matches = append(e.search.matches, matchRange{pos, pos + len(q)})
		start = pos + len(q)
	}
	// Pick the match at or after the current caret.
	e.search.current = -1
	for i, mr := range e.search.matches {
		if mr.lo >= e.caret {
			e.search.current = i
			break
		}
	}
	if e.search.current == -1 && len(e.search.matches) > 0 {
		e.search.current = 0
	}
}

func (e *Editor) searchNext() {
	if len(e.search.matches) == 0 {
		return
	}
	e.search.current = (e.search.current + 1) % len(e.search.matches)
	e.jumpToMatch(e.search.current)
}

func (e *Editor) searchPrev() {
	if len(e.search.matches) == 0 {
		return
	}
	n := len(e.search.matches)
	e.search.current = (e.search.current - 1 + n) % n
	e.jumpToMatch(e.search.current)
}

func (e *Editor) jumpToMatch(idx int) {
	if idx < 0 || idx >= len(e.search.matches) {
		return
	}
	mr := e.search.matches[idx]
	e.selAnchor = mr.lo
	e.caret = mr.hi
	e.ensureCaretVisible()
}

// searchField returns pointers to the focused field's text and caret.
func (e *Editor) searchField() (*string, *int) {
	if e.search.replaceMode && e.search.focusField == 1 {
		return &e.search.replace, &e.search.replaceCaret
	}
	return &e.search.query, &e.search.queryCaret
}

func (e *Editor) searchHandleText(runes []rune) {
	field, caret := e.searchField()
	s := string(runes)
	*field = (*field)[:*caret] + s + (*field)[*caret:]
	*caret += len(s)
	if e.search.focusField == 0 {
		e.runSearch()
		if e.search.current >= 0 {
			e.jumpToMatch(e.search.current)
		}
	}
}

func (e *Editor) searchHandleKey(ev input.KeyEvent) {
	shift := ev.Mods.Has(input.ModShift)
	switch ev.Key {
	case input.KeyEscape:
		e.closeSearch()
	case input.KeyEnter:
		if e.search.replaceMode && e.search.focusField == 1 {
			e.doReplace()
		} else if shift {
			e.searchPrev()
		} else {
			e.searchNext()
		}
	case input.KeyTab:
		if e.search.replaceMode {
			e.search.focusField = 1 - e.search.focusField
		}
	case input.KeyBackspace:
		field, caret := e.searchField()
		if *caret > 0 {
			_, sz := utf8.DecodeLastRune([]byte((*field)[:*caret]))
			*field = (*field)[:*caret-sz] + (*field)[*caret:]
			*caret -= sz
			if e.search.focusField == 0 {
				e.runSearch()
			}
		}
	case input.KeyDelete:
		field, caret := e.searchField()
		if *caret < len(*field) {
			_, sz := utf8.DecodeRune([]byte((*field)[*caret:]))
			*field = (*field)[:*caret] + (*field)[*caret+sz:]
			if e.search.focusField == 0 {
				e.runSearch()
			}
		}
	case input.KeyLeft:
		field, caret := e.searchField()
		if *caret > 0 {
			_, sz := utf8.DecodeLastRune([]byte((*field)[:*caret]))
			*caret -= sz
		}
	case input.KeyRight:
		field, caret := e.searchField()
		if *caret < len(*field) {
			_, sz := utf8.DecodeRune([]byte((*field)[*caret:]))
			*caret += sz
		}
	case input.KeyHome:
		_, caret := e.searchField()
		*caret = 0
	case input.KeyEnd:
		field, caret := e.searchField()
		*caret = len(*field)
	}
	// Cmd/Ctrl shortcuts while search bar is open.
	if ev.Mods.Primary() {
		switch ev.Key {
		case input.KeyH:
			e.search.replaceMode = !e.search.replaceMode
		case input.KeyF:
			e.search.focusField = 0
		}
	}
}

func (e *Editor) doReplace() {
	cur := e.search.current
	if cur < 0 || cur >= len(e.search.matches) {
		return
	}
	mr := e.search.matches[cur]
	e.applyEdit(mr.lo, mr.hi-mr.lo, e.search.replace, false)
	e.runSearch()
	if len(e.search.matches) > 0 {
		if cur >= len(e.search.matches) {
			cur = 0
		}
		e.search.current = cur
		e.jumpToMatch(cur)
	}
}

func (e *Editor) doReplaceAll() {
	for i := len(e.search.matches) - 1; i >= 0; i-- {
		mr := e.search.matches[i]
		e.applyEdit(mr.lo, mr.hi-mr.lo, e.search.replace, false)
	}
	e.runSearch()
}

// ---------------------------------------------------------------------------
// Search bar painting
// ---------------------------------------------------------------------------

const (
	searchBarW = float32(310)
	searchRowH = float32(30)
)

func (e *Editor) paintSearchBar(dl *render.DrawList, engine *shape.Engine) {
	if !e.search.open {
		return
	}
	th := theme.Current()
	m := engine.MetricsMono()

	bar := e.searchBarRect()
	barX, barY, barH := bar.X, bar.Y, bar.H

	// Background + border.
	dl.AddRoundedRectBorder(
		render.Rect{X: barX, Y: barY, W: searchBarW, H: barH},
		4, 1, th.Chrome, th.Border,
	)

	textTopY := func(row int) float32 {
		return barY + float32(row)*searchRowH + (searchRowH-m.LineHeight)/2
	}

	const labelW = float32(44)
	const rightPad = float32(20) // space for the × close button
	inputX := barX + labelW
	inputW := searchBarW - labelW - rightPad - 60 // 60 = count area

	noMatch := e.search.query != "" && len(e.search.matches) == 0
	searchFieldBg := th.Background
	if noMatch {
		// Tint the input red when query has no matches.
		searchFieldBg = render.Color{R: 0.40, G: 0.10, B: 0.10, A: 1}
	}

	// --- Search row ---
	engine.DrawStringTopMono(dl, "Find:", barX+5, textTopY(0), th.TextDim)
	dl.AddRoundedRect(render.Rect{X: inputX, Y: barY + 4, W: inputW, H: searchRowH - 8}, 2, searchFieldBg)

	// Search text + caret (clipped to field).
	dl.PushClip(render.Rect{X: inputX, Y: barY, W: inputW, H: searchRowH})
	textColor := th.Foreground
	if noMatch {
		textColor = render.Color{R: 0.95, G: 0.40, B: 0.40, A: 1}
	}
	engine.DrawStringTopMono(dl, e.search.query, inputX+4, textTopY(0), textColor)
	if e.search.focused && e.search.focusField == 0 {
		qw, _ := engine.MeasureMono(e.search.query[:e.search.queryCaret])
		cx := inputX + 4 + qw
		dl.AddRect(render.Rect{X: cx, Y: barY + 5, W: 1.5, H: searchRowH - 10}, th.Accent)
	}
	dl.PopClip()

	// Match count.
	var countStr string
	switch {
	case len(e.search.matches) > 0:
		countStr = strconv.Itoa(e.search.current+1) + "/" + strconv.Itoa(len(e.search.matches))
	case e.search.query != "":
		countStr = "0/0"
	}
	countX := inputX + inputW + 4
	engine.DrawStringTopMono(dl, countStr, countX, textTopY(0), th.TextDim)

	// Close button.
	closeX := barX + searchBarW - 16
	engine.DrawStringTopMono(dl, "x", closeX, textTopY(0), th.ForegroundMuted)

	// --- Replace row ---
	if e.search.replaceMode {
		engine.DrawStringTopMono(dl, "Repl:", barX+5, textTopY(1), th.TextDim)
		replFieldY := barY + searchRowH
		dl.AddRoundedRect(render.Rect{X: inputX, Y: replFieldY + 4, W: inputW, H: searchRowH - 8}, 2, th.Background)

		dl.PushClip(render.Rect{X: inputX, Y: replFieldY, W: inputW, H: searchRowH})
		engine.DrawStringTopMono(dl, e.search.replace, inputX+4, textTopY(1), th.Foreground)
		if e.search.focused && e.search.focusField == 1 {
			rw, _ := engine.MeasureMono(e.search.replace[:e.search.replaceCaret])
			cx := inputX + 4 + rw
			dl.AddRect(render.Rect{X: cx, Y: replFieldY + 5, W: 1.5, H: searchRowH - 10}, th.Accent)
		}
		dl.PopClip()

		// Replace-one and replace-all buttons.
		btnX := countX
		engine.DrawStringTopMono(dl, "->1", btnX, textTopY(1), th.Accent)
		bw, _ := engine.MeasureMono("->1")
		engine.DrawStringTopMono(dl, "->*", btnX+bw+4, textTopY(1), th.Accent)
	}
}
