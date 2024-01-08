package widgets

import (
	"gioui.org/layout"
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
	theme *material.Theme

	Icon         *widget.Icon
	IconPosition int
	spaceBetween int

	label widget.Label

	clickable *widget.Clickable

	Text string
}

func NewFlatButton(theme *material.Theme, clickable *widget.Clickable, text string) *FlatButton {
	return &FlatButton{
		theme: theme,
		Text:  text,

		clickable: clickable,
	}
}

func (f *FlatButton) SetIcon(icon *widget.Icon, position int, spaceBetween int) {
	f.Icon = icon
	f.IconPosition = position
	f.spaceBetween = spaceBetween
}

func (f *FlatButton) Layout(gtx layout.Context) layout.Dimensions {
	color := f.theme.Palette.ContrastFg
	if f.clickable.Hovered() {
		color = f.theme.Palette.ContrastBg
	}

	axis := layout.Horizontal
	labelLayout := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		l := material.Label(f.theme, unit.Sp(12), f.Text)
		l.Color = color
		return l.Layout(gtx)
	})

	widgets := []layout.FlexChild{labelLayout}

	if f.Icon != nil {
		iconLayout := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(f.spaceBetween)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return f.Icon.Layout(gtx, color)
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

	return f.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: axis, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx, widgets...)
		})
	})
}
