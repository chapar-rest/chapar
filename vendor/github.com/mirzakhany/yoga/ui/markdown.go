package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
	east "github.com/yuin/goldmark/extension/ast"
)

// markdownView renders Markdown source. See Markdown.
type markdownView struct {
	id     string
	src    string
	onLink func(url string)
}

// markdownState keeps the parse of the last source and the highlighters of
// its code blocks, so a steady frame does no parsing.
type markdownState struct {
	src    string
	parsed bool
	blocks []*mdBlock
	code   map[string]*mdCodeState
	gen    int

	// sel is one selection over the whole view: its text is every block's
	// text joined in reading order, and spans place each block in it.
	sel   paragraphSelection
	spans []*mdSpan
}

// mdSpan is one block of text in the view's selection: where its text
// starts in the joined text, and where it was laid out, for hit testing.
type mdSpan struct {
	base  int
	v     *richView
	st    *richState
	el    *layout.Element
	lineH float32
}

// hit maps a point to an offset in the joined text: in the block under it,
// the nearest block on its row, or the start of the next block below.
func (s *markdownState) hit(eng *shape.Engine, x, y float32) int {
	var row *mdSpan
	var rowDist float32
	for _, sp := range s.spans {
		if sp.el == nil {
			continue
		}
		f := sp.el.Frame
		if y < f.Y || y >= f.Y+f.H {
			continue
		}
		d := max(f.X-x, x-(f.X+f.W), 0)
		if row == nil || d < rowDist {
			row, rowDist = sp, d
		}
	}
	if row != nil {
		f := row.el.Frame
		return row.base + row.v.offsetAt(eng, row.st, x-f.X, y-f.Y, row.lineH)
	}
	for _, sp := range s.spans {
		if sp.el != nil && sp.el.Frame.Y > y {
			return sp.base
		}
	}
	return len(s.sel.text)
}

// mdCodeState highlights one code block. Its key is the block's position,
// so a block that keeps its place while a response streams in keeps its
// highlighter and only reparses.
type mdCodeState struct {
	lang    string
	src     string
	hl      highlight.Highlighter
	tokens  []highlight.Token
	pending bool
	gen     int
}

// release stops the code highlighters when the view leaves the tree.
func (s *markdownState) release() {
	for k, cs := range s.code {
		if cs.hl != nil {
			cs.hl.Close()
		}
		delete(s.code, k)
	}
}

// Markdown renders CommonMark source with the GitHub extensions: headings,
// paragraphs with bold, italic, strikethrough, inline code and links, block
// quotes, ordered, bullet and task lists, fenced code with syntax colors
// and a copy button, tables and rules. Images show as links with their alt
// text.
//
// The source is parsed again only when it changes, so it suits both a file
// preview and a reply that streams in token by token. id keys the parse and
// its selection; give every Markdown in a window its own. Its text can be
// selected and copied as one document, across paragraphs, lists, tables and
// code.
//
// It fills the width it is given and is as tall as its content; put it in a
// Scroll for long documents. Links do nothing until OnLink is set.
func Markdown(id, src string) *Node {
	return &Node{kind: kindWrap, inner: &markdownView{id: id, src: src}}
}

// OnLink sets the handler for a click on a link in a Markdown view, called
// with the link's destination as written.
func (n *Node) OnLink(fn func(url string)) *Node {
	if v, ok := n.inner.(*markdownView); ok {
		v.onLink = fn
	}
	return n
}

func (v *markdownView) Layout(c *Ctx) *layout.Element {
	st := c.Widget(v.id, func() any { return &markdownState{code: map[string]*mdCodeState{}} }).(*markdownState)
	if !st.parsed || st.src != v.src {
		st.src, st.parsed = v.src, true
		st.blocks = parseMarkdown(v.src)
	}
	st.gen++
	r := &mdRender{c: c, st: st, id: v.id, onLink: v.onLink, sep: "\n\n"}
	th := c.Theme()
	view := Column(r.blocks(st.blocks, "", false, 0)...).Gap(th.Spacing.M)
	doc := r.doc.String()
	sel := &st.sel
	if strings.HasPrefix(doc, sel.text) {
		sel.text = doc // a streaming reply grows; keep what is selected
	} else {
		sel.setText(doc)
	}
	st.spans = r.spans
	// Drop highlighters of code blocks that are gone.
	for k, cs := range st.code {
		if cs.gen != st.gen {
			if cs.hl != nil {
				cs.hl.Close()
			}
			delete(st.code, k)
		}
	}
	el := view.Layout(c)
	sel.host = el
	sel.hit = st.hit
	c.Focus().Add(sel)
	sel.menu.layout(c)
	return el
}

// mdRender turns parsed blocks into views for one frame.
type mdRender struct {
	c      *Ctx
	st     *markdownState
	id     string
	onLink func(string)

	// doc collects the text of every block in reading order, each after
	// sep, for the view's one selection.
	doc   strings.Builder
	sep   string
	spans []*mdSpan
}

