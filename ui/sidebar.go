package ui

import (
	"image/color"

	"example.com/gio_test/ui/widgets"
	"gioui.org/f32"
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
	sidebarButtons := layout.Flex{Axis: layout.Vertical, Spacing: 0, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := widgets.NewFlatButton(s.Theme, &s.homeButton, "Requests")
			h.SetIcon(widgets.SwapHoriz, widgets.FlatButtonIconTop, 5)
			return h.Layout(gtx)
		}),
		//layout.Rigid(layout.Spacer{Height: 10}.Layout),
		layout.Rigid(func(gtx C) D {
			return widgets.Rect{
				// gray 300
				Color: color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
				Size:  f32.Point{X: float32(90), Y: 2},
				Radii: 1,
			}.Layout(gtx)
		}),
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
		verticalLine(gtx),
	)
}
