package theme

import (
	"fmt"
	"image/color"
	"reflect"

	"gioui.org/text"
	"gioui.org/widget/material"
)

var (
	DefaultHover    = 48
	DefaultSelected = 96
)

type Palette struct {
	// Bg is the background color atop which content is currently being
	// drawn.
	Bg color.NRGBA

	// Fg is a color suitable for drawing on top of Bg.
	Fg color.NRGBA

	// ContrastBg is a color used to draw attention to active,
	// important, interactive widgets such as buttons.
	ContrastBg color.NRGBA

	// ContrastFg is a color suitable for content drawn on top of
	// ContrastBg.
	ContrastFg color.NRGBA

	// Bg2 specifies the background color for components like navibar
	Bg2 color.NRGBA

	HoverAlpha, SelectedAlpha uint8
}

type SubThemeID string

type Theme struct {
	*material.Theme

	// Alpha is the set of alpha values to be applied for certain
	// states like hover, selected, etc...
	HoverAlpha, SelectedAlpha uint8

	// Bg2 specifies the background color for components like navibar
	Bg2 color.NRGBA

	// sub theme maps and their type info used to
	// accept dynamic subtheme registration.
	subThemes     map[SubThemeID]interface{}
	subThemeTypes map[SubThemeID]reflect.Type
}

// NewTheme instantiates a theme, extending material theme.
func NewTheme(fontDir string, embeddedFonts [][]byte, noSystemFonts bool) *Theme {
	th := material.NewTheme()

	var options = []text.ShaperOption{
		text.WithCollection(LoadBuiltin(fontDir, embeddedFonts)),
	}

	if noSystemFonts {
		options = append(options, text.NoSystemFonts())
	}

	th.Shaper = text.NewShaper(options...)

	theme := &Theme{
		Theme:         th,
		HoverAlpha:    uint8(DefaultHover),
		SelectedAlpha: uint8(DefaultSelected),
		Bg2:           th.Bg,
	}

	return theme
}

func (t *Theme) WithPalette(p Palette) *Theme {
	t.Theme.Palette = material.Palette{
		Bg:         p.Bg,
		Fg:         p.Fg,
		ContrastFg: p.ContrastFg,
		ContrastBg: p.ContrastBg,
	}

	if p.HoverAlpha > 0 {
		t.HoverAlpha = p.HoverAlpha
	}
	if p.SelectedAlpha > 0 {
		t.SelectedAlpha = p.SelectedAlpha
	}

	t.Bg2 = p.Bg2
	return t
}

func (th *Theme) Register(ID SubThemeID, sub interface{}) error {
	if th.subThemes == nil {
		th.subThemes = make(map[SubThemeID]interface{})
		th.subThemeTypes = make(map[SubThemeID]reflect.Type)
	}

	// confliction check
	if t, ok := th.subThemeTypes[ID]; ok {
		if t != reflect.TypeOf(sub) {
			return fmt.Errorf("type %v already registered as %s", ID, t.Name())
		}
	}

	th.subThemes[ID] = sub
	th.subThemeTypes[ID] = reflect.TypeOf(sub)
	return nil
}

func (th *Theme) Get(ID SubThemeID) interface{} {
	if _, exist := th.subThemeTypes[ID]; !exist {
		panic(fmt.Sprintf("%v not registered", ID))
	}

	return th.subThemes[ID]
}