// add places a block's text in the joined text after the pending separator
// and ties the block to the view's selection.
func (r *mdRender) add(v *richView) View {
	sep := r.sep
	if r.doc.Len() == 0 {
		sep = strings.TrimLeft(sep, "\n")
	}
	r.doc.WriteString(sep)
	sp := &mdSpan{base: r.doc.Len(), v: v}
	r.doc.WriteString(v.rt.text)
	r.spans = append(r.spans, sp)
	r.sep = "\n\n"
	v.sel, v.span = &r.st.sel, sp
	return ViewOf(v)
}

func (r *mdRender) blocks(bs []*mdBlock, path string, muted bool, depth int) []View {
	out := make([]View, 0, len(bs))
	for i, b := range bs {
		out = append(out, r.block(b, path+"."+strconv.Itoa(i), muted, depth))
	}
	return out
}

func (r *mdRender) text(path string, rt *richText, size float32, weight int, muted bool) View {
	return r.add(&richView{id: r.id + path, rt: rt, size: size, weight: weight, muted: muted, onLink: r.onLink})
}

func (r *mdRender) block(b *mdBlock, path string, muted bool, depth int) View {
	th := r.c.Theme()
	body := th.Typography.Body
	switch b.kind {
	case mdHeading:
		size := body.Size
		switch b.level {
		case 1:
			size = th.Typography.Title.Size * 1.4
		case 2:
			size = th.Typography.Title.Size * 1.15
		case 3:
			size = th.Typography.Title.Size
		case 4:
			size = th.Typography.Subtitle.Size
		}
		h := r.text(path, &b.text, size, 600, muted)
		if b.level > 2 {
			return h
		}
		return Column(h, HLine(th.Stroke.Thin, th.Border)).Gap(th.Spacing.XS)
	case mdParagraph:
		return r.text(path, &b.text, body.Size, body.Weight, muted)
	case mdCode:
		return r.code(b, path)
	case mdHTML:
		rt := &richText{text: b.code, runs: []mdRun{{hi: len(b.code)}}}
		return r.add(&richView{id: r.id + path, rt: rt, size: body.Size, weight: body.Weight, mono: true, pre: true, muted: true})
	case mdQuote:
		return Row(
			VLine(3, th.BorderStrong).Radius(th.Radius.Small),
			Column(r.blocks(b.children, path, true, depth)...).Gap(th.Spacing.S).Grow(1),
		).Align(AlignStretch).Gap(th.Spacing.M)
	case mdList:
		return r.list(b, path, muted, depth)
	case mdRule:
		return HLine(th.Stroke.Thin, th.Border)
	case mdTable:
		return r.table(b, path, muted)
	}
	return nil
}

func (r *mdRender) list(b *mdBlock, path string, muted bool, depth int) View {
	th := r.c.Theme()
	body := th.Typography.Body
	gap := th.Spacing.XS
	if !b.tight {
		gap = th.Spacing.S
	}
	// Markers share one width so item text lines up.
	markerW := body.Size * 1.2
	if b.ordered {
		last := strconv.Itoa(b.start+len(b.children)-1) + "."
		if eng := r.c.Text(); eng != nil {
			w, _ := eng.MeasureAt(last, body.Size)
			markerW = max(markerW, w)
		}
	}
	fg := th.Foreground
	if muted {
		fg = th.ForegroundMuted
	}
	top := mdLeading(body.Size, false) / 2
	items := make([]View, 0, len(b.children))
	indent := strings.Repeat("  ", depth)
	for i, it := range b.children {
		// Copied text keeps a plain-text marker for each item.
		sep := "\n"
		if i == 0 && depth == 0 {
			sep = "\n\n"
		}
		var marker *Node
		switch {
		case it.task != 0:
			icon := icons.Square
			mark := "- [ ] "
			if it.task == 2 {
				icon = icons.SquareCheck
				mark = "- [x] "
			}
			sep += indent + mark
			marker = Icon(icon, body.Size, fg).PaddingTop(top + 2)
		case b.ordered:
			marker = Text(fmt.Sprintf("%d.", b.start+i)).Style(Spec{}.TextColor(TokenForegroundMuted)).PaddingTop(top)
			sep += fmt.Sprintf("%s%d. ", indent, b.start+i)
		default:
			sep += indent + "- "
			bullet := "•"
			if depth%2 == 1 {
				bullet = "◦"
			}
			marker = Text(bullet).Style(Spec{}.TextColor(TokenForegroundMuted)).PaddingTop(top)
		}
		ipath := path + "." + strconv.Itoa(i)
		r.sep = sep
		items = append(items, Row(
			Row(marker).Width(markerW).Justify(JustifyEnd).Align(AlignStart),
			Column(r.blocks(it.children, ipath, muted, depth+1)...).Gap(gap).Grow(1),
		).Align(AlignStart).Gap(th.Spacing.S))
	}
	return Column(items...).Gap(gap)
}

