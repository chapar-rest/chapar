package ui

import (
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/theme"
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

// paintFocusRing strokes the keyboard focus indicator around a control whose
// corner radius is radius. The ring sits outside rect, separated by a
// Stroke.Thin gap, so it is drawn against the surface behind the control rather
// than over the control's own fill or border: it reads on a primary button as
// well as on a ghost one, and never paints a fill of its own. Use it for
// free-standing controls; rows and tabs that fill a clipped strip use
// paintFocusRingInset instead.
func paintFocusRing(dl *render.DrawList, rect render.Rect, radius float32, t *theme.Theme) {
	w := t.Stroke.Thick
	out := t.Stroke.Thin + w
	ring := render.Rect{X: rect.X - out, Y: rect.Y - out, W: rect.W + 2*out, H: rect.H + 2*out}
	dl.AddRoundedRectStroke(ring, render.UniformCorners(radius+out), render.UniformBorder(w),
		t.FocusRing, render.BorderSolid)
}

// PaintFocusRingInset strokes the focus indicator just inside rect, for
// targets that span a clipped strip (tabs, accordion headers, list rows) where
// an outside ring would be cut off or overlap a neighbor. fill is the color
// under the ring, so FocusRingOn can pick a ring that clears it.
func PaintFocusRingInset(dl *render.DrawList, rect render.Rect, radius float32, fill render.Color, t *theme.Theme) {
	dl.AddRoundedRectStroke(rect, render.UniformCorners(radius), render.UniformBorder(t.Stroke.Thick),
		t.FocusRingOn(fill), render.BorderSolid)
}

// DrawElevationShadow approximates an elevation shadow behind rect.
func DrawElevationShadow(dl *render.DrawList, rect render.Rect, radius float32, shadow theme.Shadow) {
	dl.AddElevationShadow(rect, radius, render.Shadow{
		OffsetX: shadow.OffsetX,
		OffsetY: shadow.OffsetY,
		Blur:    shadow.Blur,
		Color:   shadow.Color,
	})
}
