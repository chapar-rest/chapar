package widgets

import (
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
	"gioui.org/x/richtext"

	giovieweditor "github.com/oligo/gioview/editor"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/fonts"
)

type CodeEditor struct {
	editor *giovieweditor.Editor
	code   string

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
		editor:  new(giovieweditor.Editor),
		code:    code,
		font:    fonts.MustGetCodeEditorFont(),
		rhState: richtext.InteractiveText{},
	}

	c.border = widget.Border{
		Color:        theme.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	c.editor.WrapPolicy = text.WrapGraphemes
	c.editor.SetText(code, false)

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
			if c.onChange != nil {
				c.onChange(c.editor.Text())
				c.code = c.editor.Text()
			}
		}
	}

	flexH := layout.Flex{Axis: layout.Horizontal}
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
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return inset4.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

							editorConf := &giovieweditor.EditorConf{
								Shaper:          theme.Shaper,
								TextColor:       theme.Fg,
								Bg:              theme.Bg,
								SelectionColor:  theme.TextSelectionColor,
								TypeFace:        c.font.Font.Typeface,
								TextSize:        unit.Sp(14),
								LineHeightScale: 1.2,
								ColorScheme:     "default",
							}

							c.editor.UpdateTextStyles(stylingText(c.editor.Text()))
							return giovieweditor.NewEditor(c.editor, editorConf, hint).Layout(gtx)
						})
					}),
				)
			})
		}),
	)
}

var (
	// List of Python keywords
	keywords = []string{
		"False", "await", "else", "import", "pass",
		"None", "break", "except", "in", "raise",
		"True", "class", "finally", "is", "return",
		"and", "continue", "for", "lambda", "try",
		"as", "def", "from", "nonlocal", "while",
		"assert", "del", "global", "not", "with",
		"async", "elif", "if", "or", "yield",
	}
	keyColor     = color.NRGBA{R: 255, G: 165, B: 0, A: 255}   // Orange
	stringColor  = color.NRGBA{R: 42, G: 161, B: 152, A: 255}  // #2aa198
	numberColor  = color.NRGBA{R: 42, G: 161, B: 152, A: 255}  // #2aa198
	booleanColor = color.NRGBA{R: 128, G: 0, B: 128, A: 255}   // Purple
	nullColor    = color.NRGBA{R: 128, G: 128, B: 128, A: 255} // Gray
	commentColor = color.NRGBA{R: 88, G: 110, B: 117, A: 255}  // #586e75
	// Define colors for different JSON elements
	pythonKeywordsColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255} // Orange

	pythonPattern = regexp.MustCompile(`\b(` + strings.Join(keywords, "|") + `)\b`)

	// Define regex patterns for different JSON elements
	keyPattern                   = regexp.MustCompile(`"(\\\"|[^"])*"\s*:`)
	stringPattern                = regexp.MustCompile(`"(\\\"|[^"])*"`)
	numberPattern                = regexp.MustCompile(`\b\d+(\.\d+)?([eE][+-]?\d+)?\b`)
	booleanPattern               = regexp.MustCompile(`\btrue\b|\bfalse\b`)
	nullPattern                  = regexp.MustCompile(`\bnull\b`)
	jsonSingleLineCommentPattern = regexp.MustCompile(`//.*`)
	pythonCommentPattern         = regexp.MustCompile(`#.*`)
)

func stylingText(text string) []*giovieweditor.TextStyle {
	var styles []*giovieweditor.TextStyle

	// Apply styles based on matches
	applyStyles := func(re *regexp.Regexp, col color.NRGBA) {
		matches := re.FindAllStringIndex(text, -1)
		for _, match := range matches {
			styles = append(styles, &giovieweditor.TextStyle{
				Start: match[0],
				End:   match[1],
				Color: colorToOp(col),
			})
		}
	}

	// Apply styles for each JSON element
	applyStyles(keyPattern, keyColor)
	applyStyles(stringPattern, stringColor)
	applyStyles(numberPattern, numberColor)
	applyStyles(booleanPattern, booleanColor)
	applyStyles(nullPattern, nullColor)
	applyStyles(pythonPattern, pythonKeywordsColor)
	applyStyles(jsonSingleLineCommentPattern, commentColor)
	applyStyles(pythonCommentPattern, commentColor)

	return styles
}

func colorToOp(textColor color.NRGBA) op.CallOp {
	ops := new(op.Ops)

	m := op.Record(ops)
	paint.ColorOp{Color: textColor}.Add(ops)
	return m.Stop()
}
