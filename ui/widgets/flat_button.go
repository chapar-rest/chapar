package widgets

import (
	"image"
	"image/color"

	"gioui.org/io/input"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const (
	FlatButtonIconStart = 0
	FlatButtonIconEnd   = 1
	FlatButtonIconTop   = 2
	FlatButtonIconDown  = 3
)

type FlatButton struct {
	Icon         *widget.Icon
	IconPosition int
	SpaceBetween unit.Dp

	label widget.Label

	Clickable widget.Clickable

	MinWidth        unit.Dp
	BackgroundColor color.NRGBA
	TextColor       color.NRGBA
	Text            string

	BackgroundPadding unit.Dp
	ContentPadding    unit.Dp
}

func (f *FlatButton) SetIcon(icon *widget.Icon, position int, spaceBetween unit.Dp) {
	f.Icon = icon
	f.IconPosition = position
	f.SpaceBetween = spaceBetween
}

func (f *FlatButton) SetColor(background, text color.NRGBA) {
	f.BackgroundColor = background
	f.TextColor = text
}

func (f *FlatButton) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if f.BackgroundColor == (color.NRGBA{}) {
		f.BackgroundColor = theme.Palette.ContrastBg
	}

	if f.TextColor == (color.NRGBA{}) {
		f.TextColor = theme.Palette.ContrastFg
	}

	axis := layout.Horizontal
	labelLayout := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		l := material.Label(theme, unit.Sp(12), f.Text)
		l.Color = f.TextColor
		return l.Layout(gtx)
	})

	widgets := []layout.FlexChild{labelLayout}

	if f.Icon != nil {
		iconLayout := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(f.SpaceBetween).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return f.Icon.Layout(gtx, f.TextColor)
			})
		})

		if f.IconPosition == FlatButtonIconTop || f.IconPosition == FlatButtonIconDown {
			axis = layout.Vertical
		}

		switch f.IconPosition {
		case FlatButtonIconStart, FlatButtonIconTop:
			widgets = []layout.FlexChild{iconLayout, labelLayout}
		case FlatButtonIconEnd, FlatButtonIconDown:
			widgets = []layout.FlexChild{labelLayout, iconLayout}
		}
	}

	return f.Clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(f.BackgroundPadding).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(f.MinWidth)
					defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 4).Push(gtx.Ops).Pop()
					background := f.BackgroundColor
					if gtx.Source == (input.Source{}) {
						background = Disabled(f.BackgroundColor)
					} else if f.Clickable.Hovered() || gtx.Focused(f.Clickable) {
						background = Hovered(f.BackgroundColor)
					}
					paint.Fill(gtx.Ops, background)
					return layout.Dimensions{Size: gtx.Constraints.Min}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(f.ContentPadding).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: axis, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx, widgets...)
					})
				},
			)
		})
	})
}
