package misc

import (
	"image"
	"image/color"

	"github.com/oligo/gioview/theme"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type Icon struct {
	*widget.Icon
	Color color.NRGBA
	Size  unit.Dp
}

func (i Icon) Layout(gtx C, th *theme.Theme) D {
	if i.Size <= 0 {
		i.Size = unit.Dp(18)
	}
	if i.Color == (color.NRGBA{}) {
		i.Color = th.ContrastBg
	}

	iconSize := gtx.Dp(i.Size)
	gtx.Constraints = layout.Exact(image.Pt(iconSize, iconSize))

	return i.Icon.Layout(gtx, i.Color)
}

func IconButton(th *theme.Theme, icon *widget.Icon, button *widget.Clickable, description string) material.IconButtonStyle {
	return material.IconButtonStyle{
		Background:  th.Palette.Bg,
		Color:       th.Palette.ContrastBg,
		Icon:        icon,
		Size:        18,
		Inset:       layout.UniformInset(4),
		Button:      button,
		Description: description,
	}
}

func LayoutErrorLabel(gtx C, th *theme.Theme, err error) D {
	if err != nil {
		return layout.Inset{
			Top:    unit.Dp(10),
			Bottom: unit.Dp(10),
			Left:   unit.Dp(15),
			Right:  unit.Dp(15),
		}.Layout(gtx, func(gtx C) D {
			label := material.Label(th.Theme, th.TextSize*0.8, err.Error())
			label.Color = color.NRGBA{R: 255, A: 255}
			label.Alignment = text.Middle
			return label.Layout(gtx)
		})
	} else {
		return layout.Dimensions{}
	}
}

// WithAlpha returns the input color with the new alpha value.
func WithAlpha(c color.NRGBA, a uint8) color.NRGBA {
	return color.NRGBA{
		R: c.R,
		G: c.G,
		B: c.B,
		A: a,
	}
}
