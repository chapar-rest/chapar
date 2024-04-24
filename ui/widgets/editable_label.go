package widgets

import (
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/chapartheme"
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

func (e *EditableLabel) SetText(text string) {
	e.Text = text
}

func (e *EditableLabel) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	for e.clickable.Clicked(gtx) {
		if !e.isEditing {
			e.isEditing = true
			e.editor.SetText(e.Text)
		}
	}

	if e.clickable.Hovered() {
		// set cursor to pointer
		pointer.CursorText.Add(gtx.Ops)
	}

	for {
		ev, ok := gtx.Event(
			key.Filter{
				Focus: e.editor,
				Name:  key.NameEscape,
			},
		)
		if !ok {
			break
		}
		ee, ok := ev.(key.Event)
		if !ok {
			continue
		}

		if ee.Name == key.NameEscape {
			e.isEditing = false
			e.editor.SetText(e.Text)
		}
	}

	return e.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if e.isEditing {
			if ev, ok := e.editor.Update(gtx); ok {
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
				Color:        theme.BorderColorFocused,
				Width:        1,
				CornerRadius: unit.Dp(4),
			}
			return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					editor := material.Editor(theme.Material(), e.editor, "")
					editor.SelectionColor = theme.TextSelectionColor
					return editor.Layout(gtx)
				})
			})
		}

		return layout.Inset{
			Top:    unit.Dp(5),
			Bottom: unit.Dp(5),
			Left:   unit.Dp(5),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme.Material(), theme.TextSize, e.Text).Layout(gtx)
		})
	})
}
