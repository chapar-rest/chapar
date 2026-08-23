package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type radioState struct {
	hovered, focused bool
	el               *layout.Element
	selectFn         func()
}

func (b *radioState) Focus()                      { b.focused = true }
func (b *radioState) Blur()                       { b.focused = false }
func (b *radioState) Focused() bool               { return b.focused }
func (b *radioState) HandleKeys([]input.KeyEvent) {}
func (b *radioState) CapturesTab() bool           { return false }
func (b *radioState) FocusOnClick() bool          { return true }
func (b *radioState) FocusEl() *layout.Element    { return b.el }

func (b *radioState) HandleText(runes []rune) {
	if !b.focused {
		return
	}
	for _, r := range runes {
		if r == ' ' && b.selectFn != nil {
			b.selectFn()
		}
	}
}

// Radio is one mutually exclusive option. selected is controlled by the app.
func Radio(id, label string) *Node {
	return &Node{kind: kindRadio, id: id, text: label}
}

func (n *Node) layoutRadio(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "radio")
	}
	st := c.Widget(id, func() any { return &radioState{} }).(*radioState)
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	box := th.Metrics.IconSizeSM
	style := th.Typography.Body
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = eng.MeasureAt(n.text, style.Size)
	} else {
		lh = style.LineHeight
		tw = style.Size * 0.5 * float32(len(n.text))
	}
	h := f32max(box, lh)
	w := box + th.Spacing.S + tw
	el := layout.New(applyLayoutSpec(layout.Box().Size(w, h).FlexShrink(0), n.spec))
	st.el = el
	selected := n.checked
	onClick := n.onClick
	st.selectFn = onClick
	label := n.text
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		bx := f.X
		by := f.Y + (f.H-box)/2
		br := render.Rect{X: bx, Y: by, W: box, H: box}
		border := th.Border
		if st.focused {
			border = th.FocusRing
		}
		fill := th.Chrome
		if selected {
			border = th.Accent
		}
		if st.hovered {
			fill = th.ListHover
		}
		dl.AddRoundedRectBorder(br, box/2, th.Stroke.Thin, fill, border)
		if selected {
			dot := box * 0.35
			dl.AddRoundedRect(render.Rect{
				X: bx + (box-dot)/2, Y: by + (box-dot)/2, W: dot, H: dot,
			}, dot/2, th.Accent)
		}
		tx := bx + box + th.Spacing.S
		_, hl := text.MeasureAt(label, style.Size)
		text.DrawStringTopAt(dl, label, tx, f.Y+(f.H-hl)/2, th.Foreground, style.Size)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		st.hovered = e.Frame.Contains(m.X, m.Y)
		if st.hovered {
			m.SetCursor(CursorPointer)
		}
		if st.hovered && m.Released && onClick != nil {
			onClick()
			m.Consumed = true
		}
	}
	return el
}
