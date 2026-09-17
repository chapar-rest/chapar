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
	"sort"
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

// DefaultMaxBytes is the source-size ceiling for Tree-sitter highlighting.
// Documents larger than this are shown unhighlighted.
//
// A syntax tree costs far more than the text it describes: a 10 MB JSON body
// parses to ~3.5M nodes and ~210 MB of resident C memory, which the allocator
// does not hand back to the OS when the tree is freed. Parsing it also takes
// ~600 ms, and classifying it emits millions of tokens — all to color the ~50
// lines an editor viewport can actually show.
//
// 2 MB keeps the worst case near ~40 MB and ~120 ms while covering the source
// files and API responses people actually read.
const DefaultMaxBytes = 2 << 20

// MaxBytes is the limit applied to highlighters created from here on. Set it
// before constructing them to trade memory for highlighting on larger
// documents; values <= 0 mean DefaultMaxBytes.
var MaxBytes = DefaultMaxBytes

// classifyFunc walks a parsed syntax tree and emits a flat, ordered list of
// colored token ranges for the walker's byte range. Each language supplies one.
type classifyFunc func(w *rangeWalker, root *tree_sitter.Node)

// rangeWalker collects tokens for the byte range a classifier is asked about.
//
// Classifiers describe a language by recursing over nodes; the walker supplies
// the two operations that make that recursion proportional to the range rather
// than to the document: add drops tokens outside it, and children skips whole
// subtrees before it.
type rangeWalker struct {
	lo, hi int
	toks   []Token
}

// newRangeWalker returns a walker scoped to [lo, hi).
func newRangeWalker(lo, hi int) *rangeWalker {
	if hi < lo {
		hi = lo
	}
	return &rangeWalker{lo: lo, hi: hi}
}

// overlaps reports whether n intersects the requested range.
func (w *rangeWalker) overlaps(n *tree_sitter.Node) bool {
	return int(n.EndByte()) > w.lo && int(n.StartByte()) < w.hi
}

// add records a token for n, unless n lies outside the range.
func (w *rangeWalker) add(n *tree_sitter.Node, c ColorClass) {
	if n == nil || !w.overlaps(n) {
		return
	}
	w.toks = append(w.toks, Token{Start: int(n.StartByte()), End: int(n.EndByte()), Class: c})
}

// children visits the children of n that intersect the range, in source order.
//
// Children are source-ordered, so the first one reaching into the range is
// found by binary search instead of by walking every sibling. That is what
// makes a viewport-sized request cheap in a document whose root has hundreds of
// thousands of direct children — a large JSON array, for example.
func (w *rangeWalker) children(n *tree_sitter.Node, visit func(*tree_sitter.Node)) {
	count := int(n.ChildCount())
	if count == 0 {
		return
	}
	start := 0
	if count > childScanLimit {
		start = sort.Search(count, func(i int) bool {
			c := n.Child(uint(i))
			return c == nil || int(c.EndByte()) > w.lo
		})
	}
	for i := start; i < count; i++ {
		c := n.Child(uint(i))
		if c == nil {
			continue
		}
		if int(c.StartByte()) >= w.hi {
			return
		}
		if int(c.EndByte()) <= w.lo {
			continue
		}
		visit(c)
	}
}

// childScanLimit is the child count above which children binary-searches for
// its starting sibling. Below it a linear scan is cheaper than the extra
// Child() calls a search costs, since every one crosses into C.
const childScanLimit = 32

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
	maxBytes int
	// wantLo/wantHi is the byte range the consumer last asked for, and
	// rangeSet records whether it ever asked. Until it does, the whole
	// document is classified, so a consumer that never calls SetRange behaves
	// as before.
	wantLo, wantHi int
	rangeSet       bool
	// oversize records that the last source exceeded maxBytes, so crossing the
	// limit clears any tokens the editor is still holding exactly once.
	oversize bool
}

// newTS starts a worker loop for the given grammar/classifier and returns it.
func newTS(langFn func() unsafe.Pointer, classify classifyFunc) Highlighter {
	h := &tsHighlighter{
		wake:     make(chan struct{}, 1),
		results:  make(chan []Token, 1),
		done:     make(chan struct{}),
		langFn:   langFn,
		classify: classify,
		maxBytes: MaxBytes,
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
	// doneLo/doneHi is the range the delivered tokens cover, so a wake that
	// only repeats the current request does no work.
	doneLo, doneHi := 0, 0
	haveResult := false
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
				// A wake with no parse work is either an oversize transition or
				// a range request against the tree already in hand.
				if h.wasOversize() {
					// Drop the retained tree so its native memory is freed
					// rather than held for the editor's lifetime.
					if prev != nil {
						prev.Close()
						prev = nil
						prevLen = 0
					}
					haveResult = false
					continue
				}
				if prev == nil {
					continue
				}
				lo, hi := h.wantRange()
				if haveResult && lo == doneLo && hi == doneHi {
					continue
				}
				doneLo, doneHi = lo, hi
				haveResult = true
				deliver(h.results, h.classifyRange(prev, lo, hi))
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

			doneLo, doneHi = h.wantRange()
			haveResult = true
			deliver(h.results, h.classifyRange(tree, doneLo, doneHi))
		}
	}
}

// classifyRange walks tree for [lo, hi) and returns the tokens in it.
func (h *tsHighlighter) classifyRange(tree *tree_sitter.Tree, lo, hi int) []Token {
	w := newRangeWalker(lo, hi)
	h.classify(w, tree.RootNode())
	return w.toks
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
	if h.skipOversize(source) {
		return
	}
	cp := append([]byte(nil), source...)
	h.enqueue(parseJob{src: cp, edit: nil})
}

