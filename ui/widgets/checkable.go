// SPDX-License-Identifier: Unlicense OR MIT
// Copied from: gioui material/checkable.go with some modifications

package widgets

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

type checkable struct {
	Label              string
	Color              color.NRGBA
	Font               font.Font
	TextSize           unit.Sp
	IconColor          color.NRGBA
	Size               unit.Dp
	shaper             *text.Shaper
	checkedStateIcon   *widget.Icon
	uncheckedStateIcon *widget.Icon
}

func (c *checkable) layout(gtx layout.Context, checked bool) layout.Dimensions {
	var icon *widget.Icon
	if checked {
		icon = c.checkedStateIcon
	} else {
		icon = c.uncheckedStateIcon
	}

	dims := layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(2).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				size := gtx.Dp(c.Size)
				col := c.IconColor
				if !gtx.Enabled() {
					col = Disabled(col)
				}
				gtx.Constraints.Min = image.Point{X: size}
				icon.Layout(gtx, col)
				return layout.Dimensions{
					Size: image.Point{X: size, Y: size},
				}
			})
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(2).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				colMacro := op.Record(gtx.Ops)
				paint.ColorOp{Color: c.Color}.Add(gtx.Ops)
				return widget.Label{}.Layout(gtx, c.shaper, c.Font, c.TextSize, c.Label, colMacro.Stop())
			})
		}),
	)
	return dims
}
