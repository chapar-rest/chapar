package notifications

import (
	"sort"
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

type NotificationType string

const (
	NotificationTypeInfo  NotificationType = "info"
	NotificationTypeWarn  NotificationType = "warn"
	NotificationTypeError NotificationType = "error"
)

type Notification struct {
	Text      string
	CreatedAt time.Time
	EndAt     time.Time
	Type      NotificationType

	closeButton widget.Clickable

	// Indicates if the notification is floating
	// Notifications are floating when the main notification list is not visible and when they closed they will turn into docked notifications
	// and will be visible in the main notification list
	isFloating bool

	removed bool // Indicates if the notification is removed from the list
}

var DefaultNotifications *Notifications

type Notifications struct {
	isVisible bool

	notifications []*Notification
	state         widget.List

	w        *app.Window
	progress float32

	closeButton widget.Clickable
	clearButton widget.Clickable

	mx *sync.Mutex
}

func (n *Notifications) refreshWindow() {
	if n.w == nil {
		return
	}
	n.w.Invalidate()
}

func (n *Notifications) floatingItems() []*Notification {
	var floating []*Notification
	for _, notif := range n.notifications {
		if notif.isFloating && !notif.removed && notif.EndAt.After(time.Now()) {
			floating = append(floating, notif)
		}
	}

	// only return last 5 floating notifications to avoid cluttering the UI
	if len(floating) > 5 {
		return floating[:5]
	}
	return floating
}

func (n *Notifications) dockedItems() []*Notification {
	var docked []*Notification
	for _, notif := range n.notifications {
		if !notif.removed && notif.EndAt.After(time.Now()) {
			docked = append(docked, notif)
		}
	}
	return docked
}

func New(w *app.Window) *Notifications {
	n := &Notifications{
		isVisible: false,
		w:         w,
		state: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},

		notifications: nil,
		mx:            &sync.Mutex{},
	}

	DefaultNotifications = n

	go n.curator()
	return n
}

func Send(text string, notifType NotificationType, duration time.Duration) {
	if DefaultNotifications == nil {
		panic("AddNotification called before creating DefaultNotifications")
	}

	DefaultNotifications.mx.Lock()
	defer DefaultNotifications.mx.Unlock()

	n := &Notification{
		Text:        text,
		CreatedAt:   time.Now(),
		EndAt:       time.Now().Add(duration),
		Type:        notifType,
		closeButton: widget.Clickable{},
		isFloating:  true,
	}

	if DefaultNotifications.notifications == nil {
		DefaultNotifications.notifications = make([]*Notification, 0)
	}

	DefaultNotifications.notifications = append(DefaultNotifications.notifications, n)
	// sort the notifications by CreatedAt in descending order
	sort.Slice(DefaultNotifications.notifications, func(i, j int) bool {
		return DefaultNotifications.notifications[i].CreatedAt.After(DefaultNotifications.notifications[j].CreatedAt)
	})

	DefaultNotifications.refreshWindow()
}

func (n *Notifications) curator() {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	invalidated := false

	for {
		select {
		case <-t.C:
			n.mx.Lock()
			for _, notif := range n.notifications {
				if notif.EndAt.Before(time.Now()) && notif.isFloating {
					notif.isFloating = false
					invalidated = true
				}
			}
			n.mx.Unlock()
			if invalidated {
				invalidated = false
				if n.w != nil {
					n.w.Invalidate()
				}
			}
		}
	}
}

func IsVisible() bool {
	if DefaultNotifications == nil {
		panic("IsVisible called before creating DefaultNotifications")
	}
	return DefaultNotifications.isVisible
}

func ToggleVisibility() {
	if DefaultNotifications == nil {
		panic("ToggleVisibility called before creating DefaultNotifications")
	}

	DefaultNotifications.isVisible = !DefaultNotifications.isVisible

	// set all floating notifications to docked if the main notification list is visible
	if DefaultNotifications.isVisible {
		for _, notif := range DefaultNotifications.notifications {
			notif.isFloating = false
		}
	}
}

func Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if DefaultNotifications == nil {
		panic("Layout called before creating DefaultNotifications")
	}

	return DefaultNotifications.Layout(gtx, theme)
}

func (n *Notifications) notificationLayout(gtx layout.Context, theme *chapartheme.Theme, notification *Notification) layout.Dimensions {
	if notification.closeButton.Clicked(gtx) {
		if notification.isFloating {
			notification.isFloating = false
		} else {
			notification.removed = true
		}
	}

	return widget.Border{
		Color:        widgets.MulAlpha(theme.ContrastFg, 0x22),
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(1),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(5), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(16))
						switch notification.Type {
						case NotificationTypeError:
							return widgets.ErrorIcon.Layout(gtx, chapartheme.LightRed)
						case NotificationTypeWarn:
							return widgets.WarningIcon.Layout(gtx, chapartheme.LightYellow)
						default:
							return widgets.InfoIcon.Layout(gtx, chapartheme.LightBlue)
						}
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme.Material(), theme.TextSize, notification.Text).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := widgets.Button(theme.Material(), &notification.closeButton, widgets.CloseIcon, widgets.IconPositionStart, "")
					btn.TextSize = unit.Sp(12)
					btn.IconSize = unit.Sp(12)
					btn.Background = theme.DropDownMenuBgColor
					btn.IconInset = layout.Inset{Right: unit.Dp(3)}
					btn.Inset = layout.UniformInset(unit.Dp(3))
					return btn.Layout(gtx, theme)
				}),
			)
		})
	})
}

func (n *Notifications) listLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	items := n.dockedItems()

	if len(items) == 0 {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme.Material(), theme.TextSize, "No Notification").Layout(gtx)
		})
	}

	return material.List(theme.Material(), &n.state).
		Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return n.notificationLayout(gtx, theme, items[index])
			})
		})
}

func (n *Notifications) actionsLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &n.clearButton, widgets.CleanIcon, widgets.IconPositionStart, "Clear")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.Background = theme.DropDownMenuBgColor
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.UniformInset(unit.Dp(3))
			return btn.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &n.closeButton, widgets.CloseIcon, widgets.IconPositionStart, "")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.Background = theme.DropDownMenuBgColor
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.UniformInset(unit.Dp(3))
			return btn.Layout(gtx, theme)
		}),
	)
}

func (n *Notifications) floatingNotifications(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	items := n.floatingItems()

	return material.List(theme.Material(), &n.state).
		Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				sf := component.Surface(theme.Material())
				sf.Fill = theme.DropDownMenuBgColor
				return sf.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return n.notificationLayout(gtx, theme, items[index])
				})
			})
		})
}

func (n *Notifications) floatingLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if n.isVisible || len(n.floatingItems()) == 0 {
		return layout.Dimensions{}
	}

	return layout.Inset{Right: unit.Dp(10), Bottom: unit.Dp(30)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.SE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(400)

			return n.floatingNotifications(gtx, theme)
		})
	})
}

func (n *Notifications) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if len(n.notifications) > 0 && !n.isVisible {
		return n.floatingLayout(gtx, theme)
	}

	if !n.isVisible {
		return layout.Dimensions{}
	}

	if n.closeButton.Clicked(gtx) {
		n.isVisible = false
	}

	if n.clearButton.Clicked(gtx) {
		n.notifications = nil
	}

	return layout.Inset{Right: unit.Dp(20), Bottom: unit.Dp(30)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.SE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			sf := component.Surface(theme.Material())
			sf.Fill = theme.DropDownMenuBgColor
			return sf.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
}
