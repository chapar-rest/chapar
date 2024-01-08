package ui

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type Header struct {
	Theme *material.Theme
	//dropDown *Dropdown

	combo *ComboBox
}

func NewHeader(theme *material.Theme) *Header {
	h := &Header{
		Theme: theme,
	}

	h.combo = NewComboBox(theme,
		NewOption("No Environment", func() { fmt.Println("none") }),
		NewDivider(),
		NewOption("Local", func() { fmt.Println("Local") }),
		NewOption("Dev", func() { fmt.Println("Dev") }),
		NewOption("Prod", func() { fmt.Println("Prod") }),
	)

	return h
}

func (h *Header) Layout(gtx C) D {
	inset := layout.UniformInset(unit.Dp(4))

	headerBar := inset.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Left: unit.Dp(10), Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.H6(h.Theme, "Chapar").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					//return h.dropDown.Layout(gtx, h.Theme)
					return h.combo.Layout(gtx)
				})
			}),
		)
	})

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.Y = 200
			return headerBar
		}),
		horizontalLine(gtx),
	)
}
