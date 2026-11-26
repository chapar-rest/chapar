//go:build !js

package highlight

import (
	tree_sitter_yaml "github.com/tree-sitter-grammars/tree-sitter-yaml/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_bash "github.com/tree-sitter/tree-sitter-bash/bindings/go"
	tree_sitter_css "github.com/tree-sitter/tree-sitter-css/bindings/go"
	tree_sitter_html "github.com/tree-sitter/tree-sitter-html/bindings/go"
)

// langSpec describes how to color a grammar's nodes, so a language is a table
// rather than a hand-written walk.
type langSpec struct {
	// kinds colors a named node, and everything under it, with one class.
	kinds map[string]ColorClass
	// interp lists the child kinds inside a colored string (template
	// substitutions, shell expansions) that keep their own colors.
	interp map[string]bool
	// fields colors one field child of a parent kind; the parent's other
	// children are walked as usual.
	fields []fieldClass
	// atKeywords also treats anonymous tokens starting with '@' (CSS
	// at-rules) as keywords.
	atKeywords bool
}

type fieldClass struct {
	parent, field string
	class         ColorClass
}

// specClassifier returns a classifier for spec. Anonymous leaf tokens made only
// of letters and underscores are the grammar's keywords; punctuation is not.
func specClassifier(spec langSpec) classifyFunc {
	fields := map[string][]fieldClass{}
	for _, f := range spec.fields {
		fields[f.parent] = append(fields[f.parent], f)
	}
	return func(w *rangeWalker, root *tree_sitter.Node) {
		var walk func(n *tree_sitter.Node)
		walk = func(n *tree_sitter.Node) {
			kind := n.Kind()
			if c, ok := spec.kinds[kind]; ok && n.IsNamed() {
				if len(spec.interp) == 0 || !hasChildKind(n, spec.interp) {
					w.add(n, c)
					return
				}
				// Color the string around its interpolations, which recurse.
				pos := int(n.StartByte())
				w.children(n, func(ch *tree_sitter.Node) {
					if spec.interp[ch.Kind()] {
						w.addSpan(pos, int(ch.StartByte()), c)
						walk(ch)
						pos = int(ch.EndByte())
					}
				})
				w.addSpan(pos, int(n.EndByte()), c)
				return
			}
			if fs := fields[kind]; len(fs) > 0 {
				colored := map[uintptr]bool{}
				for _, f := range fs {
					if ch := n.ChildByFieldName(f.field); ch != nil {
						w.add(ch, f.class)
						colored[ch.Id()] = true
					}
				}
				w.children(n, func(ch *tree_sitter.Node) {
					if !colored[ch.Id()] {
						walk(ch)
					}
				})
				return
			}
			if n.ChildCount() == 0 {
				if !n.IsNamed() && isKeywordToken(kind, spec.atKeywords) {
					w.add(n, ClassKeyword)
				}
				return
			}
			w.children(n, walk)
		}
		walk(root)
	}
}

