package widgets

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/mirzakhany/chapar/ui/fonts"

	"gioui.org/x/richtext"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type JsonViewer struct {
	data string

	lines []string

	selectables []*widget.Selectable
	states      []richtext.InteractiveText

	list *widget.List

	monoFont text.FontFace
}

func NewJsonViewer() *JsonViewer {
	return &JsonViewer{
		monoFont: fonts.MustGetMono(),
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (j *JsonViewer) SetData(data string) {
	j.data = data
	j.lines = strings.Split(data, "\n")

	j.selectables = make([]*widget.Selectable, len(j.lines))
	j.states = make([]richtext.InteractiveText, len(j.lines))
	for i := range j.selectables {
		j.selectables[i] = &widget.Selectable{}
		j.states[i] = richtext.InteractiveText{}
	}
}

func (j *JsonViewer) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        Gray400,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(3).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.List(theme, j.list).Layout(gtx, len(j.lines), func(gtx layout.Context, i int) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(theme, theme.TextSize, fmt.Sprintf("%d", i+1))
							l.Font.Weight = font.Medium
							l.Color = Gray800
							l.Alignment = text.End
							return l.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							// l := material.Label(theme, theme.TextSize, j.lines[i])

							return richtext.Text(&j.states[i], theme.Shaper, j.parseLine(j.lines[i])...).Layout(gtx)
							//rch := richtext.Text(&j.states[i], theme.Shaper)
							//rch.Typeface = "RobotoMono"
							//l.Font.Typeface = "RobotoMono"
							//l.State = j.selectables[i]
							// return l.Layout(gtx)
						})
					}),
				)
			})
		})
	})
}

func (j *JsonViewer) parseLine(line string) []richtext.SpanStyle {
	// Define colors for different elements for visual distinction
	keyColor := color.NRGBA{R: 0x55, G: 0x95, B: 0x5C, A: 255}         // Red for keys
	symbolColor := color.NRGBA{R: 0xb0, G: 0xb3, B: 0xb8, A: 0xff}     // Green for symbols (":", "{", "}", ",")
	stringValueColor := color.NRGBA{R: 0xBA, G: 0x64, B: 0xAD, A: 255} // Blue for string values
	size := unit.Sp(16)                                                // Default text size

	var spans []richtext.SpanStyle
	var currentPart strings.Builder
	inString := false
	isKey := true // Assume the first token is a key

	for _, runeValue := range line {
		switch runeValue {
		case '"':
			if inString {
				// End of string
				inString = false
				// Determine color based on whether the current part is a key or value
				color := keyColor
				if !isKey {
					color = stringValueColor
				}
				// Add the string part
				spans = append(spans, richtext.SpanStyle{
					Content: currentPart.String() + "\"",
					Color:   color,
					Size:    size,
					Font:    j.monoFont.Font,
				})
				currentPart.Reset()
			} else {
				// Beginning of string, prepare for new token
				if currentPart.Len() > 0 {
					// Non-string part before a string starts is always a symbol
					spans = append(spans, richtext.SpanStyle{
						Content: currentPart.String(),
						Color:   symbolColor,
						Size:    size,
						Font:    j.monoFont.Font,
					})
					currentPart.Reset()
				}
				currentPart.WriteRune('"')
				inString = true
			}
		case ':':
			if !inString {
				// Colon indicates that the next token is a value
				isKey = false
			}
			currentPart.WriteRune(runeValue)
		case '{', '}', ',':
			if !inString {
				// These symbols reset to expecting a key (except within arrays or nested objects)
				isKey = true
				// Symbol found, treat as a separate part
				if currentPart.Len() > 0 {
					spans = append(spans, richtext.SpanStyle{
						Content: currentPart.String(),
						Color:   symbolColor,
						Size:    size,
						Font:    j.monoFont.Font,
					})
					currentPart.Reset()
				}
				spans = append(spans, richtext.SpanStyle{
					Content: string(runeValue),
					Color:   symbolColor,
					Size:    size,
					Font:    j.monoFont.Font,
				})
			} else {
				currentPart.WriteRune(runeValue) // Still inside a string, keep accumulating
			}
		default:
			currentPart.WriteRune(runeValue)
		}
	}

	// Handle any remaining part
	if currentPart.Len() > 0 {
		color := symbolColor
		if inString {
			// Final part is a string without closing quote
			color = stringValueColor
		}
		spans = append(spans, richtext.SpanStyle{
			Content: currentPart.String(),
			Color:   color,
			Size:    size,
			Font:    j.monoFont.Font,
		})
	}

	return spans
}
