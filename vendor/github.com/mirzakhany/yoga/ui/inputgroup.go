package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// joined is a control's place in an InputGroup. The zero value means the
// control stands alone and draws its own border and focus ring.
type joined struct {
	in, first, last bool
}

// corners rounds only the corners on the group's outer edge.
func (j joined) corners(r float32) render.Corners {
	var c render.Corners
	if j.first {
		c.TopLeft, c.BottomLeft = r, r
	}
	if j.last {
		c.TopRight, c.BottomRight = r, r
	}
	return c
}

// paintJoinedFill fills a grouped control's frame. The group draws the
// border, the dividers and the focus ring over it.
func paintJoinedFill(dl *render.DrawList, f render.Rect, j joined, fill render.Color, radius float32) {
	if fill.A == 0 {
		return
	}
	dl.AddRoundedRectCorners(f, j.corners(radius), fill)
}

// InputGroup joins controls (Select, TextField, Button) into one bordered
// control: one outer border and radius, 1 px dividers between the parts, and
// a single focus ring around the whole group while any part has focus. Give
// the part that should take the spare width Grow(1).
//
//	ui.InputGroup("req",
//	    ui.Select("m", methods).FitContent(),
//	    ui.TextField("url", url).Grow(1),
//	    ui.Button("send", ui.Text("Send")).Primary(),
//	).Grow(1)
func InputGroup(id string, children ...View) *Node {
	return &Node{kind: kindInputGroup, id: id, children: children}
}

func (n *Node) layoutInputGroup(c *Ctx) *layout.Element {
	th := c.Theme()
	h := c.controlHeight()
	radius := th.Radius.Medium
	stroke := th.Stroke.Thin

	var parts []Focusable
	from := 0
	if c.Focus() != nil {
		from = len(c.Focus().items)
	}
	kids := make([]*layout.Element, 0, len(n.children)+1)
	old := c.env.joined
	for i, ch := range n.children {
		if ch == nil {
			continue
		}
		c.env.joined = joined{in: true, first: i == 0, last: i == len(n.children)-1}
		if el := ch.Layout(c); el != nil {
			kids = append(kids, el)
		}
	}
	c.env.joined = old
	if c.Focus() != nil {
		parts = append(parts, c.Focus().items[from:]...)
	}
	focused := func() bool {
		for _, p := range parts {
			if p.Focused() {
				return true
			}
		}
		return false
	}
	controls := kids

	// The frame paints after the parts (it is the last child), so the border
	// and dividers sit on top of their fills.
	frame := layout.New(layout.Box().AbsLeft(0).AbsTop(0).AbsRight(0).AbsBottom(0))
	host := layout.New(applyLayoutSpec(
		layout.Box().Direction(layout.Row).AlignItems(layout.AlignStretch).H(h).FlexShrink(0),
		n.spec), append(kids, frame)...)
	host.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		if focused() {
			// The same soft glow a lone text field has.
			f := host.Frame
			glow := render.Rect{X: f.X - 2, Y: f.Y - 2, W: f.W + 4, H: f.H + 4}
			dl.AddRoundedRect(glow, radius+2, alpha(th.FocusRing, 0.25))
		}
	}
	frame.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		f := host.Frame
		for _, k := range controls[1:] {
			dl.AddRect(render.Rect{X: k.Frame.X, Y: f.Y, W: stroke, H: f.H}, th.Border)
		}
		border := th.Border
		if focused() {
			border = th.FocusRing
		}
		r := render.Corners{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}
		w := render.BorderEdges{Top: stroke, Right: stroke, Bottom: stroke, Left: stroke}
		dl.AddRoundedRectStroke(f, r, w, border, render.BorderSolid)
	}
	return host
}

func alpha(c render.Color, a float32) render.Color {
	c.A = a
	return c
}
