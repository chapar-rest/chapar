package widget

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/oligo/gvcode"
)

type EditorStyle struct {
	Font font.Font
	// LineHeight controls the distance between the baselines of lines of text.
	// If zero, a sensible default will be used.
	LineHeight unit.Sp
	// LineHeightScale applies a scaling factor to the LineHeight. If zero, a
	// sensible default will be used.
	LineHeightScale float32
	// TabWidth set how many spaces to represent a tab character.
	TabWidth int
	// TextSize set the text size.
	TextSize unit.Sp
	// Color is the text color.
	Color color.NRGBA
	// SelectionColor is the color of the background for selected text.
	SelectionColor color.NRGBA
	//LineHighlightColor is the color used to highlight the clicked logical line.
	// If not set, line will not be highlighted.
	LineHighlightColor color.NRGBA
	// TextHighlightColor use the color used to highlight the interested substring.
	TextHighlightColor color.NRGBA
	// Gap size between the line number bar and the main text area.
	LineNumberGutter unit.Dp
	LineNumberColor  color.NRGBA

	Editor *gvcode.Editor
	shaper *text.Shaper
}

func NewEditor(th *material.Theme, editor *gvcode.Editor) EditorStyle {
	es := EditorStyle{
		Editor: editor,
		shaper: th.Shaper,
		Font: font.Font{
			Typeface: th.Face,
		},
		LineHeightScale:    1.2,
		TabWidth:           4,
		TextSize:           th.TextSize,
		Color:              th.Fg,
		SelectionColor:     mulAlpha(th.ContrastBg, 0x60),
		LineHighlightColor: mulAlpha(th.ContrastBg, 0x30),
		LineNumberColor:    mulAlpha(th.Fg, 0xb6),
		LineNumberGutter:   unit.Dp(24),
	}

	return es
}

func (e EditorStyle) Layout(gtx layout.Context) layout.Dimensions {
	// Choose colors.
	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.Color}.Add(gtx.Ops)
	textColor := textColorMacro.Stop()

	selectionColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: blendDisabledColor(!gtx.Enabled(), e.SelectionColor)}.Add(gtx.Ops)
	selectionColor := selectionColorMacro.Stop()

	lineColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.LineHighlightColor}.Add(gtx.Ops)
	lineColor := lineColorMacro.Stop()

	textHighlightColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.TextHighlightColor}.Add(gtx.Ops)
	textHighlightColor := textHighlightColorMacro.Stop()

	lineNumColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.LineNumberColor}.Add(gtx.Ops)
	lineNumColor := lineNumColorMacro.Stop()


	e.Editor.WithOptions(
		gvcode.WithShaperParams(e.Font, e.TextSize, text.Start, e.LineHeight, e.LineHeightScale),
		gvcode.WithTabWidth(e.TabWidth),
	)

	e.Editor.LineNumberGutter = e.LineNumberGutter
	e.Editor.TextMaterial = textColor
	e.Editor.SelectMaterial = selectionColor
	e.Editor.LineMaterial = lineColor
	e.Editor.LineNumberMaterial = lineNumColor
	e.Editor.TextHighlightMaterial = textHighlightColor

	return e.Editor.Layout(gtx, e.shaper)
}

func blendDisabledColor(disabled bool, c color.NRGBA) color.NRGBA {
	if disabled {
		return disabledColor(c)
	}
	return c
}

// mulAlpha applies the alpha to the color.
func mulAlpha(c color.NRGBA, alpha uint8) color.NRGBA {
	c.A = uint8(uint32(c.A) * uint32(alpha) / 0xFF)
	return c
}

// approxLuminance is a fast approximate version of RGBA.Luminance.
func approxLuminance(c color.NRGBA) byte {
	const (
		r = 13933 // 0.2126 * 256 * 256
		g = 46871 // 0.7152 * 256 * 256
		b = 4732  // 0.0722 * 256 * 256
		t = r + g + b
	)
	return byte((r*int(c.R) + g*int(c.G) + b*int(c.B)) / t)
}

// Disabled blends color towards the luminance and multiplies alpha.
// Blending towards luminance will desaturate the color.
// Multiplying alpha blends the color together more with the background.
func disabledColor(c color.NRGBA) (d color.NRGBA) {
	const r = 80 // blend ratio
	lum := approxLuminance(c)
	d = mix(c, color.NRGBA{A: c.A, R: lum, G: lum, B: lum}, r)
	d = mulAlpha(d, 128+32)
	return
}

// mix mixes c1 and c2 weighted by (1 - a/256) and a/256 respectively.
func mix(c1, c2 color.NRGBA, a uint8) color.NRGBA {
	ai := int(a)
	return color.NRGBA{
		R: byte((int(c1.R)*ai + int(c2.R)*(256-ai)) / 256),
		G: byte((int(c1.G)*ai + int(c2.G)*(256-ai)) / 256),
		B: byte((int(c1.B)*ai + int(c2.B)*(256-ai)) / 256),
		A: byte((int(c1.A)*ai + int(c2.A)*(256-ai)) / 256),
	}
}
