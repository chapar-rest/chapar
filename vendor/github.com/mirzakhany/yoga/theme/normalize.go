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
	// FocusRing is deliberately not aliased to Accent here; deriveTokens fits it
	// against the surfaces it is painted on instead.

	deriveTokens(t)

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

// deriveTokens fills the tokens that are defined by a contrast relationship
// rather than by taste: control outlines, the focus ring, status text and
// fills, and the editor washes. A palette author sets the handful of brand
// colors; everything derived here is computed so it passes WCAG against the
// surfaces it is actually painted on. Explicit values are never overwritten,
// so any theme can opt out token by token.
func deriveTokens(t *Theme) {
	if t.LightSibling == "" {
		t.LightSibling = t.Name
	}
	if t.DarkSibling == "" {
		t.DarkSibling = t.Name
	}
	if colorUnset(t.Surface) {
		return // nothing to derive against
	}
	surface, chrome := t.Surface, t.Chrome
	if colorUnset(chrome) {
		chrome = surface
	}

	// Control boundaries must stay visible; Border stays free to be decorative.
	if colorUnset(t.BorderControl) {
		base := t.BorderStrong
		if colorUnset(base) {
			base = t.Border
		}
		if colorUnset(base) {
			base = t.ForegroundMuted
		}
		t.BorderControl = fitAll(base, ContrastNonText, surface, chrome)
	}

	// The ring reads against the surfaces behind a control. It cannot also be
	// guaranteed against every fill a control can have — an accent ring on an
	// accent button is the classic invisible-focus bug — so FocusRingInverse
	// covers the complement, and FocusRingOn picks between them per control.
	if colorUnset(t.FocusRing) {
		t.FocusRing = fitAll(t.Accent, ContrastNonText, surface, chrome)
	}
	if colorUnset(t.FocusRingInverse) {
		t.FocusRingInverse = complementRing(t.FocusRing, []render.Color{
			t.Accent, t.AccentHover, t.AccentPressed, t.ListActive, t.ChromeMuted, chrome,
		})
	}

	if colorUnset(t.Info) {
		t.Info = t.Accent
	}
	if colorUnset(t.Link) {
		t.Link = fitAll(t.Accent, ContrastText, surface, chrome)
	}
	if colorUnset(t.LinkHover) {
		// A link brightens or deepens under the pointer without dropping below
		// text contrast, so hovering never makes it harder to read.
		t.LinkHover = fitAll(mix(t.Link, awayFrom(surface), 0.25), ContrastText, surface, chrome)
	}
	if colorUnset(t.LinkVisited) {
		t.LinkVisited = fitAll(mix(t.Link, t.Foreground, 0.35), ContrastText, surface, chrome)
	}

	// Status fills, then the text that sits on them. Dark surfaces need a
	// heavier mix before a hue registers.
	fillAmount := float32(0.14)
	if t.Dark {
		fillAmount = 0.22
	}
	status := []struct {
		base, surf, fg *render.Color
	}{
		{&t.Error, &t.ErrorSurface, &t.ErrorForeground},
		{&t.Warning, &t.WarningSurface, &t.WarningForeground},
		{&t.Success, &t.SuccessSurface, &t.SuccessForeground},
		{&t.Info, &t.InfoSurface, &t.InfoForeground},
	}
	for _, s := range status {
		if colorUnset(*s.base) {
			continue
		}
		if colorUnset(*s.surf) {
			// Status fills carry text, so the tint backs off if it would push
			// the foreground under the text threshold.
			*s.surf = highlightFill(*s.base, surface, t.Foreground, fillAmount)
		}
		if colorUnset(*s.fg) {
			*s.fg = fitAll(*s.base, ContrastText, surface, chrome, *s.surf)
		}
	}

	if colorUnset(t.Scrim) {
		a := float32(0.45)
		if !t.Dark {
			a = 0.35
		}
		t.Scrim = alpha(black, a)
	}

	// Editor washes. Each is painted under live text, so highlightFill backs
	// off until the foreground still clears 4.5:1.
	fg := t.Foreground
	if colorUnset(t.SelectionInactive) && !colorUnset(t.Selection) {
		// A washed-out copy of the focused selection. Weakening it moves it
		// toward the surface and away from Selection, so the loop stops at the
		// strongest version that is still clearly the "unfocused" one.
		t.SelectionInactive = separate(t.Selection, t.Selection, surface, fg, 0.65)
	}
	if colorUnset(t.CurrentLine) {
		t.CurrentLine = alpha(fg, 0.07)
	}
	if colorUnset(t.IndentGuide) {
		guide := t.ForegroundSubtle
		if colorUnset(guide) {
			guide = t.ForegroundMuted
		}
		t.IndentGuide = alpha(guide, 0.35)
	}
	if colorUnset(t.Caret) {
		t.Caret = fitAll(t.Accent, ContrastNonText, surface, chrome)
	}
	if colorUnset(t.SearchMatch) && !colorUnset(t.Warning) {
		t.SearchMatch = highlightFill(t.Warning, surface, fg, 0.30)
	}
	if colorUnset(t.SearchMatchActive) && !colorUnset(t.Warning) {
		// The active hit must be findable among the others. Both washes are
		// capped by the text that sits on them, so the separation comes from
		// hue — a shift toward Error — rather than from going darker.
		t.SearchMatchActive = separate(
			mix(t.Warning, t.Error, 0.6), t.SearchMatch, surface, fg, 0.55)
	}
	if colorUnset(t.BracketMatch) && !colorUnset(t.Success) {
		t.BracketMatch = highlightFill(t.Success, surface, fg, 0.40)
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
