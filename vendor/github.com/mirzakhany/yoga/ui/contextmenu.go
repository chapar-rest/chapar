package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

type contextMenuData struct {
	child View
	items []MenuItem
}

type contextMenuState struct {
	menu *Menu
}

// ContextMenu wraps child and opens a Menu on right-click.
func ContextMenu(id string, child View, items []MenuItem) *Node {
	return &Node{
		kind:  kindContextMenu,
		id:    id,
		child: child,
		extra: &contextMenuData{child: child, items: items},
	}
}

func (n *Node) layoutContextMenu(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "ctxmenu")
	}
	d, _ := n.extra.(*contextMenuData)
	items := []MenuItem{}
	if d != nil {
		items = d.items
	}
	st := c.Widget(id, func() any { return &contextMenuState{} }).(*contextMenuState)
	w := float32(180)
	if n.spec.hasW {
		w = n.spec.width
	}
	if st.menu == nil {
		st.menu = NewMenu(w, items)
	} else {
		st.menu.SetItems(items)
		st.menu.width = w
	}

	var childEl *layout.Element
	child := n.child
	if d != nil && d.child != nil {
		child = d.child
	}
	if child != nil {
		childEl = child.Layout(c)
	}
	box := applyLayoutSpec(layout.Box().Direction(layout.Column).FlexShrink(0), n.spec)
	el := layout.New(box)
	if childEl != nil {
		el.Children = []*layout.Element{childEl}
	}

	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if e.Frame.Contains(m.X, m.Y) && m.RightPressed {
			st.menu.OpenAt(m.X, m.Y)
			c.MarkNeedsPaint()
			m.Consumed = true
		}
	}

	if st.menu.Open {
		if kb := c.Keyboard(); kb != nil {
			for _, ev := range kb.Keys {
				if ev.Key == input.KeyEscape {
					st.menu.Close()
					c.MarkNeedsPaint()
					break
				}
			}
		}
		st.menu.BindPaint(c)
		c.Overlay(st.menu.overlay())
	}
	return el
}
