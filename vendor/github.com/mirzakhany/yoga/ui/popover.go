package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type popoverData struct {
	trigger, content View
	open             bool
	onOpenChange     func(bool)
	placement        Placement
}

type popoverState struct {
	triggerFrame render.Rect
}

// Popover is a click-anchored overlay with arbitrary content (no scrim).
// Open is controlled via .Open(v); dismiss via outside click or Escape.
//
// Width/Height on the Popover node size the overlay panel only — the trigger
// keeps its natural size and AlignSelf(Start) so parent Columns do not stretch
// it (stretch would make placeAnchor center far from the control).
func Popover(id string, trigger, content View) *Node {
	return &Node{
		kind: kindPopover,
		id:   id,
		extra: &popoverData{
			trigger:   trigger,
			content:   content,
			placement: PlacementBottom,
		},
	}
}

// Open sets whether a Popover or Disclosure is shown/expanded (controlled).
func (n *Node) Open(v bool) *Node {
	switch d := n.extra.(type) {
	case *popoverData:
		d.open = v
	case *disclosureData:
		d.open = v
	}
	return n
}

// OnOpenChange is called when the popover wants to open or close.
func (n *Node) OnOpenChange(fn func(bool)) *Node {
	if d, ok := n.extra.(*popoverData); ok {
		d.onOpenChange = fn
	}
	return n
}

// Placement sets the preferred side for a Popover.
func (n *Node) Placement(p Placement) *Node {
	if d, ok := n.extra.(*popoverData); ok {
		d.placement = p
	}
	return n
}

func (n *Node) layoutPopover(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "popover")
	}
	d, _ := n.extra.(*popoverData)
	if d == nil {
		d = &popoverData{placement: PlacementBottom}
	}
	st := c.Widget(id, func() any { return &popoverState{} }).(*popoverState)

	var triggerEl *layout.Element
	if d.trigger != nil {
		triggerEl = d.trigger.Layout(c)
	}
	// Trigger is content-sized. Do not apply Width/Height (those size the panel).
	// AlignSelf(Start) prevents parent Column stretch from widening the hit box
	// used as the placeAnchor rect.
	box := layout.Box().
		Direction(layout.Row).
		AlignItems(layout.AlignCenter).
		AlignSelf(layout.AlignStart).
		FlexShrink(0)
	el := layout.New(box)
	if triggerEl != nil {
		el.Children = []*layout.Element{triggerEl}
	}

	open := d.open
	onChange := d.onOpenChange
	placement := d.placement
	content := d.content

	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		st.triggerFrame = e.Frame
		if e.Frame.Contains(m.X, m.Y) && m.Released {
			m.Consumed = true
			if onChange != nil {
				onChange(!open)
			}
			c.MarkNeedsPaint()
		}
	}
	// Keep anchor fresh while the trigger is painted (scroll / layout shifts).
	el.Paint = func(_ *render.DrawList, _ *shape.Engine) {
		if el.Frame.W > 0 {
			st.triggerFrame = el.Frame
		}
	}

	if !open {
		return el
	}

	if kb := c.Keyboard(); kb != nil {
		for _, ev := range kb.Keys {
			if ev.Key == input.KeyEscape {
				if onChange != nil {
					onChange(false)
				}
				c.MarkNeedsPaint()
				break
			}
		}
	}

	th := c.Theme()
	pad := th.Spacing.M
	cw := float32(240)
	ch := float32(160)
	if n.spec.hasW {
		cw = n.spec.width
	}
	if n.spec.hasH {
		ch = n.spec.height
	}
	anchor := st.triggerFrame
	x, y := placeAnchorStart(anchor, cw, ch, placement)

	var innerEl *layout.Element
	if content != nil {
		innerEl = content.Layout(c)
	}
	host := layout.New(layout.Box().Absolute(x, y).Size(cw, ch).PaddingAll(pad), innerEl)
	host.Overlay = true
	host.Frame = render.Rect{X: x, Y: y, W: cw, H: ch}
	host.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		f := host.Frame
		r := th.Radius.Large
		drawElevationShadow(dl, f, r, th.Elevation.ShadowMd)
		dl.AddRoundedRectBorder(f, r, th.Stroke.Thin, th.Chrome, th.Border)
	}
	host.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if e.Frame.Contains(m.X, m.Y) {
			m.Consumed = true
			return
		}
		if m.Pressed && !anchor.Contains(m.X, m.Y) {
			if onChange != nil {
				onChange(false)
			}
			m.Consumed = true
		}
	}
	c.Overlay(host)
	return el
}
