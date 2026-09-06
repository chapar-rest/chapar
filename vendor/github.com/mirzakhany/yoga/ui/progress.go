package ui

import (
	"math"
	"time"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type progressData struct {
	value         float32
	indeterminate bool
	ring          bool
}

type progressState struct {
	phase float32
}

// ProgressBar is a determinate (or indeterminate) linear progress indicator.
// value is in [0, 1].
func ProgressBar(id string, value float32) *Node {
	return &Node{kind: kindProgress, id: id, extra: &progressData{value: clampf(value, 0, 1)}}
}

// ProgressRing is a circular progress indicator. value is in [0, 1].
func ProgressRing(id string, value float32) *Node {
	return &Node{kind: kindProgress, id: id, extra: &progressData{value: clampf(value, 0, 1), ring: true}}
}

// Indeterminate switches ProgressBar/ProgressRing to an animated sweep.
func (n *Node) Indeterminate() *Node {
	if d, ok := n.extra.(*progressData); ok {
		d.indeterminate = true
	}
	return n
}

func (n *Node) layoutProgress(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "progress")
	}
	d, _ := n.extra.(*progressData)
	if d == nil {
		d = &progressData{}
	}
	st := c.Widget(id, func() any { return &progressState{} }).(*progressState)
	th := c.Theme()

	if d.ring {
		sz := float32(32)
		if n.iconSize > 0 {
			sz = n.iconSize
		} else if n.spec.hasW {
			sz = n.spec.width
		} else if n.spec.hasH {
			sz = n.spec.height
		}
		el := layout.New(applyLayoutSpec(layout.Box().Size(sz, sz).FlexShrink(0), n.spec))
		value := d.value
		indet := d.indeterminate
		if indet {
			st.phase += 0.08
			c.Animate(16 * time.Millisecond)
		}
		el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
			paintProgressRing(dl, el.Frame, th.ChromeMuted, th.Accent, value, indet, st.phase)
		}
		return el
	}

	h := float32(6)
	w := float32(200)
	if n.spec.hasW {
		w = n.spec.width
	}
	if n.spec.hasH {
		h = n.spec.height
	}
	el := layout.New(applyLayoutSpec(layout.Box().W(w).H(h).FlexShrink(0), n.spec))
	value := d.value
	indet := d.indeterminate
	if indet {
		st.phase += 0.04
		if st.phase > 1 {
			st.phase -= 1
		}
		c.Animate(16 * time.Millisecond)
	}
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		f := el.Frame
		r := f.H / 2
		dl.AddRoundedRect(f, r, th.ChromeMuted)
		if indet {
			barW := f.W * 0.35
			x := f.X + (f.W+barW)*st.phase - barW
			seg := render.Rect{X: x, Y: f.Y, W: barW, H: f.H}
			dl.PushClip(f)
			dl.AddRoundedRect(seg, r, th.Accent)
			dl.PopClip()
			return
		}
		fw := f.W * clampf(value, 0, 1)
		if fw > 0 {
			dl.AddRoundedRect(render.Rect{X: f.X, Y: f.Y, W: fw, H: f.H}, r, th.Accent)
		}
	}
	return el
}

func paintProgressRing(dl *render.DrawList, f render.Rect, track, accent render.Color, value float32, indet bool, phase float32) {
	cx := f.X + f.W/2
	cy := f.Y + f.H/2
	radius := f.W/2 - 2
	const segments = 48
	stroke := float32(3)

	// Track circle as small rects (no native arc API).
	for i := 0; i < segments; i++ {
		a0 := float64(i) / segments * 2 * math.Pi
		x := cx + radius*float32(math.Cos(a0))
		y := cy + radius*float32(math.Sin(a0))
		dl.AddRect(render.Rect{X: x - stroke/2, Y: y - stroke/2, W: stroke, H: stroke}, track)
	}

	start := -math.Pi / 2
	span := float64(clampf(value, 0, 1)) * 2 * math.Pi
	if indet {
		start = float64(phase)
		span = math.Pi * 0.75
	}
	nSeg := int(span / (2 * math.Pi) * segments)
	if nSeg < 1 && (value > 0 || indet) {
		nSeg = 1
	}
	for i := 0; i < nSeg; i++ {
		a := start + float64(i)/segments*2*math.Pi
		x := cx + radius*float32(math.Cos(a))
		y := cy + radius*float32(math.Sin(a))
		dl.AddRect(render.Rect{X: x - stroke/2, Y: y - stroke/2, W: stroke, H: stroke}, accent)
	}
}
