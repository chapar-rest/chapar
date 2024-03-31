package widgets

import (
	"fmt"
	"image/color"
	"strings"

	"gioui.org/x/styledtext"

	"gioui.org/op"
	"gioui.org/x/richtext"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/mirzakhany/chapar/ui/fonts"

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
	monoFont font.FontFace

	lexer chroma.Lexer

	font font.FontFace

	rhState richtext.InteractiveText
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
		font:    fonts.MustGetJetBrainsMono(),
		rhState: richtext.InteractiveText{},
	}

	c.editor.Submit = false
	c.editor.SingleLine = false
	c.editor.WrapPolicy = text.WrapGraphemes
	c.editor.SetText(code)
	c.lines = strings.Split(code, "\n")

	lexer := lexers.Get("JSON") // Replace "go" with the language of your choice
	if lexer == nil {
		lexer = lexers.Fallback
	}
	c.lexer = chroma.Coalesce(lexer)

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

		if e.Name == key.NameTab && e.State == key.Release {
			c.editor.Insert("    ")
			gtx.Execute(op.InvalidateCmd{})
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
					return layout.Stack{}.Layout(gtx,
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							ee := material.Editor(theme, c.editor, hint)
							ee.Font = c.font.Font
							ee.Font.Typeface = "JetBrainsMono"
							ee.TextSize = unit.Sp(13)
							return ee.Layout(gtx)
						}),
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							t := styledtext.Text(theme.Shaper, c.getSpans()...)
							t.WrapPolicy = styledtext.WrapGraphemes
							return t.Layout(gtx, nil)
						}),
					)
				})
			}),
		)
	})
}

func (c *CodeEditor) getSpans() []styledtext.SpanStyle {
	iterator, err := c.lexer.Tokenise(nil, c.code) // sourceCode is a string containing your code
	if err != nil {
		panic(err)
	}
	spans := make([]styledtext.SpanStyle, 0)
	for _, t := range iterator.Tokens() {

		// fmt.Println("TOKEN", t.Type, t.Value)
		var sColor = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff} // Default color (black)

		switch t.Type {
		case chroma.Punctuation:
			// Color for punctuation (brackets, commas, colons)
			sColor = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
		case chroma.NameTag:
			// Color for keys
			sColor = color.NRGBA{R: 0x22, G: 0x8B, B: 0x22, A: 0xff}
		case chroma.LiteralString:
			// Color for strings
			sColor = color.NRGBA{R: 0xDC, G: 0x14, B: 0x3C, A: 0xff}
		case chroma.LiteralNumber:
			// Color for numbers
			sColor = color.NRGBA{R: 0x00, G: 0x00, B: 0x8B, A: 0xff}
		case chroma.KeywordConstant:
			// Color for booleans and null
			sColor = color.NRGBA{R: 0x8B, G: 0x00, B: 0x00, A: 0xff}
		// ... other token types as needed
		default:
			sColor = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
		}

		// Create your span using the determined color
		span := styledtext.SpanStyle{
			Content: t.Value,
			Size:    unit.Sp(13),
			Color:   sColor,
			Font:    c.font.Font,
		}
		span.Font.Typeface = "JetBrainsMono"
		spans = append(spans, span)
	}
	return spans
}
