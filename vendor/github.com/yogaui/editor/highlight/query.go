//go:build !js

package highlight

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unsafe"

	tree_sitter_proto "github.com/coder3101/tree-sitter-proto/bindings/go"
	tree_sitter_lua "github.com/tree-sitter-grammars/tree-sitter-lua/bindings/go"
	tree_sitter_make "github.com/tree-sitter-grammars/tree-sitter-make/bindings/go"
	tree_sitter_toml "github.com/tree-sitter-grammars/tree-sitter-toml/bindings/go"
	tree_sitter_zig "github.com/tree-sitter-grammars/tree-sitter-zig/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_csharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	tree_sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tree_sitter_cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_php "github.com/tree-sitter/tree-sitter-php/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_ruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// Highlight queries are the grammars' own queries/highlights.scm, copied
// from the module versions in go.mod; each file names its source.
//
//go:embed queries/*.scm
var queryFS embed.FS

func builtinQuery(names ...string) string {
	var b strings.Builder
	for _, n := range names {
		data, err := queryFS.ReadFile("queries/" + n + ".scm")
		if err != nil {
			panic(err) // a missing embedded query is a build mistake
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String()
}

// captureClasses maps highlight capture names to classes. A capture takes
// the class of its longest listed prefix ("function.method.call" ->
// "function.method"); unlisted names, such as @spell or @embedded, are not
// colored.
var captureClasses = map[string]ColorClass{
	"keyword": ClassKeyword, "conditional": ClassKeyword, "repeat": ClassKeyword,
	"include": ClassKeyword, "exception": ClassKeyword, "storageclass": ClassKeyword,
	"string": ClassString, "character": ClassString, "escape": ClassConstant,
	"string.escape": ClassConstant, "string.regexp": ClassString,
	"comment": ClassComment,
	"number":  ClassNumber, "float": ClassNumber,
	"boolean": ClassConstant, "constant": ClassConstant, "constant.builtin": ClassConstant,
	"type": ClassType, "constructor": ClassType,
	"function": ClassFunction, "function.method": ClassMethod, "method": ClassMethod,
	"function.builtin": ClassBuiltin, "function.macro": ClassFunction,
	"variable": ClassVariable, "variable.parameter": ClassParameter, "parameter": ClassParameter,
	"variable.builtin": ClassBuiltin, "variable.member": ClassProperty,
	"property": ClassProperty, "field": ClassProperty,
	"operator": ClassOperator, "punctuation": ClassPunctuation,
	"tag": ClassTag, "attribute": ClassAttribute, "label": ClassProperty,
	"module": ClassNamespace, "namespace": ClassNamespace,
}

// captureClass resolves a capture name; ok is false for names not colored.
func captureClass(name string) (ColorClass, bool) {
	for {
		if c, ok := captureClasses[name]; ok {
			return c, true
		}
		i := strings.LastIndexByte(name, '.')
		if i < 0 {
			return 0, false
		}
		name = name[:i]
	}
}

// compiledQuery is a query with its capture classes, shared by every
// highlighter of a language: a Query is read-only once built.
type compiledQuery struct {
	q       *tree_sitter.Query
	classes []ColorClass
	colored []bool
}

func compileQuery(langFn func() unsafe.Pointer, src string) (*compiledQuery, error) {
	q, qerr := tree_sitter.NewQuery(tree_sitter.NewLanguage(langFn()), src)
	if qerr != nil {
		return nil, fmt.Errorf("highlight query: %s (row %d, column %d)", qerr.Message, qerr.Row+1, qerr.Column+1)
	}
	names := q.CaptureNames()
	cq := &compiledQuery{q: q, classes: make([]ColorClass, len(names)), colored: make([]bool, len(names))}
	for i, n := range names {
		cq.classes[i], cq.colored[i] = captureClass(n)
	}
	return cq, nil
}

// span is one capture: a byte range, its class, and the pattern that matched.
type span struct {
	start, end int
	pattern    uint
	class      ColorClass
}

// classify colors [w.lo, w.hi) with the query's captures. Where captures
// overlap, the inner one wins, and for the same range the later pattern
// does: the grammars' queries list general patterns ((identifier) @variable)
// before the specific ones that refine them.
func (cq *compiledQuery) classify(w *rangeWalker, root *tree_sitter.Node) {
	lo, hi := w.lo, min(w.hi, len(w.src))
	if hi <= lo {
		return
	}
	qc := tree_sitter.NewQueryCursor()
	defer qc.Close()
	qc.SetByteRange(uint(lo), uint(hi))
	caps := qc.Captures(cq.q, root, w.src)
	var spans []span
	for m, ci := caps.Next(); m != nil; m, ci = caps.Next() {
		c := m.Captures[ci]
		if !cq.colored[c.Index] {
			continue
		}
		s, e := max(int(c.Node.StartByte()), lo), min(int(c.Node.EndByte()), hi)
		if e > s {
			spans = append(spans, span{s, e, m.PatternIndex, cq.classes[c.Index]})
		}
	}
	w.toks = append(w.toks, flatten(spans, lo, hi)...)
}

// flatten paints spans into non-overlapping tokens: longer spans first so
// nested ones land on top, and for equal spans the latest pattern last.
func flatten(spans []span, lo, hi int) []Token {
	if len(spans) == 0 {
		return nil
	}
	sort.SliceStable(spans, func(i, j int) bool {
		li, lj := spans[i].end-spans[i].start, spans[j].end-spans[j].start
		if li != lj {
			return li > lj
		}
		return spans[i].pattern < spans[j].pattern
	})
	const none = 0xff
	paint := make([]ColorClass, hi-lo)
	for i := range paint {
		paint[i] = none
	}
	for _, s := range spans {
		for i := s.start; i < s.end; i++ {
			paint[i-lo] = s.class
		}
	}
	var toks []Token
	for i := 0; i < len(paint); {
		j := i + 1
		for j < len(paint) && paint[j] == paint[i] {
			j++
		}
		if paint[i] != none {
			toks = append(toks, Token{Start: lo + i, End: lo + j, Class: paint[i]})
		}
		i = j
	}
	return toks
}

// queryLang is a grammar highlighted by a query, compiled on first use.
type queryLang struct {
	langFn func() unsafe.Pointer
	src    func() string
	once   sync.Once
	cq     *compiledQuery
	err    error
}

func (l *queryLang) compiled() (*compiledQuery, error) {
	l.once.Do(func() { l.cq, l.err = compileQuery(l.langFn, l.src()) })
	return l.cq, l.err
}

// highlighter starts a worker that colors with the query, or with fallback
// when the query does not compile.
func (l *queryLang) highlighter(fallback classifyFunc) Highlighter {
	cq, err := l.compiled()
	if err != nil {
		if fallback == nil {
			return Noop{}
		}
		return newTS(l.langFn, fallback)
	}
	return startTS(l.langFn, cq.classify, true)
}

// The languages colored by their highlight queries. TypeScript's query only
// adds to JavaScript's, which it is read after, as tree-sitter.json says;
// C++'s adds to C's the same way.
var (
	goQuery     = &queryLang{langFn: tree_sitter_go.Language, src: func() string { return builtinQuery("go") }}
	pythonQuery = &queryLang{langFn: tree_sitter_python.Language, src: func() string { return builtinQuery("python") }}
	rustQuery   = &queryLang{langFn: tree_sitter_rust.Language, src: func() string { return builtinQuery("rust") }}
	jsQuery     = &queryLang{langFn: tree_sitter_javascript.Language, src: func() string { return builtinQuery("javascript", "jsx") }}
	tsQuery     = &queryLang{langFn: tree_sitter_typescript.LanguageTypescript, src: func() string { return builtinQuery("javascript", "typescript") }}
	tsxQuery    = &queryLang{langFn: tree_sitter_typescript.LanguageTSX, src: func() string { return builtinQuery("javascript", "jsx", "typescript") }}
	javaQuery   = &queryLang{langFn: tree_sitter_java.Language, src: func() string { return builtinQuery("java") }}
	protoQuery  = &queryLang{langFn: tree_sitter_proto.Language, src: func() string { return builtinQuery("proto") }}
	makeQuery   = &queryLang{langFn: tree_sitter_make.Language, src: func() string { return builtinQuery("make") }}
	tomlQuery   = &queryLang{langFn: tree_sitter_toml.Language, src: func() string { return builtinQuery("toml") }}
	cQuery      = &queryLang{langFn: tree_sitter_c.Language, src: func() string { return builtinQuery("c") }}
	cppQuery    = &queryLang{langFn: tree_sitter_cpp.Language, src: func() string { return builtinQuery("c", "cpp") }}
	csharpQuery = &queryLang{langFn: tree_sitter_csharp.Language, src: func() string { return builtinQuery("csharp") }}
	luaQuery    = &queryLang{langFn: tree_sitter_lua.Language, src: func() string { return builtinQuery("lua") }}
	rubyQuery   = &queryLang{langFn: tree_sitter_ruby.Language, src: func() string { return builtinQuery("ruby") }}
	phpQuery    = &queryLang{langFn: tree_sitter_php.LanguagePHP, src: func() string { return builtinQuery("php") }}
	zigQuery    = &queryLang{langFn: tree_sitter_zig.Language, src: func() string { return builtinQuery("zig") }}
)

// NewJava returns a Tree-sitter highlighter for Java.
func NewJava() Highlighter { return javaQuery.highlighter(nil) }

// NewProto returns a Tree-sitter highlighter for Protocol Buffers.
func NewProto() Highlighter { return protoQuery.highlighter(nil) }

// NewMake returns a Tree-sitter highlighter for Makefiles.
func NewMake() Highlighter { return makeQuery.highlighter(nil) }

// NewTOML returns a Tree-sitter highlighter for TOML.
func NewTOML() Highlighter { return tomlQuery.highlighter(nil) }

// NewC returns a Tree-sitter highlighter for C.
func NewC() Highlighter { return cQuery.highlighter(nil) }

// NewCPP returns a Tree-sitter highlighter for C++.
func NewCPP() Highlighter { return cppQuery.highlighter(nil) }

// NewCSharp returns a Tree-sitter highlighter for C#.
func NewCSharp() Highlighter { return csharpQuery.highlighter(nil) }

// NewLua returns a Tree-sitter highlighter for Lua.
func NewLua() Highlighter { return luaQuery.highlighter(nil) }

// NewRuby returns a Tree-sitter highlighter for Ruby.
func NewRuby() Highlighter { return rubyQuery.highlighter(nil) }

// NewPHP returns a Tree-sitter highlighter for PHP files, which start in
// HTML text until an opening <?php tag.
func NewPHP() Highlighter { return phpQuery.highlighter(nil) }

// NewZig returns a Tree-sitter highlighter for Zig.
func NewZig() Highlighter { return zigQuery.highlighter(nil) }

// grammarQueries are the compiled-in grammars a query can color, with their
// built-in queries.
var grammarQueries = map[string]*queryLang{
	"go": goQuery, "python": pythonQuery, "rust": rustQuery, "javascript": jsQuery,
	"typescript": tsQuery, "tsx": tsxQuery, "java": javaQuery, "proto": protoQuery,
	"make": makeQuery, "toml": tomlQuery, "c": cQuery, "cpp": cppQuery, "csharp": csharpQuery,
	"lua": luaQuery, "ruby": rubyQuery, "php": phpQuery, "zig": zigQuery,
}

// Grammars lists the grammars QueryFactory accepts, sorted.
func Grammars() []string {
	out := make([]string, 0, len(grammarQueries))
	for n := range grammarQueries {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// BuiltinQuery returns the highlight query yoga uses for grammar, or "".
// Apps can extend it: patterns appended after it win over its own.
func BuiltinQuery(grammar string) string {
	l, ok := grammarQueries[strings.ToLower(grammar)]
	if !ok {
		return ""
	}
	return l.src()
}

// QueryFactory returns highlighters that color grammar with a highlight
// query, in tree-sitter's highlights.scm syntax. Capture names such as
// @function, @variable.parameter or @keyword pick the colors; a later
// pattern wins over an earlier one on the same node. The query is compiled
// once, here, so a mistake in it is reported before any file opens.
func QueryFactory(grammar, query string) (Factory, error) {
	l, ok := grammarQueries[strings.ToLower(grammar)]
	if !ok {
		return nil, fmt.Errorf("no grammar %q; known: %s", grammar, strings.Join(Grammars(), ", "))
	}
	cq, err := compileQuery(l.langFn, query)
	if err != nil {
		return nil, err
	}
	return func() Highlighter { return startTS(l.langFn, cq.classify, true) }, nil
}
