package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

type hoverGroupState struct {
	hovered bool
}

type hoverGroup struct {
	id    string
	child func(hovered bool) View
}

// HoverGroup lays out child(hovered), hovered telling whether the pointer is
// over the group: a list row that shows its actions only while hovered (see
// Reveal). id keys the hover state across frames. The group adds no box of
// its own; modifiers such as Grow apply to the child's.
func HoverGroup(id string, child func(hovered bool) View) *Node {
	return ViewOf(&hoverGroup{id: id, child: child})
}

func (g *hoverGroup) Layout(c *Ctx) *layout.Element {
	st := c.Widget(g.id, func() any { return &hoverGroupState{} }).(*hoverGroupState)
	var el *layout.Element
	if v := g.child(st.hovered); v != nil {
		el = v.Layout(c)
	}
	if el == nil {
		el = layout.New(layout.Box())
	}
	inner := el.OnMouse
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		trackHover(c, &st.hovered, e.Frame.Contains(m.X, m.Y))
		if inner != nil {
			inner(e, m)
		}
	}
	return el
}

// Reveal(false) leaves an IconButton's space empty and lets clicks through,
// until Reveal(true) shows it again; a button whose menu is open stays
// shown. Pair it with HoverGroup for actions that appear on hover.
func (n *Node) Reveal(show bool) *Node { n.hidden = !show; return n }
