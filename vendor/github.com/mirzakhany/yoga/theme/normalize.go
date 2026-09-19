package theme

import "github.com/mirzakhany/yoga/render"

// colorUnset reports whether c looks like a zero-value color token.
func colorUnset(c render.Color) bool {
	return c.A == 0 && c.R == 0 && c.G == 0 && c.B == 0
}

// normalize fills missing Yoga tokens from legacy fields and applies shared
// defaults for spacing, radius, stroke, typography, elevation, and metrics.
func normalize(t *Theme) {
	// Legacy -> Yoga when Yoga tokens are unset.
	if colorUnset(t.Surface) && !colorUnset(t.Background) {
		t.Surface = t.Background
	}
	if colorUnset(t.Chrome) && !colorUnset(t.Panel) {
		t.Chrome = t.Panel
	}
	if colorUnset(t.ChromeMuted) && !colorUnset(t.PanelAlt) {
		t.ChromeMuted = t.PanelAlt
	}
	if colorUnset(t.Foreground) && !colorUnset(t.Text) {
		t.Foreground = t.Text
	}
	if colorUnset(t.ForegroundMuted) && !colorUnset(t.TextDim) {
		t.ForegroundMuted = t.TextDim
	}
	if colorUnset(t.ForegroundSubtle) && !colorUnset(t.TextDim) {
		t.ForegroundSubtle = t.TextDim
	}
	if colorUnset(t.ForegroundDisabled) && !colorUnset(t.TextDim) {
		c := t.TextDim
		c.A *= 0.6
		t.ForegroundDisabled = c
	}
	if colorUnset(t.AccentHover) && !colorUnset(t.Accent) {
		t.AccentHover = lighten(t.Accent, 0.08)
	}
	if colorUnset(t.AccentPressed) && !colorUnset(t.Active) {
		t.AccentPressed = t.Active
	}
	if colorUnset(t.AccentForeground) && !colorUnset(t.AccentText) {
		t.AccentForeground = t.AccentText
	}
	if colorUnset(t.BorderStrong) && !colorUnset(t.Border) {
		t.BorderStrong = t.Border
	}
	if colorUnset(t.ListHover) && !colorUnset(t.Hover) {
		t.ListHover = t.Hover
	}
	if colorUnset(t.ListActive) && !colorUnset(t.Active) {
		t.ListActive = t.Active
	}
	if colorUnset(t.FocusRing) && !colorUnset(t.Accent) {
		t.FocusRing = t.Accent
	}

	syncLegacyFromYoga(t)

	if t.Spacing == (Spacing{}) {
		t.Spacing = DefaultSpacing()
	}
	if t.Radius == (Radius{}) {
		t.Radius = DefaultRadius()
	}
	if t.Stroke == (Stroke{}) {
		t.Stroke = DefaultStroke()
	}
	if t.Typography == (Typography{}) {
		t.Typography = DefaultTypography()
	}
	if t.Metrics == (ComponentMetrics{}) {
		t.Metrics = DefaultComponentMetrics()
	}
	if t.Elevation == (Elevation{}) {
		if t.Dark {
			t.Elevation = DefaultElevationDark()
		} else {
			t.Elevation = DefaultElevationLight()
		}
	}
}

// syncLegacyFromYoga copies Yoga tokens into legacy alias fields.
func syncLegacyFromYoga(t *Theme) {
	if !colorUnset(t.Surface) {
		t.Background = t.Surface
	}
	if !colorUnset(t.Chrome) {
		t.Panel = t.Chrome
	}
	if !colorUnset(t.ChromeMuted) {
		t.PanelAlt = t.ChromeMuted
	}
	if !colorUnset(t.Foreground) {
		t.Text = t.Foreground
	}
	if !colorUnset(t.ForegroundMuted) {
		t.TextDim = t.ForegroundMuted
	}
	if !colorUnset(t.AccentForeground) {
		t.AccentText = t.AccentForeground
	}
	if !colorUnset(t.ListHover) {
		t.Hover = t.ListHover
	}
	if !colorUnset(t.ListActive) {
		t.Active = t.ListActive
	}
}

func lighten(c render.Color, amount float32) render.Color {
	return render.Color{
		R: c.R + (1-c.R)*amount,
		G: c.G + (1-c.G)*amount,
		B: c.B + (1-c.B)*amount,
		A: c.A,
	}
}
