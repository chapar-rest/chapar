package ui

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/bus"
	"github.com/mirzakhany/chapar/internal/loader"
	"github.com/mirzakhany/chapar/ui/state"
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

	h.envDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("No Environment"),
		widgets.NewDropDownDivider(),
		widgets.NewDropDownOption("Local"),
		widgets.NewDropDownOption("Dev"),
		widgets.NewDropDownOption("Prod"),
	)

	h.loadEnvs(nil)

	bus.Subscribe(state.EnvironmentsChanged, h.loadEnvs)

	return h
}

func (h *Header) loadEnvs(_ any) {
	data, err := loader.ReadEnvironmentsData()
	if err != nil {
		panic(err)
	}

	options := make([]*widgets.DropDownOption, 0)
	options = append(options, widgets.NewDropDownOption("No Environment"))
	options = append(options, widgets.NewDropDownDivider())
	for _, env := range data {
		options = append(options, widgets.NewDropDownOption(env.MetaData.Name))
	}

	h.envDropDown.SetOptions(options...)
}

func (h *Header) Layout(gtx C) D {
	inset := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4)}

	headerBar := inset.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Left: unit.Dp(10), Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.H6(h.Theme, "Chapar").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return h.envDropDown.Layout(gtx, h.Theme)
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
