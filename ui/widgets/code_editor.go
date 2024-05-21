package widgets

import (
	"fmt"
	"strings"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/richtext"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/fonts"
)

type CodeEditor struct {
	editor *widget.Editor
	code   string

	lines []string
	list  *widget.List

	onChange func(text string)

	font font.FontFace

	rhState richtext.InteractiveText

	border widget.Border
}

func NewCodeEditor(code string, _ string, theme *chapartheme.Theme) *CodeEditor {
	c := &CodeEditor{
		editor: new(widget.Editor),
		code:   code,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		font:    fonts.MustGetCodeEditorFont(),
		rhState: richtext.InteractiveText{},
	}

	c.border = widget.Border{
		Color:        theme.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	c.editor.Submit = false
	c.editor.SingleLine = false
	c.editor.WrapPolicy = text.WrapGraphemes
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
	c.code = code
}

func (c *CodeEditor) SetLanguage(_ string) {
}

func (c *CodeEditor) Code() string {
	return c.editor.Text()
}

func (c *CodeEditor) Layout(gtx layout.Context, theme *chapartheme.Theme, hint string) layout.Dimensions {
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

		if e.Name == key.NameTab && e.State == key.Release {
			c.editor.Insert("    ")
			break
		}
	}

	if ev, ok := c.editor.Update(gtx); ok {
		if _, ok := ev.(widget.ChangeEvent); ok {
			c.lines = strings.Split(c.editor.Text(), "\n")

			if c.onChange != nil {
				c.onChange(c.editor.Text())
				c.code = c.editor.Text()
			}
		}
	}

	flexH := layout.Flex{Axis: layout.Horizontal}
	listInset := layout.Inset{Left: unit.Dp(10), Top: unit.Dp(4)}
	inset4 := layout.UniformInset(unit.Dp(4))
	return c.border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return flexH.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return listInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.List(theme.Material(), c.list).Layout(gtx, len(c.lines), func(gtx layout.Context, i int) layout.Dimensions {
						l := material.Label(theme.Material(), theme.TextSize, fmt.Sprintf("%*d", len(fmt.Sprintf("%d", len(c.lines))), i+1))
						l.Font.Weight = font.Medium
						l.Color = theme.TextColor
						l.TextSize = unit.Sp(14)
						l.Alignment = text.End
						return l.Layout(gtx)
					})
				})
			}),

			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return inset4.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					ee := material.Editor(theme.Material(), c.editor, hint)
					ee.TextSize = unit.Sp(14)
					ee.SelectionColor = theme.TextSelectionColor
					return ee.Layout(gtx)
				})
			}),
		)
	})
}
