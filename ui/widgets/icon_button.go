package widgets

import (
	"image"
	"image/color"

	"gioui.org/io/semantic"

	"gioui.org/widget/material"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

type IconButton struct {
	Icon                 *widget.Icon
	Size                 unit.Dp
	Color                color.NRGBA
	BackgroundColor      color.NRGBA
	BackgroundColorHover color.NRGBA

	SkipFocus bool
	Clickable *widget.Clickable

	OnClick func()
}

func (ib *IconButton) Layout(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	if ib.BackgroundColor == (color.NRGBA{}) {
		ib.BackgroundColor = theme.Palette.Bg
	}

	if ib.BackgroundColorHover == (color.NRGBA{}) {
		ib.BackgroundColorHover = Hovered(ib.BackgroundColor)
	}

	for ib.Clickable.Clicked(gtx) {
		if ib.OnClick != nil {
			go ib.OnClick()
		}
	}

	return ib.Clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.Button.Add(gtx.Ops)
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 4).Push(gtx.Ops).Pop()
				background := ib.BackgroundColor
				switch {
				case gtx.Queue == nil:
					background = Disabled(ib.BackgroundColor)
				case ib.Clickable.Hovered() || (ib.Clickable.Focused() && !ib.SkipFocus):
					background = ib.BackgroundColorHover
				}
				paint.Fill(gtx.Ops, background)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(ib.Size)
				return ib.Icon.Layout(gtx, ib.Color)
			},
		)
	})
}
