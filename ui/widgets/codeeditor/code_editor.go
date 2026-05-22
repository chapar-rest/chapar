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
	"github.com/alecthomas/chroma/v2/styles"
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
	CodeLanguageGraphQL    = "GraphQL"
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

	c.editor.SetText(code)
	c.setEditorOptions()

	tokens := chromaTokensToGvcode(c.lang, code)
	if len(tokens) > 0 {
		c.editor.SetSyntaxTokens(tokens...)
	}

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

	return c
}

func getEditorFont() font.FontFace {
	fontFamilyName := prefs.GetGlobalConfig().Spec.Editor.FontFamily
	fontFamilyName = strings.ReplaceAll(fontFamilyName, " ", "")

	if fontFamilyName == "JetBrainsMono" {
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
	var styleName string
	if c.theme.IsDark() {
		styleName = "dracula"
	} else {
		styleName = "tango"
	}

	// Build color scheme from chroma style and apply syntax highlighting
	chromaStyle := styles.Get(styleName)
	if chromaStyle == nil {
		chromaStyle = styles.Fallback
	}
	gvScheme := buildColorSchemeFromChroma(c.theme, chromaStyle)

	editorOptions := []gvcode.EditorOption{
		gvcode.WithFont(c.font.Font),
		gvcode.WithTextSize(unit.Sp(c.editorConfig.FontSize)),
		gvcode.WithTextAlignment(text.Start),
		gvcode.WithLineHeight(unit.Sp(16), 1),
		gvcode.WithTabWidth(c.editorConfig.TabWidth),
		gvcode.WithSoftTab(c.editorConfig.Indentation == domain.IndentationSpaces),
		gvcode.WrapLine(c.editorConfig.WrapLines),
		gvcode.WithLineNumber(c.editorConfig.ShowLineNumbers),
		gvcode.WithLineNumberGutterGap(unit.Dp(8)),
		gvcode.WithColorScheme(gvScheme),
	}

	if !c.editorConfig.AutoCloseBrackets {
		editorOptions = append(editorOptions, gvcode.WithQuotePairs(map[rune]rune{}))
	}

	if !c.editorConfig.AutoCloseQuotes {
		editorOptions = append(editorOptions, gvcode.WithBracketPairs(map[rune]rune{}))
	}

	c.editor.WithOptions(editorOptions...)
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
	c.code = code
	// Avoid passing empty string to the editor: harfbuzz (used by the text shaper)
	// panics when shaping an empty buffer (index out of range [0] with length 0).
	display := code
	if display == "" {
		display = "\n"
	}
	c.editor.SetText(display)

	tokens := chromaTokensToGvcode(c.lang, code)
	if len(tokens) > 0 {
		c.editor.SetSyntaxTokens(tokens...)
	}
}

func (c *CodeEditor) SetLanguage(lang string) {
	c.lang = lang
	tokens := chromaTokensToGvcode(c.lang, c.code)
	if len(tokens) > 0 {
		c.editor.SetSyntaxTokens(tokens...)
	}
}

func (c *CodeEditor) Code() string {
	return c.editor.Text()
}

func (c *CodeEditor) Layout(gtx layout.Context, theme *chapartheme.Theme, hint string) layout.Dimensions {
	scrollIndicatorColor := gvcolor.MakeColor(theme.Material().Fg).MulAlpha(0x30)

	if c.editor.Mode() != gvcode.ModeReadOnly {
		for {
			evt, ok := c.editor.Update(gtx)
			if !ok {
				break
			}
			if _, isChange := evt.(gvcode.ChangeEvent); isChange {
				c.code = c.editor.Text()
				if c.onChange != nil {
					c.onChange(c.code)
				}
				c.editor.OnTextEdit()
				tokens := chromaTokensToGvcode(c.lang, c.code)
				if len(tokens) > 0 {
					c.editor.SetSyntaxTokens(tokens...)
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

					btn := widgets.Button(theme, &c.loadExample, widgets.RefreshIcon, widgets.IconPositionStart, "Load Example")
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
							Top:    unit.Dp(8),
							Bottom: unit.Dp(0),
							Left:   unit.Dp(8),
							Right:  unit.Dp(0),
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
			btn := widgets.Button(theme, &c.beatufier, widgets.FormatIcon, widgets.IconPositionStart, "Beautify")
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
