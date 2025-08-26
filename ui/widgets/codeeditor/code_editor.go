package codeeditor

import (
	"image"
	"image/color"
	"os"
	"strings"

	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/flopp/go-findfont"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/textstyle/syntax"
	wg "github.com/oligo/gvcode/widget"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/fonts"
	"github.com/chapar-rest/chapar/ui/widgets"

	"github.com/oligo/gvcode"
)

const (
	CodeLanguageJSON       = "JSON"
	CodeLanguageYAML       = "YAML"
	CodeLanguageXML        = "XML"
	CodeLanguagePython     = "Python"
	CodeLanguageGolang     = "Golang"
	CodeLanguageJava       = "Java"
	CodeLanguageJavaScript = "JavaScript"
	CodeLanguageRuby       = "Ruby"
	CodeLanguageShell      = "Shell"
	CodeLanguageDotNet     = "Shell"
	CodeLanguageProperties = "properties"
)

type CodeEditor struct {
	editor *gvcode.Editor
	code   string

	theme *chapartheme.Theme

	styledCode string
	tokens     []syntax.Token

	lexer chroma.Lexer
	lang  string

	onChange func(text string)

	font font.FontFace

	border widget.Border

	beatufier   widget.Clickable
	loadExample widget.Clickable

	withBeautify bool

	onLoadExample func()

	xScroll widget.Scrollbar
	yScroll widget.Scrollbar

	editorConfig domain.EditorConfig
}

