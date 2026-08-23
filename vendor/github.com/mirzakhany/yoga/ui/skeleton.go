package ui

import (
	"time"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type skeletonData struct {
	circle bool
}

type skeletonState struct {
	phase float32
}

// Skeleton is a shimmering loading placeholder.
func Skeleton(id string) *Node {
	return &Node{kind: kindSkeleton, id: id, extra: &skeletonData{}}
}

// Circle makes a Skeleton a circular placeholder of diameter d.
func (n *Node) Circle(d float32) *Node {
	if sd, ok := n.extra.(*skeletonData); ok {
		sd.circle = true
	}
	return n.Width(d).Height(d)
}

func (n *Node) layoutSkeleton(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "skeleton")
	}
	d, _ := n.extra.(*skeletonData)
	st := c.Widget(id, func() any { return &skeletonState{} }).(*skeletonState)
	st.phase += 0.05
	if st.phase > 1 {
		st.phase -= 1
	}
	c.Animate(16 * time.Millisecond)

	th := c.Theme()
	w := float32(160)
	h := float32(12)
	if n.spec.hasW {
		w = n.spec.width
	}
	if n.spec.hasH {
		h = n.spec.height
	}
	circle := d != nil && d.circle
	el := layout.New(applyLayoutSpec(layout.Box().Size(w, h).FlexShrink(0), n.spec))
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		f := el.Frame
		base := th.ChromeMuted
		hi := th.ListHover
		t := st.phase
		col := render.Color{
			R: base.R + (hi.R-base.R)*t,
			G: base.G + (hi.G-base.G)*t,
			B: base.B + (hi.B-base.B)*t,
			A: 1,
		}
		r := th.Radius.Small
		if circle {
			r = f.H / 2
		}
		dl.AddRoundedRect(f, r, col)
	}
	return el
}
