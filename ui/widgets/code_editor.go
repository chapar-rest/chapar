package widgets

import (
	"fmt"
	"strings"

	"gioui.org/op"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type CodeEditor struct {
	editor *widget.Editor

	code string

	lines []string
	list  *widget.List

	onChange func(text string)
}

func NewCodeEditor(code string) *CodeEditor {
	c := &CodeEditor{
		editor: new(widget.Editor),
		code:   code,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}

	c.editor.Submit = false
	c.editor.SingleLine = false
	c.editor.SetText(code)
	c.lines = strings.Split(code, "\n")
	return c
}

func (c *CodeEditor) SetOnChanged(f func(text string)) {
	c.onChange = f
}

func (c *CodeEditor) SetCode(code string) {
	c.editor.SetText(code)
	c.lines = strings.Split(code, "\n")
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

	for {
		ev, ok := gtx.Event(
			key.Filter{
				Focus: c.editor,
				Name:  key.NameTab,
			},
		)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok {
			continue
		}

		if e.Name == key.NameTab {
			c.editor.Insert("    ")
			gtx.Execute(op.InvalidateCmd{})
		}
	}

	if ev, ok := c.editor.Update(gtx); ok {
		if _, ok := ev.(widget.ChangeEvent); ok {
			c.lines = strings.Split(c.editor.Text(), "\n")

			if c.onChange != nil {
				c.onChange(c.editor.Text())
			}
		}
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.List(theme, c.list).Layout(gtx, len(c.lines), func(gtx layout.Context, i int) layout.Dimensions {
						l := material.Label(theme, theme.TextSize, fmt.Sprintf("%*d", len(fmt.Sprintf("%d", len(c.lines))), i+1))
						l.Font.Weight = font.Medium
						l.Color = Gray800
						l.Alignment = text.End
						return l.Layout(gtx)
					})
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Editor(theme, c.editor, hint).Layout(gtx)
				})
			}),
		)
	})
}
