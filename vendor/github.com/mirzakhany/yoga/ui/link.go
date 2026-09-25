package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type linkState struct {
	hovered, focused bool
	el               *layout.Element
	activate         func()
}

func (s *linkState) Focus()                      { s.focused = true }
func (s *linkState) Blur()                       { s.focused = false }
func (s *linkState) Focused() bool               { return s.focused }
func (s *linkState) HandleKeys([]input.KeyEvent) {}
func (s *linkState) CapturesTab() bool           { return false }
func (s *linkState) FocusOnClick() bool          { return true }
func (s *linkState) FocusEl() *layout.Element    { return s.el }

func (s *linkState) HandleText(runes []rune) {
	if !s.focused {
		return
	}
	for _, r := range runes {
		if (r == ' ' || r == '\n') && s.activate != nil {
			s.activate()
		}
	}
}

var _ Focusable = (*linkState)(nil)

// Link is clickable accent text.
func Link(id, label string) *Node {
	return &Node{kind: kindLink, id: id, text: label}
}

func (n *Node) layoutLink(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "link")
	}
	st := c.Widget(id, func() any { return &linkState{} }).(*linkState)
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	style := th.Typography.Body
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = eng.MeasureAt(n.text, style.Size)
	} else {
		lh = style.LineHeight
		tw = style.Size * 0.5 * float32(len(n.text))
	}
	el := layout.New(applyLayoutSpec(layout.Box().Size(tw, lh).FlexShrink(0), n.spec))
	st.el = el
	label := n.text
	onClick := n.onClick
	disabled := n.disabled
	st.activate = onClick

	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		fg := th.Link
		if disabled {
			fg = th.ForegroundDisabled
		} else if st.hovered {
			fg = th.LinkHover
		}
		// The ring fills its rect, so it goes under the label. The frame hugs
		// the text, so the ring sits a little outside it to clear the glyphs.
		if st.focused {
			const pad = 3.0
			ring := render.Rect{X: f.X - pad, Y: f.Y - pad/2, W: f.W + 2*pad, H: f.H + pad}
			paintFocusRing(dl, ring, th.Surface, th)
		}
		_, mh := text.MeasureAt(label, style.Size)
		ty := f.Y + (f.H-mh)/2
		text.DrawStringTopAt(dl, label, f.X, ty, fg, style.Size)
		underline := render.Rect{X: f.X, Y: ty + mh - 1, W: tw, H: 1}
		dl.AddRect(underline, fg)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if disabled {
			return
		}
		inside := e.Frame.Contains(m.X, m.Y)
		trackHover(c, &st.hovered, inside)
		if inside {
			m.SetCursor(CursorPointer)
		}
		if inside && m.Released && onClick != nil {
			onClick()
			c.MarkNeedsPaint()
			m.Consumed = true
		}
		if inside && m.Pressed {
			m.Consumed = true
		}
	}
	return el
}
