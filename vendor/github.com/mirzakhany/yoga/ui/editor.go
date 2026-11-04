package ui

import (
	"bytes"
	"math"
	"slices"
	"sort"
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
	indent     int32 // cells a continuation row is shifted right (WrapIndent)
}

// WrapIndent picks where a soft-wrapped line's continuation rows start.
type WrapIndent int

const (
	// WrapIndentNone starts continuation rows at column 0 (the default).
	WrapIndentNone WrapIndent = iota
	// WrapIndentSame lines continuation rows up with the line's leading
	// whitespace.
	WrapIndentSame
	// WrapHanging indents continuation rows one tab stop past the line's
	// leading whitespace.
	WrapHanging
)

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

// AutoIndent is what Enter carries onto the new line.
type AutoIndent int

const (
	// AutoIndentDefault follows EditorDefaults.AutoIndent.
	AutoIndentDefault AutoIndent = iota
	// AutoIndentNone starts the new line at column 0.
	AutoIndentNone
	// AutoIndentKeep copies the current line's leading whitespace.
	AutoIndentKeep
	// AutoIndentBrackets keeps the indent and adds one level after an opening
	// bracket or a trailing colon. Enter between a bracket pair puts the
	// closing bracket on its own line.
	AutoIndentBrackets
)

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

// indentDepth returns the column depth of leading whitespace in s,
// counting spaces as 1 and tabs as advancing to the next tabW stop.
func indentDepth(s []byte, tabW int) int {
	n := 0
	for _, ch := range string(s) {
		switch ch {
		case ' ':
			n++
		case '\t':
			n = ((n / tabW) + 1) * tabW
		default:
			return n
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

	// Undo/redo. undoBytes is the text held by undo, bounded by undoMaxBytes
	// along with the step count undoMaxSteps; the oldest steps drop first.
	undo         []editOp
	redo         []editOp
	undoBytes    int
	undoMaxSteps int
	undoMaxBytes int
	lastMerge    editMerge // kind of the last edit, mergeNone after anything else
	// Every undo step gets a fresh id from opSeq; savedID is the id of the
	// newest step when the text was last saved (0 for none), so undoing back
	// to the saved text reads as unmodified again.
	opSeq   uint64
	savedID uint64
	// extra are the selections beyond the primary one (caret, selAnchor);
	// see editor_multi.go.
	extra      []cursor
	inMulti    bool // multi is running an action per selection
	undoGroup  int  // nonzero: edits now share this undo group
	groupSeq   int
	lastEditAt time.Time

	// tokens is the latest syntax highlight result. It covers only the byte
	// range last requested from the highlighter (hlLo..hlHi) when the
	// highlighter supports scoping, so bytes outside it have no tokens and
	// paint in the default color.
	tokens     []highlight.Token
	hlLo, hlHi int
	hlRangeSet bool

	// lsp connects this document to a language server for diagnostics, hover,
	// and completion. It stays nil until the editor first updates (see
	// lspOpen), so a server is started only for editors actually shown.
	lsp       lsp.Doc
	lspOpened bool
	lspUI     lspState

	contentSizeDirty bool
	revision         uint64
	lineMarks        map[int]LineMarkKind
	inline           inlineSuggestion

	gutterW float32
	lineH   float32
	textPad float32
	goalX   float32 // x Up/Down keep to, while goalSet
	goalSet bool

	tabW    int     // tab width in columns: tabSet, else the engine's
	tabSet  int     // SetTabWidth; 0 follows the engine config
	fontGen uint64  // last-seen engine font generation
	cellW   float32 // mono cell advance width, mirrors fontGen

	// Soft wrap: when true, long logical lines are laid out into visual rows
	// that fit the viewport width. lineRows holds each line's row segments as
	// line-relative byte ranges, so an edit only invalidates the touched
	// lines; rowPrefix maps global visual row indexes to lines. Rebuilt fully
	// on resize/font changes, incrementally on edits.
	SoftWrap                 bool
	WrapWords                bool
	WrapIndent               WrapIndent
	wrapSet                  bool // SoftWrap was chosen for this editor; ignore EditorDefaults.SoftWrap
	indentSet                bool // SetIndentSpaces was called; ignore EditorDefaults.IndentSpaces
	indentSpaces             bool
	autoIndent               AutoIndent
	noGutter                 bool // WithoutGutter: never show line numbers
	gutter                   *GutterSpec
	shareContent             bool // construction-time: build over the caller's slice, no copy
	readOnly                 bool // selection and copy work; edits are ignored
	lineRows                 [][]wrapRow
	rowPrefix                []int   // rowPrefix[i] = visual rows before logical line i (len = LineCount+1)
	revealCaret              bool    // scroll the caret into view once the wrap tables are rebuilt
	revealCenter             bool    // the pending reveal centers the caret row (CenterCaret)
	revealWoke               bool    // a frame was requested for the pending reveal
	pastEndH                 float32 // blank space added below the last line (ScrollPastEnd)
	anchorSet                bool    // restore anchorOff to anchorY once rows reflow
	anchorOff                int     // byte whose row keeps its place across a wrap change
	anchorY                  float32 // that row's y relative to the top of the view
	wrapCols                 int     // columns per visual row used when wrapping
	lastWrapW                float32 // last client width the wrap table was built for
	wrapFull                 bool    // next sync must rebuild every line
	wrapDirtyLo, wrapDirtyHi int     // line range touched by pending edits
	wrapDirtySet             bool

	search searchState
	box    boxSelect
	fold   foldState

	addGoalX float32 // x AddCursorLine keeps to while it adds cursors

	keymapOwn  []EditorBinding // SetKeymap; nil follows EditorDefaults.Keymap
	keymapC    *compiledKeymap
	keyPending *editorChord // first chord of a sequence, waiting for the next

	comment    highlight.Comment // SetLineComment
	commentSet bool

	// ContextMenu, when set, edits the right-click menu: it receives the
	// standard edit items and returns the ones to show, so an app can add its
	// own (Save…, Format, …). Returning none turns the menu off and leaves the
	// right-click to widgets behind the editor.
	ContextMenu func(items []MenuItem) []MenuItem
	// Suggest, when set, offers completions the application knows about, such
	// as template variables. It is asked after every edit, ahead of the
	// language server, and returns candidates plus the byte range they
	// replace.
	Suggest SuggestFunc
	// OnInlineAccept, when set, runs after Tab inserts the inline suggestion
	// (see SetInlineSuggestion), with the inserted text.
	OnInlineAccept func(text string)
	// HoverInfo, when set, explains the byte offset under a resting pointer.
	// It runs whether or not a language server is attached, and its card is
	// shown above any diagnostics and server hover text.
	HoverInfo HoverInfoFunc
	// OnModClick, when set, runs for a ⌘-click (Ctrl-click off macOS) on the
	// text with the byte offset under the pointer, such as to go to a
	// definition. Returning true consumes the click; the caret moves there.
	OnModClick func(off int) bool
	// KeyFilter and TextFilter see keys and typed text before the editor
	// does; returning true consumes them. A modal keymap such as Vim's sets
	// both, so that in normal mode letters are commands, not text.
	KeyFilter  func(ev input.KeyEvent) bool
	TextFilter func(runes []rune) bool
	// CaretStyle is the caret's shape; Vim's normal mode uses CaretBlock.
	CaretStyle CaretStyle
	menu       editMenu

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
	group       int // edits with the same nonzero group undo together
	id          uint64
	// selBefore and selAfter are every selection, primary first, before and
	// after a group: set on its first and last op, so undo and redo put back
	// all the cursors a multi-cursor edit had.
	selBefore, selAfter []cursor
}

func (op *editOp) size() int { return len(op.deleted) + len(op.inserted) }

const editorBarSize = 14 // scrollbar thickness (vertical and horizontal)

// EditorDefaults configure every Editor. Apps embedding the editor can set
// these once, for example from user preferences, instead of configuring each
// editor individually.
//
// WrapWords and the undo limits are read when an editor is built. The others
// are read live, so changing them updates editors that are already open.
// SoftWrap applies to editors whose wrap was not set by WithSoftWrap or
// SetSoftWrap, and LineNumbers to editors built without WithoutGutter.
var EditorDefaults = struct {
	SoftWrap          bool       // wrap long lines at the viewport width
	WrapWords         bool       // prefer breaking wrapped rows at word boundaries
	UndoSteps         int        // most undo steps kept; <= 0 means no limit
	UndoBytes         int        // most text bytes kept by undo; <= 0 means no limit
	LineNumbers       bool       // show the line-number gutter
	AutoCloseBrackets bool       // typing ( [ { inserts the closing bracket too
	AutoCloseQuotes   bool       // typing " ' ` inserts the closing quote too
	IndentSpaces      bool       // Tab inserts spaces up to the next tab stop
	AutoIndent        AutoIndent // what Enter carries to the new line; Default means None
	ScrollPastEnd     bool       // let the last line scroll up to the top of the view
	// Keymap replaces the built-in shortcuts (DefaultEditorKeymap) for
	// editors without their own SetKeymap; nil keeps the built-in ones.
	Keymap []EditorBinding
}{
	WrapWords:         true,
	UndoSteps:         1000,
	UndoBytes:         32 << 20,
	LineNumbers:       true,
	AutoCloseBrackets: true,
}

// editorGutterW is the width of the line-number gutter.
const editorGutterW = 52

// EditorOption configures an Editor at construction time.
type EditorOption func(*Editor)

// WithSoftWrap sets the initial soft-wrap state.
func WithSoftWrap(v bool) EditorOption {
	return func(e *Editor) { e.SoftWrap, e.wrapSet = v, true }
}

// WithUndoLimit bounds the undo history to steps edits and bytes of text,
// dropping the oldest steps first. A value <= 0 lifts that limit. The newest
// step is always kept, however large.
func WithUndoLimit(steps, bytes int) EditorOption {
	return func(e *Editor) { e.undoMaxSteps, e.undoMaxBytes = steps, bytes }
}

// WithWrapIndent sets where continuation rows of a wrapped line start.
func WithWrapIndent(m WrapIndent) EditorOption { return func(e *Editor) { e.WrapIndent = m } }

// WithWrapWords toggles word-boundary wrapping of wrapped rows.
func WithWrapWords(v bool) EditorOption { return func(e *Editor) { e.WrapWords = v } }

// WithSharedContent builds the editor over the content slice without copying
// it. The editor only ever reads that slice — edits are stored separately — so
// the caller must never modify it or its contents afterwards, though it may
// keep reading them.
//
// Use it when the bytes are already immutable, such as a response body held for
// display. It saves a full copy of the document, which for a large body is tens
// of megabytes per editor.
func WithSharedContent() EditorOption { return func(e *Editor) { e.shareContent = true } }

// WithReadOnly makes the editor a view: text can be selected, searched and
// copied, but typing, paste, cut, undo and replace leave it unchanged.
func WithReadOnly() EditorOption { return func(e *Editor) { e.readOnly = true } }

// WithSuggest sets the application completion source (see Editor.Suggest).
func WithSuggest(fn SuggestFunc) EditorOption { return func(e *Editor) { e.Suggest = fn } }

// WithHoverInfo sets the application hover source (see Editor.HoverInfo).
func WithHoverInfo(fn HoverInfoFunc) EditorOption { return func(e *Editor) { e.HoverInfo = fn } }

// WithoutGutter hides the line-number gutter, for text where line numbers
// carry no meaning, such as a log view.
func WithoutGutter() EditorOption { return func(e *Editor) { e.noGutter = true } }

// ReadOnly reports whether the editor ignores edits (see WithReadOnly).
func (e *Editor) ReadOnly() bool { return e.readOnly }

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
		hl:               hl,
		Path:             path,
		selAnchor:        -1,
		blinkStart:       time.Now(),
		lineH:            m.LineHeight,
		textPad:          8,
		tabW:             engine.TabWidth(),
		fontGen:          engine.FontGen(),
		cellW:            cellW,
		WrapWords:        EditorDefaults.WrapWords,
		SoftWrap:         EditorDefaults.SoftWrap,
		undoMaxSteps:     EditorDefaults.UndoSteps,
		undoMaxBytes:     EditorDefaults.UndoBytes,
		wrapFull:         true,
		contentSizeDirty: true,
	}
	for _, opt := range opts {
		opt(e)
	}
	e.gutterW = e.wantGutterW()
	// Options decide whether the content is copied, so build the storage after
	// applying them.
	if e.shareContent {
		e.pt = text.NewShared(content)
	} else {
		e.pt = text.New(content)
	}
	e.viewport = layout.New(layout.Box().FlexGrow(1))
	e.viewport.Clip = true
	e.viewport.Paint = e.paint
	e.viewport.OnMouse = e.onMouse

	e.vbar = NewScrollbar(&e.ScrollPx, &e.ContentHeight, editorBarSize)
	e.hbar = NewScrollbarAxis(Horizontal, &e.ScrollX, &e.ContentWidth, editorBarSize)

	e.host = layout.New(layout.Box().FlexGrow(1), e.viewport, e.vbar.host, e.hbar.host)
	// Ask for a plausible first screen before the initial parse: the editor has
	// not been laid out yet, and classifying the whole document just to show
	// its first lines is what this range exists to avoid. The first Update
	// widens or narrows it to the real viewport.
	if rh, ok := hl.(highlight.RangeHighlighter); ok {
		last := minInt(initialHighlightLines, maxInt(e.pt.LineCount()-1, 0))
		e.hlLo, e.hlHi = 0, e.pt.LineStart(last)+e.pt.LineLen(last)
		e.hlRangeSet = true
		rh.SetRange(e.hlLo, e.hlHi)
	}
	e.hl.Update(e.pt.Bytes())
	e.markParsePending()
	e.lspUI.init()
	return e
}

