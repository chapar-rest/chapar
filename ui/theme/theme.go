package theme

import (
	"image/color"

	"gioui.org/widget/material"
)

type Theme struct {
	*material.Theme

	isLight bool
}

func (t *Theme) Material() *material.Theme {
	return t.Theme
}

func (t *Theme) Switch() *material.Theme {
	if t.isLight {
		// set foreground color
		t.Theme.Palette.Fg = color.NRGBA{R: 0xD7, G: 0xDA, B: 0xDE, A: 0xff}
		// set background color
		t.Theme.Palette.Bg = color.NRGBA{R: 0x20, G: 0x22, B: 0x24, A: 0xff}
		t.isLight = false
		return t.Theme
	}
	// set foreground color
	t.Theme.Palette.Fg = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
	// set background color
	t.Theme.Palette.Bg = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff}
	t.isLight = true
	return t.Theme
}
