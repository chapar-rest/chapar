package ui

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Header struct {
	Theme *material.Theme

	envDropDown *widgets.DropDown
}

func NewHeader(theme *material.Theme) *Header {
	h := &Header{
		Theme: theme,
	}

	h.envDropDown = widgets.NewDropDown(theme,
		widgets.NewOption("No Environment", func() { fmt.Println("none") }),
		widgets.NewDivider(),
		widgets.NewOption("Local", func() { fmt.Println("Local") }),
		widgets.NewOption("Dev", func() { fmt.Println("Dev") }),
		widgets.NewOption("Prod", func() { fmt.Println("Prod") }),
	)

	h.envDropDown.SetBorder(theme.ContrastFg, unit.Dp(1), unit.Dp(4))

	return h
}

func (h *Header) Layout(gtx C) D {
	inset := layout.UniformInset(unit.Dp(4))

	headerBar := inset.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Left: unit.Dp(10), Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.H6(h.Theme, "Chapar").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return h.envDropDown.Layout(gtx)
				})
			}),
		)
	})

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.Y = 200
			return headerBar
		}),
		widgets.HorizontalFullLine(),
	)
}