// initialHighlightLines is the range requested before the editor knows its
// viewport — generous enough to cover a first screen on a tall window.
const initialHighlightLines = 400

// highlightMarginScreens is how many extra screenfuls above and below the
// viewport are highlighted. Scrolling within the margin needs no new request,
// so ordinary scrolling never waits on the highlighter.
const highlightMarginScreens = 2

// visibleLineRange returns the logical line range the viewport shows, widened
// by highlightMarginScreens on each side.
func (e *Editor) visibleLineRange() (first, last int) {
	lc := e.pt.LineCount()
	if lc == 0 {
		return 0, 0
	}
	vp := e.contentViewport()
	rows := 1
	if e.lineH > 0 && vp.H > 0 {
		rows = int(vp.H/e.lineH) + 2
	}
	margin := rows * highlightMarginScreens

	firstRow := 0
	if e.lineH > 0 {
		firstRow = int(e.ScrollPx / e.lineH)
	}
	if firstRow < 0 {
		firstRow = 0
	}

	if e.SoftWrap && len(e.rowPrefix) > 1 {
		total := e.rowPrefix[len(e.rowPrefix)-1]
		lo := maxInt(firstRow-margin, 0)
		hi := minInt(firstRow+rows+margin, total)
		first = e.rowOfVisual(minInt(lo, maxInt(total-1, 0)))
		last = e.rowOfVisual(minInt(maxInt(hi-1, 0), maxInt(total-1, 0)))
	} else {
		total := e.lineRowCount()
		first = e.rowLine(minInt(maxInt(firstRow-margin, 0), maxInt(total-1, 0)))
		last = e.rowLine(maxInt(minInt(firstRow+rows+margin, total)-1, 0))
	}
	if first < 0 {
		first = 0
	}
	if last < first {
		last = first
	}
	if last >= lc {
		last = lc - 1
	}
	return first, last
}

// syncHighlightRange asks the highlighter for the lines around the viewport.
//
// Tokens outside the requested range are absent, which paints as the default
// color — and nothing outside the viewport is painted, so the reader never sees
// the difference. The request is skipped while the wanted range is still inside
// the one already asked for, which is what keeps scrolling from generating a
// request per frame.
func (e *Editor) syncHighlightRange() {
	rh, ok := e.hl.(highlight.RangeHighlighter)
	if !ok {
		return
	}
	first, last := e.visibleLineRange()
	lo := e.pt.LineStart(first)
	hi := e.pt.LineStart(last) + e.pt.LineLen(last)
	if hi < lo {
		hi = lo
	}
	if e.hlRangeSet && lo >= e.hlLo && hi <= e.hlHi {
		return
	}
	e.hlLo, e.hlHi = lo, hi
	e.hlRangeSet = true
	rh.SetRange(lo, hi)
	// Results arrive on the worker goroutine; open the poll window so the next
	// wakes pick them up.
	e.markParsePending()
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
	e.lspOpened = true // never open after Close
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
	return e.lineRowCount()
}

// Bytes returns the current document content.
func (e *Editor) Bytes() []byte { return e.pt.Bytes() }

// Modified reports whether there are unsaved changes.
func (e *Editor) Modified() bool { return e.modified }

// Revision counts edits to the text, undo and redo included. Apps compare it
// across frames to notice changes, for an idle timer or a stale cache.
func (e *Editor) Revision() uint64 { return e.revision }

// MarkSaved clears the modified flag and records the current undo step as
// the saved one, so undoing or redoing back to it clears the flag again.
func (e *Editor) MarkSaved() {
	e.savedID = e.topID()
	e.modified = false
}

// topID is the id of the newest undo step, 0 when there is none.
func (e *Editor) topID() uint64 {
	if len(e.undo) == 0 {
		return 0
	}
	return e.undo[len(e.undo)-1].id
}

// stepModified recomputes the modified flag after an edit, undo or redo.
func (e *Editor) stepModified() { e.modified = e.topID() != e.savedID }

// SetText replaces the whole document with s as a single undoable edit, the
// way selecting everything and pasting s would, so the editor reports itself
// modified and Undo brings the old text back. The caret keeps its offset,
// clamped to the new text. It does nothing when s is already the content or
// the editor is read-only.
func (e *Editor) SetText(s string) {
	if e.readOnly || s == string(e.pt.Bytes()) {
		return
	}
	e.extra = nil
	caret := e.caret
	e.applyEdit(0, e.pt.Len(), s, mergeNone)
	e.lastMerge = mergeNone
	if caret > len(s) {
		caret = len(s)
	}
	e.caret = caret
	e.ensureCaretVisible()
}

