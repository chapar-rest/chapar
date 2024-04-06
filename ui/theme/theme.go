package theme

import (
	"image/color"

	"gioui.org/unit"

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

	isDark bool

	BorderColor        color.NRGBA
	BorderColorFocused color.NRGBA
	TextColor          color.NRGBA
}

func New(material *material.Theme, isDark bool) *Theme {
	t := &Theme{
		Theme: material,
	}

	t.Theme.TextSize = unit.Sp(14)
	// default theme is dark
	t.Switch(isDark)
	return t
}

func (t *Theme) Material() *material.Theme {
	return t.Theme
}

func (t *Theme) Switch(isDark bool) *material.Theme {
	t.isDark = isDark

	if isDark {
		// set foreground color
		t.Theme.Palette.Fg = color.NRGBA{R: 0xD7, G: 0xDA, B: 0xDE, A: 0xff}
		// set background color
		t.Theme.Palette.Bg = color.NRGBA{R: 0x20, G: 0x22, B: 0x24, A: 0xff}
		t.BorderColor = Gray300
		t.Theme.Palette.ContrastBg = color.NRGBA{R: 0x20, G: 0x22, B: 0x24, A: 0xff}
		t.Theme.Palette.ContrastFg = rgb(0xffffff)
		t.BorderColorFocused = t.Theme.Palette.ContrastFg
		t.TextColor = Gray700

	} else {

		t.Theme.Palette.Fg = rgb(0x000000)
		t.Theme.Palette.Bg = rgb(0xffffff)
		t.Theme.Palette.ContrastBg = Gray700
		t.Theme.Palette.ContrastFg = rgb(0x000000)
		t.BorderColor = Gray400
		t.BorderColorFocused = t.Theme.Palette.ContrastBg
		t.TextColor = Black
	}

	return t.Theme
}

func (t *Theme) IsDark() bool {
	return t.isDark
}

func rgb(c uint32) color.NRGBA {
	return argb(0xff000000 | c)
}

func argb(c uint32) color.NRGBA {
	return color.NRGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}
