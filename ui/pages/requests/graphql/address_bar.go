package graphql

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type AddressBar struct {
	url *widgets.PatternEditor

	sendClickable widget.Clickable

	onURLChanged func(url string)
	onSubmit     func()
}

func NewAddressBar(url string) *AddressBar {
	a := &AddressBar{
		url: widgets.NewPatternEditor(),
	}

	a.url.SingleLine = true
	a.url.Submit = true
	a.url.SetText(url)

	return a
}

func (a *AddressBar) SetOnURLChanged(onURLChanged func(url string)) {
	a.onURLChanged = onURLChanged
}

func (a *AddressBar) SetOnSubmit(onSubmit func()) {
	a.onSubmit = onSubmit
}

func (a *AddressBar) SetURL(url string) {
	a.url.SetText(url)
}

func (a *AddressBar) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if a.url.Changed() && a.onURLChanged != nil {
		a.onURLChanged(a.url.Text())
	}
	if a.url.Submitted() && a.onSubmit != nil {
		a.onSubmit()
	}

	borderColor := theme.BorderColor
	if gtx.Source.Focused(a.url) {
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
			gtx.Constraints.Min.Y = gtx.Dp(40)
			return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Left:   unit.Dp(10),
						Right:  unit.Dp(5),
						Top:    unit.Dp(8),
						Bottom: unit.Dp(8),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.url.Layout(gtx, theme, "https://example.com/graphql")
					})
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.sendClickable.Clicked(gtx) {
				if a.onSubmit != nil {
					a.onSubmit()
				}
			}

			gtx.Constraints.Min.X = gtx.Dp(80)
			btn := material.Button(theme.Material(), &a.sendClickable, "Send")
			btn.Background = theme.ActionButtonBgColor
			btn.Color = theme.ButtonTextColor
			return btn.Layout(gtx)
		}),
	)
}
