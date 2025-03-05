package fuzzysearch

import (
	"image"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type TextField struct {
	textEditor widget.Editor
	Icon       *widget.Icon

	Text        string
	Placeholder string

	size image.Point

	onTextChange func(text string)
	OnKeyPress   func(k key.Name)
}

func NewTextField(text, placeholder string) *TextField {
	t := &TextField{
		textEditor:  widget.Editor{},
		Text:        text,
		Placeholder: placeholder,
		Icon:        widgets.SearchIcon,
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

func (t *TextField) SetOnTextChange(f func(text string)) {
	t.onTextChange = f
}

func (t *TextField) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	for {
		event, ok := gtx.Event(
			key.FocusFilter{Target: t},
			key.Filter{Focus: &t.textEditor, Name: key.NameEscape},
			key.Filter{Focus: &t.textEditor, Name: key.NameReturn},
			key.Filter{Focus: &t.textEditor, Name: key.NameDownArrow},
		)
		if !ok {
			break
		}
		switch ev := event.(type) {
		case key.FocusEvent:
			gtx.Execute(key.FocusCmd{Tag: &t.textEditor})
		case key.Event:
			if ev.Name == key.NameReturn {
				if t.OnKeyPress != nil {
					t.OnKeyPress(ev.Name)
				}
			}

			if ev.Name == key.NameEscape {
				gtx.Execute(key.FocusCmd{Tag: nil})
				if t.OnKeyPress != nil {
					t.OnKeyPress(ev.Name)
				}
			}
		}
	}

	borderColor := theme.BorderColor
	if gtx.Source.Focused(&t.textEditor) {
		borderColor = theme.BorderColorFocused
	}

	cornerRadius := unit.Dp(4)
	border := widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: cornerRadius,
	}

	leftPadding := unit.Dp(8)

	for {
		event, ok := t.textEditor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := event.(widget.ChangeEvent); ok {
			if t.onTextChange != nil {
				t.onTextChange(t.textEditor.Text())
			}
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
				ed := material.Editor(theme.Material(), &t.textEditor, t.Placeholder)
				ed.SelectionColor = theme.TextSelectionColor
				return ed.Layout(gtx)
			})
			items := []layout.FlexChild{inputLayout}

			spacing := layout.SpaceBetween
			if t.Icon != nil {
				iconLayout := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := widgets.Button(theme.Material(), &widget.Clickable{}, t.Icon, 0, "")
					b.Inset = layout.Inset{Left: unit.Dp(8), Right: unit.Dp(2), Top: unit.Dp(2), Bottom: unit.Dp(2)}
					return b.Layout(gtx, theme)
				})

				items = []layout.FlexChild{inputLayout, iconLayout}
			}

			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: spacing}.Layout(gtx, items...)
		})
	})
}
