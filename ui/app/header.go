package app

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/state"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Header struct {
	selectedEnv string
	envDropDown *widgets.DropDown

	envState *state.Environments

	OnSelectedEnvChanged func(env *domain.Environment)
}

func NewHeader(envState *state.Environments) *Header {
	h := &Header{
		selectedEnv: "No Environment",
		envState:    envState,
	}

	h.envDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("No Environment"),
		widgets.NewDropDownDivider(),
	)
	return h
}

func (h *Header) LoadEnvs(data []*domain.Environment) {
	options := make([]*widgets.DropDownOption, 0)
	options = append(options, widgets.NewDropDownOption("No Environment"))
	options = append(options, widgets.NewDropDownDivider())
	for _, env := range data {
		options = append(options, widgets.NewDropDownOption(env.MetaData.Name).WithIdentifier(env.MetaData.ID))
	}

	h.envDropDown.SetOptions(options...)
}

func (h *Header) SetSelectedEnvironment(env *domain.Environment) {
	h.selectedEnv = env.MetaData.Name
	h.envDropDown.SetSelectedByTitle(env.MetaData.Name)
}

func (h *Header) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4)}

	if h.envDropDown.GetSelected().Text != h.selectedEnv {
		h.selectedEnv = h.envDropDown.GetSelected().Text
		id := h.envDropDown.GetSelected().Identifier
		h.envState.SetActiveEnvironment(h.envState.GetEnvironment(id))

		if h.OnSelectedEnvChanged != nil {
			h.OnSelectedEnvChanged(h.envState.GetEnvironment(id))
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
		}),
		widgets.DrawLineFlex(widgets.Gray300, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X)),
	)
}
