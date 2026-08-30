package ui

import (
	"github.com/mirzakhany/yoga/layout"
)

type dropdownData struct {
	items []MenuItem
}

type dropdownState struct {
	menu *Menu
}

// Dropdown is a labelled trigger that opens a menu of items.
func Dropdown(id, label string, items []MenuItem) *Node {
	return &Node{kind: kindDropdown, id: id, text: label, extra: &dropdownData{items: items}}
}

func (n *Node) layoutDropdown(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "dropdown")
	}
	d, _ := n.extra.(*dropdownData)
	items := []MenuItem{}
	if d != nil {
		items = d.items
	}
	st := c.Widget(id+"-menu", func() any { return &dropdownState{} }).(*dropdownState)
	w := n.spec.width
	if !n.spec.hasW {
		w = 160
	}
	if st.menu == nil {
		st.menu = NewMenu(w, items)
	} else {
		st.menu.SetItems(items)
		st.menu.width = w
	}
	btn := Button(id, Text(n.text)).OnClick(func() {
		bst := c.Widget(id, func() any { return &buttonState{} }).(*buttonState)
		if st.menu.Open {
			st.menu.Close()
			return
		}
		if bst.el != nil {
			f := bst.el.Frame
			st.menu.width = triggerMenuWidth(w, f.W)
			st.menu.OpenAt(f.X, f.Y+f.H)
		}
	}).Style(n.spec)
	if n.disabled {
		btn = btn.Disabled(true)
	}
	el := btn.Layout(c)
	if st.menu.Open {
		c.Overlay(st.menu.overlay())
	}
	return el
}
