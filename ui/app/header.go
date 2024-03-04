package app

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
	selectedEnv string
	envDropDown *widgets.DropDown
}

func NewHeader() *Header {
	h := &Header{
		selectedEnv: "No Environment",
	}

	h.envDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("No Environment"),
		widgets.NewDropDownDivider(),
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

func (h *Header) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4)}

	if h.envDropDown.GetSelected().Text != h.selectedEnv {
		h.selectedEnv = h.envDropDown.GetSelected().Text
		bus.Publish(state.SelectedEnvChanged, h.selectedEnv)
	}

	headerBar := inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.H6(theme, "Chapar").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return h.envDropDown.Layout(gtx, theme)
				})
			}),
		)
	})

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return headerBar
		}),
		widgets.DrawLineFlex(widgets.Gray300, unit.Dp(1), unit.Dp(gtx.Constraints.Max.Y)),
	)
}