func isKeywordToken(kind string, at bool) bool {
	if len(kind) < 2 {
		return false
	}
	start := 0
	if at && kind[0] == '@' {
		start = 1
	}
	for i := start; i < len(kind); i++ {
		c := kind[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '_' {
			return false
		}
	}
	return true
}

func hasChildKind(n *tree_sitter.Node, kinds map[string]bool) bool {
	for i := uint(0); i < n.ChildCount(); i++ {
		if c := n.Child(i); c != nil && kinds[c.Kind()] {
			return true
		}
	}
	return false
}

// addSpan records a token for [start, end) clipped to the walker's range.
func (w *rangeWalker) addSpan(start, end int, c ColorClass) {
	start, end = max(start, w.lo), min(end, w.hi)
	if end > start {
		w.toks = append(w.toks, Token{Start: start, End: end, Class: c})
	}
}

var jsSpec = langSpec{
	kinds: map[string]ColorClass{
		"comment": ClassComment, "hash_bang_line": ClassComment,
		"string": ClassString, "template_string": ClassString, "regex": ClassString,
		"number":          ClassNumber,
		"type_identifier": ClassType, "predefined_type": ClassType,
		"true": ClassKeyword, "false": ClassKeyword, "null": ClassKeyword,
		"undefined": ClassKeyword, "this": ClassKeyword, "super": ClassKeyword,
	},
	interp: map[string]bool{"template_substitution": true},
	fields: []fieldClass{
		{"class_declaration", "name", ClassType},
		{"class", "name", ClassType},
		{"jsx_opening_element", "name", ClassKeyword},
		{"jsx_closing_element", "name", ClassKeyword},
		{"jsx_self_closing_element", "name", ClassKeyword},
	},
}

var (
	classifyJS   = specClassifier(jsSpec)
	classifyRust = specClassifier(langSpec{
		kinds: map[string]ColorClass{
			"line_comment": ClassComment, "block_comment": ClassComment,
			"string_literal": ClassString, "raw_string_literal": ClassString, "char_literal": ClassString,
			"integer_literal": ClassNumber, "float_literal": ClassNumber,
			"type_identifier": ClassType, "primitive_type": ClassType, "lifetime": ClassType,
			"boolean_literal": ClassKeyword, "self": ClassKeyword, "mutable_specifier": ClassKeyword,
			"crate": ClassKeyword, "super": ClassKeyword,
		},
		fields: []fieldClass{{"macro_invocation", "macro", ClassKeyword}},
	})
	classifyBash = specClassifier(langSpec{
		kinds: map[string]ColorClass{
			"comment": ClassComment,
			"string":  ClassString, "raw_string": ClassString, "ansi_c_string": ClassString, "heredoc_body": ClassString,
			"number":        ClassNumber,
			"variable_name": ClassType, "simple_expansion": ClassType, "expansion": ClassType,
		},
		interp: map[string]bool{"simple_expansion": true, "expansion": true, "command_substitution": true},
		fields: []fieldClass{{"function_definition", "name", ClassType}},
	})
	classifyCSS = specClassifier(langSpec{
		kinds: map[string]ColorClass{
			"comment": ClassComment, "js_comment": ClassComment,
			"string_value":  ClassString,
			"integer_value": ClassNumber, "float_value": ClassNumber, "color_value": ClassNumber,
			"property_name": ClassType, "class_name": ClassType, "id_name": ClassType,
			"tag_name": ClassKeyword, "important": ClassKeyword, "at_keyword": ClassKeyword,
		},
		atKeywords: true,
	})
	classifyHTML = specClassifier(langSpec{
		kinds: map[string]ColorClass{
			"comment":  ClassComment,
			"tag_name": ClassKeyword, "doctype": ClassKeyword, "erroneous_end_tag_name": ClassKeyword,
			"attribute_name":  ClassType,
			"attribute_value": ClassString, "quoted_attribute_value": ClassString,
		},
	})
	classifyYAML = specClassifier(langSpec{
		kinds: map[string]ColorClass{
			"comment":             ClassComment,
			"double_quote_scalar": ClassString, "single_quote_scalar": ClassString, "block_scalar": ClassString,
			"integer_scalar": ClassNumber, "float_scalar": ClassNumber,
			"boolean_scalar": ClassKeyword, "null_scalar": ClassKeyword,
			"anchor": ClassType, "alias": ClassType, "tag": ClassType,
		},
		fields: []fieldClass{
			{"block_mapping_pair", "key", ClassType},
			{"flow_pair", "key", ClassType},
		},
	})
)

// NewTypeScript returns a Tree-sitter highlighter for TypeScript.
func NewTypeScript() Highlighter { return tsQuery.highlighter(classifyJS) }

// NewTSX returns a Tree-sitter highlighter for TypeScript with JSX.
func NewTSX() Highlighter { return tsxQuery.highlighter(classifyJS) }

// NewJavaScript returns a Tree-sitter highlighter for JavaScript and JSX.
func NewJavaScript() Highlighter { return jsQuery.highlighter(classifyJS) }

// NewRust returns a Tree-sitter highlighter for Rust.
func NewRust() Highlighter { return rustQuery.highlighter(classifyRust) }

// NewBash returns a Tree-sitter highlighter for shell scripts.
func NewBash() Highlighter { return newTS(tree_sitter_bash.Language, classifyBash) }

// NewCSS returns a Tree-sitter highlighter for CSS.
func NewCSS() Highlighter { return newTS(tree_sitter_css.Language, classifyCSS) }

// NewHTML returns a Tree-sitter highlighter for HTML.
func NewHTML() Highlighter { return newTS(tree_sitter_html.Language, classifyHTML) }

// NewYAML returns a Tree-sitter highlighter for YAML.
func NewYAML() Highlighter { return newTS(tree_sitter_yaml.Language, classifyYAML) }
