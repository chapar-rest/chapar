package ui

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Sidebar struct {
	Theme *material.Theme

	protoFilesButton *widgets.FlatButton
	requestsButton   *widgets.FlatButton
	envButton        *widgets.FlatButton
	settingsButton   *widgets.FlatButton
}

func NewSidebar(theme *material.Theme) *Sidebar {
	s := &Sidebar{
		Theme:            theme,
		requestsButton:   widgets.NewFlatButton(theme, "Requests"),
		envButton:        widgets.NewFlatButton(theme, "Envs"),
		protoFilesButton: widgets.NewFlatButton(theme, "Proto"),
		settingsButton:   widgets.NewFlatButton(theme, "Settings"),
	}

	s.requestsButton.SetIcon(widgets.SwapHoriz, widgets.FlatButtonIconTop, 5)
	s.requestsButton.SetColor(theme.Palette.Bg, theme.Palette.Fg)
	s.requestsButton.MinWidth = 130

	s.envButton.SetIcon(widgets.MenuIcon, widgets.FlatButtonIconTop, 5)
	s.envButton.SetColor(theme.Palette.Bg, theme.Palette.Fg)
	s.envButton.MinWidth = 130

	s.protoFilesButton.SetIcon(widgets.FileFolderIcon, widgets.FlatButtonIconTop, 5)
	s.protoFilesButton.SetColor(theme.Palette.Bg, theme.Palette.Fg)
	s.protoFilesButton.MinWidth = 130

	s.settingsButton.SetIcon(widgets.SettingsIcon, widgets.FlatButtonIconTop, 5)
	s.settingsButton.SetColor(theme.Palette.Bg, theme.Palette.Fg)
	s.settingsButton.MinWidth = 130

	return s
}

func (s *Sidebar) Layout(gtx C) D {
	sidebarButtons := layout.Flex{Axis: layout.Vertical, Spacing: 0, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return s.requestsButton.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 3}.Layout),
		widgets.HorizontalLine(90.0),
		layout.Rigid(layout.Spacer{Height: 3}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return s.envButton.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 3}.Layout),
		widgets.HorizontalLine(90.0),
		layout.Rigid(layout.Spacer{Height: 3}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return s.protoFilesButton.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 3}.Layout),
		widgets.HorizontalLine(90.0),
		layout.Rigid(layout.Spacer{Height: 3}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return s.settingsButton.Layout(gtx)
		}),
	)

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return sidebarButtons
		}),
		widgets.VerticalFullLine(),
	)
}
