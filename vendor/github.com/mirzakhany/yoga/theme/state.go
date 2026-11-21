package theme

import "github.com/mirzakhany/yoga/render"

// State layers. ListHover and ListActive are fills chosen for rows on Chrome;
// a row on any other surface (a menu on Overlay, a list on Surface) needs the
// same feedback relative to its own background. HoverOn and PressedOn build
// that fill as a translucent layer of Foreground composited over bg, at the
// strength the theme's own hover and press already have on Chrome. A theme's
// hand-picked colors are kept exactly where they were designed to be used.

const (
	// minHoverStep and minPressStep are the weakest feedback a layer may
	// give, whatever the theme's own colors measure.
	minHoverStep = 1.15
	minPressStep = 1.12 // pressed over hovered
	// maxHoverStep caps the copied hover strength: a theme whose ListHover
	// is bold against Chrome still gets a quiet wash in menus and lists.
	maxHoverStep = 1.30
	// maxLayerAlpha bounds the search; no theme needs a heavier wash.
	maxLayerAlpha = 0.5
)

// HoverOn returns the hover fill for an element whose resting background is
// bg. On Chrome it is ListHover itself.
func (t *Theme) HoverOn(bg render.Color) render.Color {
	if bg == t.Chrome || colorUnset(bg) {
		return t.ListHover
	}
	return t.layerOn(bg, t.hoverStep())
}

// PressedOn returns the pressed fill for an element whose resting background
// is bg. On Chrome it is ListActive itself.
func (t *Theme) PressedOn(bg render.Color) render.Color {
	if bg == t.Chrome || colorUnset(bg) {
		return t.ListActive
	}
	return t.layerOn(bg, t.pressStep())
}

// hoverStep is how far the theme's hover sits from Chrome.
func (t *Theme) hoverStep() float64 {
	return min(max(ContrastRatio(t.ListHover, t.Chrome), minHoverStep), maxHoverStep)
}

// pressStep is how far the theme's press sits from Chrome, and always at
// least one step beyond hover.
func (t *Theme) pressStep() float64 {
	h := t.hoverStep()
	return min(max(ContrastRatio(t.ListActive, t.Chrome), h*minPressStep), h*maxHoverStep)
}

// layerOn composites the lightest wash of Foreground over bg that reaches
// step:1 against bg while Foreground text on it still clears 4.5:1. If no
// wash reaches step without costing text contrast, it returns the strongest
// one that keeps the text readable.
func (t *Theme) layerOn(bg render.Color, step float64) render.Color {
	ink := t.Foreground
	if colorUnset(ink) {
		ink = awayFrom(bg)
	}
	best := bg
	for a := float32(0.02); a <= maxLayerAlpha; a += 0.01 {
		c := Flatten(alpha(ink, a), bg)
		if ContrastRatio(t.Foreground, c) < ContrastText {
			break
		}
		best = c
		if ContrastRatio(c, bg) >= step {
			return c
		}
	}
	return best
}
