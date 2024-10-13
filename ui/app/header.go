package app

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Header struct {
	w *app.Window

	theme       *chapartheme.Theme
	selectedEnv string
	envDropDown *widgets.DropDown

	// modal is used to show error and messages to the user
	modal *widgets.MessageModal

	selectedWorkspace string
	workspaceDropDown *widgets.DropDown

	envState        *state.Environments
	workspacesState *state.Workspaces

	themeSwitcherClickable widget.Clickable

	isDarkMode    bool
	iconDarkMode  material.LabelStyle
	iconLightMode material.LabelStyle

	OnSelectedEnvChanged       func(env *domain.Environment) error
	OnSelectedWorkspaceChanged func(env *domain.Workspace) error
	OnThemeSwitched            func(isDark bool) error
}

const (
	none          = "none"
	noEnvironment = "No Environment"
)

func NewHeader(w *app.Window, envState *state.Environments, workspacesState *state.Workspaces, theme *chapartheme.Theme) *Header {
	h := &Header{
		w:               w,
		theme:           theme,
		selectedEnv:     noEnvironment,
		envState:        envState,
		workspacesState: workspacesState,
	}
	h.iconDarkMode = widgets.MaterialIcons("dark_mode", theme)
	h.iconLightMode = widgets.MaterialIcons("light_mode", theme)

	h.envDropDown = widgets.NewDropDown(theme)
	h.workspaceDropDown = widgets.NewDropDownWithoutBorder(
		theme,
		widgets.NewDropDownOption(domain.DefaultWorkspaceName).WithIdentifier(domain.DefaultWorkspaceName),
	)
	h.workspaceDropDown.SetSelectedByIdentifier(domain.DefaultWorkspaceName)
	h.envDropDown.MaxWidth = unit.Dp(150)
	h.workspaceDropDown.MaxWidth = unit.Dp(150)
	return h
}

func (h *Header) showError(err error) {
	h.modal = widgets.NewMessageModal("Error", err.Error(), widgets.MessageModalTypeErr, func(_ string) {
		h.modal.Hide()
	}, widgets.ModalOption{Text: "Ok"})
	h.modal.Show()
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

func (h *Header) LoadWorkspaces(data []*domain.Workspace) {
	options := make([]*widgets.DropDownOption, 0)
	selectWsExist := false
	for _, ws := range data {
		if h.selectedWorkspace == ws.MetaData.ID {
			selectWsExist = true
		}
		options = append(options, widgets.NewDropDownOption(ws.MetaData.Name).WithIdentifier(ws.MetaData.ID))
	}

	h.workspaceDropDown.SetOptions(options...)

	if selectWsExist {
		h.SetSelectedWorkspace(h.workspacesState.GetWorkspace(h.selectedWorkspace))
	}
}

func (h *Header) SetSelectedWorkspace(ws *domain.Workspace) {
	h.selectedWorkspace = ws.MetaData.ID
	h.workspaceDropDown.SetSelectedByTitle(ws.MetaData.Name)
}

func (h *Header) SetSelectedEnvironment(env *domain.Environment) {
	h.selectedEnv = env.MetaData.ID
	h.envDropDown.SetSelectedByTitle(env.MetaData.Name)
}

func (h *Header) SetTheme(isDark bool) {
	h.isDarkMode = isDark
}

func (h *Header) themeSwitchIcon() material.LabelStyle {
	if h.isDarkMode {
		h.iconDarkMode = widgets.MaterialIcons("dark_mode", h.theme)
		return h.iconDarkMode
	}
	h.iconLightMode = widgets.MaterialIcons("light_mode", h.theme)
	return h.iconLightMode
}

func (h *Header) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4)}

	if h.envDropDown.GetSelected().Identifier != h.selectedEnv {
		h.selectedEnv = h.envDropDown.GetSelected().Identifier
		if h.selectedEnv != none {
			id := h.envDropDown.GetSelected().Identifier
			h.envState.SetActiveEnvironment(h.envState.GetEnvironment(id))

			if h.OnSelectedEnvChanged != nil {
				if err := h.OnSelectedEnvChanged(h.envState.GetEnvironment(id)); err != nil {
					h.showError(err)
				}
			}
		} else {
			h.envState.ClearActiveEnvironment()
			if h.OnSelectedEnvChanged != nil {
				if err := h.OnSelectedEnvChanged(nil); err != nil {
					h.showError(err)
				}
			}
		}
	}

	if h.themeSwitcherClickable.Clicked(gtx) {
		h.isDarkMode = !h.isDarkMode
		if h.OnThemeSwitched != nil {
			if err := h.OnThemeSwitched(h.isDarkMode); err != nil {
				h.showError(err)
			}
			h.w.Invalidate()
		}
	}

	selectedWorkspace := h.workspaceDropDown.GetSelected().Identifier
	if selectedWorkspace != h.selectedWorkspace {
		h.selectedWorkspace = selectedWorkspace
		ws := h.workspacesState.GetWorkspace(selectedWorkspace)
		h.workspacesState.SetActiveWorkspace(ws)

		if h.OnSelectedWorkspaceChanged != nil {
			if err := h.OnSelectedWorkspaceChanged(ws); err != nil {
				h.showError(err)
			}
		}
	}

	content := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return material.H6(h.theme.Material(), "Chapar").Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(20), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return h.workspaceDropDown.Layout(gtx, theme)
							})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return widgets.Clickable(gtx, &h.themeSwitcherClickable, unit.Dp(4), h.themeSwitchIcon().Layout)
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
