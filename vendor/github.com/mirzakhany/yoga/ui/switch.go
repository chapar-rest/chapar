package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

const (
	switchTrackW float32 = 36
	switchTrackH float32 = 20
)

type switchState struct {
	hovered, focused bool
	el               *layout.Element
	toggle           func()
}

func (s *switchState) Focus()                      { s.focused = true }
func (s *switchState) Blur()                       { s.focused = false }
func (s *switchState) Focused() bool               { return s.focused }
func (s *switchState) HandleKeys([]input.KeyEvent) {}
func (s *switchState) CapturesTab() bool           { return false }
func (s *switchState) FocusOnClick() bool          { return true }
func (s *switchState) FocusEl() *layout.Element    { return s.el }

func (s *switchState) HandleText(runes []rune) {
	if !s.focused {
		return
	}
	for _, r := range runes {
		if r == ' ' && s.toggle != nil {
			s.toggle()
		}
	}
}

var _ Focusable = (*switchState)(nil)

// Switch is an unlabeled pill toggle. id keys hover/focus; checked is controlled by the app.
func Switch(id string) *Node {
	return &Node{kind: kindSwitch, id: id}
}

func (n *Node) layoutSwitch(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "switch")
	}
	st := c.Widget(id, func() any { return &switchState{} }).(*switchState)
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	el := layout.New(applyLayoutSpec(layout.Box().Size(switchTrackW, switchTrackH).FlexShrink(0), n.spec))
	st.el = el

	checked := n.checked
	onToggle := n.onToggle
	st.toggle = func() {
		if onToggle != nil {
			onToggle(!checked)
		}
	}
	spec := c.styles().Switch.merge(n.spec)

	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		f := el.Frame
		inter := interactState{hovered: st.hovered, focused: st.focused}
		r := spec.resolve(th, inter)
		trackR := switchTrackH / 2
		fill := th.ChromeMuted
		if r.hasBg {
			fill = r.bg
		}
		border := th.Border
		if r.hasBorder {
			border = r.border
		}
		if checked {
			fill = th.Accent
			border = th.Accent
			if st.hovered {
				fill = th.AccentHover
			}
		} else if st.hovered {
			fill = th.ListHover
		}
		if st.focused {
			border = th.FocusRing
		}
		bw := th.Stroke.Thin
		if r.borderW > 0 {
			bw = r.borderW
		}
		dl.AddRoundedRectBorder(f, trackR, bw, fill, border)

		pad := float32(2)
		thumbD := switchTrackH - 2*pad
		thumbX := f.X + pad
		if checked {
			thumbX = f.X + f.W - pad - thumbD
		}
		thumbY := f.Y + pad
		thumb := render.Rect{X: thumbX, Y: thumbY, W: thumbD, H: thumbD}
		thumbCol := th.Foreground
		if checked {
			thumbCol = th.AccentForeground
		}
		dl.AddRoundedRect(thumb, thumbD/2, thumbCol)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		st.hovered = e.Frame.Contains(m.X, m.Y)
		if st.hovered && m.Released {
			st.toggle()
			m.Consumed = true
		}
	}
	return el
}
