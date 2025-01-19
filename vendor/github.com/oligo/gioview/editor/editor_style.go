// SPDX-License-Identifier: Unlicense OR MIT

package editor

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/oligo/gioview/misc"
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
	//LineHighlightColor is the color used to highlight the clicked logical line.
	// If not set, line will not be highlighted.
	LineHighlightColor color.NRGBA
	// TextMatchColor use the color used to highlight the matched substring.
	TextMatchColor color.NRGBA

	Editor      *Editor
	ShowLineNum bool

	shaper  *text.Shaper
	lineBar *lineNumberBar
}

type lineNumberBar struct {
	shaper          *text.Shaper
	lineHeight      unit.Sp
	lineHeightScale float32
	// Color is the text color.
	color    color.NRGBA
	typeFace font.Typeface
	textSize unit.Sp
	// padding between line number and the editor content.
	padding unit.Dp
}

type EditorConf struct {
	Shaper             *text.Shaper
	TextColor          color.NRGBA
	Bg                 color.NRGBA
	SelectionColor     color.NRGBA
	LineHighlightColor color.NRGBA
	LineNumberColor    color.NRGBA
	TextMatchColor     color.NRGBA
	// typeface for editing
	TypeFace        font.Typeface
	TextSize        unit.Sp
	Weight          font.Weight
	LineHeight      unit.Sp
	LineHeightScale float32
	//May be helpful for code syntax highlighting.
	ColorScheme string
	ShowLineNum bool
	// padding between line number and the editor content.
	LineNumPadding unit.Dp
}

func NewEditor(editor *Editor, conf *EditorConf, hint string) EditorStyle {

	es := EditorStyle{
		Editor: editor,
		Font: font.Font{
			Typeface: conf.TypeFace,
			Weight:   conf.Weight,
		},
		LineHeightScale:    conf.LineHeightScale,
		TextSize:           conf.TextSize,
		Color:              conf.TextColor,
		shaper:             conf.Shaper,
		Hint:               hint,
		HintColor:          MulAlpha(conf.TextColor, 0xbb),
		SelectionColor:     MulAlpha(conf.SelectionColor, 0x60),
		LineHighlightColor: MulAlpha(conf.LineHighlightColor, 0x25),
		TextMatchColor:     conf.TextMatchColor,
		ShowLineNum:        conf.ShowLineNum,
		lineBar: &lineNumberBar{
			shaper:          conf.Shaper,
			lineHeight:      conf.LineHeight,
			lineHeightScale: conf.LineHeightScale,
			color:           conf.LineNumberColor,
			typeFace:        conf.TypeFace,
			textSize:        conf.TextSize,
			padding:         conf.LineNumPadding,
		},
	}

	if conf.LineNumPadding <= 0 {
		es.lineBar.padding = unit.Dp(32)
	}

	if conf.LineNumberColor == (color.NRGBA{}) {
		es.lineBar.color = misc.WithAlpha(conf.TextColor, 0xb6)
	}

	return es
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
	lineColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.LineHighlightColor}.Add(gtx.Ops)
	lineColor := lineColorMacro.Stop()

	matchColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: e.TextMatchColor}.Add(gtx.Ops)
	matchColor := matchColorMacro.Stop()

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

	if !e.ShowLineNum {
		d := e.Editor.Layout(gtx, e.shaper, e.Font, e.TextSize, textColor, selectionColor, lineColor, matchColor)
		if e.Editor.Len() == 0 {
			call.Add(gtx.Ops)
		}
		return d
	}

	// clip line number bar.
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	dims = layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return e.lineBar.Layout(gtx, e.Editor)
		}),

		layout.Rigid(layout.Spacer{Width: e.lineBar.padding}.Layout),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			d := e.Editor.Layout(gtx, e.shaper, e.Font, e.TextSize, textColor, selectionColor, lineColor, matchColor)
			if e.Editor.Len() == 0 {
				call.Add(gtx.Ops)
			}
			return d
		}),
	)

	return dims
}

func (bar lineNumberBar) layoutLine(gtx layout.Context, pos *LineInfo, textColor op.CallOp) layout.Dimensions {
	stack := op.Offset(image.Point{Y: pos.YOffset}).Push(gtx.Ops)

	tl := widget.Label{
		Alignment:       text.End,
		MaxLines:        1,
		LineHeight:      bar.lineHeight,
		LineHeightScale: bar.lineHeightScale,
	}

	d := tl.Layout(gtx, bar.shaper,
		font.Font{Typeface: bar.typeFace, Weight: font.Normal},
		bar.textSize,
		fmt.Sprintf("%d", pos.LineNum),
		textColor)
	stack.Pop()
	return d
}

func (bar lineNumberBar) Layout(gtx layout.Context, e *Editor) layout.Dimensions {
	dims := layout.Dimensions{Size: image.Point{X: gtx.Constraints.Min.X}}

	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: bar.color}.Add(gtx.Ops)
	textColor := textColorMacro.Stop()

	fake := gtx
	fake.Ops = &op.Ops{}

	positions, _ := e.VisibleLines()
	maxWidth := 0
	{
		for _, pos := range positions {
			d := bar.layoutLine(fake, pos, textColor)
			maxWidth = max(maxWidth, d.Size.X)
		}

	}

	gtx.Constraints.Max.X = maxWidth
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	for _, pos := range positions {
		d := bar.layoutLine(gtx, pos, textColor)
		dims.Size = image.Point{X: maxWidth, Y: dims.Size.Y + d.Size.Y}
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