func NewCodeEditor(code string, lang string, theme *chapartheme.Theme) *CodeEditor {
	globalConfig := prefs.GetGlobalConfig()
	editorFont := getEditorFont()

	c := &CodeEditor{
		theme:        theme,
		editor:       wg.NewEditor(theme.Material()),
		code:         code,
		font:         editorFont,
		lang:         lang,
		editorConfig: globalConfig.Spec.Editor,
	}

	c.lexer = getLexer(lang)
	c.setEditorOptions()

	prefs.AddGlobalConfigChangeListener(func(old, updated domain.GlobalConfig) {
		if old.Spec.Editor.Changed(updated.Spec.Editor) {
			c.editorConfig = updated.Spec.Editor
			c.updateEditorOptions(old.Spec.Editor, updated.Spec.Editor)
		}
	})

	c.border = widget.Border{
		Color:        theme.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	c.editor.SetText(code)
	return c
}

func getEditorFont() font.FontFace {
	fontFamilyName := prefs.GetGlobalConfig().Spec.Editor.FontFamily
	fontFamilyName = strings.ReplaceAll(fontFamilyName, " ", "")

	if "JetBrainsMono" == fontFamilyName {
		return fonts.MustGetCodeEditorFont()
	}

	fontPath, err := findfont.Find(fontFamilyName)
	if err != nil {
		// fallback to default font
		return fonts.MustGetCodeEditorFont()
	}

	data, err := os.ReadFile(fontPath)
	if err != nil {
		// fallback to default font
		return fonts.MustGetCodeEditorFont()
	}

	monoFont, err := opentype.ParseCollection(data)
	if err != nil {
		panic(err)
	}

	return font.FontFace{Font: monoFont[0].Font, Face: monoFont[0].Face}
}

func (c *CodeEditor) updateEditorOptions(old, updated domain.EditorConfig) {
	switch {
	case old.AutoCloseBrackets != updated.AutoCloseBrackets:
		if !updated.AutoCloseBrackets {
			c.editor.WithOptions(gvcode.WithQuotePairs(map[rune]rune{}))
		}

		if !c.editorConfig.AutoCloseQuotes {
			c.editor.WithOptions(gvcode.WithBracketPairs(map[rune]rune{}))
		}
	case old.FontFamily != updated.FontFamily:
		if updated.FontFamily != "" {
			c.font = getEditorFont()
			c.editor.WithOptions(gvcode.WithFont(c.font.Font))
		}
	case old.FontSize != updated.FontSize:
		if updated.FontSize > 0 {
			c.editor.WithOptions(gvcode.WithTextSize(unit.Sp(updated.FontSize)))
		}
	case old.TabWidth != updated.TabWidth:
		c.editor.WithOptions(gvcode.WithTabWidth(updated.TabWidth))
	case old.Indentation != updated.Indentation:
		c.editor.WithOptions(gvcode.WithSoftTab(updated.Indentation == domain.IndentationSpaces))
	case old.ShowLineNumbers != updated.ShowLineNumbers:
		c.editor.WithOptions(gvcode.WithLineNumber(updated.ShowLineNumbers))
	case old.WrapLines != updated.WrapLines:
		c.editor.WithOptions(gvcode.WrapLine(updated.WrapLines))
	}
}

func (c *CodeEditor) setEditorOptions() {
	// color scheme
	colorScheme := syntax.ColorScheme{}
	colorScheme.Foreground = gvcolor.MakeColor(c.theme.Fg)
	colorScheme.SelectColor = gvcolor.MakeColor(c.theme.TextSelectionColor)
	colorScheme.LineColor = gvcolor.MakeColor(c.theme.ContrastBg).MulAlpha(0x30)
	colorScheme.LineNumberColor = gvcolor.MakeColor(c.theme.ContrastFg).MulAlpha(0xb6)

	// TODO make the color scheme configurable
	syntaxStyles := getColorStyles("dracula", c.theme)
	for _, style := range syntaxStyles {
		colorScheme.AddStyle(style.scope, style.textStyle, style.color, gvcolor.Color{})
	}

	editorOptions := []gvcode.EditorOption{
		gvcode.WithFont(c.font.Font),
		gvcode.WithTextSize(unit.Sp(c.editorConfig.FontSize)),
		gvcode.WithTextAlignment(text.Start),
		gvcode.WithLineHeight(unit.Sp(16), 1),
		gvcode.WithTabWidth(c.editorConfig.TabWidth),
		gvcode.WithSoftTab(c.editorConfig.Indentation == domain.IndentationSpaces),
		gvcode.WrapLine(c.editorConfig.WrapLines),
		gvcode.WithLineNumber(c.editorConfig.ShowLineNumbers),
		gvcode.WithColorScheme(colorScheme),
	}

	if !c.editorConfig.AutoCloseBrackets {
		editorOptions = append(editorOptions, gvcode.WithQuotePairs(map[rune]rune{}))
	}

	if !c.editorConfig.AutoCloseQuotes {
		editorOptions = append(editorOptions, gvcode.WithBracketPairs(map[rune]rune{}))
	}

	c.editor.WithOptions(editorOptions...)
}

func getLexer(lang string) chroma.Lexer {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	return chroma.Coalesce(lexer)
}

func (c *CodeEditor) WithBeautifier(enabled bool) {
	c.withBeautify = enabled
}

func (c *CodeEditor) SetOnChanged(f func(text string)) {
	c.onChange = f
}

func (c *CodeEditor) SetReadOnly(readOnly bool) {
	c.editor.WithOptions(gvcode.ReadOnlyMode(readOnly))
}

func (c *CodeEditor) SetOnLoadExample(f func()) {
	c.onLoadExample = f
}

func (c *CodeEditor) SetCode(code string) {
	c.editor.SetText(code)
	c.code = code
	c.editor.SetSyntaxTokens(c.stylingText(c.editor.Text())...)
}

func (c *CodeEditor) SetLanguage(lang string) {
	c.lang = lang
	c.lexer = getLexer(lang)
	c.editor.SetSyntaxTokens(c.stylingText(c.editor.Text())...)
}

func (c *CodeEditor) Code() string {
	return c.editor.Text()
}

func (c *CodeEditor) Layout(gtx layout.Context, theme *chapartheme.Theme, hint string) layout.Dimensions {
	if c.styledCode == "" {
		// First time styling
		c.editor.SetSyntaxTokens(c.stylingText(c.editor.Text())...)
	}

	scrollIndicatorColor := gvcolor.MakeColor(theme.Material().Fg).MulAlpha(0x30)

	if !c.editor.ReadOnly() {
		if ev, ok := c.editor.Update(gtx); ok {
			if _, ok := ev.(gvcode.ChangeEvent); ok {
				st := c.stylingText(c.editor.Text())
				c.tokens = st
				c.editor.SetSyntaxTokens(st...)
				if c.onChange != nil {
					c.onChange(c.editor.Text())
					c.code = c.editor.Text()
				}
			}
		}
	}

	if c.loadExample.Clicked(gtx) {
		c.onLoadExample()
	}

	xScrollDist := c.xScroll.ScrollDistance()
	yScrollDist := c.yScroll.ScrollDistance()
	if xScrollDist != 0.0 || yScrollDist != 0.0 {
		c.editor.Scroll(gtx, xScrollDist, yScrollDist)
	}

	flexH := layout.Flex{Axis: layout.Horizontal}

	if c.withBeautify {
		macro := op.Record(gtx.Ops)
		c.beautyButton(gtx, theme)
		defer op.Defer(gtx.Ops, macro.Stop())
	}

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

					btn := widgets.Button(theme.Material(), &c.loadExample, widgets.RefreshIcon, widgets.IconPositionStart, "Load Example")
					btn.Color = theme.ButtonTextColor
					btn.Inset = layout.Inset{
						Top: unit.Dp(4), Bottom: unit.Dp(4),
						Left: unit.Dp(4), Right: unit.Dp(4),
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
							dims := c.editor.Layout(gtx, theme.Material().Shaper)

							macro := op.Record(gtx.Ops)
							scrollbarDims := func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Left: gtx.Metric.PxToDp(c.editor.GutterWidth()),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									minX, maxX, _, _ := c.editor.ScrollRatio()
									bar := makeScrollbar(theme.Material(), &c.xScroll, scrollIndicatorColor.NRGBA())
									return bar.Layout(gtx, layout.Horizontal, minX, maxX)
								})
							}(gtx)

							scrollbarOp := macro.Stop()
							defer op.Offset(image.Point{Y: dims.Size.Y - scrollbarDims.Size.Y}).Push(gtx.Ops).Pop()
							scrollbarOp.Add(gtx.Ops)
							return dims
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						_, _, minY, maxY := c.editor.ScrollRatio()
						bar := makeScrollbar(theme.Material(), &c.yScroll, scrollIndicatorColor.NRGBA())
						return bar.Layout(gtx, layout.Vertical, minY, maxY)
					}),
				)
			})
		}),
	)
}

