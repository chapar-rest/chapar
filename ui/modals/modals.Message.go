package modals

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets/card"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type MessageType string

const (
	MessageTypeInfo MessageType = "info"
	MessageTypeWarn MessageType = "warn"
	MessageTypeErr  MessageType = "err"
)

type Message struct {
	Title string
	Body  string
	Type  MessageType

	OKBtn widget.Clickable
}

func (m *Message) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	textColor := th.Fg
	switch m.Type {
	case MessageTypeErr:
		textColor = th.ErrorColor
	case MessageTypeInfo:
		textColor = th.InfoColor
	case MessageTypeWarn:
		textColor = th.WarningColor
	}

	marginTop := layout.Inset{Top: unit.Dp(90)}

	return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X /= 3
			return marginTop.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return card.Card{
					Title: m.Title,
					Body: func(gtx layout.Context) layout.Dimensions {
						lb := material.Label(th.Material(), unit.Sp(14), m.Body)
						lb.Color = textColor
						return lb.Layout(gtx)
					},
					Actions: []card.Action{
						{
							Clickable: &m.OKBtn,
							Label:     "Ok",
							Fg:        th.ButtonTextColor,
							Bg:        th.ActionButtonBgColor,
							Float:     card.FloatRight,
						},
					},
				}.Layout(gtx, th.Material())
			})
		})
	})
}
