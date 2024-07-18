// SPDX-License-Identifier: Unlicense OR MIT

package editor

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

type EditorStyle struct {
	Font font.Font
	// LineHeight controls the distance between the baselines of lines of text.
	// If zero, a sensible default will be used.
	LineHeight unit.Sp
	// LineHeightScale applies a scaling factor to the LineHeight. If zero, a
	// sensible default will be used.
	LineHeightScale float32
	TextSize        unit.Sp
	// Color is the text color.
	Color color.NRGBA
	// Hint contains the text displayed when the editor is empty.
	Hint string
	// HintColor is the color of hint text.
	HintColor color.NRGBA
	// SelectionColor is the color of the background for selected text.
	SelectionColor color.NRGBA
	Editor         *Editor

	shaper *text.Shaper
}

type EditorConf struct {
	Shaper         *text.Shaper
	TextColor      color.NRGBA
	Bg             color.NRGBA
	SelectionColor color.NRGBA
	// typeface for editing
	TypeFace        font.Typeface
	TextSize        unit.Sp
	Weight          font.Weight
	LineHeight      unit.Sp
	LineHeightScale float32
	//May be helpful for code syntax highlighting.
	ColorScheme string
}

func NewEditor(editor *Editor, conf *EditorConf, hint string) EditorStyle {
	return EditorStyle{
		Editor: editor,
		Font: font.Font{
			Typeface: conf.TypeFace,
			Weight:   conf.Weight,
		},
		LineHeightScale: conf.LineHeightScale,
		TextSize:        conf.TextSize,
		Color:           conf.TextColor,
		shaper:          conf.Shaper,
		Hint:            hint,
		HintColor:       MulAlpha(conf.TextColor, 0xbb),
		SelectionColor:  MulAlpha(conf.SelectionColor, 0x60),
	}
}

func (e EditorStyle) Layout(gtx layout.Context) layout.Dimensions {
	// Choose colors.
	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.Color}.Add(gtx.Ops)
	textColor := textColorMacro.Stop()
	hintColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.HintColor}.Add(gtx.Ops)
	hintColor := hintColorMacro.Stop()
	selectionColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: blendDisabledColor(!gtx.Enabled(), e.SelectionColor)}.Add(gtx.Ops)
	selectionColor := selectionColorMacro.Stop()

	macro := op.Record(gtx.Ops)
	tl := widget.Label{
		Alignment:       e.Editor.Alignment,
		MaxLines:        0,
		LineHeight:      e.LineHeight,
		LineHeightScale: e.LineHeightScale,
	}
	dims := tl.Layout(gtx, e.shaper, e.Font, e.TextSize, e.Hint, hintColor)
	call := macro.Stop()

	if w := dims.Size.X; gtx.Constraints.Min.X < w {
		gtx.Constraints.Min.X = w
	}
	if h := dims.Size.Y; gtx.Constraints.Min.Y < h {
		gtx.Constraints.Min.Y = h
	}
	e.Editor.LineHeight = e.LineHeight
	e.Editor.LineHeightScale = e.LineHeightScale
	dims = e.Editor.Layout(gtx, e.shaper, e.Font, e.TextSize, textColor, selectionColor)
	if e.Editor.Len() == 0 {
		call.Add(gtx.Ops)
	}
	return dims
}

func blendDisabledColor(disabled bool, c color.NRGBA) color.NRGBA {
	if disabled {
		return Disabled(c)
	}
	return c
}

// MulAlpha applies the alpha to the color.
func MulAlpha(c color.NRGBA, alpha uint8) color.NRGBA {
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
func Disabled(c color.NRGBA) (d color.NRGBA) {
	const r = 80 // blend ratio
	lum := approxLuminance(c)
	d = mix(c, color.NRGBA{A: c.A, R: lum, G: lum, B: lum}, r)
	d = MulAlpha(d, 128+32)
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