func (c *CodeEditor) beautyButton(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if c.beatufier.Clicked(gtx) {
		c.SetCode(BeautifyCode(c.lang, c.code))
		if c.onChange != nil {
			c.onChange(c.editor.Text())
		}
	}

	return layout.Inset{Bottom: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.SE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &c.beatufier, widgets.CleanIcon, widgets.IconPositionStart, "Beautify")
			btn.Color = theme.ButtonTextColor
			btn.Inset = layout.Inset{
				Top: 4, Bottom: 4,
				Left: 4, Right: 4,
			}
			return btn.Layout(gtx, theme)
		})
	})
}

func makeScrollbar(th *material.Theme, scroll *widget.Scrollbar, color color.NRGBA) material.ScrollbarStyle {
	bar := material.Scrollbar(th, scroll)
	bar.Indicator.Color = color
	bar.Indicator.CornerRadius = unit.Dp(0)
	bar.Indicator.MinorWidth = unit.Dp(8)
	bar.Track.MajorPadding = unit.Dp(0)
	bar.Track.MinorPadding = unit.Dp(1)
	return bar
}

func (c *CodeEditor) stylingText(text string) []syntax.Token {
	if c.styledCode == text {
		return c.tokens
	}

	// nolint:prealloc
	var tokens []syntax.Token

	offset := 0

	iterator, err := c.lexer.Tokenise(nil, text)
	if err != nil {
		return tokens
	}

	for _, token := range iterator.Tokens() {
		gtoken := syntax.Token{
			Start: offset,
			End:   offset + len([]rune(token.Value)),
			Scope: syntax.StyleScope(token.Type.String()),
		}
		tokens = append(tokens, gtoken)
		offset = gtoken.End
	}

	c.styledCode = text
	c.tokens = tokens

	return tokens
}

func chromaColorToNRGBA(textColor chroma.Colour) color.NRGBA {
	return color.NRGBA{
		R: textColor.Red(),
		G: textColor.Green(),
		B: textColor.Blue(),
		A: 0xff,
	}
}
