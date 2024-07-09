package widgets

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type Notification struct {
	Text  string
	EndAt time.Time
}

type Notif struct {
	// Text is the text to display in the notification.
	Text string
	// Duration is the duration to display the notification.
	Duration time.Duration
}

func (n *Notification) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if n.Text == "" || n.EndAt == (time.Time{}) || time.Now().After(n.EndAt) {
		return layout.Dimensions{}
	}

	// set max width for the notification
	gtx.Constraints.Max.X = gtx.Dp(300)
	// set max height for the notification
	gtx.Constraints.Max.Y = gtx.Dp(40)

	macro := op.Record(gtx.Ops)
	dim := layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 8).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, theme.NotificationBgColor)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				bd := material.Body1(theme.Material(), n.Text)
				bd.Color = theme.NotificationTextColor
				return bd.Layout(gtx)
			})
		},
	)
	call := macro.Stop()

	return layout.SE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Bottom: unit.Dp(40), Right: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			call.Add(gtx.Ops)
			return dim
		})
	})
}
