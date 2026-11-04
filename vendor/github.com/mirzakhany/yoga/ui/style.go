package ui

import (
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// Variant selects the Yoga control visual style.
type Variant int

const (
	VariantPrimary Variant = iota
	VariantSecondary
	VariantSubtle
	VariantTransparent
)

// State selects the interaction state for resolving control fills.
type State int

const (
	StateRest State = iota
	StateHover
	StatePressed
	StateDisabled
)

// drawFocusRing draws a Yoga focus stroke around rect, preserving fill as the
// interior background. fill must be opaque — a transparent fill lets the focus
// border color bleed through the entire rect.
func drawFocusRing(dl *render.DrawList, rect render.Rect, fill render.Color, t *theme.Theme) {
	r := t.Radius.Medium
	if r <= 0 {
		r = 4
	}
	dl.AddRoundedRectBorder(rect, r, t.Stroke.Thick, fill, t.FocusRingOn(fill))
}

// drawElevationShadow approximates an elevation shadow behind rect.
func drawElevationShadow(dl *render.DrawList, rect render.Rect, radius float32, shadow theme.Shadow) {
	dl.AddElevationShadow(rect, radius, render.Shadow{
		OffsetX: shadow.OffsetX,
		OffsetY: shadow.OffsetY,
		Blur:    shadow.Blur,
		Color:   shadow.Color,
	})
}
