package widgets

import (
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type CodeEditor struct {
	editor *widget.Editor

	code string
}

func NewCodeEditor(code string) *CodeEditor {
	c := &CodeEditor{
		editor: new(widget.Editor),
		code:   code,
	}

	c.editor.Submit = false
	c.editor.SingleLine = false

	c.editor.SetText(code)
	return c
}

func (c *CodeEditor) SetCode(code string) {
	c.editor.SetText(code)
}

func (c *CodeEditor) Code() string {
	return c.editor.Text()
}

func (c *CodeEditor) Layout(gtx layout.Context, theme *material.Theme, hint string) layout.Dimensions {
	border := widget.Border{
		Color:        Gray300,
		Width:        unit.Dp(1),
		CornerRadius: 0,
	}
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	key.InputOp{Tag: area, Keys: key.NameTab}.Add(gtx.Ops)
	defer area.Pop()

	// check for presses of the escape key and close the window if we find them.
	for _, event := range gtx.Events(area) {
		switch event := event.(type) {
		case key.Event:
			if event.Name == key.NameTab {
				c.editor.Insert("\t")
			}
		}
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Editor(theme, c.editor, hint).Layout(gtx)
		})
	})
}
