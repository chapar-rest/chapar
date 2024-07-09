package component

import (
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type AddressBar struct {
	url *widget.Editor

	lastSelectedMethod string
	methodDropDown     *widgets.DropDown

	sendClickable widget.Clickable

	onURLChanged    func(url string)
	onMethodChanged func(method string)
	onSubmit        func()
}

func NewAddressBar(theme *chapartheme.Theme, address, method string) *AddressBar {
	a := &AddressBar{
		url:                &widget.Editor{},
		methodDropDown:     widgets.NewDropDownWithoutBorder(theme),
		lastSelectedMethod: method,
	}

	a.url.SingleLine = true
	a.url.Submit = true
	a.url.SetText(address)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	opts := make([]*widgets.DropDownOption, 0, len(methods))
	for _, m := range methods {
		opts = append(opts, widgets.NewDropDownOption(m))
	}
	a.methodDropDown.SetOptions(opts...)
	a.methodDropDown.SetSelectedByTitle(strings.ToUpper(method))
	a.methodDropDown.MaxWidth = unit.Dp(90)

	return a
}

func (a *AddressBar) SetSelectedMethod(method string) {
	a.methodDropDown.SetSelectedByTitle(method)
	a.lastSelectedMethod = method
}

func (a *AddressBar) SetOnURLChanged(onURLChanged func(url string)) {
	a.onURLChanged = onURLChanged
}

func (a *AddressBar) SetOnMethodChanged(onMethodChanged func(method string)) {
	a.onMethodChanged = onMethodChanged
	a.methodDropDown.SetOnChanged(func(selected string) {
		a.onMethodChanged(selected)
	})
}

func (a *AddressBar) SetOnSubmit(onSubmit func()) {
	a.onSubmit = onSubmit
}

func (a *AddressBar) SetURL(url string) {
	a.url.SetText(url)
}

func (a *AddressBar) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	borderColor := theme.BorderColor
	if gtx.Source.Focused(a.url) {
		borderColor = theme.BorderColorFocused
	}

	border := widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	for {
		event, ok := a.url.Update(gtx)
		if !ok {
			break
		}

		switch event.(type) {
		// on carriage return event
		case widget.SubmitEvent:
			if a.onSubmit != nil {
				// goroutine to prevent blocking the ui update
				go a.onSubmit()
			}
		// on change event
		case widget.ChangeEvent:
			if a.onURLChanged != nil {
				a.onURLChanged(a.url.Text())
			}
		}
	}

	if a.methodDropDown.GetSelected().Text != a.lastSelectedMethod {
		a.lastSelectedMethod = a.methodDropDown.GetSelected().Text
		if a.onMethodChanged != nil {
			a.onMethodChanged(a.lastSelectedMethod)
		}
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = gtx.Dp(20)
			return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.Y = gtx.Dp(20)
							return a.methodDropDown.Layout(gtx, theme)
						}),
						widgets.DrawLineFlex(theme.SeparatorColor, unit.Dp(20), unit.Dp(1)),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints.Min.Y = gtx.Dp(20)
								editor := material.Editor(theme.Material(), a.url, "https://example.com")
								editor.SelectionColor = theme.TextSelectionColor
								editor.TextSize = unit.Sp(14)
								return editor.Layout(gtx)
							})
						}),
					)
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.sendClickable.Clicked(gtx) {
				if a.onSubmit != nil {
					go a.onSubmit()
				}
			}

			gtx.Constraints.Min.X = gtx.Dp(80)
			btn := material.Button(theme.Material(), &a.sendClickable, "Send")
			btn.Background = theme.SendButtonBgColor
			btn.Color = theme.ButtonTextColor
			return btn.Layout(gtx)
		}),
	)
}
