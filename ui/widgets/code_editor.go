package widgets

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	giovieweditor "github.com/oligo/gioview/editor"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/fonts"
)

const (
	CodeLanguageJSON       = "JSON"
	CodeLanguageYAML       = "YAML"
	CodeLanguageXML        = "XML"
	CodeLanguagePython     = "Python"
	CodeLanguageProperties = "properties"
)

type CodeEditor struct {
	editor *giovieweditor.Editor
	code   string

	styledCode string
	styles     []*giovieweditor.TextStyle
	editorConf *giovieweditor.EditorConf

	lexer     chroma.Lexer
	codeStyle *chroma.Style

	lang string

	onChange func(text string)

	font font.FontFace

	border widget.Border

	beatufier   widget.Clickable
	loadExample widget.Clickable

	onBeautify    func()
	onLoadExample func()
}

func NewCodeEditor(code string, lang string, theme *chapartheme.Theme) *CodeEditor {
	eFont := fonts.MustGetCodeEditorFont()
	shaper := text.NewShaper(text.WithCollection([]font.FontFace{eFont}))

	c := &CodeEditor{
		editor: new(giovieweditor.Editor),
		code:   code,
		font:   eFont,
		lang:   lang,

		editorConf: &giovieweditor.EditorConf{
			Shaper:          shaper,
			TextColor:       theme.Fg,
			Bg:              theme.Bg,
			SelectionColor:  theme.TextSelectionColor,
			TypeFace:        eFont.Font.Typeface,
			TextSize:        unit.Sp(14),
			LineHeightScale: 1.2,
			ShowLineNum:     true,
			LineNumPadding:  unit.Dp(10),
			LineHighlightColor: color.NRGBA{
				R: 0xbb,
				G: 0xbb,
				B: 0xbb,
				A: 0x33,
			},
		},
	}

	c.border = widget.Border{
		Color:        theme.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	c.lexer = getLexer(lang)

	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}

	c.codeStyle = style

	c.editor.WrapPolicy = text.WrapGraphemes
	c.editor.SetText(code, false)

	return c
}

func getLexer(lang string) chroma.Lexer {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	return chroma.Coalesce(lexer)
}

func (c *CodeEditor) SetOnChanged(f func(text string)) {
	c.onChange = f
}

func (c *CodeEditor) SetOnBeautify(f func()) {
	c.onBeautify = f
}

func (c *CodeEditor) SetReadOnly(readOnly bool) {
	c.editor.ReadOnly = readOnly
}

func (c *CodeEditor) SetOnLoadExample(f func()) {
	c.onLoadExample = f
}

func (c *CodeEditor) SetCode(code string) {
	c.editor.SetText(code, false)
	c.code = code
	c.editor.UpdateTextStyles(c.stylingText(c.editor.Text()))
}

func (c *CodeEditor) SetLanguage(lang string) {
	c.lang = lang
	c.lexer = getLexer(lang)
	c.editor.UpdateTextStyles(c.stylingText(c.editor.Text()))
}

func (c *CodeEditor) Code() string {
	return c.editor.Text()
}

func (c *CodeEditor) Layout(gtx layout.Context, theme *chapartheme.Theme, hint string) layout.Dimensions {
	if c.styledCode == "" {
		// First time styling
		c.editor.UpdateTextStyles(c.stylingText(c.editor.Text()))
	}

	if !c.editor.ReadOnly {
		if ev, ok := c.editor.Update(gtx); ok {
			if _, ok := ev.(giovieweditor.ChangeEvent); ok {
				c.editor.UpdateTextStyles(c.stylingText(c.editor.Text()))
				if c.onChange != nil {
					c.onChange(c.editor.Text())
					c.code = c.editor.Text()
				}
			}
		}
	}

	flexH := layout.Flex{Axis: layout.Horizontal}

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
						Top: unit.Dp(4), Bottom: unit.Dp(4),
						Left: unit.Dp(4), Right: unit.Dp(4),
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
						return layout.Inset{
							Top:    unit.Dp(4),
							Bottom: unit.Dp(4),
							Left:   unit.Dp(8),
							Right:  unit.Dp(4),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return giovieweditor.NewEditor(c.editor, c.editorConf, hint).Layout(gtx)
						})
					}),
				)
			})
		}),
	)
}

func (c *CodeEditor) stylingText(text string) []*giovieweditor.TextStyle {
	if c.styledCode == text {
		return c.styles
	}

	// nolint:prealloc
	var textStyles []*giovieweditor.TextStyle

	offset := 0

	iterator, err := c.lexer.Tokenise(nil, text)
	if err != nil {
		return textStyles
	}

	for _, token := range iterator.Tokens() {
		entry := c.codeStyle.Get(token.Type)

		textStyle := &giovieweditor.TextStyle{
			Start: offset,
			End:   offset + len([]rune(token.Value)),
		}

		if entry.Colour.IsSet() {
			textStyle.Color = chromaColorToOp(entry.Colour)
		}

		textStyles = append(textStyles, textStyle)
		offset = textStyle.End
	}

	c.styledCode = text
	c.styles = textStyles

	return textStyles
}

func chromaColorToOp(textColor chroma.Colour) op.CallOp {
	ops := new(op.Ops)

	m := op.Record(ops)
	paint.ColorOp{Color: color.NRGBA{
		R: textColor.Red(),
		G: textColor.Green(),
		B: textColor.Blue(),
		A: 0xff,
	}}.Add(ops)
	return m.Stop()
}
