package highlight

import "github.com/yogaui/yoga/syntax"

// Register maps a file extension (with its leading dot, case-insensitive) to
// a highlighter. See syntax.Register.
func Register(ext string, f Factory) { syntax.Register(ext, f) }

// Unregister removes an extension added with Register.
func Unregister(ext string) { syntax.Unregister(ext) }

// ByName returns the highlighter for a language name such as "go",
// "typescript", "tsx" or "yaml".
func ByName(name string) (Factory, bool) { return syntax.ByName(name) }

// Names lists the language names, sorted.
func Names() []string { return syntax.Names() }

// ForPath returns a highlighter for a file path, chosen by its file name or
// extension; extensions added with Register take precedence. Unknown types
// fall back to plain text (Noop).
func ForPath(path string) Highlighter { return syntax.ForPath(path) }
