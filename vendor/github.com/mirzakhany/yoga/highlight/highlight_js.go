//go:build js

package highlight

import "path/filepath"

// ForPath returns Noop on js/wasm — Tree-sitter grammars require CGO.
func ForPath(path string) Highlighter {
	_ = filepath.Ext(path)
	return Noop{}
}

// NewGo returns Noop on js/wasm.
func NewGo() Highlighter { return Noop{} }

// NewJSON returns Noop on js/wasm.
func NewJSON() Highlighter { return Noop{} }

// NewXML returns Noop on js/wasm.
func NewXML() Highlighter { return Noop{} }
