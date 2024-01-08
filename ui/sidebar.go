package ui

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
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
	sidebarButtons := layout.Flex{Axis: layout.Vertical, Spacing: 0, Alignment: layout.Baseline}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(s.Theme, &s.homeButton, "Home").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: 10}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(s.Theme, &s.environmentsButton, "Env").Layout(gtx)
		}),
	)

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return sidebarButtons
		}),
		verticalLine(gtx),
	)
}
