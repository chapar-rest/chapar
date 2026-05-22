package widget

import (
	"gioui.org/font"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/oligo/gvcode"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/textstyle/syntax"
)

// NewEditor is a helper function to setup a editor with the
// provided theme.
func NewEditor(th *material.Theme) *gvcode.Editor {
	editor := &gvcode.Editor{}

	colorScheme := syntax.ColorScheme{}
	colorScheme.Foreground = gvcolor.MakeColor(th.Fg)
	colorScheme.Background = gvcolor.MakeColor(th.Bg)
	colorScheme.SelectColor = gvcolor.MakeColor(th.ContrastBg).MulAlpha(0x60)
	colorScheme.LineColor = gvcolor.MakeColor(th.ContrastBg).MulAlpha(0x30)
	colorScheme.LineNumberColor = gvcolor.MakeColor(th.Fg).MulAlpha(0xb6)

	editor.WithOptions(
		gvcode.WrapLine(false),
		gvcode.WithFont(font.Font{Typeface: th.Face}),
		gvcode.WithTextSize(th.TextSize),
		gvcode.WithTextAlignment(text.Start),
		gvcode.WithLineHeight(0, 1.2),
		gvcode.WithTabWidth(4),
		gvcode.WithLineNumberGutterGap(unit.Dp(24)),
		gvcode.WithColorScheme(colorScheme),
	)

	return editor
}
