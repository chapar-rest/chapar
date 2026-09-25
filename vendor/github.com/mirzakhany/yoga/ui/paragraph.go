package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// paragraphData holds Paragraph-only options.
type paragraphData struct {
	align layout.Align
}

// paragraphState keeps the last wrap so steady frames lay out at the right
// height on the first pass instead of relayouting every frame.
type paragraphState struct {
	width  float32 // content width the lines were wrapped at; <0 until known
	text   string
	size   float32
	weight int
	lines  []string
}

func (st *paragraphState) wrap(eng *shape.Engine, text string, size float32, weight int, width float32) []string {
	if st.lines != nil && st.width == width && st.text == text && st.size == size && st.weight == weight {
		return st.lines
	}
	st.width, st.text, st.size, st.weight = width, text, size, weight
	st.lines = wrapTextWeight(eng, text, size, weight, width)
	return st.lines
}

// Paragraph renders multi-line text: explicit newlines are kept and lines are
// word-wrapped to the width the parent gives it, with words too wide for a
// line split between characters. Its height follows the wrapped line count.
//
// Unlike Text it has no intrinsic width: it fills the cross axis of a Column
// (the default stretch alignment) and needs Grow or Width in a Row or under a
// centering parent. Size, Weight and Style (TextColor) work as on Text.
func Paragraph(s string) *Node {
	return &Node{kind: kindParagraph, text: s, extra: &paragraphData{align: AlignStart}}
}

// TextAlign sets the horizontal alignment of each Paragraph line (AlignStart,
// AlignCenter or AlignEnd).
func (n *Node) TextAlign(a layout.Align) *Node {
	if d, ok := n.extra.(*paragraphData); ok {
		d.align = a
	}
	return n
}

func (n *Node) layoutParagraph(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "paragraph")
	}
	st := c.Widget(id, func() any { return &paragraphState{width: -1} }).(*paragraphState)
	d, _ := n.extra.(*paragraphData)
	if d == nil {
		d = &paragraphData{align: AlignStart}
	}

	th := c.Theme()
	size := c.env.fontSize
	if size <= 0 {
		size = th.Typography.Body.Size
	}
	if n.spec.hasFontSize {
		size = n.spec.fontSize
	}
	weight := shape.WeightRegular
	if n.spec.hasFontWeight {
		weight = n.spec.fontWeight
	}
	col := c.env.textColor
	if !c.env.hasColor {
		col = th.Foreground
	}
	r := n.spec.resolve(th, interactState{})
	if r.hasFg {
		col = r.fg
	}
	if r.hasFontSize {
		size = r.fontSize
	}
	if r.hasFontWeight {
		weight = r.fontWeight
	}

	eng := c.Text()
	lineH := size
	if eng != nil {
		_, lineH = eng.MeasureAtWeight("Ag", size, weight)
	}
	var padL, padR, padT, padB float32
	if n.spec.hasPad {
		padL, padR = n.spec.pad.Left, n.spec.pad.Right
		padT, padB = n.spec.pad.Top, n.spec.pad.Bottom
	}
	heightFor := func(width float32) float32 {
		lines := 1
		if eng != nil && width >= 0 {
			lines = len(st.wrap(eng, n.text, size, weight, width))
		}
		return float32(lines)*lineH + padT + padB
	}

	// Height is a guess from the last known width until AfterLayout sees the
	// solved one; an explicit Height from the caller wins.
	box := layout.Box().FlexShrink(0)
	if !n.spec.hasH {
		box = box.H(heightFor(st.width))
	}
	el := layout.New(applyLayoutSpec(box, n.spec))
	text := n.text
	if !n.spec.hasH {
		el.AfterLayout = func(e *layout.Element) bool {
			w, _ := e.LayoutSize()
			w = f32max(0, w-padL-padR)
			if w == st.width && st.lines != nil {
				return false
			}
			h := heightFor(w)
			if h == e.Style.Height {
				return false
			}
			e.Style.Height = h
			// The engine allows a single relayout per frame; ask for another
			// frame in case this one was the second pass.
			c.Invalidate()
			return true
		}
	}
	el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		fr := el.Frame
		x0 := fr.X + padL
		w := f32max(0, fr.W-padL-padR)
		lines := st.wrap(eng, text, size, weight, w)
		dl.PushClip(fr)
		y := fr.Y + padT
		for _, line := range lines {
			if y >= fr.Y+fr.H {
				break
			}
			x := x0
			if d.align != AlignStart {
				lw, _ := eng.MeasureAtWeight(line, size, weight)
				if d.align == AlignCenter {
					x += (w - lw) / 2
				} else if d.align == AlignEnd {
					x += w - lw
				}
			}
			eng.DrawStringTopAtWeight(dl, line, x, y, col, size, weight)
			y += lineH
		}
		dl.PopClip()
	}
	return el
}
