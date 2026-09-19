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

// resolveBg returns the background fill for a variant and state.
func resolveBg(t *theme.Theme, v Variant, s State) render.Color {
	if s == StateDisabled {
		return t.ChromeMuted
	}
	switch v {
	case VariantPrimary:
		switch s {
		case StatePressed:
			return t.AccentPressed
		case StateHover:
			return t.AccentHover
		default:
			return t.Accent
		}
	case VariantSecondary:
		switch s {
		case StatePressed:
			return t.ListActive
		case StateHover:
			return t.ListHover
		default:
			return t.ChromeMuted
		}
	case VariantSubtle:
		switch s {
		case StatePressed:
			return t.ListActive
		case StateHover:
			return t.ListHover
		default:
			return render.Color{} // transparent
		}
	case VariantTransparent:
		return render.Color{}
	default:
		return t.ChromeMuted
	}
}

// resolveFg returns the foreground color for a variant and state.
func resolveFg(t *theme.Theme, v Variant, s State) render.Color {
	if s == StateDisabled {
		return t.ForegroundDisabled
	}
	switch v {
	case VariantPrimary:
		return t.AccentForeground
	default:
		return t.Foreground
	}
}

// resolveButtonBorder returns the border color for a button variant and state.
// Primary and Subtle buttons have no visible border; Secondary buttons show a
// 1-px border that gives them definition without visual weight.
func resolveButtonBorder(t *theme.Theme, v Variant, s State) render.Color {
	if s == StateDisabled {
		return render.Color{} // no border when disabled
	}
	if v == VariantSecondary {
		if s == StatePressed {
			return t.BorderStrong
		}
		return t.Border
	}
	return render.Color{}
}

// drawFocusRing draws a Yoga focus stroke around rect, preserving fill as the
// interior background. fill must be opaque — a transparent fill lets the focus
// border color bleed through the entire rect.
func drawFocusRing(dl *render.DrawList, rect render.Rect, fill render.Color, t *theme.Theme) {
	r := t.Radius.Medium
	if r <= 0 {
		r = 4
	}
	dl.AddRoundedRectBorder(rect, r, t.Stroke.Thick, fill, t.FocusRing)
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
