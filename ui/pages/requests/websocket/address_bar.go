package websocket

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type AddressBar struct {
	theme         *chapartheme.Theme
	serverAddress *widgets.PatternEditor

	ConnectClickable widget.Clickable
}

func NewAddressBar(theme *chapartheme.Theme, address string) *AddressBar {
	a := &AddressBar{
		theme:         theme,
		serverAddress: widgets.NewPatternEditor(),
	}

	a.serverAddress.SingleLine = true
	a.serverAddress.Submit = true
	a.serverAddress.SetText(address)

	return a
}

func (a *AddressBar) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	borderColor := theme.BorderColor
	if gtx.Source.Focused(a.serverAddress) {
		borderColor = theme.BorderColorFocused
	}

	border := widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = gtx.Dp(20)
			return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.serverAddress.Layout(gtx, theme, "wss://localhost:8080")
					})
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.ConnectClickable.Clicked(gtx) {
				//if a.onSubmit != nil {
				//	go a.onSubmit()
				//}
			}

			gtx.Constraints.Min.X = gtx.Dp(80)
			btn := material.Button(theme.Material(), &a.ConnectClickable, "Connect")
			btn.Background = theme.ActionButtonBgColor
			btn.Color = theme.ButtonTextColor
			return btn.Layout(gtx)
		}),
	)
}
