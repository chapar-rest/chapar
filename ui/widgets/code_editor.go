package widgets

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/richtext"

	"github.com/oligo/gioview/editor"
	giovieweditor "github.com/oligo/gioview/editor"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/fonts"
)

type CodeEditor struct {
	editor *giovieweditor.Editor
	code   string

	lines []string
	list  *widget.List

	onChange func(text string)

	font font.FontFace

	rhState richtext.InteractiveText

	border widget.Border

	beatufier   widget.Clickable
	loadExample widget.Clickable

	onBeautify    func()
	onLoadExample func()
}

func NewCodeEditor(code string, _ string, theme *chapartheme.Theme) *CodeEditor {
	c := &CodeEditor{
		editor: new(giovieweditor.Editor),
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

	//	c.editor.Submit = false
	// 	c.editor.SingleLine = false
	c.editor.WrapPolicy = text.WrapGraphemes
	c.editor.SetText(code, false)
	c.lines = strings.Split(code, "\n")

	return c
}

func (c *CodeEditor) SetOnChanged(f func(text string)) {
	c.onChange = f
}

func (c *CodeEditor) SetOnBeautify(f func()) {
	c.onBeautify = f
}

func (c *CodeEditor) SetOnLoadExample(f func()) {
	c.onLoadExample = f
}

func (c *CodeEditor) SetCode(code string) {
	c.editor.SetText(code, false)
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
		if _, ok := ev.(giovieweditor.ChangeEvent); ok {
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

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:    layout.Horizontal,
				Spacing: layout.SpaceStart,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if c.onLoadExample == nil {
						return layout.Dimensions{}
					}

					btn := Button(theme.Material(), &c.loadExample, RefreshIcon, IconPositionStart, "Load Example")
					btn.Color = theme.ButtonTextColor
					btn.Inset = layout.Inset{
						Top: 4, Bottom: 4,
						Left: 4, Right: 4,
					}

					if c.loadExample.Clicked(gtx) {
						c.onLoadExample()
					}

					return btn.Layout(gtx, theme)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if c.onBeautify == nil {
						return layout.Dimensions{}
					}

					btn := Button(theme.Material(), &c.beatufier, CleanIcon, IconPositionStart, "Beautify")
					btn.Color = theme.ButtonTextColor
					btn.Inset = layout.Inset{
						Top: 4, Bottom: 4,
						Left: 4, Right: 4,
					}

					if c.beatufier.Clicked(gtx) {
						c.onBeautify()
					}

					return btn.Layout(gtx, theme)
				}),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
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

							th := theme.Material()
							editorConf := &giovieweditor.EditorConf{
								Shaper:          th.Shaper,
								TextColor:       th.Fg,
								Bg:              th.Bg,
								SelectionColor:  theme.TextSelectionColor,
								TypeFace:        "Go, Helvetica, Arial, sans-serif",
								TextSize:        unit.Sp(14),
								LineHeightScale: 1,
								ColorScheme:     "default",
							}

							c.editor.UpdateTextStyles(stylingText(c.editor.Text(), th.Bg))
							//ee := material.Editor(theme.Material(), c.editor, hint)
							//ee.TextSize = unit.Sp(14)
							//ee.SelectionColor = theme.TextSelectionColor
							return giovieweditor.NewEditor(c.editor, editorConf, hint).Layout(gtx)
							// return ee.Layout(gtx)
						})
					}),
				)
			})
		}),
	)
}

func stylingText(text string, bk color.NRGBA) []*giovieweditor.TextStyle {
	var styles []*giovieweditor.TextStyle

	// List of Python keywords
	keywords := []string{
		"False", "await", "else", "import", "pass",
		"None", "break", "except", "in", "raise",
		"True", "class", "finally", "is", "return",
		"and", "continue", "for", "lambda", "try",
		"as", "def", "from", "nonlocal", "while",
		"assert", "del", "global", "not", "with",
		"async", "elif", "if", "or", "yield",
	}

	// Create a single regex pattern for all keywords
	pattern := `\b(` + strings.Join(keywords, "|") + `)\b`
	re := regexp.MustCompile(pattern)

	matches := re.FindAllIndex([]byte(text), -1)
	for _, match := range matches {
		styles = append(styles, &editor.TextStyle{
			Start:      match[0],
			End:        match[1],
			Color:      colorToOp(color.NRGBA{R: 255, G: 165, B: 0, A: 255}),
			Background: colorToOp(bk),
		})
	}

	return styles
}

func colorToOp(textColor color.NRGBA) op.CallOp {
	ops := new(op.Ops)

	m := op.Record(ops)
	paint.ColorOp{Color: textColor}.Add(ops)
	return m.Stop()
}