func (h *tsHighlighter) UpdateEdit(source []byte, edit Edit) {
	if h.skipOversize(source) {
		return
	}
	cp := append([]byte(nil), source...)
	ed := edit
	h.enqueue(parseJob{src: cp, edit: &ed})
}

// skipOversize reports whether source is too large to highlight. The check runs
// before the source is copied, so an oversized document costs nothing. The
// first update that crosses the limit delivers an empty token set and drops any
// retained tree, releasing both the editor's tokens and the worker's native
// memory; the first one back under the limit reparses normally.
func (h *tsHighlighter) skipOversize(source []byte) bool {
	limit := h.maxBytes
	if limit <= 0 {
		limit = DefaultMaxBytes
	}
	over := len(source) > limit

	// Read and flip the flag under one lock so concurrent updates cannot both
	// see the transition and clear twice.
	h.mu.Lock()
	crossed := over && !h.oversize
	h.oversize = over
	if crossed {
		h.pending = nil
	}
	h.mu.Unlock()

	if crossed {
		// Clear the stale highlighting and wake the worker so it releases its
		// retained tree.
		deliver(h.results, nil)
		select {
		case h.wake <- struct{}{}:
		default:
		}
	}
	return over
}

func (h *tsHighlighter) wasOversize() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.oversize
}

// SetRange restricts classification to [lo, hi), and asks for a fresh result
// if that range is not the one already reported.
//
// Only the range has to be reclassified — the parsed tree is retained — so
// scrolling costs a range walk rather than a reparse.
func (h *tsHighlighter) SetRange(lo, hi int) {
	if hi < lo {
		lo, hi = hi, lo
	}
	h.mu.Lock()
	changed := !h.rangeSet || lo != h.wantLo || hi != h.wantHi
	h.wantLo, h.wantHi = lo, hi
	h.rangeSet = true
	oversize := h.oversize
	h.mu.Unlock()

	if !changed || oversize {
		return
	}
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

// wantRange returns the requested range, or the whole document when the
// consumer has not asked for one.
func (h *tsHighlighter) wantRange() (lo, hi int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.rangeSet {
		return 0, maxInt
	}
	return h.wantLo, h.wantHi
}

const maxInt = int(^uint(0) >> 1)

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
func classifyGo(w *rangeWalker, root *tree_sitter.Node) {
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "comment":
			w.add(n, ClassComment)
			return
		case "interpreted_string_literal", "raw_string_literal", "rune_literal":
			w.add(n, ClassString)
			return
		case "int_literal", "float_literal", "imaginary_literal":
			w.add(n, ClassNumber)
			return
		case "type_identifier":
			w.add(n, ClassType)
			return
		}

		if n.ChildCount() == 0 {
			if !n.IsNamed() && goKeywords[n.Kind()] {
				w.add(n, ClassKeyword)
			}
			return
		}
		w.children(n, walk)
	}
	walk(root)
}

// classifyJSON walks a JSON syntax tree. Object keys are colored distinctly
// (ClassType) from string values (ClassString); numbers, the literals
// true/false/null, and comments (JSONC) get their own classes.
func classifyJSON(w *rangeWalker, root *tree_sitter.Node) {
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "comment":
			w.add(n, ClassComment)
			return
		case "number":
			w.add(n, ClassNumber)
			return
		case "true", "false", "null":
			w.add(n, ClassKeyword)
			return
		case "string":
			w.add(n, ClassString)
			return
		case "pair":
			// A key/value member: color the key like a property name and recurse
			// only into the value, so the key string is not re-colored generically.
			w.add(n.ChildByFieldName("key"), ClassType)
			if v := n.ChildByFieldName("value"); v != nil && w.overlaps(v) {
				walk(v)
			}
			return
		}
		w.children(n, walk)
	}
	walk(root)
}

// classifyXML walks an XML syntax tree (the tree-sitter-grammars XML grammar,
// which also covers XML dialects like SVG and XSD). Tag names are colored as
// keywords, attribute names like property names (ClassType), and attribute
// values as strings; text content keeps the default color.
func classifyXML(w *rangeWalker, root *tree_sitter.Node) {
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "Comment":
			w.add(n, ClassComment)
			return
		case "CharData":
			return // plain text keeps the default color
		case "AttValue", "PseudoAttValue", "SystemLiteral", "PubidLiteral":
			w.add(n, ClassString)
			return
		case "PI", "XMLDecl", "CDSect", "EntityRef", "CharRef":
			w.add(n, ClassKeyword)
			return
		case "Attribute", "PseudoAtt":
			// Color the name (ClassType) directly and recurse for the value,
			// so AttValue gets its string class without re-coloring the name.
			w.children(n, func(c *tree_sitter.Node) {
				if c.Kind() == "Name" {
					w.add(c, ClassType)
				} else {
					walk(c)
				}
			})
			return
		case "STag", "ETag", "EmptyElemTag":
			// Color the element name; attributes recurse for their own rules.
			w.children(n, func(c *tree_sitter.Node) {
				if c.Kind() == "Name" {
					w.add(c, ClassKeyword)
				} else {
					walk(c)
				}
			})
			return
		}
		w.children(n, walk)
	}
	walk(root)
}

var _ RangeHighlighter = (*tsHighlighter)(nil)
