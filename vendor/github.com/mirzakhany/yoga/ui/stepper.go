package ui

import (
	"fmt"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type stepperData struct {
	value, min, max, step float64
	onChange              func(float64)
}

type stepperState struct {
	focused               bool
	disabled              bool
	el                    *layout.Element
	dec, inc              func()
	value, min, max, step float64
	onChange              func(float64)
}

func (s *stepperState) Focus()                   { s.focused = true }
func (s *stepperState) Blur()                    { s.focused = false }
func (s *stepperState) Focused() bool            { return s.focused }
func (s *stepperState) HandleText(_ []rune)      {}
func (s *stepperState) CapturesTab() bool        { return false }
func (s *stepperState) FocusOnClick() bool       { return true }
func (s *stepperState) FocusEl() *layout.Element { return s.el }

func (s *stepperState) HandleKeys(keys []input.KeyEvent) {
	if !s.focused || s.disabled {
		return
	}
	for _, ev := range keys {
		if ev.Mods != 0 {
			continue
		}
		switch ev.Key {
		case input.KeyLeft, input.KeyDown:
			if s.dec != nil {
				s.dec()
			}
		case input.KeyRight, input.KeyUp:
			if s.inc != nil {
				s.inc()
			}
		}
	}
}

func (s *stepperState) apply(delta float64) {
	v := snapFloat(clampFloat(s.value+delta, s.min, s.max), s.min, s.step)
	s.value = v
	if s.onChange != nil {
		s.onChange(v)
	}
}

var _ Focusable = (*stepperState)(nil)

// NumberStepper is a minus / value / plus control for numeric values.
func NumberStepper(id string, value float64) *Node {
	return &Node{kind: kindStepper, id: id, extra: &stepperData{value: value, min: 0, max: 100, step: 1}}
}

func (n *Node) layoutStepper(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "stepper")
	}
	d, _ := n.extra.(*stepperData)
	if d == nil {
		d = &stepperData{min: 0, max: 100, step: 1}
	}
	st := c.Widget(id, func() any { return &stepperState{} }).(*stepperState)
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	st.min, st.max, st.step = d.min, d.max, d.step
	if st.max <= st.min {
		st.max = st.min + 1
	}
	if st.step <= 0 {
		st.step = 1
	}
	st.value = clampFloat(d.value, st.min, st.max)
	st.onChange = d.onChange
	st.dec = func() { st.apply(-st.step) }
	st.inc = func() { st.apply(st.step) }

	label := formatStepper(st.value)
	dec := IconButton(id+"-dec", icons.Minus).OnClick(st.dec)
	inc := IconButton(id+"-inc", icons.Plus).OnClick(st.inc)
	if n.disabled {
		dec = dec.Disabled(true)
		inc = inc.Disabled(true)
	}
	val := Text(label).Style(Spec{}.TextColor(TokenForeground))
	if n.disabled {
		val = val.Style(Spec{}.TextColor(TokenForegroundDisabled))
	}

	row := Row(dec, val, inc).Gap(th.Spacing.S).Align(AlignCenter)
	el := row.Style(n.spec).Layout(c)
	st.el = el
	st.disabled = n.disabled
	if n.disabled {
		st.focused = false
	}
	if st.focused {
		prev := el.Paint
		el.Paint = func(dl *render.DrawList, text *shape.Engine) {
			if prev != nil {
				prev(dl, text)
			}
			paintFocusRing(dl, el.Frame, th.Surface, th)
		}
	}
	return el
}

func formatStepper(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%g", v)
}
