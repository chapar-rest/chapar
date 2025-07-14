package codeeditor

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/textstyle/syntax"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type colorStyle struct {
	scope     syntax.StyleScope
	textStyle syntax.TextStyle
	color     gvcolor.Color
}

// registry holds the color styles for styles
var registry = make(map[string][]colorStyle)

func getColorStyles(name string, theme *chapartheme.Theme) []colorStyle {
	if st, ok := registry[name]; ok {
		return st
	}

	style := styles.Get(name)
	if style == nil {
		style = styles.Fallback
	}

	out := make([]colorStyle, 0)
	for _, token := range style.Types() {
		if ok, styleColor := getColorStyle(token, style.Get(token), theme); ok {
			out = append(out, styleColor)
		} else {
			// If the token type is not recognized, we can skip it
			continue
		}
	}

	registry[name] = out
	return out
}

func getColorStyle(token chroma.TokenType, style chroma.StyleEntry, theme *chapartheme.Theme) (bool, colorStyle) {
	th := theme.Material()

	styleStr := style.String()
	cc := strings.Split(styleStr, " ")
	if len(cc) < 1 {
		return false, colorStyle{}
	}

	var textStyle syntax.TextStyle = 0
	switch strings.ToLower(cc[0]) {
	case "bold":
		textStyle = syntax.Bold
	case "italic":
		textStyle = syntax.Italic
	case "underline":
		textStyle = syntax.Underline
	case "border":
		textStyle = syntax.Border
	}

	var setColor gvcolor.Color
	if style.Colour.IsSet() {
		setColor = gvcolor.MakeColor(chromaColorToNRGBA(style.Colour))
	} else {
		setColor = gvcolor.MakeColor(th.Fg)
	}

	return true, colorStyle{
		scope:     syntax.StyleScope(fmt.Sprintf("%s", token)),
		textStyle: textStyle,
		color:     setColor,
	}
}
