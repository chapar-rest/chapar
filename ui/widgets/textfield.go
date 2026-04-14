package widgets

import (
	"image"
	"image/color"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

const (
	IconPositionStart = 0
	IconPositionEnd   = 1
)

type TextField struct {
	textEditor widget.Editor
	Icon       *widget.Icon
	iconClick  widget.Clickable

	IconPosition       int
	BorderColor        color.NRGBA
	BorderColorFocused color.NRGBA

	Text        string
	Placeholder string

	size image.Point

	changed     bool
	iconClicked bool
}

func NewTextField(text, placeholder string) *TextField {
	t := &TextField{
		textEditor:  widget.Editor{},
		Text:        text,
		Placeholder: placeholder,
	}

	t.textEditor.SetText(text)
	t.textEditor.SingleLine = true
	return t
}

func (t *TextField) GetText() string {
	return t.textEditor.Text()
}

func (t *TextField) SetText(text string) {
	t.textEditor.SetText(text)
}

func (t *TextField) SetIcon(icon *widget.Icon, position int) {
	t.Icon = icon
	t.IconPosition = position
}

func (t *TextField) SetSize(s image.Point) {
	t.size = s
}

func (t *TextField) SetMinWidth(width int) {
	t.size.X = width
}

func (t *TextField) SetBorderColor(color color.NRGBA) {
	t.BorderColor = color
}

func (t *TextField) Changed() bool {
	out := t.changed
	t.changed = false
	return out
}

func (t *TextField) IconClicked() bool {
	out := t.iconClicked
	t.iconClicked = false
	return out
}

func (t *TextField) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	for {
		event, ok := gtx.Event(key.FocusFilter{Target: t}, key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}

		if ev, ok := event.(key.Event); ok && (ev.Name == key.NameEscape) {
			gtx.Execute(key.FocusCmd{Tag: nil})
		}
	}

	borderColor := t.BorderColor
	if t.BorderColor == (color.NRGBA{}) {
		borderColor = theme.BorderColor
	}

	if gtx.Source.Focused(&t.textEditor) {
		if t.BorderColorFocused == (color.NRGBA{}) {
			borderColor = theme.BorderColorFocused
		} else {
			borderColor = t.BorderColorFocused
		}
	}

	cornerRadius := unit.Dp(4)
	border := widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: cornerRadius,
	}

	leftPadding := unit.Dp(8)
	if t.Icon != nil && t.IconPosition == IconPositionStart {
		leftPadding = unit.Dp(0)
	}

	for {
		event, ok := t.textEditor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := event.(widget.ChangeEvent); ok {
			t.changed = true
		}
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if t.size.X == 0 {
			t.size.X = gtx.Constraints.Min.X
		}

		gtx.Constraints.Min = t.size
		return layout.Inset{
			Top:    4,
			Bottom: 4,
			Left:   leftPadding,
			Right:  4,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			inputLayout := layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(theme.Material(), &t.textEditor, t.Placeholder).Layout(gtx)
			})
			widgets := []layout.FlexChild{inputLayout}

			spacing := layout.SpaceBetween
			if t.Icon != nil {
				iconLayout := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if t.iconClick.Clicked(gtx) {
						t.iconClicked = true
					}

					ib := &IconButton{
						Icon:            t.Icon,
						Color:           theme.TextColor,
						BackgroundColor: theme.Palette.Bg,
						Size:            unit.Dp(18),
						Clickable:       &t.iconClick,
					}
					return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(2), Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ib.Layout(gtx, theme)
					})
				})

				if t.IconPosition == IconPositionEnd {
					widgets = []layout.FlexChild{inputLayout, iconLayout}
				} else {
					widgets = []layout.FlexChild{iconLayout, inputLayout}
					spacing = layout.SpaceEnd
				}
			}

			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: spacing}.Layout(gtx, widgets...)
		})
	})
}