func (r *mdRender) table(b *mdBlock, path string, muted bool) View {
	th := r.c.Theme()
	body := th.Typography.Body
	cols := 0
	for _, row := range b.rows {
		cols = max(cols, len(row))
	}
	if cols == 0 {
		return nil
	}
	var cells []View
	for i, row := range b.rows {
		for j := 0; j < cols; j++ {
			var cell View
			if j < len(row) {
				weight := body.Weight
				if i == 0 {
					weight = 600
				}
				align := AlignStart
				if j < len(b.align) {
					switch b.align[j] {
					case east.AlignRight:
						align = AlignEnd
					case east.AlignCenter:
						align = AlignCenter
					}
				}
				switch {
				case i == 0 && j == 0:
					r.sep = "\n\n"
				case j == 0:
					r.sep = "\n"
				default:
					r.sep = "\t"
				}
				cell = r.add(&richView{id: r.id + fmt.Sprintf("%s.%d.%d", path, i, j), rt: &row[j], size: body.Size,
					weight: weight, muted: muted, align: align, onLink: r.onLink})
			}
			n := Column(cell).PaddingXY(th.Spacing.S, th.Spacing.XS).BorderBottom(TokenBorder, th.Stroke.Thin)
			if i == 0 {
				n = n.Background(TokenChromeMuted)
			}
			cells = append(cells, n)
		}
	}
	return Grid(cols, cells...).Border(TokenBorder, th.Stroke.Thin).Radius(th.Radius.Small)
}

// mdLangAliases maps fence names to the highlighter names they mean.
var mdLangAliases = map[string]string{
	"golang": "go", "js": "javascript", "ts": "typescript", "py": "python",
	"sh": "bash", "shell": "bash", "zsh": "bash", "console": "bash",
	"yml": "yaml", "rs": "rust", "jsonc": "json",
}

// mdHighlighter returns a highlighter for a fence language, or nil.
func mdHighlighter(lang string) highlight.Highlighter {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return nil
	}
	if a, ok := mdLangAliases[lang]; ok {
		lang = a
	}
	if f, ok := highlight.ByName(lang); ok {
		return f()
	}
	// An extension an app registered, such as "toml" for ".toml".
	if h := highlight.ForPath("x." + lang); h != nil {
		if _, noop := h.(highlight.Noop); !noop {
			return h
		}
	}
	return nil
}

func (r *mdRender) code(b *mdBlock, path string) View {
	c, st := r.c, r.st
	th := c.Theme()
	cs := st.code[path]
	if cs == nil || cs.lang != b.lang {
		if cs != nil && cs.hl != nil {
			cs.hl.Close()
		}
		cs = &mdCodeState{lang: b.lang, hl: mdHighlighter(b.lang)}
		st.code[path] = cs
	}
	cs.gen = st.gen
	src := strings.ReplaceAll(b.code, "\t", "    ")
	if cs.hl != nil {
		if src != cs.src {
			cs.src = src
			cs.hl.Update([]byte(src))
			cs.pending = true
		}
		if toks, ok := cs.hl.Poll(); ok {
			cs.tokens, cs.pending = toks, false
		}
		if cs.pending {
			c.Animate(30 * time.Millisecond)
		}
	}
	// Tokens from an older source may run past the text while a reply
	// streams in; they are dropped by the segment split.
	toks := cs.tokens
	for len(toks) > 0 && toks[len(toks)-1].End > len(src) {
		toks = toks[:len(toks)-1]
	}

	code := b.code
	label := b.lang
	if label == "" {
		label = "text"
	}
	rt := &richText{text: src, runs: []mdRun{{hi: len(src)}}}
	size := th.Typography.Body.Size - 1
	return Column(
		Row(
			Caption(label).Style(Spec{}.TextColor(TokenForegroundMuted)),
			Spacer(),
			IconButton(r.id+path+"/copy", icons.Copy).Tooltip("Copy").OnClick(func() {
				if clip := c.Clipboard(); clip != nil {
					clip.Set(code)
				}
			}),
		).PaddingLeft(th.Spacing.M).PaddingRight(th.Spacing.XXS).Height(28).
			Background(TokenChromeMuted).RadiusTopLeft(th.Radius.Medium).RadiusTopRight(th.Radius.Medium).BorderBottom(TokenBorder, th.Stroke.Thin),
		Column(r.add(&richView{id: r.id + path, rt: rt, size: size, weight: th.Typography.Body.Weight,
			mono: true, pre: true, tokens: toks})).Padding(th.Spacing.M),
	).Background(TokenChrome).Border(TokenBorder, th.Stroke.Thin).Radius(th.Radius.Medium)
}
