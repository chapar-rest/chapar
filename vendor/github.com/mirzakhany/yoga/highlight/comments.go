package highlight

import (
	"path/filepath"
	"strings"
)

// Comment is how a language writes comments, for the editor's toggle
// comment: Line starts a line comment ("//"); Open and Close wrap a block
// ("/*", "*/"), used per line where a language has no line comment.
type Comment struct {
	Line        string
	Open, Close string
}

var (
	slashes = Comment{Line: "//", Open: "/*", Close: "*/"}
	hashes  = Comment{Line: "#"}
	dashes  = Comment{Line: "--"}
	markup  = Comment{Open: "<!--", Close: "-->"}
)

// builtinComments maps lowercase extensions (and whole file names) to their
// comment syntax.
var builtinComments = map[string]Comment{
	".go": slashes, ".c": slashes, ".h": slashes, ".cc": slashes, ".cpp": slashes, ".hpp": slashes,
	".cs": slashes, ".java": slashes, ".kt": slashes, ".kts": slashes, ".scala": slashes, ".swift": slashes,
	".js": slashes, ".jsx": slashes, ".mjs": slashes, ".cjs": slashes, ".ts": slashes, ".tsx": slashes,
	".rs": slashes, ".dart": slashes, ".zig": slashes, ".proto": slashes, ".jsonc": slashes, ".json5": slashes,
	".php": slashes, ".groovy": slashes, ".gradle": slashes, ".scss": slashes, ".less": slashes,
	".css": {Open: "/*", Close: "*/"},
	".py":  hashes, ".rb": hashes, ".sh": hashes, ".bash": hashes, ".zsh": hashes, ".fish": hashes,
	".yaml": hashes, ".yml": hashes, ".toml": hashes, ".ini": {Line: ";"}, ".conf": hashes, ".env": hashes,
	".pl": hashes, ".r": hashes, ".ex": hashes, ".exs": hashes, ".nim": hashes, ".tf": hashes, ".mk": hashes,
	".cmake": hashes, ".dockerfile": hashes, "makefile": hashes, "dockerfile": hashes, ".gitignore": hashes,
	".sql": dashes, ".lua": dashes, ".hs": dashes, ".elm": dashes,
	".html": markup, ".htm": markup, ".xml": markup, ".svg": markup, ".vue": markup, ".md": markup, ".markdown": markup,
	".vim": {Line: `"`}, ".el": {Line: ";"}, ".clj": {Line: ";"}, ".lisp": {Line: ";"}, ".erl": {Line: "%"}, ".tex": {Line: "%"},
}

var userComments = map[string]Comment{}

// RegisterComment sets the comment syntax for an extension (with its dot) or
// a whole file name, overriding the built-in one.
func RegisterComment(ext string, c Comment) {
	regMu.Lock()
	defer regMu.Unlock()
	userComments[strings.ToLower(ext)] = c
}

// CommentFor returns the comment syntax for path, judged by its file name or
// extension; the zero Comment when it is unknown.
func CommentFor(path string) Comment {
	if path == "" {
		return Comment{}
	}
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))
	regMu.RLock()
	defer regMu.RUnlock()
	for _, k := range []string{base, ext} {
		if c, ok := userComments[k]; ok {
			return c
		}
	}
	for _, k := range []string{base, ext} {
		if c, ok := builtinComments[k]; ok {
			return c
		}
	}
	return Comment{}
}
