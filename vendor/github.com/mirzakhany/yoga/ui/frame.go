package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// BuildFrame runs one build pass: it resets the context with the frame's
// viewport and input, builds the body View, composes any registered overlays
// into a synthetic root, and solves layout.
func BuildFrame(c *Ctx, body func(*Ctx) View, w, h float32, m *input.Mouse, kb *input.Keyboard) *layout.Element {
	c.BeginFrame(w, h, m, kb)
	var root *layout.Element
	if v := body(c); v != nil {
		root = v.Layout(c)
	}
	c.layoutWindowOverlays()
	if c.Focus() != nil {
		c.Focus().finishBuild()
	}
	if root == nil {
		root = layout.New(layout.Box().FlexGrow(1))
	}
	root = compose(root, c.overlays)
	root.Calculate(w, h)
	return root
}

func compose(body *layout.Element, overlays []*layout.Element) *layout.Element {
	if len(overlays) == 0 {
		return body
	}
	children := make([]*layout.Element, 0, len(overlays)+1)
	children = append(children, body)
	children = append(children, overlays...)
	return layout.New(layout.Box().FlexGrow(1), children...)
}
