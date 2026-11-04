package ui

import "github.com/mirzakhany/yoga/layout"

type responsiveState struct {
	width float32
}

type responsive struct {
	id    string
	child func(width float32) View
}

// Responsive lays out child(width), width being what the view was given at
// its last layout: 0 until the first one. When the width changes the frame
// is built again, so a view can pick a compact form for a narrow pane, such
// as a menu in place of a side list. id keys the width across frames. Like
// HoverGroup, it adds no box of its own.
func Responsive(id string, child func(width float32) View) *Node {
	return ViewOf(&responsive{id: id, child: child})
}

func (r *responsive) Layout(c *Ctx) *layout.Element {
	st := c.Widget(r.id, func() any { return &responsiveState{} }).(*responsiveState)
	var el *layout.Element
	if v := r.child(st.width); v != nil {
		el = v.Layout(c)
	}
	if el == nil {
		el = layout.New(layout.Box())
	}
	inner := el.AfterLayout
	el.AfterLayout = func(e *layout.Element) bool {
		relayout := inner != nil && inner(e)
		if w, _ := e.LayoutSize(); w != st.width {
			st.width = w
			c.Invalidate()
		}
		return relayout
	}
	return el
}