// ApplyText changes the document to s as one undoable edit that replaces
// only the span that differs, so the caret, selection and scroll position
// outside that span stay where they were. Formatters use it to rewrite a
// buffer without making the view jump. It does nothing when s is already the
// content or the editor is read-only.
func (e *Editor) ApplyText(s string) {
	old := e.pt.Bytes()
	if e.readOnly || s == string(old) {
		return
	}
	e.extra = nil
	pre := 0
	for pre < len(old) && pre < len(s) && old[pre] == s[pre] {
		pre++
	}
	suf := 0
	for suf < len(old)-pre && suf < len(s)-pre && old[len(old)-1-suf] == s[len(s)-1-suf] {
		suf++
	}
	for pre > 0 && pre < len(old) && !utf8.RuneStart(old[pre]) {
		pre--
	}
	for suf > 0 && !utf8.RuneStart(old[len(old)-suf]) {
		suf--
	}
	lo, oldHi := pre, len(old)-suf
	ins := s[pre : len(s)-suf]
	delta := len(ins) - (oldHi - lo)
	shift := func(off int) int {
		switch {
		case off <= lo:
			return off
		case off >= oldHi:
			return off + delta
		}
		return lo + len(ins) // inside the change: end of the new text
	}
	caret, anchor := shift(e.caret), e.selAnchor
	if anchor >= 0 {
		anchor = shift(anchor)
	}
	scroll := e.ScrollPx
	e.applyEdit(lo, oldHi-lo, ins, mergeNone)
	e.lastMerge = mergeNone
	e.caret, e.selAnchor = caret, anchor
	e.ScrollPx = scroll
}

// HighlightOversize reports whether the document is larger than its
// highlighter's size limit and is shown without syntax colors. Only
// highlighters that implement highlight.SizeLimited ever report it.
func (e *Editor) HighlightOversize() bool {
	sl, ok := e.hl.(highlight.SizeLimited)
	return ok && sl.Oversize()
}

// HighlightAnyway lifts the highlighter's size limit for this editor and
// highlights the document, however large. Parsing a big document costs memory
// and time on the highlighter's worker (see highlight.DefaultMaxBytes), so it
// is meant for a reader who asked for it.
func (e *Editor) HighlightAnyway() {
	sl, ok := e.hl.(highlight.SizeLimited)
	if !ok {
		return
	}
	sl.SetMaxBytes(math.MaxInt)
	sl.Update(e.pt.Bytes())
	e.markParsePending()
}

