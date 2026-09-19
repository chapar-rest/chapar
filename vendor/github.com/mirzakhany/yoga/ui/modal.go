package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// layoutModalPanel shows scrim, lays out inner content in a centered overlay panel,
// and inserts the panel immediately after the scrim in the overlay stack.
func layoutModalPanel(c *Ctx, scrim *Scrim, inner View, dw, dh float32) *layout.Element {
	vw, vh := c.Viewport()
	x := f32max(0, (vw-dw)/2)
	y := f32max(0, (vh-dh)/2)

	scrim.Show(0, 0, vw, vh)
	scrimAt := len(c.overlays)
	c.Overlay(scrim.host)

	if c.Focus() != nil {
		c.Focus().BeginModal()
	}
	th := c.Theme()
	var innerEl *layout.Element
	if inner != nil {
		innerEl = inner.Layout(c)
	}
	host := layout.New(layout.Box().Absolute(x, y).Size(dw, dh), innerEl)
	host.Overlay = true
	host.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		r := th.Radius.Large
		drawElevationShadow(dl, host.Frame, r, th.Elevation.ShadowLg)
	}
	host.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if e.Frame.Contains(m.X, m.Y) {
			m.ScrollY = 0
			m.ScrollX = 0
			m.Consumed = true
		}
	}
	insert := scrimAt + 1
	if insert >= len(c.overlays) {
		c.Overlay(host)
	} else {
		c.overlays = append(c.overlays[:insert], append([]*layout.Element{host}, c.overlays[insert:]...)...)
	}
	return host
}

func clampModalSize(vw, vh, wantW, wantH, minW, minH float32, th *theme.Theme) (dw, dh float32) {
	margin := float32(th.Spacing.XXXL) * 2
	dw = wantW
	dh = wantH
	if dw <= 0 {
		dw = 480
	}
	if dh <= 0 {
		dh = 360
	}
	dw = f32min(dw, vw-margin)
	dh = f32min(dh, vh-margin)
	if minW > 0 && dw < minW {
		dw = f32max(minW, vw-margin)
	}
	if minH > 0 && dh < minH {
		dh = f32max(minH, vh-margin)
	}
	return dw, dh
}
