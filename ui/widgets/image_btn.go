package widgets

import (
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type ImageButton struct {
	Image     paint.ImageOp
	Clickable widget.Clickable
	Title     string
}

func NewImageButton(image paint.ImageOp, title string) *ImageButton {
	return &ImageButton{
		Image:     image,
		Clickable: widget.Clickable{},
		Title:     title,
	}
}

func (b *ImageButton) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        th.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	padding := layout.UniformInset(unit.Dp(8))

	gtx.Constraints.Min.X = gtx.Dp(80) // Ensure a minimum width for the button

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Clickable(gtx, &b.Clickable, func(gtx layout.Context) layout.Dimensions {
			return padding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Middle,
					Spacing:   layout.SpaceAround,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return widget.Image{
							Src:      b.Image,
							Fit:      widget.Unscaled,
							Position: layout.Center,
							Scale:    0.5,
						}.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lb := material.Label(th.Material(), unit.Sp(14), b.Title)
						lb.Alignment = text.Middle
						return lb.Layout(gtx)
					}),
				)
			})
		})
	})
}
