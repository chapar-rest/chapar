package widgets

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

func SaveButtonLayout(gtx layout.Context, theme *chapartheme.Theme, clickable *widget.Clickable) layout.Dimensions {
	border := widget.Border{
		Color:        theme.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return layout.Inset{Left: unit.Dp(15)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return Clickable(gtx, clickable, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Max.X = gtx.Dp(16)
									return SaveIcon.Layout(gtx, theme.Palette.ContrastFg)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.Body1(theme.Material(), "Save").Layout(gtx)
								}),
							)
						}),
					)
				})
			})
		})
	})
}
