package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type iconButtonState struct {
	hovered, pressed bool
	el               *layout.Element
}

// IconButton is a square icon-only control.
func IconButton(id, icon string) *Node {
	return &Node{kind: kindIconButton, id: id, icon: icon}
}

func (n *Node) layoutIconButton(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "iconbutton")
	}
	st := c.Widget(id, func() any { return &iconButtonState{} }).(*iconButtonState)
	th := c.Theme()
	sz := th.Metrics.ControlHeight
	if n.spec.hasH {
		sz = n.spec.height
	}
	if n.iconSize > 0 {
		sz = n.iconSize
	}
	el := layout.New(applyLayoutSpec(layout.Box().Size(sz, sz).FlexShrink(0), n.spec))
	st.el = el
	icon := n.icon
	onClick := n.onClick
	disabled := n.disabled
	spec := c.styles().ButtonSubtle.merge(n.spec)
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		r := spec.resolve(th, interactState{hovered: st.hovered, pressed: st.pressed, disabled: disabled})
		frame := scaledFrame(el.Frame, r.scaleX, r.scaleY)
		radius := th.Radius.Medium
		if r.hasRadius {
			radius = r.radius
		}
		if r.hasBg && r.bg.A > 0 {
			dl.AddRoundedRect(frame, radius, r.bg)
		} else if st.hovered {
			dl.AddRoundedRect(frame, radius, th.ListHover)
		}
		col := th.Foreground
		if r.hasFg {
			col = r.fg
		}
		inset := sz * 0.22
		inner := render.Rect{X: frame.X + inset, Y: frame.Y + inset, W: frame.W - 2*inset, H: frame.H - 2*inset}
		if sheet := frameIcons(); sheet != nil {
			sheet.Draw(dl, icon, inner, col)
		}
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if disabled {
			return
		}
		inside := e.Frame.Contains(m.X, m.Y)
		st.hovered = inside
		if inside {
			m.SetCursor(CursorPointer)
		}
		if inside && m.Pressed {
			st.pressed = true
			m.Consumed = true
		}
		if m.Released {
			if st.pressed && inside && onClick != nil {
				onClick()
			}
			st.pressed = false
		}
	}
	return el
}
