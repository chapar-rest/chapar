package widget

import (
	"gioui.org/font"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/oligo/gvcode"
)

// NewEditor is a helper function to setup a editor with the
// provided theme.
func NewEditor(th *material.Theme) *gvcode.Editor {
	editor := &gvcode.Editor{}

	editor.WithOptions(
		gvcode.WrapLine(false),
		gvcode.WithFont(font.Font{Typeface: th.Face}),
		gvcode.WithTextSize(th.TextSize),
		gvcode.WithTextAlignment(text.Start),
		gvcode.WithLineHeight(0, 1.2),
		gvcode.WithTabWidth(4),
		gvcode.WithLineNumberGutterGap(unit.Dp(24)),
	)

	return editor
}
