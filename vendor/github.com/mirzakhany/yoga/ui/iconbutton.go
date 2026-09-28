package ui

import (
	"github.com/mirzakhany/yoga/icons"
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
func IconButton(id string, icon icons.Icon) *Node {
	return &Node{kind: kindIconButton, id: id, icon: icon}
}

// Menu makes an IconButton open a menu of items when clicked, instead of
// running OnClick: the "more actions" button of a row or toolbar. The menu
// hangs below the button with its right edge on the button's, so it stays
// in view when the button sits at the right of its container.
func (n *Node) Menu(items []MenuItem) *Node {
	n.menuItems = items
	n.hasMenu = true
	return n
}

func (n *Node) layoutIconButton(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "iconbutton")
	}
	st := c.Widget(id, func() any { return &iconButtonState{} }).(*iconButtonState)
	th := c.Theme()
	sz := c.controlHeight()
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

	var menu *Menu
	if n.hasMenu {
		mst := c.Widget(id+"-menu", func() any { return &dropdownState{} }).(*dropdownState)
		if mst.menu == nil {
			mst.menu = NewMenu(iconMenuWidth, n.menuItems)
		} else {
			mst.menu.SetItems(n.menuItems)
		}
		menu = mst.menu
		if disabled {
			menu.Close()
		}
		onClick = func() {
			if menu.Open {
				menu.Close()
				return
			}
			f := el.Frame
			menu.OpenAt(f.X+f.W-menu.width, f.Y+f.H)
		}
	}

	spec := c.styles().ButtonSubtle.merge(n.spec)
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		// An open menu keeps its button lit, as a hovered one.
		hovered := st.hovered || (menu != nil && menu.Open)
		r := spec.resolve(th, interactState{hovered: hovered, pressed: st.pressed, disabled: disabled})
		frame := scaledFrame(el.Frame, r.scaleX, r.scaleY)
		radius := th.Radius.Medium
		if r.hasRadii {
			radius = uniformRadius(r.radii, th.Radius.Medium)
		}
		if r.hasBg && r.bg.A > 0 {
			dl.AddRoundedRect(frame, radius, r.bg)
		} else if hovered {
			dl.AddRoundedRect(frame, radius, th.ListHover)
		}
		col := th.Foreground
		if r.hasFg {
			col = r.fg
		}
		inset := sz * 0.22
		inner := render.Rect{X: frame.X + inset, Y: frame.Y + inset, W: frame.W - 2*inset, H: frame.H - 2*inset}
		if sheet := frameIcons(); sheet != nil && !icon.Empty() {
			sheet.Draw(dl, icon, inner, col)
		}
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
		if inside && m.Pressed {
			trackBool(c, &st.pressed, true)
			m.Consumed = true
		}
		if m.Released {
			if st.pressed && inside && onClick != nil {
				onClick()
				c.MarkNeedsPaint()
			}
			trackBool(c, &st.pressed, false)
		}
	}
	if menu != nil && menu.Open {
		menu.BindPaint(c)
		c.Overlay(menu.overlay())
	}
	return el
}

// iconMenuWidth is the width of an IconButton's menu, which cannot take
// the width of its small trigger as a MenuButton's does.
const iconMenuWidth = 180
