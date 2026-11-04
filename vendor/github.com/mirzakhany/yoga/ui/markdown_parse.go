package ui

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// mdKind is the kind of a Markdown block.
type mdKind uint8

const (
	mdParagraph mdKind = iota
	mdHeading
	mdCode
	mdQuote
	mdList
	mdItem
	mdRule
	mdTable
	mdHTML
)

// mdBlock is one block of parsed Markdown. The tree is built once per source
// and rendered every frame, so it holds plain data only.
type mdBlock struct {
	kind  mdKind
	level int      // heading level 1–6
	text  richText // inline content of paragraphs and headings

	code, lang string // fenced or indented code; HTML blocks use code too

	ordered bool
	start   int  // first number of an ordered list
	tight   bool // list items have no blank lines between them
	task    int  // list item checkbox: 0 none, 1 open, 2 checked

	children []*mdBlock

	// Tables: rows[0] is the header row.
	rows  [][]richText
	align []east.Alignment
}

// mdStyle is the inline style of a run of text.
type mdStyle struct {
	bold, italic, code, strike bool
	link                       string
}

// mdRun styles text[lo:hi] of a richText.
type mdRun struct {
	lo, hi int
	style  mdStyle
}

// richText is inline text with styled runs over one plain string, so
// selection and copy work on the text a reader sees.
type richText struct {
	text string
	runs []mdRun
}

func (r *richText) add(s string, st mdStyle) {
	if s == "" {
		return
	}
	lo := len(r.text)
	r.text += s
	if n := len(r.runs); n > 0 && r.runs[n-1].style == st && r.runs[n-1].hi == lo {
		r.runs[n-1].hi = len(r.text)
		return
	}
	r.runs = append(r.runs, mdRun{lo: lo, hi: len(r.text), style: st})
}

var mdParser = goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser()

// parseMarkdown parses CommonMark with the GitHub extensions (tables,
// strikethrough, task lists, autolinks) into blocks.
func parseMarkdown(src string) []*mdBlock {
	b := []byte(src)
	doc := mdParser.Parse(text.NewReader(b))
	return mdBlocks(doc, b)
}

func mdBlocks(parent ast.Node, src []byte) []*mdBlock {
	var out []*mdBlock
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		if blk := mdBlockOf(n, src); blk != nil {
			out = append(out, blk)
		}
	}
	return out
}

func mdBlockOf(n ast.Node, src []byte) *mdBlock {
	switch n := n.(type) {
	case *ast.Heading:
		return &mdBlock{kind: mdHeading, level: n.Level, text: mdInline(n, src)}
	case *ast.Paragraph, *ast.TextBlock:
		return &mdBlock{kind: mdParagraph, text: mdInline(n, src)}
	case *ast.FencedCodeBlock:
		return &mdBlock{kind: mdCode, code: mdLines(n, src), lang: string(n.Language(src))}
	case *ast.CodeBlock:
		return &mdBlock{kind: mdCode, code: mdLines(n, src)}
	case *ast.HTMLBlock:
		s := mdLines(n, src)
		if n.HasClosure() {
			s += "\n" + strings.TrimRight(string(n.ClosureLine.Value(src)), "\n")
		}
		return &mdBlock{kind: mdHTML, code: strings.TrimSpace(s)}
	case *ast.Blockquote:
		return &mdBlock{kind: mdQuote, children: mdBlocks(n, src)}
	case *ast.List:
		blk := &mdBlock{kind: mdList, ordered: n.IsOrdered(), start: n.Start, tight: n.IsTight}
		for it := n.FirstChild(); it != nil; it = it.NextSibling() {
			item := &mdBlock{kind: mdItem, children: mdBlocks(it, src)}
			// A task item starts with a checkbox inside its first paragraph.
			if len(item.children) > 0 {
				if p := it.FirstChild(); p != nil {
					if cb, ok := p.FirstChild().(*east.TaskCheckBox); ok {
						item.task = 1
						if cb.IsChecked {
							item.task = 2
						}
						t := &item.children[0].text
						trimmed := strings.TrimLeft(t.text, " ")
						cut := len(t.text) - len(trimmed)
						t.text = trimmed
						for i := range t.runs {
							t.runs[i].lo = max(0, t.runs[i].lo-cut)
							t.runs[i].hi = max(0, t.runs[i].hi-cut)
						}
					}
				}
			}
			blk.children = append(blk.children, item)
		}
		return blk
	case *ast.ThematicBreak:
		return &mdBlock{kind: mdRule}
	case *east.Table:
		blk := &mdBlock{kind: mdTable, align: n.Alignments}
		for row := n.FirstChild(); row != nil; row = row.NextSibling() {
			var cells []richText
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				cells = append(cells, mdInline(cell, src))
			}
			blk.rows = append(blk.rows, cells)
		}
		return blk
	}
	return nil
}

// mdLines joins a block's source lines, without the final newline.
func mdLines(n ast.Node, src []byte) string {
	var sb strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		sb.Write(seg.Value(src))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// mdInline flattens a block's inline children into rich text.
func mdInline(n ast.Node, src []byte) richText {
	var r richText
	mdWalkInline(&r, n, src, mdStyle{})
	return r
}

func mdWalkInline(r *richText, parent ast.Node, src []byte, st mdStyle) {
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		switch n := n.(type) {
		case *ast.Text:
			r.add(string(n.Segment.Value(src)), st)
			switch {
			case n.HardLineBreak():
				r.add("\n", st)
			case n.SoftLineBreak():
				r.add(" ", st)
			}
		case *ast.String:
			r.add(string(n.Value), st)
		case *ast.CodeSpan:
			s := st
			s.code = true
			var sb strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					sb.Write(t.Segment.Value(src))
				}
			}
			r.add(sb.String(), s)
		case *ast.Emphasis:
			s := st
			if n.Level >= 2 {
				s.bold = true
			} else {
				s.italic = true
			}
			mdWalkInline(r, n, src, s)
		case *east.Strikethrough:
			s := st
			s.strike = true
			mdWalkInline(r, n, src, s)
		case *ast.Link:
			s := st
			s.link = string(n.Destination)
			mdWalkInline(r, n, src, s)
		case *ast.AutoLink:
			s := st
			s.link = string(n.URL(src))
			if n.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(s.link, "mailto:") {
				s.link = "mailto:" + s.link
			}
			r.add(string(n.Label(src)), s)
		case *ast.Image:
			// Images are shown by their alt text; a link opens the source.
			s := st
			s.link = string(n.Destination)
			before := len(r.text)
			mdWalkInline(r, n, src, s)
			if len(r.text) == before {
				r.add("image", s)
			}
		case *ast.RawHTML:
			segs := n.Segments
			for i := 0; i < segs.Len(); i++ {
				seg := segs.At(i)
				r.add(string(seg.Value(src)), st)
			}
		case *east.TaskCheckBox:
			// Rendered by the list item.
		default:
			mdWalkInline(r, n, src, st)
		}
	}
}
