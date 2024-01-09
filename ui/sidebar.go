package ui

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Sidebar struct {
	Theme *material.Theme

	// homeButton is the button that takes the user to the home screen.
	homeButton         widget.Clickable
	environmentsButton widget.Clickable
}

func NewSidebar(theme *material.Theme) *Sidebar {
	return &Sidebar{
		Theme: theme,
	}
}

func (s *Sidebar) Layout(gtx C) D {
	sidebarButtons := layout.Flex{Axis: layout.Vertical, Spacing: 0, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := widgets.NewFlatButton(s.Theme, &s.homeButton, "Requests")
			h.SetIcon(widgets.SwapHoriz, widgets.FlatButtonIconTop, 5)
			return h.Layout(gtx)
		}),
		widgets.HorizontalLine(90.0),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := widgets.NewFlatButton(s.Theme, &s.environmentsButton, "Envs")
			h.SetIcon(widgets.MenuIcon, widgets.FlatButtonIconTop, 5)
			return h.Layout(gtx)
		}),
	)

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return sidebarButtons
		}),
		widgets.VerticalFullLine(),
	)
}
