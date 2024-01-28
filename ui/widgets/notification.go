package widgets

import (
	"image"
	"sync"
	"time"

	"gioui.org/op"

	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type Notification struct {
	Text  string
	EndAt time.Time

	Mtx sync.Mutex
}

type Notif struct {
	// Text is the text to display in the notification.
	Text string
	// Duration is the duration to display the notification.
	Duration time.Duration
}

func (n *Notification) Layout(gtx layout.Context, theme *material.Theme, windowWidth int) layout.Dimensions {
	n.Mtx.Lock()
	defer n.Mtx.Unlock()

	if n.Text == "" || n.EndAt == (time.Time{}) || time.Now().After(n.EndAt) {
		return layout.Dimensions{}
	}

	macro := op.Record(gtx.Ops)
	dim := layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 8).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, theme.Palette.ContrastBg)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Body1(theme, n.Text).Layout(gtx)
			})
		},
	)
	call := macro.Stop()
	// change the offset to move the notification to the right side of the screen
	offset := layout.Inset{
		Top:    0,
		Left:   unit.Dp(windowWidth/2 - dim.Size.X),
		Right:  0,
		Bottom: unit.Dp(40),
	}

	return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		call.Add(gtx.Ops)
		return dim
	})
}
