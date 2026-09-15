package ui

import (
	"math"
	"time"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// spinnerFrame is the wake interval for a visible spinner. 60fps rebuilds the
// whole app (double layout + paint) for a 24px indicator; ~20fps is smooth
// enough and cuts that cost roughly 3×.
const spinnerFrame = 50 * time.Millisecond

// spinnerRadPerSec keeps rotation speed independent of the frame interval
// (legacy was +0.12 rad every 16ms ≈ 7.5 rad/s).
const spinnerRadPerSec = 0.12 / 0.016

type spinnerState struct {
	angle     float32
	lastTick  time.Time
	lastFrame render.Rect // previous paint; used to skip off-screen wakes
	haveFrame bool
}

func spinnerIntersectsViewport(f render.Rect, vw, vh float32) bool {
	if f.W <= 0 || f.H <= 0 {
		return false
	}
	return f.X+f.W > 0 && f.Y+f.H > 0 && f.X < vw && f.Y < vh
}

// Spinner is an indeterminate loading indicator. id keys rotation across frames.
func Spinner(id string, size float32) *Node {
	if size <= 0 {
		size = 24
	}
	return &Node{kind: kindSpinner, id: id, iconSize: size}
}

func (n *Node) layoutSpinner(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "spinner")
	}
	st := c.Widget(id, func() any { return &spinnerState{} }).(*spinnerState)
	vw, vh := c.Viewport()
	// Advance and request frames only when the spinner is (or has not yet
	// been) on screen. An off-screen demo spinner was pinning the whole app
	// at 16ms forever.
	visible := !st.haveFrame || spinnerIntersectsViewport(st.lastFrame, vw, vh)
	if visible {
		now := c.Now()
		dt := spinnerFrame.Seconds()
		if !st.lastTick.IsZero() {
			dt = now.Sub(st.lastTick).Seconds()
		}
		st.angle += float32(dt * spinnerRadPerSec)
		st.lastTick = now
		c.Animate(spinnerFrame)
	} else {
		st.lastTick = time.Time{}
	}

	sz := n.iconSize
	el := layout.New(applyLayoutSpec(layout.Box().Size(sz, sz).FlexShrink(0), n.spec))
	angle := st.angle
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		th := theme.Current()
		f := el.Frame
		st.lastFrame, st.haveFrame = f, true
		cx := f.X + f.W/2
		cy := f.Y + f.H/2
		r := f.W/2 - 2
		for i := 0; i < 12; i++ {
			t := float32(i) / 12
			a := angle + t*2*math.Pi
			col := th.Accent
			col.A *= 0.2 + 0.8*t
			x0 := cx + r*float32(math.Cos(float64(a)))
			y0 := cy + r*float32(math.Sin(float64(a)))
			dl.AddRect(render.Rect{X: x0 - 1, Y: y0 - 1, W: 2, H: 2}, col)
		}
		for i := 0; i < 8; i++ {
			t := float32(i) / 8
			a := angle + t*2*math.Pi
			col := th.Accent
			col.A *= 0.15 + 0.85*(1-t)
			outer := r
			inner := r - 3
			x0 := cx + inner*float32(math.Cos(float64(a)))
			y0 := cy + inner*float32(math.Sin(float64(a)))
			x1 := cx + outer*float32(math.Cos(float64(a)))
			y1 := cy + outer*float32(math.Sin(float64(a)))
			mx := (x0 + x1) / 2
			my := (y0 + y1) / 2
			dl.AddRect(render.Rect{X: mx - 1.5, Y: my - 1.5, W: 3, H: 3}, col)
		}
	}
	return el
}
