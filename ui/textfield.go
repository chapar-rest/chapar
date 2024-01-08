package ui

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const (
	IconPositionStart = 0
	IconPositionEnd   = 1
)

type TextField struct {
	textEditor widget.Editor
	textField  material.EditorStyle
	icon       *widget.Icon

	iconPosition int

	text string
}

func (t *TextField) SetText(text string) {
	t.textEditor.SetText(text)
}

func (t *TextField) SetIcon(icon *widget.Icon, position int) {
	t.icon = icon
	t.iconPosition = position
}

func (t *TextField) Layout(gtx layout.Context) layout.Dimensions {
	borderColor := color.NRGBA{R: 0xc0, G: 0xc3, B: 0xc8, A: 0xff}
	if t.textEditor.Focused() {
		borderColor = color.NRGBA{R: 0x3f, G: 0x7e, B: 0xca, A: 0xff}
	}

	cornerRadius := unit.Dp(4)
	border := widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: cornerRadius,
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = 300
		gtx.Constraints.Min.Y = 60
		return layout.Inset{
			Top:    10,
			Bottom: 0,
			Left:   10,
			Right:  5,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return t.textField.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return t.icon.Layout(gtx, borderColor)
				}),
			)
		})
	})
}
