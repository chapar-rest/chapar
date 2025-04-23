package widget

import (
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
	"github.com/oligo/gioview/theme"
)

type TagVariant uint8

const (
	Solid TagVariant = iota
	Outline
)

// Tag is used for items that need to be labeled using keywords
// that describe them.
type Tag struct {
	Text     string
	TextSize unit.Sp
	Font     font.Font
	// Text color the of label. For outine variant, this is also the border of the tag.
	TextColor color.NRGBA
	// Background color of the label. Only valid in the case of Solid variant.
	Background color.NRGBA
	Radius     unit.Dp
	Inset      layout.Inset
	Variant    TagVariant
}

func (t Tag) Layout(gtx C, th *theme.Theme) D {
	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: t.TextColor}.Add(gtx.Ops)
	textColor := textColorMacro.Stop()

	textFont := t.Font
	if t.Font == (font.Font{}) {
		textFont = font.Font{
			Typeface: th.Face,
			Weight:   font.Normal,
		}
	}

	switch t.Variant {
	case Outline:
		return t.layoutOutline(gtx, th.Shaper, textFont, textColor)
	case Solid:
		return t.layoutSolid(gtx, th.Shaper, textFont, textColor)
	}

	return D{}
}

func (t Tag) layoutText(gtx C, shaper *text.Shaper, font font.Font, size unit.Sp, textMaterial op.CallOp, txt string) D {
	gtx.Constraints.Min.X = 0

	tl := widget.Label{
		Alignment: text.Start,
		MaxLines:  1,
	}

	return tl.Layout(gtx, shaper, font, size, txt, textMaterial)
}

func (t Tag) layoutSolid(gtx C, shaper *text.Shaper, font font.Font, textMaterial op.CallOp) D {
	macro := op.Record(gtx.Ops)
	dims := t.Inset.Layout(gtx, func(gtx C) D {
		return t.layoutText(gtx, shaper, font, t.TextSize, textMaterial, t.Text)
	})
	callOps := macro.Stop()

	defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, int(t.Radius)).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, t.Background)
	callOps.Add(gtx.Ops)

	return dims
}

func (t Tag) layoutOutline(gtx C, shaper *text.Shaper, font font.Font, textMaterial op.CallOp) D {
	return widget.Border{
		Color:        t.TextColor,
		CornerRadius: t.Radius,
		Width:        unit.Dp(1),
	}.Layout(gtx, func(gtx C) D {
		return t.Inset.Layout(gtx, func(gtx C) D {
			return t.layoutText(gtx, shaper, font, t.TextSize, textMaterial, t.Text)
		})
	})
}
