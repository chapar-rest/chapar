package widgets

import (
	"sync"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var NotificationController = &Notification{
	mtx: sync.Mutex{},
}

type Notification struct {
	Text  string
	EndAt time.Time

	mtx sync.Mutex
}

type Notif struct {
	// Text is the text to display in the notification.
	Text string
	// Duration is the duration to display the notification.
	Duration time.Duration
}

func Noifiy(text string, duration time.Duration) {
	NotificationController.mtx.Lock()
	defer NotificationController.mtx.Unlock()

	NotificationController.EndAt = time.Now().Add(duration)
	NotificationController.Text = text
}

func (n *Notification) Layout(gtx layout.Context, theme *material.Theme, windowWidth int) layout.Dimensions {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	if n.Text == "" || n.EndAt == (time.Time{}) || time.Now().After(n.EndAt) {
		return layout.Dimensions{}
	}

	border := widget.Border{
		Color:        Gray600,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	// change the offset to move the notification to the right side of the screen
	offset := layout.Inset{
		Left:   unit.Dp((windowWidth / 2) - gtx.Metric.Dp(100)),
		Bottom: unit.Dp(40),
	}

	return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Body1(theme, n.Text).Layout(gtx)
			})
		})
	})
}
