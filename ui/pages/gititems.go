package pages

import (
	"bytes"
	"path"
	"strings"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"gopkg.in/yaml.v2"
)

// gitItem is the workspace item a changed file holds.
type gitItem struct {
	// Kind is the file's kind field, such as "Request", or "" for a file
	// that is not a Chapar item.
	Kind string
	ID   string
	Name string
	// Type names the item for the list: "Request", "Collection", ...
	Type string
	// Where is the collection a request belongs to, or "".
	Where string
}

// itemHeader is the part of a workspace file that names its item.
type itemHeader struct {
	Kind     string `yaml:"kind"`
	MetaData struct {
		ID   string `yaml:"id"`
		Name string `yaml:"name"`
	} `yaml:"metadata"`
}

// itemFor maps a workspace file to the item it holds. data is the file's
// content, from the work tree or, for a deleted file, the last commit; it
// may be nil, and then the name comes from the path.
func itemFor(p string, data []byte) gitItem {
	it := gitItem{Name: strings.TrimSuffix(path.Base(p), ".yaml"), Type: "File"}
	parts := strings.Split(p, "/")
	isYAML := strings.HasSuffix(p, ".yaml")
	switch {
	case p == "_workspace.yaml":
		it.Type = "Workspace"
	case len(parts) == 3 && parts[0] == "collections" && parts[2] == "_collection.yaml":
		it.Type, it.Name = "Collection", parts[1]
	case len(parts) == 3 && parts[0] == "collections" && isYAML:
		it.Type, it.Where = "Request", parts[1]
	case len(parts) == 2 && parts[0] == "requests" && isYAML:
		it.Type = "Request"
	case len(parts) == 2 && parts[0] == "envs" && isYAML:
		it.Type = "Environment"
	case len(parts) == 2 && parts[0] == "testcases" && isYAML:
		it.Type = "Test case"
	case len(parts) == 2 && parts[0] == "protofiles" && isYAML:
		it.Type = "Proto file"
	default:
		it.Name = p
		return it
	}
	var h itemHeader
	if len(data) > 0 && yaml.Unmarshal(data, &h) == nil {
		it.Kind, it.ID = h.Kind, h.MetaData.ID
		if h.MetaData.Name != "" {
			it.Name = h.MetaData.Name
		}
	}
	return it
}

// icon is the glyph for the item's type.
func (it gitItem) icon() icons.Icon {
	switch it.Type {
	case "Request":
		return icons.Send
	case "Collection":
		return icons.Folder
	case "Environment":
		return icons.FolderPlus
	case "Test case":
		return icons.FlaskConical
	case "Workspace":
		return icons.Boxes
	case "Proto file":
		return icons.FileCode
	}
	return icons.File
}

// label is the item's name for the list, with its collection.
func (it gitItem) label() string {
	if it.Where != "" {
		return it.Where + " / " + it.Name
	}
	return it.Name
}

// diffTokens colors a unified diff by line: added and removed lines, hunk
// headers, and the file header lines git writes before the first hunk.
func diffTokens(data []byte) []highlight.Token {
	var toks []highlight.Token
	inHeader := false
	for start := 0; start < len(data); {
		end := bytes.IndexByte(data[start:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += start
		}
		line := data[start:end]
		class, ok := highlight.ClassDefault, false
		switch {
		case bytes.HasPrefix(line, []byte("diff ")):
			inHeader = true
			class, ok = highlight.ClassMuted, true
		case bytes.HasPrefix(line, []byte("@@")):
			inHeader = false
			class, ok = highlight.ClassType, true
		case inHeader:
			class, ok = highlight.ClassMuted, true
		case len(line) > 0 && line[0] == '+':
			class, ok = highlight.ClassSuccess, true
		case len(line) > 0 && line[0] == '-':
			class, ok = highlight.ClassError, true
		case len(line) > 0 && line[0] == '\\':
			class, ok = highlight.ClassMuted, true
		}
		if ok && end > start {
			toks = append(toks, highlight.Token{Start: start, End: end, Class: class})
		}
		start = end + 1
	}
	return toks
}

// staticHighlighter hands the editor tokens worked out up front. The diff
// editors are read-only and rebuilt for new content, so the tokens never
// go stale.
type staticHighlighter struct {
	tokens []highlight.Token
	sent   bool
}

func (h *staticHighlighter) Update([]byte)                     {}
func (h *staticHighlighter) UpdateEdit([]byte, highlight.Edit) {}
func (h *staticHighlighter) Close()                            {}
func (h *staticHighlighter) Poll() ([]highlight.Token, bool) {
	if h.sent {
		return nil, false
	}
	h.sent = true
	return h.tokens, true
}
