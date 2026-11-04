//go:build js

package highlight

import "path/filepath"

// ForPath returns Noop on js/wasm — Tree-sitter grammars require CGO.
func ForPath(path string) Highlighter {
	_ = filepath.Ext(path)
	if h, ok := registered(path); ok {
		return h
	}
	return Noop{}
}

// NewGo returns Noop on js/wasm.
func NewGo() Highlighter { return Noop{} }

// NewJSON returns Noop on js/wasm.
func NewJSON() Highlighter { return Noop{} }

// NewXML returns Noop on js/wasm.
func NewXML() Highlighter { return Noop{} }

// NewPython returns Noop on js/wasm.
func NewPython() Highlighter { return Noop{} }

// Tree-sitter grammars below return Noop on js/wasm.
func NewTypeScript() Highlighter { return Noop{} }
func NewTSX() Highlighter        { return Noop{} }
func NewJavaScript() Highlighter { return Noop{} }
func NewRust() Highlighter       { return Noop{} }
func NewBash() Highlighter       { return Noop{} }
func NewCSS() Highlighter        { return Noop{} }
func NewHTML() Highlighter       { return Noop{} }
func NewYAML() Highlighter       { return Noop{} }
func NewSQL() Highlighter        { return Noop{} }
func NewMarkdown() Highlighter   { return Noop{} }
func NewJava() Highlighter       { return Noop{} }
func NewProto() Highlighter      { return Noop{} }
func NewMake() Highlighter       { return Noop{} }
func NewTOML() Highlighter       { return Noop{} }
func NewC() Highlighter          { return Noop{} }
func NewCPP() Highlighter        { return Noop{} }
func NewCSharp() Highlighter     { return Noop{} }
func NewLua() Highlighter        { return Noop{} }
func NewRuby() Highlighter       { return Noop{} }
func NewPHP() Highlighter        { return Noop{} }
func NewZig() Highlighter        { return Noop{} }
func NewGitignore() Highlighter  { return Noop{} }
func NewDotenv() Highlighter     { return Noop{} }
func NewINI() Highlighter        { return Noop{} }
func NewDockerfile() Highlighter { return Noop{} }
func NewGoMod() Highlighter      { return Noop{} }
func NewDiff() Highlighter       { return Noop{} }

// Grammars lists none on js/wasm.
func Grammars() []string { return nil }

// BuiltinQuery returns "" on js/wasm.
func BuiltinQuery(string) string { return "" }

// QueryFactory returns Noop highlighters on js/wasm.
func QueryFactory(string, string) (Factory, error) {
	return func() Highlighter { return Noop{} }, nil
}
