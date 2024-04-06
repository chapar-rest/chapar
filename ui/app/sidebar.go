package app

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/mirzakhany/chapar/ui/theme"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Sidebar struct {
	Theme *theme.Theme

	flatButtons []*widgets.FlatButton
	Buttons     []*SideBarButton
	list        *widget.List

	clickables []*widget.Clickable

	selectedIndex int
}

type SideBarButton struct {
	Icon *widget.Icon
	Text string
}

func NewSidebar(theme *theme.Theme) *Sidebar {
	s := &Sidebar{
		Theme: theme,

		Buttons: []*SideBarButton{
			{Icon: widgets.SwapHoriz, Text: "Requests"},
			{Icon: widgets.MenuIcon, Text: "Envs"},
			// {Icon: widgets.FileFolderIcon, Text: "Proto"},
			// {Icon: widgets.TunnelIcon, Text: "Tunnels"},
			// {Icon: widgets.ConsoleIcon, Text: "Console"},
			// {Icon: widgets.LogsIcon, Text: "Logs"},
			// {Icon: widgets.SettingsIcon, Text: "Settings"},
		},
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}

	s.clickables = make([]*widget.Clickable, 0)
	for range s.Buttons {
		s.clickables = append(s.clickables, &widget.Clickable{})
	}
	return s
}

func (s *Sidebar) makeButtons(theme *theme.Theme) {
	s.flatButtons = make([]*widgets.FlatButton, 0)
	for i, b := range s.Buttons {
		s.flatButtons = append(s.flatButtons, &widgets.FlatButton{
			Icon:              b.Icon,
			Text:              b.Text,
			IconPosition:      widgets.FlatButtonIconTop,
			Clickable:         s.clickables[i],
			SpaceBetween:      unit.Dp(5),
			BackgroundPadding: unit.Dp(1),
			CornerRadius:      0,
			MinWidth:          unit.Dp(60),
			BackgroundColor:   theme.Palette.ContrastBg,
			TextColor:         theme.TextColor,
			ContentPadding:    unit.Dp(5),
		})
	}
}

func (s *Sidebar) SelectedIndex() int {
	return s.selectedIndex
}

func (s *Sidebar) Layout(gtx layout.Context, theme *theme.Theme) layout.Dimensions {
	gtx.Constraints.Max.X = gtx.Dp(70)
	s.makeButtons(theme)
	dims := s.list.Layout(gtx, len(s.Buttons), func(gtx layout.Context, i int) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Spacing: 0, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := s.flatButtons[i]
				if btn.Clickable.Clicked(gtx) {
					s.selectedIndex = i
				}

				if s.selectedIndex == i {
					btn.TextColor = theme.Palette.ContrastFg
				} else {
					btn.TextColor = widgets.Gray700
				}

				return btn.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if i == len(s.Buttons)-1 {
					return layout.Dimensions{}
				}
				return widgets.DrawLine(gtx, widgets.Gray300, unit.Dp(2), unit.Dp(45))
			}),
		)
	})

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dims
		}),
		widgets.DrawLineFlex(widgets.Gray300, unit.Dp(gtx.Constraints.Max.Y), unit.Dp(1)),
	)
}
