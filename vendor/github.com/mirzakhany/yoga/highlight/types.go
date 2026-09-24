package highlight

// ColorClass is a semantic token category the renderer maps to a theme color.
type ColorClass uint8

const (
	ClassDefault ColorClass = iota
	ClassKeyword
	ClassString
	ClassComment
	ClassNumber
	ClassType
	// Status classes color text by meaning rather than syntax, for example
	// log lines by level. Themes map them to their status colors unless the
	// Syntax map overrides them.
	ClassError
	ClassWarning
	ClassSuccess
	ClassMuted
)

// Token is a half-open byte range [Start, End) with a color class.
type Token struct {
	Start, End int
	Class      ColorClass
}

// Pt is a zero-based row/column position in the document. Column is a byte offset
// within the row (UTF-8), matching Tree-sitter's Point for UTF-8 grammars.
type Pt struct {
	Row, Col int
}

// Edit describes a single source mutation for incremental Tree-sitter parsing.
// Byte offsets are half-open [StartByte, OldEndByte) replaced by text ending at
// NewEndByte in the new buffer.
type Edit struct {
	StartByte, OldEndByte, NewEndByte int
	Start, OldEnd, NewEnd             Pt
}

// Highlighter is the async syntax-highlighting interface the editor depends on.
// Swapping in a different engine (or the Noop highlighter) only requires
// implementing these methods.
type Highlighter interface {
	// Update requests a full reparse (initial load / external content set).
	Update(source []byte)
	// UpdateEdit requests an incremental reparse after a single edit.
	UpdateEdit(source []byte, edit Edit)
	// Poll returns the most recent finished token set, or ok=false if nothing
	// new has completed since the last call.
	Poll() (tokens []Token, ok bool)
	// Close stops the worker and frees its native resources.
	Close()
}

// RangeHighlighter is an optional Highlighter capability: the consumer can
// restrict classification to a byte range.
//
// A viewport shows a few dozen lines, so classifying a whole document on every
// reparse does work proportional to the file instead of to the screen — on an
// 8 MB body that was ~740 ms and 135 MB per keystroke, against ~0.6 ms and
// 66 kB for the visible range. Highlighters that cannot scope their work simply
// do not implement this, and consumers fall back to whole-document results.
type RangeHighlighter interface {
	Highlighter
	// SetRange asks for tokens covering [lo, hi). Bytes outside the range have
	// no tokens, so consumers must treat them as ClassDefault. Calling it with
	// an unchanged range does nothing.
	SetRange(lo, hi int)
}

// SizeLimited is an optional Highlighter capability: the highlighter leaves
// documents over a size limit uncolored, reports when it did, and lets the
// consumer move the limit.
//
// A consumer can use it to tell the reader why a large document has no colors
// and to offer highlighting it anyway.
type SizeLimited interface {
	Highlighter
	// Oversize reports whether the last source passed to Update or UpdateEdit
	// was over the limit and so left unhighlighted.
	Oversize() bool
	// SetMaxBytes changes the limit for sources passed from now on. Values
	// <= 0 mean DefaultMaxBytes; math.MaxInt removes the limit. The current
	// source is not reparsed until the next Update.
	SetMaxBytes(n int)
}

// Noop is a highlighter that produces no tokens; the editor falls back to the
// default text color. Useful for tests, non-code text, web/WASM builds (no
// Tree-sitter CGO), or when Tree-sitter is undesirable.
type Noop struct{}

func (Noop) Update([]byte)           {}
func (Noop) UpdateEdit([]byte, Edit) {}
func (Noop) Poll() ([]Token, bool)   { return nil, false }
func (Noop) Close()                  {}
