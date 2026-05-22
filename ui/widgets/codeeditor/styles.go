package codeeditor

import (
	"image/color"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/textstyle/syntax"
)

// registry holds the color styles for styles
var registry = make(map[string]syntax.ColorScheme)

// buildColorSchemeFromChroma creates a gvcode ColorScheme from a chroma style.
func buildColorSchemeFromChroma(theme *chapartheme.Theme, chromaStyle *chroma.Style) syntax.ColorScheme {
	if st, ok := registry[chromaStyle.Name]; ok {
		return st
	}

	mat := theme.Material()

	cs := syntax.ColorScheme{}
	cs.Foreground = gvcolor.MakeColor(mat.Fg)
	cs.Background = gvcolor.MakeColor(mat.Bg)

	cs.SelectColor = gvcolor.MakeColor(theme.TextSelectionColor)
	cs.LineColor = gvcolor.MakeColor(mat.ContrastBg).MulAlpha(0x80)
	cs.LineNumberColor = gvcolor.MakeColor(mat.ContrastFg).MulAlpha(0xb6)

	for _, tt := range chromaStyle.Types() {
		entry := chromaStyle.Get(tt)
		if !entry.Colour.IsSet() {
			continue
		}
		fg := gvcolor.MakeColor(color.NRGBA{
			R: entry.Colour.Red(),
			G: entry.Colour.Green(),
			B: entry.Colour.Blue(),
			A: 255,
		})
		cs.AddStyle(syntax.StyleScope(tt.String()), chromaTextStyle(entry), fg, gvcolor.Color{})
	}
	registry[chromaStyle.Name] = cs
	return cs
}

// chromaTokensToGvcode tokenizes content with chroma and returns gvcode syntax tokens.
func chromaTokensToGvcode(lang, content string) []syntax.Token {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	it, err := lexer.Tokenise(nil, content)
	if err != nil {
		return nil
	}

	var tokens []syntax.Token
	runeOffset := 0
	for t := it(); t != chroma.EOF; t = it() {
		if t.Value == "" {
			continue
		}
		start := runeOffset
		runeOffset += utf8.RuneCountInString(t.Value)
		end := runeOffset
		scope := syntax.StyleScope(t.Type.String())
		if scope.IsValid() {
			tokens = append(tokens, syntax.Token{Start: start, End: end, Scope: scope})
		}
	}
	return tokens
}

func chromaTextStyle(entry chroma.StyleEntry) syntax.TextStyle {
	var textStyle syntax.TextStyle
	if entry.Bold == chroma.Yes {
		textStyle |= syntax.Bold
	}
	if entry.Italic == chroma.Yes {
		textStyle |= syntax.Italic
	}
	if entry.Underline == chroma.Yes {
		textStyle |= syntax.Underline
	}
	if entry.Border.IsSet() {
		textStyle |= syntax.Border
	}
	return textStyle
}
