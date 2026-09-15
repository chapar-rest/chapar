//go:build !js

// Package highlight maps source text to colored token ranges using Tree-sitter.
//
// Parsing runs on a dedicated goroutine (the "worker loop") so the UI thread is
// never blocked by a parse. The editor pushes source via Update (full reparse)
// or UpdateEdit (incremental) and polls for finished results via Poll; both are
// non-blocking. Results cross the goroutine boundary as a plain Go slice of byte
// ranges, so the UI never touches a live Tree-sitter tree.
//
// Incremental reparsing: UpdateEdit supplies an InputEdit descriptor; the worker
// applies each queued edit to the previous tree with Tree.Edit, then calls
// Parse(latestSrc, prev). Update requests a full reparse (initial load). Pending
// jobs accumulate (not coalesced away) so the edit chain stays consistent.
//
// Cgo lifecycle: the Tree-sitter Parser and every Tree allocate C memory that
// the Go GC does not track. Per the binding's documentation, SetFinalizer is
// unreliable here, so the worker owns these objects and calls Close()
// deterministically — the previous tree is closed once its successor is parsed,
// and the parser/tree are closed when the loop exits on Close().
package highlight

import (
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	tree_sitter_xml "github.com/tree-sitter-grammars/tree-sitter-xml/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_json "github.com/tree-sitter/tree-sitter-json/bindings/go"
)

func (e Edit) toInputEdit() *tree_sitter.InputEdit {
	return &tree_sitter.InputEdit{
		StartByte:      uint(e.StartByte),
		OldEndByte:     uint(e.OldEndByte),
		NewEndByte:     uint(e.NewEndByte),
		StartPosition:  tree_sitter.NewPoint(uint(e.Start.Row), uint(e.Start.Col)),
		OldEndPosition: tree_sitter.NewPoint(uint(e.OldEnd.Row), uint(e.OldEnd.Col)),
		NewEndPosition: tree_sitter.NewPoint(uint(e.NewEnd.Row), uint(e.NewEnd.Col)),
	}
}

// ForPath returns a highlighter appropriate for a file path, chosen by its
// extension. This is the single place to register new languages: add a grammar
// binding and a classifier, then map the extension here. Unknown types fall
// back to plain text (Noop).
func ForPath(path string) Highlighter {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return NewGo()
	case ".json":
		return NewJSON()
	case ".xml", ".xsd", ".xsl", ".xslt", ".svg":
		return NewXML()
	default:
		return Noop{}
	}
}

// classifyFunc walks a parsed syntax tree and emits a flat, ordered list of
// colored token ranges. Each language supplies one.
type classifyFunc func(root *tree_sitter.Node, src []byte) []Token

type parseJob struct {
	src  []byte
	edit *Edit // nil means full reparse
}

// tsHighlighter is a generic Tree-sitter-backed highlighter. It is parameterized
// by a grammar (langFn returns the C language pointer) and a classifier, so a
// new language is just those two values — the async worker loop, result
// coalescing, and Cgo tree lifecycle are shared.
type tsHighlighter struct {
	mu       sync.Mutex
	pending  []parseJob
	wake     chan struct{}
	results  chan []Token
	done     chan struct{}
	langFn   func() unsafe.Pointer
	classify classifyFunc
}

// newTS starts a worker loop for the given grammar/classifier and returns it.
func newTS(langFn func() unsafe.Pointer, classify classifyFunc) Highlighter {
	h := &tsHighlighter{
		wake:     make(chan struct{}, 1),
		results:  make(chan []Token, 1),
		done:     make(chan struct{}),
		langFn:   langFn,
		classify: classify,
	}
	go h.loop()
	return h
}

// NewGo starts a worker loop highlighting Go source and returns its handle.
func NewGo() Highlighter { return newTS(tree_sitter_go.Language, classifyGo) }

// NewJSON starts a worker loop highlighting JSON and returns its handle.
func NewJSON() Highlighter { return newTS(tree_sitter_json.Language, classifyJSON) }

// NewXML starts a worker loop highlighting XML (and XML dialects such as SVG,
// XSD, and XSL) and returns its handle.
func NewXML() Highlighter { return newTS(tree_sitter_xml.LanguageXML, classifyXML) }

func (h *tsHighlighter) loop() {
	parser := tree_sitter.NewParser()
	defer parser.Close()

	lang := tree_sitter.NewLanguage(h.langFn())
	if err := parser.SetLanguage(lang); err != nil {
		return
	}

	var prev *tree_sitter.Tree
	var prevLen int
	defer func() {
		if prev != nil {
			prev.Close()
		}
	}()

	for {
		select {
		case <-h.done:
			return
		case <-h.wake:
			batch := h.drainPending()
			if len(batch) == 0 {
				continue
			}
			latestSrc := batch[len(batch)-1].src

			full := false
			for _, j := range batch {
				if j.edit == nil {
					full = true
					break
				}
			}

			var tree *tree_sitter.Tree
			if full {
				tree = parser.Parse(latestSrc, nil)
			} else if prev != nil {
				expectedLen := prevLen
				for _, j := range batch {
					e := j.edit
					prev.Edit(e.toInputEdit())
					expectedLen += e.NewEndByte - e.OldEndByte
				}
				if expectedLen != len(latestSrc) {
					tree = parser.Parse(latestSrc, nil)
				} else {
					tree = parser.Parse(latestSrc, prev)
				}
			} else {
				tree = parser.Parse(latestSrc, nil)
			}

			if tree == nil {
				continue
			}
			if prev != nil {
				prev.Close()
			}
			prev = tree
			prevLen = len(latestSrc)

			toks := h.classify(tree.RootNode(), latestSrc)
			deliver(h.results, toks)
		}
	}
}

