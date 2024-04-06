package theme

import (
	"image/color"

	"gioui.org/widget/material"
)

var (
	Gray300 = color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff}
	Gray400 = color.NRGBA{R: 0x3c, G: 0x3f, B: 0x46, A: 0xff}
	Gray600 = color.NRGBA{R: 0x6c, G: 0x6f, B: 0x76, A: 0xff}
	Gray700 = color.NRGBA{R: 0x8b, G: 0x8e, B: 0x95, A: 0xff}
	Gray800 = color.NRGBA{R: 0xb0, G: 0xb3, B: 0xb8, A: 0xff}

	White       = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	Black       = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
	LightGreen  = color.NRGBA{R: 0x8b, G: 0xc3, B: 0x4a, A: 0xff}
	LightRed    = color.NRGBA{R: 0xff, G: 0x73, B: 0x73, A: 0xff}
	LightYellow = color.NRGBA{R: 0xff, G: 0xe0, B: 0x73, A: 0xff}
)

type Theme struct {
	*material.Theme

	isLight bool
}

func (t *Theme) Material() *material.Theme {
	return t.Theme
}

func (t *Theme) Switch(light bool) *material.Theme {
	t.isLight = light

	if light {
		// set foreground color
		t.Theme.Palette.Fg = color.NRGBA{R: 0xD7, G: 0xDA, B: 0xDE, A: 0xff}
		// set background color
		t.Theme.Palette.Bg = color.NRGBA{R: 0x20, G: 0x22, B: 0x24, A: 0xff}
	} else {
		// set foreground color
		t.Theme.Palette.Fg = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
		// set background color
		t.Theme.Palette.Bg = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff}
	}
	return t.Theme
}

func (t *Theme) IsLight() bool {
	return t.isLight
}

func (t *Theme) GetBorderColor() color.NRGBA {
	if t.isLight {
		return Gray400
	} else {
		return Gray600
	}
}
