package theme

import (
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/syntax"
)

// Ramp is a 12-step neutral scale from the deepest background (step 0) to
// the strongest text (step 11). FromPalette reads fixed steps as roles, so a
// theme author picks twelve grays and every surface, border, state and text
// color keeps the same relationship to the others in every theme built this
// way:
//
//	0  scroll tracks, wells             6  hovered row
//	1  Surface: editor, workspace       7  pressed row, strong borders
//	2  Chrome: sidebars, tab strips     8  subtle icons, scroll thumb
//	3  ChromeMuted: secondary chrome    9  muted text (4.5:1 on 1-4)
//	4  Overlay: menus, popovers        10  disabled text, before its alpha
//	5  Border: dividers, card edges    11  text
//
// In a dark theme the steps get lighter from 0 to 11; in a light theme step 0
// and 2-3 are darker than the white Surface and Overlay, and 5-11 get darker
// toward the text.
type Ramp [12]render.Color

// Palette is the input to FromPalette: the few colors a theme author chooses.
// Everything else (selection, focus ring, link, status surfaces, editor
// washes, Selected, hover layers) is derived to meet the contrast targets the
// builtin themes are tested against.
type Palette struct {
	Name string
	Dark bool

	Neutral Ramp

	// Accent is the primary fill. AccentHover and AccentPressed are optional:
	// left zero, they are stepped from Accent in whichever direction keeps
	// the label on it at 4.5:1.
	Accent        render.Color
	AccentHover   render.Color
	AccentPressed render.Color

	Error, Warning, Success render.Color

	// Syntax is optional; classes it leaves out fall back to Foreground.
	Syntax map[syntax.ColorClass]render.Color
}

// FromPalette builds a theme from a neutral ramp, an accent and three status
// colors. Register the result (or call Use after registering) so the derived
// tokens are filled in.
func FromPalette(p Palette) Theme {
	t := baseTheme(p.Name, p.Dark)
	n := p.Neutral

	t.ScrollTrack = n[0]
	t.Surface = n[1]
	t.Chrome = n[2]
	t.ChromeMuted = n[3]
	t.Overlay = n[4]
	t.Border = n[5]
	t.ListHover = n[6]
	t.ListActive = n[7]
	t.BorderStrong = n[7]
	t.ForegroundSubtle = n[8]
	t.ScrollThumb = n[8]
	t.ForegroundMuted = n[9]
	t.ForegroundDisabled = alpha(n[10], 0.5)
	t.Foreground = n[11]

	t.Accent = p.Accent
	t.AccentForeground = onAccent(p.Accent)
	t.AccentHover = p.AccentHover
	if colorUnset(t.AccentHover) {
		t.AccentHover = stepAccent(p.Accent, t.AccentForeground, 0.08)
	}
	t.AccentPressed = p.AccentPressed
	if colorUnset(t.AccentPressed) {
		t.AccentPressed = stepAccent(p.Accent, t.AccentForeground, -0.14)
	}
	t.ScrollThumbHover = p.Accent
	amount := float32(0.22)
	if p.Dark {
		amount = 0.32
	}
	t.Selection = highlightFill(p.Accent, t.Surface, t.Foreground, amount)

	t.Error, t.Warning, t.Success = p.Error, p.Warning, p.Success
	if p.Syntax != nil {
		t.Syntax = p.Syntax
	}
	return finishTheme(t)
}

// onAccent returns the label color for an accent fill: white, or near-black
// when white would not clear 4.5:1.
func onAccent(accent render.Color) render.Color {
	if ContrastRatio(white, accent) >= ContrastText {
		return white
	}
	if ContrastRatio(black, accent) > ContrastRatio(white, accent) {
		return black
	}
	return white
}

// stepAccent moves accent by amount (positive lightens, negative darkens)
// for a hover or press state. If the step would drop the label below 4.5:1 it
// goes the other way, so states never cost the label its contrast.
func stepAccent(accent, label render.Color, amount float32) render.Color {
	move := func(a float32) render.Color {
		if a >= 0 {
			return lighten(accent, a)
		}
		return darken(accent, -a)
	}
	c := move(amount)
	if ContrastRatio(label, c) < ContrastText {
		c = move(-amount)
	}
	return c
}

// darken moves c toward black by amount.
func darken(c render.Color, amount float32) render.Color {
	k := 1 - amount
	return render.Color{R: c.R * k, G: c.G * k, B: c.B * k, A: c.A}
}
