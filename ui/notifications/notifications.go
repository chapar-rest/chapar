package notifications

import (
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Notification struct {
	Text  string
	EndAt time.Time

	closeButton widget.Clickable
}

var DefaultNotifications *Notifications

type Notifications struct {
	isVisible bool

	notifications []Notification
	state         widget.List

	w        *app.Window
	progress float32

	closeButton widget.Clickable
	clearButton widget.Clickable

	mx *sync.Mutex
}

func New(w *app.Window) *Notifications {
	n := &Notifications{
		isVisible: true,
		w:         w,
		state: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},

		notifications: []Notification{
			{
				Text:        "Welcome to Chapar!",
				closeButton: widget.Clickable{},
			},
			{
				Text:        "An update is available. Please check the release page.",
				closeButton: widget.Clickable{},
			},
		},

		mx: &sync.Mutex{},
	}

	DefaultNotifications = n
	return n
}

func IsVisible() bool {
	if DefaultNotifications == nil {
		panic("IsVisible called before creating DefaultNotifications")
	}
	return DefaultNotifications.isVisible
}

func SetVisible(visible bool) {
	if DefaultNotifications == nil {
		panic("SetVisible called before creating DefaultNotifications")
	}

	DefaultNotifications.isVisible = visible
}

func ToggleVisibility() {
	if DefaultNotifications == nil {
		panic("ToggleVisibility called before creating DefaultNotifications")
	}

	DefaultNotifications.isVisible = !DefaultNotifications.isVisible
}

func Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if DefaultNotifications == nil {
		panic("Layout called before creating DefaultNotifications")
	}

	return DefaultNotifications.Layout(gtx, theme)
}

func (n *Notifications) notificationLayout(gtx layout.Context, theme *chapartheme.Theme, index int) layout.Dimensions {
	if n.notifications[index].closeButton.Clicked(gtx) {
		n.mx.Lock()
		n.notifications = append(n.notifications[:index], n.notifications[index+1:]...)
		n.mx.Unlock()
		return layout.Dimensions{}
	}

	return widget.Border{
		Color:        theme.TableBorderColor,
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(1),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme.Material(), theme.TextSize, n.notifications[index].Text).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := widgets.Button(theme.Material(), &n.notifications[index].closeButton, widgets.CloseIcon, widgets.IconPositionStart, "")
					btn.TextSize = unit.Sp(12)
					btn.IconSize = unit.Sp(12)
					btn.IconInset = layout.Inset{Right: unit.Dp(3)}
					btn.Inset = layout.UniformInset(unit.Dp(3))
					return btn.Layout(gtx, theme)
				}),
			)
		})
	})
}

func (n *Notifications) listLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if len(n.notifications) == 0 {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme.Material(), theme.TextSize, "No Notification").Layout(gtx)
		})
	}

	return material.List(theme.Material(), &n.state).
		Layout(gtx, len(n.notifications), func(gtx layout.Context, index int) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return n.notificationLayout(gtx, theme, index)
			})
		})
}

func (n *Notifications) actionsLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &n.clearButton, widgets.CleanIcon, widgets.IconPositionStart, "Clear")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.UniformInset(unit.Dp(3))
			return btn.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &n.closeButton, widgets.CloseIcon, widgets.IconPositionStart, "")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.UniformInset(unit.Dp(3))
			return btn.Layout(gtx, theme)
		}),
	)
}

func (n *Notifications) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if !n.isVisible {
		return layout.Dimensions{}
	}

	if n.closeButton.Clicked(gtx) {
		n.isVisible = false
	}

	if n.clearButton.Clicked(gtx) {
		n.notifications = []Notification{}
	}

	return layout.Inset{Right: unit.Dp(20), Bottom: unit.Dp(30)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.SE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return component.Surface(theme.Material()).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return widget.Border{
					Color:        theme.TableBorderColor,
					CornerRadius: unit.Dp(4),
					Width:        unit.Dp(1),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Max.X = gtx.Dp(400)
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return n.listLayout(gtx, theme)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return n.actionsLayout(gtx, theme)
							}),
						)
					})
				})
			})
		})
	})
}
