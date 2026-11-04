package ui

import (
	"unicode/utf8"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// richView lays out and paints rich text: word-wrapped styled runs, inline
// code, links and syntax colors, with mouse selection and copy.
type richView struct {
	id     string
	rt     *richText
	size   float32
	weight int  // weight of unstyled text
	mono   bool // the whole text is code
	pre    bool // keep whitespace; lines break at newlines and when too wide
	muted  bool
	align  layout.Align // AlignStart, AlignCenter or AlignEnd per line
	tokens []highlight.Token
	onLink func(url string)
	// sel is the Markdown view's selection and span this block's place in
	// its text.
	sel  *paragraphSelection
	span *mdSpan
}

// richFrag is a piece of text drawn in one style on one line.
type richFrag struct {
	lo, hi int
	line   int
	x, w   float32
	style  mdStyle
	class  highlight.ColorClass
}

type richState struct {
	// The last layout and what it was computed from.
	width  float32
	text   string
	nruns  int
	ntoks  int
	size   float32
	align  layout.Align
	frags  []richFrag
	lines  int
	hover  string // link under the pointer
	press  string // link the button went down on
	inited bool
}

// mdLeading is the extra space between lines of prose at size.
func mdLeading(size float32, mono bool) float32 {
	if mono {
		return float32(int(size*0.3 + 0.5))
	}
	return float32(int(size*0.45 + 0.5))
}

func (v *richView) styleWeight(st mdStyle) int {
	if st.bold {
		return shape.WeightSemiBold
	}
	return v.weight
}

func (v *richView) isMono(st mdStyle) bool { return v.mono || st.code }

// sizeOf is the font size of a style. Inline code is a step smaller, since
// monospace glyphs run larger than prose at the same size.
func (v *richView) sizeOf(st mdStyle) float32 {
	if st.code && !v.mono {
		return float32(int(v.size*0.9 + 0.5))
	}
	return v.size
}

func (v *richView) measure(eng *shape.Engine, s string, st mdStyle) float32 {
	if v.isMono(st) {
		w, _ := eng.MeasureMonoAt(s, v.sizeOf(st))
		return w
	}
	w, _ := eng.MeasureAtWeight(s, v.size, v.styleWeight(st))
	return w
}

// segment is a maximal piece of text with one style and color class.
type richSeg struct {
	lo, hi int
	style  mdStyle
	class  highlight.ColorClass
}

// segments splits the text at run and token boundaries.
func (v *richView) segments() []richSeg {
	rt := v.rt
	var out []richSeg
	ti := 0
	for _, r := range rt.runs {
		lo := r.lo
		for lo < r.hi {
			hi := r.hi
			class := highlight.ClassDefault
			for ti < len(v.tokens) && v.tokens[ti].End <= lo {
				ti++
			}
			if ti < len(v.tokens) {
				t := v.tokens[ti]
				if t.Start <= lo {
					class = t.Class
					hi = min(hi, t.End)
				} else {
					hi = min(hi, t.Start)
				}
			}
			out = append(out, richSeg{lo: lo, hi: hi, style: r.style, class: class})
			lo = hi
		}
	}
	return out
}

// wrap lays the text out into fragments at width.
func (v *richView) wrap(eng *shape.Engine, st *richState, width float32) {
	if st.inited && st.width == width && st.text == v.rt.text && st.nruns == len(v.rt.runs) &&
		st.ntoks == len(v.tokens) && st.size == v.size && st.align == v.align {
		return
	}
	st.inited = true
	st.width, st.text, st.nruns, st.ntoks, st.size, st.align = width, v.rt.text, len(v.rt.runs), len(v.tokens), v.size, v.align
	st.frags = st.frags[:0]
	text := v.rt.text
	line, x := 0, float32(0)
	emit := func(sg richSeg, lo, hi int, w float32) {
		st.frags = append(st.frags, richFrag{lo: lo, hi: hi, line: line, x: x, w: w, style: sg.style, class: sg.class})
		x += w
	}
	newline := func() { line, x = line+1, 0 }
	// place puts text[lo:hi] (no spaces in it) on the current line,
	// splitting it between characters when it is wider than a whole line.
	place := func(sg richSeg, lo, hi int) {
		for lo < hi {
			w := v.measure(eng, text[lo:hi], sg.style)
			if x+w <= width || width <= 0 {
				emit(sg, lo, hi, w)
				return
			}
			if x > 0 {
				newline()
				continue
			}
			// Longest prefix that fits, at least one character.
			cut := lo
			for cut < hi {
				_, n := utf8.DecodeRuneInString(text[cut:])
				if cut > lo && v.measure(eng, text[lo:cut+n], sg.style) > width {
					break
				}
				cut += n
			}
			emit(sg, lo, cut, v.measure(eng, text[lo:cut], sg.style))
			newline()
			lo = cut
		}
	}

	segs := v.segments()
	for i := 0; i < len(segs); i++ {
		sg := segs[i]
		pos := sg.lo
		for pos < sg.hi {
			c := text[pos]
			switch c {
			case '\n':
				newline()
				pos++
			case ' ', '\t':
				end := pos
				for end < sg.hi && (text[end] == ' ' || text[end] == '\t') {
					end++
				}
				if !v.pre && x == 0 {
					pos = end // no leading space on a wrapped line
					continue
				}
				w := v.measure(eng, text[pos:end], sg.style)
				if v.pre && x+w > width && x > 0 {
					newline()
				}
				emit(sg, pos, end, w)
				pos = end
			default:
				// A word may continue into the next segments (**bold**ly), so
				// measure all of it before choosing to break the line.
				end := pos
				for end < sg.hi && text[end] != ' ' && text[end] != '\t' && text[end] != '\n' {
					end++
				}
				if x > 0 && end == sg.hi {
					w := v.measure(eng, text[pos:end], sg.style)
					for j := i + 1; j < len(segs); j++ {
						n := segs[j]
						e := n.lo
						for e < n.hi && text[e] != ' ' && text[e] != '\t' && text[e] != '\n' {
							e++
						}
						w += v.measure(eng, text[n.lo:e], n.style)
						if e < n.hi {
							break
						}
					}
					if x+w > width {
						newline()
					}
				} else if x > 0 && x+v.measure(eng, text[pos:end], sg.style) > width {
					newline()
				}
				place(sg, pos, end)
				pos = end
			}
		}
	}
	st.lines = line + 1
	if text == "" {
		st.lines = 1
	}
	if (v.align == AlignCenter || v.align == AlignEnd) && width > 0 {
		v.alignLines(st, width)
	}
}

// alignLines moves each line's fragments right by the room left on it, or
// half of it when centered.
func (v *richView) alignLines(st *richState, width float32) {
	for i := 0; i < len(st.frags); {
		j := i
		for j < len(st.frags) && st.frags[j].line == st.frags[i].line {
			j++
		}
		last := st.frags[j-1]
		shift := width - (last.x + last.w)
		if v.align == AlignCenter {
			shift /= 2
		}
		for k := i; k < j; k++ {
			st.frags[k].x += max(0, shift)
		}
		i = j
	}
}

// fragAt returns the fragment under (x, y) relative to the text's origin.
func (st *richState) fragAt(x, y, lineH float32) int {
	line := int(y / lineH)
	for i, f := range st.frags {
		if f.line == line && x >= f.x && x < f.x+f.w {
			return i
		}
	}
	return -1
}

// offsetAt maps a point relative to the text's origin to a byte offset.
func (v *richView) offsetAt(eng *shape.Engine, st *richState, x, y, lineH float32) int {
	if len(st.frags) == 0 || y < 0 {
		return 0
	}
	line := int(y / lineH)
	if line >= st.lines {
		return len(v.rt.text)
	}
	// The last fragment at or before the point on its line.
	last := -1
	for i, f := range st.frags {
		if f.line > line {
			break
		}
		if f.line < line {
			last = i
			continue
		}
		if x < f.x {
			if last < 0 || st.frags[last].line < line {
				return f.lo
			}
			break
		}
		last = i
		if x < f.x+f.w {
			text := v.rt.text
			prev := float32(0)
			for off := f.lo; off < f.hi; {
				_, n := utf8.DecodeRuneInString(text[off:])
				w := v.measure(eng, text[f.lo:off+n], f.style)
				if x < f.x+(prev+w)/2 {
					return off
				}
				prev = w
				off += n
			}
			return f.hi
		}
	}
	if last < 0 {
		return 0
	}
	return st.frags[last].hi
}

func (v *richView) Layout(c *Ctx) *layout.Element {
	st := c.Widget(v.id, func() any { return &richState{width: -1} }).(*richState)
	th := c.Theme()
	eng := c.Text()
	lead := mdLeading(v.size, v.mono)
	textH := v.size
	if eng != nil {
		if v.mono {
			_, textH = eng.MeasureMonoAt("Ag", v.size)
		} else {
			_, textH = eng.MeasureAtWeight("Ag", v.size, v.weight)
		}
	}
	lineH := textH + lead
	heightFor := func(width float32) float32 {
		if eng != nil && width >= 0 {
			v.wrap(eng, st, width)
			return float32(st.lines) * lineH
		}
		return lineH
	}

	el := layout.New(layout.Box().FlexShrink(0).H(heightFor(st.width)))
	sel, span := v.sel, v.span
	span.st, span.el, span.lineH = st, el, lineH

	el.AfterLayout = func(e *layout.Element) bool {
		w, _ := e.LayoutSize()
		if w == st.width && st.inited {
			return false
		}
		h := heightFor(w)
		if h == e.Style.Height {
			return false
		}
		e.Style.Height = h
		c.Invalidate()
		return true
	}
	onLink := v.onLink
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		inside := e.Frame.Contains(m.X, m.Y)
		link := ""
		if inside {
			if i := st.fragAt(m.X-e.Frame.X, m.Y-e.Frame.Y, lineH); i >= 0 {
				link = st.frags[i].style.link
			}
		}
		if link != st.hover {
			st.hover = link
			c.MarkNeedsPaint()
		}
		if m.Pressed && inside {
			st.press = link
		}
		released := m.Released && inside
		sel.onMouse(e, m, c.Text())
		if link != "" {
			m.SetCursor(CursorPointer)
		}
		if released && link != "" && link == st.press && !sel.HasSelection() {
			if onLink != nil {
				onLink(link)
			}
			m.Consumed = true
		}
		if !m.Down {
			st.press = ""
		}
	}

	muted := v.muted
	mono := v.mono
	el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		fr := el.Frame
		v.wrap(eng, st, fr.W)
		base := th.Foreground
		if muted {
			base = th.ForegroundMuted
		}
		dl.PushClip(fr)
		// Inline code backgrounds go under the selection and the text.
		for _, f := range st.frags {
			if !f.style.code || mono {
				continue
			}
			y := fr.Y + float32(f.line)*lineH + lead/2
			r := render.Rect{X: fr.X + f.x - 2, Y: y, W: f.w + 4, H: textH}
			dl.AddRoundedRect(r, th.Radius.Small, th.ChromeMuted)
		}
		if lo, hi := sel.selRange(); sel.focused && lo != hi {
			lo, hi = lo-span.base, hi-span.base
			text := v.rt.text
			for _, f := range st.frags {
				a, b := max(lo, f.lo), min(hi, f.hi)
				if a >= b {
					continue
				}
				xa := v.measure(eng, text[f.lo:a], f.style)
				xb := v.measure(eng, text[f.lo:b], f.style)
				y := fr.Y + float32(f.line)*lineH
				dl.AddRect(render.Rect{X: fr.X + f.x + xa, Y: y, W: xb - xa, H: lineH}, th.Selection)
			}
		}
		for _, f := range st.frags {
			y := fr.Y + float32(f.line)*lineH + lead/2
			if y >= fr.Y+fr.H {
				break
			}
			s := v.rt.text[f.lo:f.hi]
			col := base
			switch {
			case f.style.link != "":
				col = th.Link
				if f.style.link == st.hover {
					col = th.LinkHover
				}
			case f.class != highlight.ClassDefault:
				col = th.SyntaxColor(f.class)
			}
			x := fr.X + f.x
			if v.isMono(f.style) {
				size := v.sizeOf(f.style)
				_, h := eng.MeasureMonoAt("Ag", size)
				eng.DrawStringTopMonoAt(dl, s, x, y+(textH-h)/2, col, size)
			} else {
				eng.DrawStringTopAtWeight(dl, s, x, y, col, v.size, v.styleWeight(f.style))
			}
			if f.style.link != "" {
				dl.AddRect(render.Rect{X: x, Y: y + textH - 1, W: f.w, H: 1}, col)
			}
			if f.style.strike {
				dl.AddRect(render.Rect{X: x, Y: y + textH*0.55, W: f.w, H: 1}, col)
			}
		}
		dl.PopClip()
	}
	return el
}
