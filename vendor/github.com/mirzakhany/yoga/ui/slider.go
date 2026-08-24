package ui

import (
	"math"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type sliderData struct {
	value, min, max, step float64
	onChange              func(float64)
}

type sliderState struct {
	hovered, dragging, focused bool
	el                         *layout.Element
	setValue                   func(float64)
	min, max, step, value      float64
}

func (s *sliderState) Focus()                   { s.focused = true }
func (s *sliderState) Blur()                    { s.focused = false }
func (s *sliderState) Focused() bool            { return s.focused }
func (s *sliderState) HandleText(_ []rune)      {}
func (s *sliderState) CapturesTab() bool        { return false }
func (s *sliderState) FocusOnClick() bool       { return true }
func (s *sliderState) FocusEl() *layout.Element { return s.el }

func (s *sliderState) HandleKeys(keys []input.KeyEvent) {
	if !s.focused || s.setValue == nil {
		return
	}
	step := s.step
	if step <= 0 {
		step = (s.max - s.min) / 100
		if step <= 0 {
			step = 1
		}
	}
	for _, ev := range keys {
		if ev.Mods != 0 {
			continue
		}
		switch ev.Key {
		case input.KeyLeft, input.KeyDown:
			s.setValue(s.cur() - step)
		case input.KeyRight, input.KeyUp:
			s.setValue(s.cur() + step)
		case input.KeyHome:
			s.setValue(s.min)
		case input.KeyEnd:
			s.setValue(s.max)
		}
	}
}

func (s *sliderState) cur() float64 { return s.value }

var _ Focusable = (*sliderState)(nil)

// Slider is a controlled numeric range control. value is clamped to [Min, Max].
func Slider(id string, value float64) *Node {
	return &Node{kind: kindSlider, id: id, extra: &sliderData{value: value, min: 0, max: 100, step: 1}}
}

// Min sets the Slider/NumberStepper lower bound.
func (n *Node) Min(v float64) *Node {
	switch d := n.extra.(type) {
	case *sliderData:
		d.min = v
	case *stepperData:
		d.min = v
	}
	return n
}

// Max sets the Slider/NumberStepper upper bound.
func (n *Node) Max(v float64) *Node {
	switch d := n.extra.(type) {
	case *sliderData:
		d.max = v
	case *stepperData:
		d.max = v
	}
	return n
}

// Step sets the Slider/NumberStepper increment.
func (n *Node) Step(v float64) *Node {
	switch d := n.extra.(type) {
	case *sliderData:
		d.step = v
	case *stepperData:
		d.step = v
	}
	return n
}

// OnFloatChange sets the Slider/NumberStepper change handler.
func (n *Node) OnFloatChange(fn func(float64)) *Node {
	switch d := n.extra.(type) {
	case *sliderData:
		d.onChange = fn
	case *stepperData:
		d.onChange = fn
	}
	return n
}

func (n *Node) layoutSlider(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "slider")
	}
	d, _ := n.extra.(*sliderData)
	if d == nil {
		d = &sliderData{min: 0, max: 100, step: 1}
	}
	st := c.Widget(id, func() any { return &sliderState{} }).(*sliderState)
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	trackH := float32(4)
	thumbR := float32(8)
	h := f32max(th.Metrics.ControlHeight, thumbR*2+4)
	w := float32(200)
	if n.spec.hasW {
		w = n.spec.width
	}
	el := layout.New(applyLayoutSpec(layout.Box().W(w).H(h).FlexShrink(0), n.spec))
	st.el = el
	st.min, st.max, st.step = d.min, d.max, d.step
	if st.max <= st.min {
		st.max = st.min + 1
	}
	value := clampFloat(d.value, st.min, st.max)
	st.value = value
	onChange := d.onChange
	st.setValue = func(v float64) {
		v = snapFloat(clampFloat(v, st.min, st.max), st.min, st.step)
		st.value = v
		if onChange != nil {
			onChange(v)
		}
	}

	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		f := el.Frame
		ty := f.Y + (f.H-trackH)/2
		track := render.Rect{X: f.X + thumbR, Y: ty, W: f.W - 2*thumbR, H: trackH}
		dl.AddRoundedRect(track, trackH/2, th.ChromeMuted)
		t := float32(0)
		if st.max > st.min {
			t = float32((st.value - st.min) / (st.max - st.min))
		}
		fillW := track.W * t
		if fillW > 0 {
			dl.AddRoundedRect(render.Rect{X: track.X, Y: track.Y, W: fillW, H: track.H}, trackH/2, th.Accent)
		}
		tx := track.X + fillW
		cy := f.Y + f.H/2
		thumb := render.Rect{X: tx - thumbR, Y: cy - thumbR, W: thumbR * 2, H: thumbR * 2}
		col := th.Accent
		if st.hovered || st.dragging {
			col = th.AccentHover
		}
		dl.AddRoundedRect(thumb, thumbR, col)
		if st.focused {
			paintFocusRing(dl, thumb, col, th)
		}
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		f := e.Frame
		trackX0 := f.X + thumbR
		trackW := f.W - 2*thumbR
		inside := e.Frame.Contains(m.X, m.Y)
		st.hovered = inside || st.dragging
		if inside {
			m.SetCursor(CursorPointer)
		}
		fromX := func(x float32) float64 {
			if trackW <= 0 {
				return st.min
			}
			t := float64((x - trackX0) / trackW)
			return st.min + clampFloat(t, 0, 1)*(st.max-st.min)
		}
		if m.Pressed && inside {
			st.dragging = true
			st.setValue(fromX(m.X))
			m.Consumed = true
		}
		if st.dragging && m.Down {
			st.setValue(fromX(m.X))
			m.Consumed = true
		}
		if m.Released {
			st.dragging = false
		}
	}
	return el
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func snapFloat(v, min, step float64) float64 {
	if step <= 0 {
		return v
	}
	n := math.Round((v - min) / step)
	return min + n*step
}
