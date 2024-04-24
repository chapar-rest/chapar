package app

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/state"
	"github.com/mirzakhany/chapar/ui/chapartheme"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Header struct {
	materialTheme *material.Theme
	selectedEnv   string
	envDropDown   *widgets.DropDown

	envState      *state.Environments
	switchState   *widget.Bool
	themeSwitcher material.SwitchStyle

	iconDarkMode  material.LabelStyle
	iconLightMode material.LabelStyle

	OnSelectedEnvChanged func(env *domain.Environment)
	OnThemeSwitched      func(isLight bool)
}

const (
	none          = "none"
	noEnvironment = "No Environment"
)

func NewHeader(envState *state.Environments, theme *chapartheme.Theme) *Header {
	h := &Header{
		materialTheme: theme.Material(),
		selectedEnv:   noEnvironment,
		envState:      envState,
		switchState:   new(widget.Bool),
	}
	h.iconDarkMode = widgets.MaterialIcons("dark_mode", theme)
	h.iconLightMode = widgets.MaterialIcons("light_mode", theme)

	h.themeSwitcher = material.Switch(theme.Material(), h.switchState, "")
	h.envDropDown = widgets.NewDropDown(theme)
	h.envDropDown.MinWidth = unit.Dp(150)
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

func (h *Header) SetTheme(isDark bool) {
	h.switchState.Value = !isDark
}

func (h *Header) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
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

	if h.switchState.Update(gtx) {
		if h.OnThemeSwitched != nil {
			go h.OnThemeSwitched(!h.switchState.Value)
		}
	}

	content := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.H6(h.materialTheme, "Chapar").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							h.iconDarkMode.Color = theme.TextColor
							return h.iconDarkMode.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Right: unit.Dp(10), Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return h.themeSwitcher.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							h.iconLightMode.Color = theme.TextColor
							return h.iconLightMode.Layout(gtx)
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
	})

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		content,
		widgets.DrawLineFlex(theme.SeparatorColor, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X)),
	)
}
