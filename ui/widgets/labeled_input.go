package widgets

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/theme"
)

type LabeledInput struct {
	Label          string
	SpaceBetween   int
	MinEditorWidth unit.Dp
	MinLabelWidth  unit.Dp
	Editor         *widget.Editor
}

func (l *LabeledInput) Layout(gtx layout.Context, theme *theme.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(l.SpaceBetween)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(l.MinLabelWidth)
				return material.Label(theme.Material(), theme.TextSize, l.Label).Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(l.MinEditorWidth)
			return widget.Border{
				Color:        Gray600,
				Width:        unit.Dp(1),
				CornerRadius: unit.Dp(4),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Editor(theme.Material(), l.Editor, "            ").Layout(gtx)
				})
			})
		}),
	)
}
