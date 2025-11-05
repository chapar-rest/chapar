package codeeditor

import (
	"fmt"
	"image/color"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/textstyle/syntax"
)

type colorStyle struct {
	scope     syntax.StyleScope
	textStyle syntax.TextStyle
	color     gvcolor.Color
	bg        gvcolor.Color
}

// registry holds the color styles for styles
var registry = make(map[string][]colorStyle)

func extractStylesFromChroma(styleName string) ([]colorStyle, error) {
	if st, ok := registry[styleName]; ok {
		return st, nil
	}

	// Get the Chroma style
	chromaStyle := styles.Get(styleName)
	if chromaStyle == nil {
		return nil, fmt.Errorf("style %s not found", styleName)
	}

	var customStyles = make([]colorStyle, 0, len(chromaStyle.Types()))

	// Iterate through all style entries
	for _, tokenType := range chromaStyle.Types() {
		entry := chromaStyle.Get(tokenType)
		custom := colorStyle{
			scope:     syntax.StyleScope(tokenType.String()),
			textStyle: extractTextStyle(entry),
			color:     extractColor(entry.Colour),
			//bg:        extractColor(entry.Background),
			bg: gvcolor.Color{},
		}

		customStyles = append(customStyles, custom)
	}

	registry[styleName] = customStyles
	return customStyles, nil
}

func extractTextStyle(entry chroma.StyleEntry) syntax.TextStyle {
	var textStyle syntax.TextStyle = 0
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

func extractColor(color chroma.Colour) gvcolor.Color {
	if !color.IsSet() {
		return gvcolor.Color{}
	}

	return gvcolor.MakeColor(chromaColorToNRGBA(color))
}

func chromaColorToNRGBA(textColor chroma.Colour) color.NRGBA {
	return color.NRGBA{
		R: textColor.Red(),
		G: textColor.Green(),
		B: textColor.Blue(),
		A: 0xff,
	}
}