func (h *tsHighlighter) drainPending() []parseJob {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.pending) == 0 {
		return nil
	}
	batch := h.pending
	h.pending = nil
	return batch
}

func (h *tsHighlighter) enqueue(job parseJob) {
	h.mu.Lock()
	h.pending = append(h.pending, job)
	h.mu.Unlock()
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

// deliver pushes the latest result, discarding any unconsumed older one so the
// editor always sees the freshest tokens.
func deliver(ch chan []Token, toks []Token) {
	select {
	case ch <- toks:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- toks:
		default:
		}
	}
}

func (h *tsHighlighter) Update(source []byte) {
	cp := append([]byte(nil), source...)
	h.enqueue(parseJob{src: cp, edit: nil})
}

func (h *tsHighlighter) UpdateEdit(source []byte, edit Edit) {
	cp := append([]byte(nil), source...)
	ed := edit
	h.enqueue(parseJob{src: cp, edit: &ed})
}

func (h *tsHighlighter) Poll() ([]Token, bool) {
	select {
	case toks := <-h.results:
		return toks, true
	default:
		return nil, false
	}
}

func (h *tsHighlighter) Close() { close(h.done) }

// goKeywords is the set of Go keywords; Tree-sitter emits these as anonymous
// leaf nodes whose Kind() is the literal keyword text.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// classifyGo walks a Go syntax tree and emits a flat, ordered list of colored
// token ranges. Container nodes recurse; leaf-ish nodes are classified directly.
func classifyGo(root *tree_sitter.Node, src []byte) []Token {
	var toks []Token
	add := func(n *tree_sitter.Node, c ColorClass) {
		toks = append(toks, Token{Start: int(n.StartByte()), End: int(n.EndByte()), Class: c})
	}

	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "comment":
			add(n, ClassComment)
			return
		case "interpreted_string_literal", "raw_string_literal", "rune_literal":
			add(n, ClassString)
			return
		case "int_literal", "float_literal", "imaginary_literal":
			add(n, ClassNumber)
			return
		case "type_identifier":
			add(n, ClassType)
			return
		}

		count := n.ChildCount()
		if count == 0 {
			if !n.IsNamed() && goKeywords[n.Kind()] {
				add(n, ClassKeyword)
			}
			return
		}
		for i := uint(0); i < count; i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}
	walk(root)
	return toks
}

// classifyJSON walks a JSON syntax tree. Object keys are colored distinctly
// (ClassType) from string values (ClassString); numbers, the literals
// true/false/null, and comments (JSONC) get their own classes.
func classifyJSON(root *tree_sitter.Node, src []byte) []Token {
	var toks []Token
	add := func(n *tree_sitter.Node, c ColorClass) {
		toks = append(toks, Token{Start: int(n.StartByte()), End: int(n.EndByte()), Class: c})
	}

	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "comment":
			add(n, ClassComment)
			return
		case "number":
			add(n, ClassNumber)
			return
		case "true", "false", "null":
			add(n, ClassKeyword)
			return
		case "string":
			add(n, ClassString)
			return
		case "pair":
			// A key/value member: color the key like a property name and recurse
			// only into the value, so the key string is not re-colored generically.
			if k := n.ChildByFieldName("key"); k != nil {
				add(k, ClassType)
			}
			if v := n.ChildByFieldName("value"); v != nil {
				walk(v)
			}
			return
		}

		count := n.ChildCount()
		for i := uint(0); i < count; i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}
	walk(root)
	return toks
}

// classifyXML walks an XML syntax tree (the tree-sitter-grammars XML grammar,
// which also covers XML dialects like SVG and XSD). Tag names are colored as
// keywords, attribute names like property names (ClassType), and attribute
// values as strings; text content keeps the default color.
func classifyXML(root *tree_sitter.Node, src []byte) []Token {
	var toks []Token
	add := func(n *tree_sitter.Node, c ColorClass) {
		toks = append(toks, Token{Start: int(n.StartByte()), End: int(n.EndByte()), Class: c})
	}

	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "Comment":
			add(n, ClassComment)
			return
		case "CharData":
			return // plain text keeps the default color
		case "AttValue", "PseudoAttValue", "SystemLiteral", "PubidLiteral":
			add(n, ClassString)
			return
		case "PI", "XMLDecl", "CDSect", "EntityRef", "CharRef":
			add(n, ClassKeyword)
			return
		case "Attribute", "PseudoAtt":
			// Color the name (ClassType) directly and recurse for the value,
			// so AttValue gets its string class without re-coloring the name.
			count := n.ChildCount()
			for i := uint(0); i < count; i++ {
				if c := n.Child(i); c != nil && c.Kind() == "Name" {
					add(c, ClassType)
				}
			}
			for i := uint(0); i < count; i++ {
				if c := n.Child(i); c != nil && c.Kind() != "Name" {
					walk(c)
				}
			}
			return
		case "STag", "ETag", "EmptyElemTag":
			// Color the element name; attributes recurse for their own rules.
			count := n.ChildCount()
			for i := uint(0); i < count; i++ {
				c := n.Child(i)
				if c == nil {
					continue
				}
				if c.Kind() == "Name" {
					add(c, ClassKeyword)
				} else {
					walk(c)
				}
			}
			return
		}

		count := n.ChildCount()
		for i := uint(0); i < count; i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}
	walk(root)
	return toks
}
