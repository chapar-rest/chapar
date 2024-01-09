package widgets

import (
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
	theme *material.Theme

	textEditor widget.Editor
	textField  material.EditorStyle
	Icon       *widget.Icon

	IconPosition int

	Text string
}

func NewTextField(theme *material.Theme, text string) *TextField {
	t := &TextField{
		theme:      theme,
		textEditor: widget.Editor{},
		Text:       text,
	}

	t.textEditor.SetText(text)
	t.textEditor.SingleLine = true
	return t
}

func (t *TextField) SetText(text string) {
	t.textEditor.SetText(text)
}

func (t *TextField) SetIcon(icon *widget.Icon, position int) {
	t.Icon = icon
	t.IconPosition = position
}

func (t *TextField) Layout(gtx layout.Context) layout.Dimensions {
	borderColor := t.theme.Palette.ContrastFg
	if t.textEditor.Focused() {
		borderColor = t.theme.Palette.ContrastBg
	}

	cornerRadius := unit.Dp(4)
	border := widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: cornerRadius,
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.Y = 60
		return layout.Inset{
			Top:    10,
			Bottom: 0,
			Left:   10,
			Right:  5,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			inputLayout := layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(t.theme, &t.textEditor, "").Layout(gtx)
			})
			widgets := []layout.FlexChild{inputLayout}

			spacing := layout.SpaceBetween
			if t.Icon != nil {
				iconLayout := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return t.Icon.Layout(gtx, borderColor)
				})

				if t.IconPosition == IconPositionEnd {
					widgets = []layout.FlexChild{inputLayout, iconLayout}
				} else {
					widgets = []layout.FlexChild{iconLayout, inputLayout}
					spacing = layout.SpaceEnd
				}
			}

			return layout.Flex{Axis: layout.Horizontal, Spacing: spacing}.Layout(gtx, widgets...)
		})
	})
}
