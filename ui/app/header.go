package app

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/state"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Header struct {
	selectedEnv string
	envDropDown *widgets.DropDown

	envState      *state.Environments
	themeSwitcher *widget.Bool

	OnSelectedEnvChanged func(env *domain.Environment)

	OnThemeSwitched func(isLight bool)
}

const (
	none          = "none"
	noEnvironment = "No Environment"
)

func NewHeader(envState *state.Environments) *Header {
	h := &Header{
		selectedEnv:   noEnvironment,
		envState:      envState,
		themeSwitcher: new(widget.Bool),
	}

	h.envDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption(noEnvironment),
		widgets.NewDropDownDivider(),
	)
	return h
}

func (h *Header) LoadEnvs(data []*domain.Environment) {
	options := make([]*widgets.DropDownOption, 0)
	options = append(options, widgets.NewDropDownOption(noEnvironment).WithIdentifier(none))
	options = append(options, widgets.NewDropDownDivider())

	selectEnvExist := false
	for _, env := range data {
		if h.selectedEnv == env.MetaData.ID {
			selectEnvExist = true
		}
		options = append(options, widgets.NewDropDownOption(env.MetaData.Name).WithIdentifier(env.MetaData.ID))
	}

	h.envDropDown.SetOptions(options...)

	if selectEnvExist {
		h.SetSelectedEnvironment(h.envState.GetEnvironment(h.selectedEnv))
	}
}

func (h *Header) SetSelectedEnvironment(env *domain.Environment) {
	h.selectedEnv = env.MetaData.ID
	h.envDropDown.SetSelectedByTitle(env.MetaData.Name)
}

func (h *Header) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4)}

	if h.envDropDown.GetSelected().Identifier != h.selectedEnv {
		h.selectedEnv = h.envDropDown.GetSelected().Identifier
		if h.selectedEnv != none {
			id := h.envDropDown.GetSelected().Identifier
			h.envState.SetActiveEnvironment(h.envState.GetEnvironment(id))

			if h.OnSelectedEnvChanged != nil {
				h.OnSelectedEnvChanged(h.envState.GetEnvironment(id))
			}
		} else {
			h.envState.ClearActiveEnvironment()
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
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return widgets.MaterialIcons("dark_mode", theme).Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Right: unit.Dp(10), Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									if h.themeSwitcher.Update(gtx) {
										if h.OnThemeSwitched != nil {
											h.OnThemeSwitched(h.themeSwitcher.Value)
										}
									}

									return material.Switch(theme, h.themeSwitcher, "").Layout(gtx)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return widgets.MaterialIcons("light_mode", theme).Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Left: unit.Dp(20), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return h.envDropDown.Layout(gtx, theme)
								})
							}),
						)
					}),
				)
			})
		}),
		widgets.DrawLineFlex(widgets.Gray300, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X)),
	)
}
