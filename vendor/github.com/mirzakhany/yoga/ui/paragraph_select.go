package ui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// Selectable lets the user select a Paragraph's text with the mouse and copy
// it: drag, double-click a word, triple-click a line, Cmd/Ctrl+A and
// Cmd/Ctrl+C once it is focused, or Copy and Select All from the right-click
// menu. id keeps the selection with this paragraph across frames.
//
// A selectable paragraph keeps the text as given when wrapping: runs of spaces
// and indentation are drawn, not collapsed, so a traceback lines up.
func (n *Node) Selectable(id string) *Node {
	if d, ok := n.extra.(*paragraphData); ok {
		d.selectable = true
		n.id = id
	}
	return n
}

// textSpan is one wrapped line: the byte range [lo, hi) of the text it shows.
type textSpan struct{ lo, hi int }

// wrapSpans wraps s at maxW like wrapTextWeight, but as byte ranges of s with
// its spacing intact. Newlines end a line and belong to no span; spaces where
// a line breaks stay at the end of the line they follow.
func wrapSpans(eng *shape.Engine, s string, size float32, weight int, maxW float32) []textSpan {
	var out []textSpan
	fits := func(t string) bool {
		w, _ := eng.MeasureAtWeight(strings.TrimRight(t, " \t"), size, weight)
		return w <= maxW
	}
	for start := 0; ; {
		end := len(s)
		if i := strings.IndexByte(s[start:], '\n'); i >= 0 {
			end = start + i
		}
		if eng == nil {
			out = append(out, textSpan{start, end})
		} else {
			out = wrapSpanLine(out, s, start, end, fits)
		}
		if end == len(s) {
			return out
		}
		start = end + 1
	}
}

// wrapSpanLine appends the wrapped lines of s[lo:hi], which has no newline.
func wrapSpanLine(out []textSpan, s string, lo, hi int, fits func(string) bool) []textSpan {
	line, pos := lo, lo
	for pos < hi {
		// The next token is a run of non-spaces and the spaces after it.
		word := pos
		for word < hi && s[word] != ' ' && s[word] != '\t' {
			_, sz := utf8.DecodeRuneInString(s[word:])
			word += sz
		}
		tok := word
		for tok < hi && (s[tok] == ' ' || s[tok] == '\t') {
			tok++
		}
		if fits(s[line:tok]) {
			pos = tok
			continue
		}
		if pos > line {
			out = append(out, textSpan{line, pos})
			line = pos
			continue
		}
		// One word wider than the line: split it between characters.
		cut := splitToFit(s[line:word], fits)
		out = append(out, textSpan{line, line + cut})
		line += cut
		pos = line
	}
	return append(out, textSpan{line, hi})
}

// paragraphSelection is the selection state of a selectable Paragraph. It is
// the Focusable the focus scope routes copy keys to.
type paragraphSelection struct {
	host    *layout.Element
	text    string
	focused bool
	anchor  int // selection is [min(anchor, caret), max(anchor, caret))
	caret   int
	drag    bool
	menu    editMenu

	clicks    int
	lastClick time.Time
	lastX     float32
	lastY     float32

	// Geometry of the last layout, for hit testing.
	spans      []textSpan
	lineX      func(i int) float32 // left edge of line i
	top, lineH float32
	size       float32
	weight     int
}

func (p *paragraphSelection) selRange() (int, int) {
	return min(p.anchor, p.caret), max(p.anchor, p.caret)
}

// setText resets the selection when the paragraph shows new text.
func (p *paragraphSelection) setText(s string) {
	if s != p.text {
		p.text = s
		p.anchor, p.caret, p.drag = 0, 0, false
	}
}

// offsetAt maps a point to the nearest byte offset in the text.
func (p *paragraphSelection) offsetAt(eng *shape.Engine, x, y float32) int {
	if len(p.spans) == 0 || p.lineH <= 0 {
		return 0
	}
	i := int((y - p.top) / p.lineH)
	if y < p.top {
		return 0
	}
	if i >= len(p.spans) {
		return len(p.text)
	}
	sp := p.spans[i]
	x0 := p.lineX(i)
	prev := float32(0)
	for off := sp.lo; off < sp.hi; {
		_, sz := utf8.DecodeRuneInString(p.text[off:])
		w, _ := eng.MeasureAtWeight(p.text[sp.lo:off+sz], p.size, p.weight)
		if x < x0+(prev+w)/2 {
			return off
		}
		prev = w
		off += sz
	}
	return sp.hi
}

// lineRange is the source line (between newlines) holding off.
func (p *paragraphSelection) lineRange(off int) (int, int) {
	lo := strings.LastIndexByte(p.text[:off], '\n') + 1
	hi := len(p.text)
	if i := strings.IndexByte(p.text[off:], '\n'); i >= 0 {
		hi = off + i
	}
	return lo, hi
}

