package widgets

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Tab struct {
	clickable *widget.Clickable

	IsSelected bool

	BackgroundColor color.NRGBA
	IndicatorColor  color.NRGBA
	Text            string
}

func NewTab(text string, clickable *widget.Clickable) *Tab {
	return &Tab{
		Text:      text,
		clickable: clickable,
	}
}

func (t *Tab) Layout(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	border := widget.Border{
		Color:        Gray300,
		Width:        1,
		CornerRadius: 0,
	}

	return t.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					background := t.BackgroundColor
					switch {
					case gtx.Queue == nil:
						background = Disabled(t.BackgroundColor)
					case t.clickable.Hovered() || t.clickable.Focused():
						if !t.IsSelected {
							background = Hovered(t.BackgroundColor)
						}
					}
					paint.Fill(gtx.Ops, background)
					return layout.Dimensions{Size: gtx.Constraints.Min}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !t.IsSelected {
								return layout.Dimensions{Size: image.Pt(1, 5)}
							}
							return Rect{
								Color: t.IndicatorColor,
								Size:  f32.Point{X: 200.0, Y: 5},
								Radii: 0,
							}.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return material.Label(theme, unit.Sp(13), t.Text).Layout(gtx)
								})
							})
						}),
					)
				},
			)
		})
	})
}
