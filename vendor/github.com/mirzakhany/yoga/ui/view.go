package ui

import "github.com/mirzakhany/yoga/layout"

// View is the universal widget interface. Body and every directive return a
// View; the runtime calls Layout once per build pass to materialize geometry.
type View interface {
	Layout(c *Ctx) *layout.Element
}