func (p *paragraphSelection) onMouse(e *layout.Element, m *input.Mouse, eng *shape.Engine) {
	inside := e.Frame.Contains(m.X, m.Y)
	if inside {
		m.SetCursor(input.CursorText)
	}
	if m.RightPressed && inside {
		// Keep a selection the click lands in, so Copy acts on it.
		off := p.offsetAt(eng, m.X, m.Y)
		if lo, hi := p.selRange(); lo == hi || off < lo || off > hi {
			p.anchor, p.caret = off, off
		}
		items := editMenuItems(p, editMenuFlags{copyable: true})
		if p.menu.open(items, m.X, m.Y) {
			m.Consumed = true
		}
		return
	}
	if m.Pressed && inside {
		off := p.offsetAt(eng, m.X, m.Y)
		now := time.Now()
		if now.Sub(p.lastClick) < doubleClickInterval && absF(m.X-p.lastX) <= multiClickSlop && absF(m.Y-p.lastY) <= multiClickSlop {
			p.clicks = p.clicks%3 + 1
		} else {
			p.clicks = 1
		}
		p.lastClick, p.lastX, p.lastY = now, m.X, m.Y
		switch p.clicks {
		case 2:
			p.anchor, p.caret = wordRangeIn(p.text, off)
		case 3:
			p.anchor, p.caret = p.lineRange(off)
		default:
			p.anchor, p.caret = off, off
			p.drag = true
		}
		m.Consumed = true
	}
	// A drag keeps selecting past the paragraph's edges.
	if p.drag && m.Down {
		p.caret = p.offsetAt(eng, m.X, m.Y)
	}
	if !m.Down {
		p.drag = false
	}
}

// paintSelection fills the selected part of each visible line.
func (p *paragraphSelection) paintSelection(dl *render.DrawList, eng *shape.Engine, fr render.Rect, color render.Color) {
	lo, hi := p.selRange()
	if !p.focused || lo == hi {
		return
	}
	for i, sp := range p.spans {
		y := p.top + float32(i)*p.lineH
		if y >= fr.Y+fr.H {
			break
		}
		a, b := max(lo, sp.lo), min(hi, sp.hi)
		// A selection running on past this line also covers its line break.
		wraps := hi > sp.hi && sp.hi < len(p.text) && p.text[sp.hi] == '\n'
		if a > b || (a == b && !wraps) {
			continue
		}
		x0 := p.lineX(i)
		xa, _ := eng.MeasureAtWeight(p.text[sp.lo:a], p.size, p.weight)
		xb, _ := eng.MeasureAtWeight(p.text[sp.lo:b], p.size, p.weight)
		if wraps {
			sw, _ := eng.MeasureAtWeight(" ", p.size, p.weight)
			xb += sw
		}
		dl.AddRect(render.Rect{X: x0 + xa, Y: y, W: xb - xa, H: p.lineH}, color)
	}
}

// ── Focusable ────────────────────────────────────────────────────────────────

func (p *paragraphSelection) Focus()            { p.focused = true }
func (p *paragraphSelection) Focused() bool     { return p.focused }
func (p *paragraphSelection) HandleText([]rune) {}
func (p *paragraphSelection) CapturesTab() bool { return false }
func (p *paragraphSelection) FocusOnClick() bool {
	return true
}
func (p *paragraphSelection) FocusEl() *layout.Element { return p.host }

func (p *paragraphSelection) Blur() {
	p.focused = false
	p.menu.close()
}

func (p *paragraphSelection) HandleKeys(keys []input.KeyEvent) {
	if !p.focused {
		return
	}
	for _, ev := range keys {
		if !ev.Mods.Primary() {
			continue
		}
		switch ev.Key {
		case input.KeyA:
			p.SelectAll()
		case input.KeyC:
			p.Copy()
		}
	}
}

// ── textEditing (the right-click menu) ───────────────────────────────────────

func (p *paragraphSelection) CanUndo() bool { return false }
func (p *paragraphSelection) CanRedo() bool { return false }
func (p *paragraphSelection) Undo()         {}
func (p *paragraphSelection) Redo()         {}
func (p *paragraphSelection) Cut()          {}
func (p *paragraphSelection) Paste()        {}

func (p *paragraphSelection) HasSelection() bool {
	lo, hi := p.selRange()
	return lo != hi
}

// Copy puts the selection on the clipboard.
func (p *paragraphSelection) Copy() bool {
	clip := frameClipboard()
	lo, hi := p.selRange()
	if clip == nil || lo == hi {
		return false
	}
	clip.Set(p.text[lo:hi])
	return true
}

func (p *paragraphSelection) SelectAll() {
	p.anchor, p.caret = 0, len(p.text)
}