// Update polls the highlighter, recomputes scroll extents, and drives scrollbars.
func (e *Editor) Update(m *input.Mouse) {
	if gen := frameText().FontGen(); gen != e.fontGen {
		e.fontGen = gen
		e.lineH = frameText().MetricsMono().LineHeight
		if e.tabSet == 0 {
			e.tabW = frameText().TabWidth()
		}
		w, _ := frameText().MeasureMono("n")
		e.cellW = w
		e.wrapFull = true
		e.contentSizeDirty = true
	}
	e.followDefaults()
	if e.SoftWrap {
		// Wrapping depends on the viewport width; relayout when it changes.
		if cw, _, _, _ := e.scrollMetrics(); cw != e.lastWrapW {
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
	e.lspOpen()
	e.lspUpdate(m)
	if e.contentSizeDirty {
		e.recomputeContentSize()
	}
	e.restoreScrollAnchor()
	if e.revealCaret {
		e.revealCaret = false
		e.ensureCaretVisible()
	}
	if m != nil {
		_, _, vShow, hShow := e.scrollMetrics()
		e.syncScrollbarLayout(vShow, hShow)
		vp := e.contentViewport()
		e.vbar.Update(m, vp)
		e.hbar.Update(m, vp)
	}
	e.clampScroll()
	e.findUpdate()
	// Scroll and the wrap tables are settled, so the visible range is now known.
	e.syncHighlightRange()
}

const editorLargeDocLines = 500

// maxShapedLine bounds how many bytes of a line the editor ever shapes in one
// piece. Files like minified JSON/XML can hold megabytes on a single line;
// shaping such a line takes minutes and gigabytes, so lines longer than this
// are shaped one visible window at a time and use mono-cell arithmetic for
// caret/hit-test geometry instead of the shaped line.
const maxShapedLine = 8192

// followDefaults applies the live EditorDefaults (line numbers, soft wrap,
// scroll past end) that this editor does not override.
func (e *Editor) followDefaults() {
	if w := e.wantGutterW(); w != e.gutterW {
		e.gutterW = w
		e.contentSizeDirty = true
	}
	if h := e.wantPastEndH(); h != e.pastEndH {
		e.contentSizeDirty = true
	}
	if !e.wrapSet {
		e.setSoftWrap(EditorDefaults.SoftWrap)
	}
}

// wantPastEndH is the blank space below the last line: with
// EditorDefaults.ScrollPastEnd, enough for the last line to reach the top of
// the view. Editors built WithoutGutter (inputs, logs, previews) never get it.
func (e *Editor) wantPastEndH() float32 {
	if !EditorDefaults.ScrollPastEnd || e.noGutter {
		return 0
	}
	return f32max(0, e.viewport.Frame.H-e.lineH)
}

// wantGutterW is the gutter width the editor should currently have.
func (e *Editor) wantGutterW() float32 {
	if e.noGutter || !EditorDefaults.LineNumbers {
		return 0
	}
	if g := e.gutter; g != nil {
		digits := g.MinDigits
		if digits <= 0 {
			digits = 2
		}
		if e.pt != nil {
			digits = max(digits, len(strconv.Itoa(e.pt.LineCount())))
		}
		return 2*e.gutterPad() + float32(digits)*e.cellW + e.foldGutterW()
	}
	return editorGutterW + e.foldGutterW()
}

// foldGutterW is the room the gutter keeps for fold chevrons.
func (e *Editor) foldGutterW() float32 {
	if e.fold.off {
		return 0
	}
	return foldGutterW
}

// GutterSpec styles an editor's line-number gutter (see SetGutter).
type GutterSpec struct {
	// MinDigits is how many digits the gutter always has room for; it
	// grows past that as the document does. 0 means 2.
	MinDigits int
	// Padding is the space on each side of the numbers; 0 means Spacing.S.
	Padding float32
	// Fill is the gutter background. TokenUnset leaves it transparent, the
	// same as the text area.
	Fill Token
	// Separator draws a 1 px border line between the gutter and the text.
	Separator bool
}

// WithGutter styles the line-number gutter; see SetGutter.
func WithGutter(g GutterSpec) EditorOption { return func(e *Editor) { e.gutter = &g } }

// SetGutter sizes the line-number gutter to the document's digit count
// instead of a fixed width, and sets its padding, fill and separator.
// Editors without it keep the default 52 px filled gutter with a border.
func (e *Editor) SetGutter(g GutterSpec) {
	e.gutter = &g
	if w := e.wantGutterW(); w != e.gutterW {
		e.gutterW = w
		e.contentSizeDirty = true
	}
}

func (e *Editor) gutterPad() float32 {
	if e.gutter != nil && e.gutter.Padding > 0 {
		return e.gutter.Padding
	}
	return theme.Current().Spacing.S
}

// SetSoftWrap toggles soft wrapping of long lines at the viewport width. The
// editor then keeps v and no longer follows EditorDefaults.SoftWrap.
func (e *Editor) SetSoftWrap(v bool) {
	e.wrapSet = true
	e.setSoftWrap(v)
}

func (e *Editor) setSoftWrap(v bool) {
	if e.SoftWrap == v {
		return
	}
	e.keepScrollAnchor()
	e.SoftWrap = v
	if !v {
		e.lineRows = nil
		e.rowPrefix = nil
		e.ScrollX = 0
	}
	e.wrapFull = true
	e.contentSizeDirty = true
}

// keepScrollAnchor remembers the caret's row, or the top row when the caret
// is off screen, and its y in the view, so the next layout puts it back in
// the same place after the rows reflow.
func (e *Editor) keepScrollAnchor() {
	if e.anchorSet || e.lineH <= 0 {
		return
	}
	_, clientH, _, _ := e.scrollMetrics()
	if clientH <= 0 {
		return
	}
	off := e.caret
	y := float32(e.rowOfByte(off))*e.lineH - e.ScrollPx
	if y < 0 || y+e.lineH > clientH {
		r := int(e.ScrollPx / e.lineH)
		off = e.rowStartByte(r)
		y = float32(r)*e.lineH - e.ScrollPx
	}
	e.anchorOff, e.anchorY, e.anchorSet = off, y, true
}

// restoreScrollAnchor scrolls so the anchored row is back at its old y.
func (e *Editor) restoreScrollAnchor() {
	if !e.anchorSet || e.wrapStale() {
		return
	}
	e.anchorSet = false
	e.ScrollPx = float32(e.rowOfByte(e.anchorOff))*e.lineH - e.anchorY
}

// rowStartByte is the offset where visual row r starts.
func (e *Editor) rowStartByte(r int) int {
	if e.SoftWrap && len(e.rowPrefix) > 1 {
		r = max(0, min(r, e.rowPrefix[len(e.rowPrefix)-1]-1))
		_, absStart, _ := e.wrapRowAbs(r)
		return absStart
	}
	return e.pt.LineStart(e.rowLine(max(0, min(r, e.lineRowCount()-1))))
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

// SetWrapIndent sets where continuation rows of a soft-wrapped line start:
// at column 0, under the line's leading whitespace, or one tab stop past it.
// Indents are measured in rendered columns, and capped at half the row so
// deeply nested lines still have room for text.
func (e *Editor) SetWrapIndent(m WrapIndent) {
	if e.WrapIndent == m {
		return
	}
	e.keepScrollAnchor()
	e.WrapIndent = m
	e.wrapFull = true
	e.contentSizeDirty = true
}

// wrapIndentCells is how far continuation rows of a line starting with view
// are indented, for rows of cols cells.
func (e *Editor) wrapIndentCells(view []byte, cols int) int {
	if e.WrapIndent == WrapIndentNone {
		return 0
	}
	n := indentDepth(view, max(e.tabW, 1))
	if e.WrapIndent == WrapHanging {
		n += max(e.tabW, 1)
	}
	return min(n, cols/2)
}

// rowIndentPx is the x shift of global visual row r in wrap mode.
func (e *Editor) rowIndentPx(r int) float32 {
	if !e.SoftWrap || len(e.rowPrefix) < 2 {
		return 0
	}
	ln := e.rowOfVisual(r)
	ri := r - e.rowPrefix[ln]
	if ln >= len(e.lineRows) || ri < 0 || ri >= len(e.lineRows[ln]) {
		return 0
	}
	return float32(e.lineRows[ln][ri].indent) * e.cellW
}

func (e *Editor) recomputeContentSize() {
	e.pastEndH = e.wantPastEndH()
	if e.SoftWrap {
		e.syncWrap()
		e.ContentHeight = float32(e.rowPrefix[len(e.rowPrefix)-1])*e.lineH + e.pastEndH
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
	e.ContentHeight = float32(e.lineRowCount())*e.lineH + e.pastEndH
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
	if len(e.fold.hidden) == 0 {
		for i := 0; i < lc; i++ {
			e.rowPrefix[i+1] = e.rowPrefix[i] + len(e.lineRows[i])
		}
		return
	}
	for i := 0; i < lc; i++ {
		e.rowPrefix[i+1] = e.rowPrefix[i] + e.shownRows(i)
	}
}

// wrapLineRows splits view into visual rows of at most cols mono cells each
// (tab stops included; no font shaping). With WrapWords on, rows break at the
// last word boundary that fits, falling back to a hard cell break for runs
// longer than a row.
func (e *Editor) wrapLineRows(view []byte) []wrapRow {
	full := e.wrapCols
	ind := e.wrapIndentCells(view, full)
	rows := make([]wrapRow, 0, len(view)/full+2)
	rowStart := 0
	for rowStart < len(view) {
		cols, indent := full, 0
		if len(rows) > 0 { // continuation row: narrower by the indent
			cols, indent = full-ind, ind
		}
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
			rows = append(rows, wrapRow{int32(rowStart), int32(len(view)), int32(indent)})
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
		rows = append(rows, wrapRow{int32(rowStart), int32(i), int32(indent)})
		rowStart = i
	}
	if len(rows) == 0 { // empty line: one empty row
		rows = append(rows, wrapRow{0, int32(len(view)), 0})
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
	e.freshWrap()
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
	return e.lineMono(e.pt.Line(ln))
}

// lineMono shapes text with this editor's tab stops.
func (e *Editor) lineMono(text string) shape.Line {
	return frameText().LineMonoTabs(text, e.tabSet)
}

// SetTabWidth sets the columns per tab stop for this editor, overriding the
// font config's width; cols <= 0 goes back to following it.
func (e *Editor) SetTabWidth(cols int) {
	cols = max(cols, 0)
	if cols == e.tabSet {
		return
	}
	e.tabSet = cols
	if cols > 0 {
		e.tabW = cols
	} else if t := frameText(); t != nil {
		e.tabW = t.TabWidth()
	}
	e.wrapFull = true
	e.contentSizeDirty = true
}

// TabWidth is the columns per tab stop this editor lays tabs out with.
func (e *Editor) TabWidth() int { return max(e.tabW, 1) }

// SetIndentSpaces makes Tab and auto-indent insert spaces (true) or tabs
// (false) in this editor, whatever EditorDefaults.IndentSpaces says.
func (e *Editor) SetIndentSpaces(v bool) { e.indentSet, e.indentSpaces = true, v }

// IndentSpaces reports whether this editor indents with spaces.
func (e *Editor) IndentSpaces() bool {
	if e.indentSet {
		return e.indentSpaces
	}
	return EditorDefaults.IndentSpaces
}

// SetAutoIndent sets what Enter carries onto the new line in this editor;
// AutoIndentDefault follows EditorDefaults.AutoIndent.
func (e *Editor) SetAutoIndent(m AutoIndent) { e.autoIndent = m }

func (e *Editor) autoIndentMode() AutoIndent {
	if e.autoIndent != AutoIndentDefault {
		return e.autoIndent
	}
	if EditorDefaults.AutoIndent != AutoIndentDefault {
		return EditorDefaults.AutoIndent
	}
	return AutoIndentNone
}

// newline is Enter: a line break carrying the indent AutoIndent asks for.
func (e *Editor) newline() {
	mode := e.autoIndentMode()
	if mode == AutoIndentNone {
		e.replaceSelection("\n", mergeNone)
		return
	}
	lo, hi := e.selRange()
	doc := e.pt.Bytes()
	ls := e.pt.LineStart(e.lineOf(lo))
	head := doc[ls:lo]
	lead := head[:len(head)-len(bytes.TrimLeft(head, " \t"))]
	ins := "\n" + string(lead)
	caret := len(ins)
	if mode == AutoIndentBrackets {
		if before := bytes.TrimRight(head, " \t"); len(before) > 0 {
			last := rune(before[len(before)-1])
			if _, open := bracketClose[last]; open || last == ':' {
				if e.IndentSpaces() {
					ins += strings.Repeat(" ", e.TabWidth())
				} else {
					ins += "\t"
				}
				caret = len(ins)
				// The closer moves down to its own line at the old indent,
				// taking the blanks before it along.
				ln := e.lineOf(hi)
				rest := doc[hi : e.pt.LineStart(ln)+len(e.pt.Line(ln))]
				trimmed := bytes.TrimLeft(rest, " \t")
				if open && len(trimmed) > 0 && rune(trimmed[0]) == bracketClose[last] {
					hi += len(rest) - len(trimmed)
					ins += "\n" + string(lead)
				}
			}
		}
	}
	e.applyEdit(lo, hi-lo, ins, mergeNone)
	e.caret, e.selAnchor = lo+caret, -1
	e.ensureCaretVisible()
}

// isHuge reports whether line ln exceeds the windowed-shaping threshold.
func (e *Editor) isHuge(ln int) bool {
	return e.pt.LineLen(ln) > maxShapedLine
}

// rowOfByte returns the visual row index containing byte offset off. With soft
// wrap off there is one row per line, so this is the logical line index.
func (e *Editor) rowOfByte(off int) int {
	e.freshWrap()
	if !e.SoftWrap || len(e.rowPrefix) < 2 {
		return e.lineRow(e.lineOf(off))
	}
	if off < 0 {
		off = 0
	}
	if n := e.pt.Len(); off > n {
		off = n
	}
	ln := e.lineOf(off)
	if e.lineHidden(ln) {
		return max(e.rowPrefix[ln]-1, 0) // the fold's own row
	}
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
	r := e.rowOfByte(off)
	ln, absStart, _ := e.wrapRowAbs(r)
	ls := e.pt.LineStart(ln)
	view := e.lineView(ln)
	rel := off - ls
	start := absStart - ls
	if rel < start {
		rel = start
	}
	return e.rowIndentPx(r) + e.monoPrefixWidth(view[start:], rel-start)
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
	vBar, hBar := e.vbar.gutter(editorBarSize), e.hbar.gutter(editorBarSize)
	if vShow {
		clientW = f.W - vBar
	}
	hShow = e.ContentWidth > clientW
	if hShow {
		clientH = f.H - hBar
	}
	if e.ContentHeight > clientH {
		vShow = true
		clientW = f.W - vBar
		hShow = e.ContentWidth > clientW
		if hShow {
			clientH = f.H - hBar
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
			bottom = e.hbar.strip(editorBarSize)
		}
		e.vbar.host.Style = layout.Box().W(e.vbar.strip(editorBarSize)).AbsTop(0).AbsRight(0).AbsBottom(bottom)
		e.vbar.host.ReapplyStyle()
	}
	if hShow {
		right := float32(0)
		if vShow {
			right = e.vbar.strip(editorBarSize)
		}
		e.hbar.host.Style = layout.Box().H(e.hbar.strip(editorBarSize)).AbsLeft(0).AbsRight(right).AbsBottom(0)
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
	if vShow && e.vbar.hit(m.X, m.Y) {
		return true
	}
	if hShow && e.hbar.hit(m.X, m.Y) {
		return true
	}
	return false
}

// SetScrollbars overrides the theme's ScrollbarMode for this editor.
func (e *Editor) SetScrollbars(m theme.ScrollbarMode) {
	e.vbar.SetMode(m)
	e.hbar.SetMode(m)
}

// Focus grants keyboard focus to the editor.
func (e *Editor) Focus() {
	e.focused = true
	e.blinkStart = time.Now()
}

// Blur removes keyboard focus from the editor.
func (e *Editor) Blur() {
	e.focused = false
	e.menu.close()
	e.findBlur()
}

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
	if e.revealCaret && !e.revealWoke {
		// Laid out this frame; reveal on the next. Once only, so an editor
		// kept at zero size does not redraw forever.
		e.revealWoke = true
		c.Invalidate()
	}
	e.vbar.tick(c, e.viewport.Frame)
	e.hbar.tick(c, e.viewport.Frame)
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
	e.menu.layout(c)
	e.findLayout(c)
	return e.host
}

// Contains reports whether (x, y) falls inside the editor's last laid-out frame.
func (e *Editor) Contains(x, y float32) bool {
	return e.host != nil && e.host.Frame.Contains(x, y)
}

// AnimationWait implements the runtime's optional animation hook.
func (e *Editor) AnimationWait() (time.Duration, bool) {
	if e.fold.anim.on {
		return 16 * time.Millisecond, true
	}
	hoverWake, hoverPending := e.lspHoverWake()
	if !e.focused && !time.Now().Before(e.parseUntil) && !hoverPending {
		return 0, false
	}
	const half = 500 * time.Millisecond
	since := time.Since(e.blinkStart) % (2 * half)
	wait := half - (since % half)
	if e.dragging || e.box.on {
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
	if fw, ok := e.findWake(); ok && fw < wait {
		wait = fw
	}
	if wait < 16*time.Millisecond && !e.dragging {
		wait = 16 * time.Millisecond
	}
	return wait, true
}

// ---------------------------------------------------------------------------
// Editing core
// ---------------------------------------------------------------------------

func (e *Editor) applyEdit(pos, delLen int, ins string, merge editMerge) int {
	if e.readOnly {
		return e.caret
	}
	e.goalSet = false
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
		group:       e.undoGroup,
	}
	if op.group != 0 {
		merge = mergeNone // one action over several selections is one step
	}

	if delLen > 0 {
		e.pt.Delete(pos, delLen)
	}
	if len(ins) > 0 {
		e.pt.Insert(pos, []byte(ins))
	}

	if strings.Contains(ins, "\n") || strings.Contains(deleted, "\n") {
		merge = mergeNone
	}
	now := time.Now()
	e.opSeq++
	if e.mergeInto(&op, merge, now) {
		// The merged step now holds different text from what was saved.
		e.undo[len(e.undo)-1].id = e.opSeq
	} else {
		op.id = e.opSeq
		e.undo = append(e.undo, op)
		e.undoBytes += op.size()
	}
	e.lastMerge = merge
	e.lastEditAt = now
	e.trimUndo()

	clear(e.redo) // release the discarded redo payloads
	e.redo = e.redo[:0]
	e.stepModified()
	e.revision++
	e.contentSizeDirty = true
	e.selAnchor = -1
	e.caret = op.caretAfter
	e.afterMutation(highlight.Edit{
		StartByte: pos, OldEndByte: pos + delLen, NewEndByte: pos + len(ins),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(pos + len(ins)),
	})
	return e.caret
}

// mergeInto folds op into the newest undo step when both are part of one run
// of typing, backspacing or forward-deleting, and reports whether it did.
func (e *Editor) mergeInto(op *editOp, merge editMerge, now time.Time) bool {
	if merge == mergeNone || merge != e.lastMerge || len(e.undo) == 0 ||
		now.Sub(e.lastEditAt) > undoMergeGap {
		return false
	}
	last := &e.undo[len(e.undo)-1]
	switch merge {
	case mergeType:
		// Typing over a selection starts a run (last.deleted is the
		// selection); later keystrokes only append.
		if op.deleted != "" || last.pos+len(last.inserted) != op.pos || startsWord(last.inserted, op.inserted) {
			return false
		}
		last.inserted += op.inserted
		e.undoBytes += len(op.inserted)
	case mergeBackspace:
		if last.inserted != "" || op.pos+len(op.deleted) != last.pos {
			return false
		}
		last.deleted = op.deleted + last.deleted
		last.pos = op.pos
		e.undoBytes += len(op.deleted)
	case mergeDelete:
		if last.inserted != "" || op.pos != last.pos {
			return false
		}
		last.deleted += op.deleted
		e.undoBytes += len(op.deleted)
	}
	last.caretAfter = op.caretAfter
	return true
}

// trimUndo drops the oldest undo steps until the history fits its limits,
// always keeping the newest one.
func (e *Editor) trimUndo() {
	drop := 0
	bytes := e.undoBytes
	for len(e.undo)-drop > 1 &&
		((e.undoMaxSteps > 0 && len(e.undo)-drop > e.undoMaxSteps) ||
			(e.undoMaxBytes > 0 && bytes > e.undoMaxBytes)) {
		bytes -= e.undo[drop].size()
		drop++
	}
	if drop == 0 {
		return
	}
	n := copy(e.undo, e.undo[drop:])
	clear(e.undo[n:]) // release the dropped payloads
	e.undo = e.undo[:n]
	e.undoBytes = bytes
}

func (e *Editor) afterMutation(edit highlight.Edit) {
	e.foldsAfterEdit(edit) // before anything maps lines to rows
	e.hl.UpdateEdit(e.pt.Bytes(), edit)
	e.contentSizeDirty = true // Undo/Redo don't set this themselves
	e.markParsePending()
	e.blinkStart = time.Now()
	// Mark the wrap tables stale before anything maps the caret to a row.
	e.markWrapDirty(edit)
	e.ensureCaretVisible()
	if e.search.open {
		e.findAfterEdit(edit)
	}
	e.lspDidChange()
}

// freshWrap re-wraps lines an edit left stale. Edits run between layouts, so
// a row lookup right after one (keeping the caret visible, Up/Down) would
// otherwise read tables sized for the old text.
func (e *Editor) freshWrap() {
	if e.wrapStale() {
		e.syncWrap()
	}
}

// wrapStale reports whether the wrap tables exist but lag the text.
func (e *Editor) wrapStale() bool {
	if !e.SoftWrap || len(e.rowPrefix) < 2 {
		return false
	}
	return e.wrapFull || e.wrapDirtySet || len(e.lineRows) != e.pt.LineCount()
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

func (e *Editor) replaceSelection(s string, merge editMerge) {
	lo, hi := e.selRange()
	e.applyEdit(lo, hi-lo, s, merge)
}

func (e *Editor) deleteSelection() bool {
	if !e.hasSelection() {
		return false
	}
	lo, hi := e.selRange()
	e.applyEdit(lo, hi-lo, "", mergeNone)
	return true
}

// Undo reverts the most recent edit, or the edits one action made at every
// selection.
func (e *Editor) Undo() {
	if f := e.findInput(); f != nil {
		f.Undo()
		return
	}
	if e.readOnly || len(e.undo) == 0 {
		return
	}
	e.extra = nil
	g := e.undo[len(e.undo)-1].group
	op := e.undoOne()
	for g != 0 && len(e.undo) > 0 && e.undo[len(e.undo)-1].group == g {
		op = e.undoOne()
	}
	if op.selBefore != nil {
		e.setCursors(slices.Clone(op.selBefore))
	}
}

func (e *Editor) undoOne() editOp {
	op := e.undo[len(e.undo)-1]
	e.undo[len(e.undo)-1] = editOp{}
	e.undo = e.undo[:len(e.undo)-1]
	e.undoBytes -= op.size()
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
	e.lastMerge = mergeNone
	e.stepModified()
	e.revision++
	e.afterMutation(highlight.Edit{
		StartByte: op.pos, OldEndByte: op.pos + len(op.inserted), NewEndByte: op.pos + len(op.deleted),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(op.pos + len(op.deleted)),
	})
	return op
}

// Redo re-applies the most recently undone edit, or group of edits.
func (e *Editor) Redo() {
	if f := e.findInput(); f != nil {
		f.Redo()
		return
	}
	if e.readOnly || len(e.redo) == 0 {
		return
	}
	e.extra = nil
	g := e.redo[len(e.redo)-1].group
	op := e.redoOne()
	for g != 0 && len(e.redo) > 0 && e.redo[len(e.redo)-1].group == g {
		op = e.redoOne()
	}
	if op.selAfter != nil {
		e.setCursors(slices.Clone(op.selAfter))
	}
}

func (e *Editor) redoOne() editOp {
	op := e.redo[len(e.redo)-1]
	e.redo[len(e.redo)-1] = editOp{}
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
	e.undoBytes += op.size()
	e.trimUndo()
	e.caret = op.caretAfter
	e.selAnchor = -1
	e.lastMerge = mergeNone
	e.stepModified()
	e.revision++
	e.afterMutation(highlight.Edit{
		StartByte: op.pos, OldEndByte: op.pos + len(op.deleted), NewEndByte: op.pos + len(op.inserted),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(op.pos + len(op.inserted)),
	})
	return op
}

// ---------------------------------------------------------------------------
// Input handling
// ---------------------------------------------------------------------------

// HandleText inserts text-producing characters.
func (e *Editor) HandleText(runes []rune) {
	if len(runes) == 0 {
		return
	}
	if e.TextFilter != nil && !e.inMulti && !e.findFocused() && e.TextFilter(runes) {
		return
	}
	if e.findFocused() {
		e.findHandleText(runes)
		return
	}
	if e.readOnly {
		return
	}
	if len(e.extra) > 0 && !e.inMulti {
		e.multi(true, func() { e.HandleText(runes) })
		return
	}
	if e.inlineLive() {
		text, at := e.inline.text, e.caret
		defer e.inlineAfterType(text, at, string(runes))
	}
	if len(runes) == 1 {
		r := runes[0]
		if isQuote(r) && EditorDefaults.AutoCloseQuotes && e.typeQuote(r) {
			return
		}
		// Auto-close opening brackets.
		if closing, ok := bracketClose[r]; ok && EditorDefaults.AutoCloseBrackets {
			if e.hasSelection() {
				lo, hi := e.selRange()
				selected := string(e.pt.Bytes()[lo:hi])
				e.applyEdit(lo, hi-lo, string(r)+selected+string(closing), mergeNone)
			} else {
				pos := e.caret
				e.applyEdit(pos, 0, string(r)+string(closing), mergeNone)
				e.caret = pos + utf8.RuneLen(r)
				e.ensureCaretVisible()
			}
			return
		}
		// Skip over auto-inserted closing bracket.
		if _, isClose := bracketOpen[r]; isClose && EditorDefaults.AutoCloseBrackets {
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
	e.replaceSelection(string(runes), mergeType)
	e.lspAfterType(runes)
}

// HandleKeys processes navigation, editing, and shortcut keys for one frame.
// Shortcuts come from the keymap (see editor_keymap.go); the keys it leaves
// unbound, such as Enter, Backspace and the arrows, are handled here.
func (e *Editor) HandleKeys(keys []input.KeyEvent) {
	for _, ev := range keys {
		shift := ev.Mods.Has(input.ModShift)
		if e.KeyFilter != nil && !e.inMulti && !e.findFocused() && !e.lspUI.compOpen && e.KeyFilter(ev) {
			continue
		}

		// The find bar takes every key while it is focused.
		if e.findFocused() {
			e.findHandleKey(ev)
			continue
		}

		// The completion popup, when open, captures navigation/accept keys.
		if e.lspUI.compOpen && e.handleCompletionKey(ev) {
			continue
		}

		// Tab takes the inline suggestion and Escape drops it.
		if e.inlineKey(ev) {
			continue
		}

		if !e.inMulti && e.runKey(ev, ScopeText) {
			continue
		}
		if len(e.extra) > 0 && !e.inMulti && e.multiKey(ev) {
			continue
		}
		if ev.Mods.Primary() {
			continue // an unbound shortcut is not text navigation
		}
		switch ev.Key {
		case input.KeyEscape:
			// Extra cursors went first (multiKey); then the find bar, then
			// the selection.
			if e.search.open {
				e.closeSearch()
			} else {
				e.selAnchor = -1
			}
		case input.KeyEnter:
			e.newline()
		case input.KeyTab:
			e.replaceSelection(e.tabText(), mergeNone)
		case input.KeyBackspace:
			e.backspace()
			// Deleting back into a placeholder should offer its names again.
			e.suggestAfterEdit()
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

// isQuote reports whether r is a quote the editor can auto-close.
func isQuote(r rune) bool { return r == '"' || r == '\'' || r == '`' }

// typeQuote handles typing quote q with auto-close on and reports whether it
// did: it wraps a selection in quotes, steps over a closing quote at the
// caret, or inserts a pair where a string can start. Other cases (such as the
// apostrophe in "don't") fall back to plain typing.
func (e *Editor) typeQuote(q rune) bool {
	qs := string(q)
	if e.hasSelection() {
		lo, hi := e.selRange()
		selected := string(e.pt.Bytes()[lo:hi])
		e.applyEdit(lo, hi-lo, qs+selected+qs, mergeNone)
		return true
	}
	b := e.pt.Bytes()
	pos := e.caret
	var next rune
	if pos < len(b) {
		next, _ = utf8.DecodeRune(b[pos:])
	}
	if next == q {
		e.moveTo(pos+len(qs), false)
		return true
	}
	if pos > 0 {
		if prev, _ := utf8.DecodeLastRune(b[:pos]); prev == '_' || unicode.IsLetter(prev) || unicode.IsDigit(prev) || isQuote(prev) {
			return false
		}
	}
	if next != 0 && !unicode.IsSpace(next) && bracketOpen[next] == 0 && next != ',' && next != ';' && next != ':' {
		return false
	}
	e.applyEdit(pos, 0, qs+qs, mergeNone)
	e.caret = pos + len(qs)
	e.ensureCaretVisible()
	return true
}

// tabText is what the Tab key inserts: a tab, or with
// EditorDefaults.IndentSpaces the spaces that reach the next tab stop.
func (e *Editor) tabText() string {
	if !e.IndentSpaces() {
		return "\t"
	}
	lo, _ := e.selRange()
	ls := e.pt.LineStart(e.lineOf(lo))
	tabW := max(e.tabW, 1)
	col := 0
	for _, ch := range string(e.pt.Bytes()[ls:lo]) {
		if ch == '\t' {
			col = (col/tabW + 1) * tabW
		} else {
			col++
		}
	}
	return strings.Repeat(" ", tabW-col%tabW)
}

func (e *Editor) backspace() {
	if e.deleteSelection() {
		return
	}
	if e.caret == 0 {
		return
	}
	prev := e.prevRune(e.caret)
	e.applyEdit(prev, e.caret-prev, "", mergeBackspace)
}

func (e *Editor) deleteForward() {
	if e.deleteSelection() {
		return
	}
	next := e.nextRune(e.caret)
	if next == e.caret {
		return
	}
	e.applyEdit(e.caret, next-e.caret, "", mergeDelete)
}

// openContextMenuAt handles a right-click in the text area: it keeps a
// selection the click lands in, otherwise moves the caret there, and shows the
// edit menu.
func (e *Editor) openContextMenuAt(m *input.Mouse) {
	if e.search.open && e.findGeom().bar.Contains(m.X, m.Y) {
		return
	}
	off := e.offsetAtPoint(m.X, m.Y)
	if lo, hi := e.selRange(); !e.hasSelection() || off < lo || off > hi {
		e.moveTo(off, false)
	}
	e.closeCompletion()
	e.clearHover()
	items := editMenuItems(e, editMenuFlags{editable: !e.readOnly, copyable: true})
	if e.ContextMenu != nil {
		items = e.ContextMenu(items)
	}
	if e.menu.open(items, m.X, m.Y) {
		m.Consumed = true
	}
}

// SelectAll selects the whole document.
func (e *Editor) SelectAll() {
	if f := e.findInput(); f != nil {
		f.SelectAll()
		return
	}
	e.extra = nil
	e.selAnchor = 0
	e.caret = e.pt.Len()
}

// HasSelection reports whether a non-empty range is selected.
func (e *Editor) HasSelection() bool { return e.hasSelection() }

// Selection returns the selected byte range [start, end) with start <= end,
// whichever way it was dragged. With nothing selected both are the caret.
func (e *Editor) Selection() (start, end int) { return e.selRange() }

// SelectedText returns the selected text, or "" with nothing selected.
func (e *Editor) SelectedText() string {
	lo, hi := e.selRange()
	return string(e.pt.Bytes()[lo:hi])
}

// Caret returns the caret's byte offset.
func (e *Editor) Caret() int { return e.caret }

// SetSelection selects from anchor to caret, byte offsets in either order, and
// scrolls the caret into view. Equal offsets place the caret with nothing
// selected. Offsets are clamped to the text and moved back to a rune start.
func (e *Editor) SetSelection(anchor, caret int) {
	e.extra = nil
	e.goalSet = false
	anchor, caret = e.clampOffset(anchor), e.clampOffset(caret)
	e.caret = caret
	e.selAnchor = -1
	if anchor != caret {
		e.selAnchor = anchor
	}
	e.ensureCaretVisible()
}

// LineCol converts a byte offset to a 0-based line and a 0-based column
// counted in runes.
func (e *Editor) LineCol(off int) (line, col int) { return e.lineColOf(e.clampOffset(off)) }

// Offset converts a 0-based line and rune column to a byte offset. Lines past
// the end clamp to the last line, columns past the end to the line's end.
func (e *Editor) Offset(line, col int) int { return e.offsetOf(line, maxInt(col, 0)) }

func (e *Editor) clampOffset(off int) int {
	b := e.pt.Bytes()
	off = maxInt(0, minInt(off, len(b)))
	for off > 0 && off < len(b) && !utf8.RuneStart(b[off]) {
		off--
	}
	return off
}

// CanUndo reports whether Undo has an edit to revert.
func (e *Editor) CanUndo() bool { return !e.readOnly && len(e.undo) > 0 }

// CanRedo reports whether Redo has an undone edit to re-apply.
func (e *Editor) CanRedo() bool { return !e.readOnly && len(e.redo) > 0 }

// Copy puts the selection on the clipboard and reports whether it did. With
// nothing selected it leaves the clipboard alone.
func (e *Editor) Copy() bool {
	if f := e.findInput(); f != nil {
		return f.Copy()
	}
	if len(e.extra) > 0 {
		e.copyAll(frameClipboard())
		return true
	}
	return e.copy(frameClipboard())
}

// Cut moves the selection to the clipboard. A read-only editor copies it and
// keeps the text, as Ctrl/Cmd+X does.
func (e *Editor) Cut() {
	if f := e.findInput(); f != nil {
		f.Cut()
		return
	}
	e.cutAll(frameClipboard())
}

// Paste replaces the selection with the clipboard text.
func (e *Editor) Paste() { e.pasteAll(frameClipboard()) }

func (e *Editor) copy(clip input.Clipboard) bool {
	if clip == nil || !e.hasSelection() {
		return false
	}
	lo, hi := e.selRange()
	clip.Set(string(e.pt.Bytes()[lo:hi]))
	return true
}

func (e *Editor) cut(clip input.Clipboard) {
	if clip == nil || !e.hasSelection() {
		return
	}
	lo, hi := e.selRange()
	clip.Set(string(e.pt.Bytes()[lo:hi]))
	e.applyEdit(lo, hi-lo, "", mergeNone)
}

func (e *Editor) paste(clip input.Clipboard) {
	if clip == nil {
		return
	}
	s := clip.Get()
	if s == "" {
		return
	}
	if f := e.findInput(); f != nil {
		f.Paste() // single-line: it keeps the first line
		return
	}
	e.replaceSelection(s, mergeNone)
}

func (e *Editor) moveTo(off int, extend bool) {
	if off < 0 {
		off = 0
	}
	if n := e.pt.Len(); off > n {
		off = n
	}
	off = e.skipFolded(off, off >= e.caret)
	if extend {
		if e.selAnchor < 0 {
			e.selAnchor = e.caret
		}
	} else {
		e.selAnchor = -1
	}
	e.caret = off
	e.lastMerge = mergeNone
	e.goalSet = false // horizontal moves and jumps reset the column Up/Down keep
	e.blinkStart = time.Now()
	e.ensureCaretVisible()
}

func (e *Editor) moveCluster(dir int, extend bool) {
	ln := e.lineOf(e.caret)
	ls := e.pt.LineStart(ln)
	off := e.caret - ls
	// At a line's edge the step crosses the line break.
	if dir > 0 && off >= e.pt.LineLen(ln) && ln+1 < e.pt.LineCount() {
		e.moveTo(e.pt.LineStart(ln+1), extend)
		return
	}
	if dir < 0 && off <= 0 && ln > 0 {
		e.moveTo(ls-1, extend)
		return
	}
	view := e.lineView(ln)
	var next int
	var gx float32
	switch {
	case e.SoftWrap:
		// Rune-step horizontally; the row's left edge anchors the goal x.
		next = monoRuneStep(view, off, dir)
		rowStart := e.rowOfByte(ls + next)
		_, absStart, _ := e.wrapRowAbs(rowStart)
		gx = e.rowIndentPx(rowStart) + e.monoPrefixWidth(view[absStart-ls:], next-(absStart-ls))
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
		if !e.goalSet {
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
		off := e.monoByteForX(view, e.goalX-e.rowIndentPx(r))
		e.moveTo(absStart+off, extend)
		e.goalSet = true
		return
	}
	line, _ := e.lineColOf(e.caret)
	if !e.goalSet {
		e.goalX = e.caretXInLine(e.lineOf(e.caret), e.caret)
	}
	// Step a visual row, so a collapsed block counts as one line.
	line = e.rowLine(clampInt(e.lineRow(line)+dir, 0, e.lineRowCount()-1))
	ls := e.pt.LineStart(line)
	var off int
	if e.isHuge(line) {
		off = e.monoByteForX(e.lineView(line), e.goalX)
	} else {
		off = e.shapedLine(line).ByteForX(e.goalX)
	}
	e.moveTo(ls+off, extend)
	e.goalSet = true
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

func (e *Editor) onMouse(el *layout.Element, m *input.Mouse) {
	if el.Frame.Contains(m.X, m.Y) && (m.ScrollY != 0 || m.ScrollX != 0) {
		e.vbar.ApplyWheel(m, el.Frame)
		e.hbar.ApplyWheel(m, el.Frame)
		e.clampScroll()
	}
	if e.overScrollbar(m) {
		return
	}

	if e.findMouse(m) {
		return
	}
	if e.foldMouse(m) {
		return
	}

	f := el.Frame
	if m.RightPressed && f.Contains(m.X, m.Y) && !m.Consumed {
		e.openContextMenuAt(m)
		return
	}
	if m.Pressed && f.Contains(m.X, m.Y) {
		off := e.offsetAtPoint(m.X, m.Y)
		if m.Consumed {
			return
		}
		if m.Mods.Primary() && e.OnModClick != nil && e.OnModClick(off) {
			e.caret, e.selAnchor = off, -1
			e.extra = nil
			m.Consumed = true
			return
		}
		if m.Mods.Has(input.ModAlt) && m.Mods.Has(input.ModShift) {
			// ⌥⇧-drag selects a column.
			e.startBoxSelect(m.X, m.Y)
			m.Consumed = true
			return
		}
		if m.Mods.Has(input.ModShift) {
			// ⇧-click extends the selection from its anchor, and a drag
			// keeps extending it.
			e.extra = nil
			if e.selAnchor < 0 {
				e.selAnchor = e.caret
			}
			e.caret = off
			e.dragging = true
			e.lastMerge = mergeNone
			e.goalSet = false
			e.blinkStart = time.Now()
			m.Consumed = true
			return
		}
		if m.Mods.Has(input.ModAlt) && !e.readOnly {
			// ⌥-click adds a caret.
			e.AddCursorAt(off)
			e.blinkStart = time.Now()
			m.Consumed = true
			return
		}
		e.extra = nil
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
		e.lastMerge = mergeNone
		e.goalSet = false
		e.blinkStart = time.Now()
		m.Consumed = true
	}
	if e.box.on && m.Down {
		e.updateBoxSelect(m.X, m.Y)
	}
	if e.dragging && m.Down {
		e.caret = e.offsetAtPoint(m.X, m.Y)
	}
	if !m.Down {
		e.dragging = false
		e.box.on = false
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
		ind := e.rowIndentPx(row)
		off := e.monoByteForX(view, px-x0-ind)
		e.goalX = ind + e.monoPrefixWidth(view, off)
		return absStart + off
	}
	row := int((py - f.Y + e.ScrollPx) / e.lineH)
	line := e.rowLine(clampInt(row, 0, e.lineRowCount()-1))
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

// CenterCaret scrolls so the caret's row sits in the middle of the view, for
// jumps to a definition or search result. An editor that has not been laid
// out yet centers once it has.
func (e *Editor) CenterCaret() {
	e.revealCenter = true
	e.ensureCaretVisible()
}

func (e *Editor) ensureCaretVisible() {
	if len(e.fold.hidden) > 0 {
		// A caret placed inside a collapsed block (a search match, a jump to
		// a definition) opens it.
		for _, c := range append([]int{e.caret}, e.extraCarets()...) {
			e.revealLine(e.lineOf(c))
		}
	}
	_, clientH, _, _ := e.scrollMetrics()
	if clientH <= 0 {
		// Not laid out yet (a freshly opened file): reveal after layout.
		e.revealCaret = true
		return
	}
	if e.wrapStale() {
		// An edit changed the text since the last wrap. Scroll once layout
		// has re-wrapped, rather than re-wrapping on every edit of a batch.
		e.revealCaret = true
		return
	}
	e.revealWoke = false
	row := e.rowOfByte(e.caret)
	top := float32(row) * e.lineH
	if e.revealCenter {
		e.revealCenter = false
		e.ScrollPx = top + e.lineH/2 - clientH/2
	} else if top < e.ScrollPx {
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
			for ln < len(e.lineRows) && ri >= e.shownRows(ln) {
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
				xoff: float32(row.indent) * cellW,
			})
			ri++
		}
	} else {
		rLast := minInt(first+visibleCount, e.lineRowCount())
		for r := first; r < rLast; r++ {
			ln := e.rowLine(r)
			ll := e.pt.LineLen(ln)
			pr := paintRow{line: ln, rowStart: 0, rowEnd: ll, lineEnd: ll, visual: r, first: true}
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
		lineY := e.rowY(r)
		if lineY >= vp.Y-e.lineH && lineY < vp.Y+vp.H {
			dl.PushClip(vp)
			dl.AddRect(render.Rect{X: vp.X, Y: lineY, W: vp.W, H: e.lineH}, th.CurrentLine)
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
	ghost := e.inlinePaintLines()
	caretRow := -1
	if ghost != nil {
		caretRow = e.rowOfByte(e.caret)
	}
	e.fold.pills = e.fold.pills[:0]
	for _, pr := range rows {
		y := e.rowY(pr.visual)
		if y >= vp.Y+vp.H {
			break
		}
		if e.foldCovers(pr.visual) {
			dl.AddRect(render.Rect{X: textArea.X, Y: y, W: textArea.W, H: e.lineH}, th.Background)
		}
		ls := e.pt.LineStart(pr.line)
		txt := e.lineView(pr.line)
		lineEnd := ls + pr.lineEnd

		// The painted slice is small in every mode: whole line, wrapped row, or
		// a huge line's visible window. Wrapped rows start at the text origin
		// plus their wrap indent; a huge line's window sits at its offset.
		sline := e.lineMono(string(txt[pr.rowStart:pr.rowEnd]))
		drawX := x0 + pr.xoff
		if !e.SoftWrap && pr.rowStart == 0 {
			if w := e.gutterW + e.textPad + sline.Width; w > e.ContentWidth {
				e.ContentWidth = w
			}
		}

		// Block guide lines at each tab-stop within the line's indentation.
		if pr.first {
			if depth := indentDepth(txt, e.tabW); depth > 0 {
				for col := e.tabW; col < depth; col += e.tabW {
					gx := x0 + float32(col)*cellW
					dl.AddRect(render.Rect{X: gx, Y: y, W: 1, H: e.lineH}, th.IndentGuide)
				}
			}
		}

		// Search match highlights.
		if e.search.open {
			matchCol := th.SearchMatch
			activeCol := th.SearchMatchActive
			ms := e.search.matches
			for i := sort.Search(len(ms), func(i int) bool { return ms[i].hi > ls }); i < len(ms); i++ {
				mr := ms[i]
				if mr.lo >= lineEnd {
					break
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
		for _, c := range e.extra {
			if lo, hi := c.lo(), c.hi(); lo < hi && lo <= lineEnd && hi >= ls {
				for _, rect := range e.windowRects(sline, maxInt(lo, ls)-ls, minInt(hi, lineEnd)-ls, pr.rowStart, pr.rowEnd, y, e.lineH) {
					dl.AddRect(render.Rect{X: drawX + rect[0], Y: rect[1], W: rect[2], H: rect[3]}, th.Selection)
				}
			}
		}

		// Bracket match highlights.
		if hasBrackets {
			bracketHL := th.BracketMatch
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
		// On the caret's row the text after the caret moves past the ghost
		// text, so the row paints only up to the caret here.
		split := len(txt)
		if pr.visual == caretRow {
			split = e.caret - tintBase
		}
		engine.DrawLineGlyphs(dl, sline, drawX, topY, func(byteOff int) render.Color {
			if byteOff >= split {
				return render.Color{}
			}
			col, h := e.colorAtHint(tintBase+byteOff, tokHint)
			tokHint = h
			return th.SyntaxColor(col)
		})

		e.paintDiagnosticsLine(dl, ls, lineEnd, sline, drawX, pr.rowStart, pr.rowEnd, y)
		e.paintFoldPlaceholder(dl, engine, pr, x0, y, vpad)
	}
	if caretRow >= 0 {
		ls := e.pt.LineStart(e.lineOf(e.caret))
		rowEnd := ls + e.pt.LineLen(e.lineOf(e.caret))
		if e.SoftWrap && len(e.rowPrefix) > 1 {
			_, _, end := e.wrapRowAbs(caretRow)
			rowEnd = end
		}
		suffixBase := e.caret
		rowEnd = min(max(rowEnd, suffixBase), suffixBase+maxShapedLine) // a huge line paints a bounded tail
		suffix := e.pt.Bytes()[suffixBase:rowEnd]
		cy := e.rowY(caretRow)
		e.paintInline(dl, engine, ghost, x0+e.xInRow(e.caret), cy, x0, cy+vpad, suffix, func(byteOff int) render.Color {
			col, _ := e.colorAtHint(suffixBase+byteOff, 0)
			return th.SyntaxColor(col)
		}, textArea)
	}
	if e.focused && e.caretVisible() {
		for _, caret := range append([]int{e.caret}, e.extraCarets()...) {
			r := e.rowOfByte(caret)
			cx := x0 + e.xInRow(caret)
			cy := e.rowY(r)
			if cy+e.lineH < vp.Y || cy > vp.Y+vp.H {
				continue
			}
			switch e.CaretStyle {
			case CaretBlock:
				w := e.cellW
				if next := e.nextRune(caret); next > caret && e.lineOf(next) == e.lineOf(caret) && e.pt.Bytes()[caret] != '\n' {
					w = f32max(w, x0+e.xInRow(next)-cx)
				}
				col := th.Accent
				col.A *= 0.55
				dl.AddRect(render.Rect{X: cx, Y: cy + vpad, W: w, H: textH}, col)
			case CaretUnderline:
				dl.AddRect(render.Rect{X: cx, Y: cy + vpad + textH - 2, W: e.cellW, H: 2}, th.Accent)
			default:
				dl.AddRect(render.Rect{X: cx, Y: cy + vpad, W: 2, H: textH}, th.Accent)
			}
		}
	}
	dl.PopClip()
	dl.PopClip()

	if e.gutterW > 0 {
		foldW := float32(0)
		if !e.fold.off {
			foldW = foldGutterW
		}
		numRight := f.X + e.gutterW - foldW - 8
		if g := e.gutter; g != nil {
			if g.Fill != TokenUnset {
				dl.AddRect(gutter, g.Fill.Resolve(th))
			}
			if g.Separator {
				dl.AddRect(render.Rect{X: gutter.X + gutter.W - 1, Y: gutter.Y, W: 1, H: vp.H}, th.Border)
			}
			numRight = f.X + e.gutterW - foldW - e.gutterPad()
		} else {
			dl.AddRect(gutter, th.ChromeMuted)
			// 1px divider between gutter and text area.
			dl.AddRect(render.Rect{X: gutter.X + gutter.W - 1, Y: gutter.Y, W: 1, H: vp.H}, th.Border)
		}
		dl.PushClip(render.Rect{X: gutter.X, Y: gutter.Y, W: gutter.W, H: vp.H})
		markX := gutter.X + gutter.W - foldW - lineMarkW - 1
		for _, pr := range rows {
			y := e.rowY(pr.visual)
			if y >= vp.Y+vp.H {
				break
			}
			if e.foldCovers(pr.visual) {
				dl.AddRect(render.Rect{X: gutter.X, Y: y, W: gutter.W, H: e.lineH}, th.ChromeMuted)
			}
			e.paintFoldChevron(dl, pr, gutter, y)
			if k, ok := e.lineMarks[pr.line]; ok {
				switch k {
				case LineAdded:
					dl.AddRect(render.Rect{X: markX, Y: y, W: lineMarkW, H: e.lineH}, th.Success)
				case LineModified:
					dl.AddRect(render.Rect{X: markX, Y: y, W: lineMarkW, H: e.lineH}, th.Accent)
				case LineRemoved:
					if pr.first {
						dl.AddRect(render.Rect{X: markX - 3, Y: y - 1, W: lineMarkW + 3, H: 3}, th.Error)
					}
				}
			}
			if !pr.first {
				continue // continuation rows carry no gutter number
			}
			num := strconv.Itoa(pr.line + 1)
			numW, _ := engine.MeasureMono(num)
			engine.DrawStringTopMono(dl, num, numRight-numW, y+vpad, th.ForegroundSubtle)
		}
		dl.PopClip()
	}

	// Search/replace overlay (drawn last, on top of everything).
	e.paintFindBar(dl, engine)
}

// LineMarkKind is how a line differs from a base version, such as the last
// commit, for the gutter's change bar.
type LineMarkKind int

const (
	LineAdded    LineMarkKind = iota + 1 // a new line
	LineModified                         // a changed line
	LineRemoved                          // lines were deleted just above this one
)

// lineMarkW is the width of the gutter change bar.
const lineMarkW = 3

// SetLineMarks shows change bars in the gutter, keyed by 0-based line. A
// LineRemoved mark sits on the line after the deleted ones (a line past the
// end marks a deletion at the end). Nil clears them.
func (e *Editor) SetLineMarks(marks map[int]LineMarkKind) { e.lineMarks = marks }

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
		switch r {
		case same:
			depth++
		case other:
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
		switch r {
		case same:
			depth++
		case other:
			depth--
			if depth == 0 {
				return i - sz
			}
		}
		i -= sz
	}
	return -1
}
