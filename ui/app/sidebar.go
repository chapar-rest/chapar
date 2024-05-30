package app

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Sidebar struct {
	Theme *chapartheme.Theme

	flatButtons []*widgets.FlatButton
	Buttons     []*SideBarButton
	list        *widget.List

	cache *op.Ops

	clickables []*widget.Clickable

	selectedIndex int
}

type SideBarButton struct {
	Icon *widget.Icon
	Text string
}

func NewSidebar(theme *chapartheme.Theme) *Sidebar {
	s := &Sidebar{
		Theme: theme,
		cache: new(op.Ops),

		Buttons: []*SideBarButton{
			{Icon: widgets.SwapHoriz, Text: "Requests"},
			{Icon: widgets.MenuIcon, Text: "Envs"},
			{Icon: widgets.WorkspacesIcon, Text: "Workspaces"},
			{Icon: widgets.FileFolderIcon, Text: "Proto files"},
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

	s.makeButtons(theme)

	return s
}

func (s *Sidebar) makeButtons(theme *chapartheme.Theme) {
	s.flatButtons = make([]*widgets.FlatButton, 0)
	for i, b := range s.Buttons {
		s.flatButtons = append(s.flatButtons, &widgets.FlatButton{
			Icon:              b.Icon,
			Text:              b.Text,
			IconPosition:      widgets.FlatButtonIconTop,
			Clickable:         s.clickables[i],
			SpaceBetween:      unit.Dp(5),
			BackgroundPadding: unit.Dp(1),
			CornerRadius:      5,
			MinWidth:          unit.Dp(70),
			BackgroundColor:   theme.SideBarBgColor,
			TextColor:         theme.SideBarTextColor,
			ContentPadding:    unit.Dp(5),
		})
	}
}

func (s *Sidebar) SelectedIndex() int {
	return s.selectedIndex
}

func (s *Sidebar) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	for i, c := range s.clickables {
		for c.Clicked(gtx) {
			s.selectedIndex = i
		}
	}

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			if !theme.IsDark() {
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 0).Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, theme.SideBarBgColor)
			}
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(2), Right: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return s.list.Layout(gtx, len(s.Buttons), func(gtx layout.Context, i int) layout.Dimensions {
							btn := s.flatButtons[i]
							if s.selectedIndex == i {
								btn.TextColor = theme.SideBarTextColor
							} else {
								btn.TextColor = widgets.Disabled(theme.SideBarTextColor)
							}

							return btn.Layout(gtx, theme)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !theme.IsDark() {
							return layout.Dimensions{}
						}
						return widgets.DrawLine(gtx, theme.SeparatorColor, unit.Dp(gtx.Constraints.Max.Y), unit.Dp(1))
					}),
				)
			})
		},
	)
}
