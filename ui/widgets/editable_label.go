package widgets

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type EditableLabel struct {
	editor *widget.Editor
	Text   string

	clickable widget.Clickable

	onChanged func(text string)

	isEditing bool
}

func NewEditableLabel(text string) *EditableLabel {
	e := &EditableLabel{
		editor:    new(widget.Editor),
		Text:      text,
		isEditing: false,
		clickable: widget.Clickable{},
	}
	e.editor.SingleLine = true
	e.editor.Submit = true
	return e
}

func (e *EditableLabel) SetOnChanged(f func(text string)) {
	e.onChanged = f
}

func (e *EditableLabel) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	for e.clickable.Clicked(gtx) {
		if !e.isEditing {
			e.isEditing = true
			e.editor.SetText(e.Text)
			e.editor.Focus()
		}
	}

	if e.clickable.Hovered() {
		// set cursor to pointer
		pointer.CursorText.Add(gtx.Ops)
	}

	return e.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if e.isEditing {
			for _, ev := range e.editor.Events() {
				switch ev.(type) {
				case widget.SubmitEvent:
					e.isEditing = false
					e.Text = e.editor.Text()
					if e.onChanged != nil {
						e.onChanged(e.Text)
					}
				}
			}

			border := widget.Border{
				Color:        theme.Palette.ContrastBg,
				Width:        2,
				CornerRadius: unit.Dp(4),
			}
			return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Editor(theme, e.editor, "").Layout(gtx)
				})
			})
		}

		return layout.Inset{
			Top:    unit.Dp(5),
			Bottom: unit.Dp(5),
			Left:   unit.Dp(5),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme, theme.TextSize, e.Text).Layout(gtx)
		})
	})
}
