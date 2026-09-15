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

// Noop is a highlighter that produces no tokens; the editor falls back to the
// default text color. Useful for tests, non-code text, web/WASM builds (no
// Tree-sitter CGO), or when Tree-sitter is undesirable.
type Noop struct{}

func (Noop) Update([]byte)           {}
func (Noop) UpdateEdit([]byte, Edit) {}
func (Noop) Poll() ([]Token, bool)   { return nil, false }
func (Noop) Close()                  {}
