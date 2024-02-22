package restcontainer

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func (r *RestContainer) requestBodyLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme, theme.TextSize, "Request body").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return r.requestBodyDropDown.Layout(gtx, theme)
				}),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				switch r.requestBodyDropDown.SelectedIndex() {
				case 1, 2, 3: // json, text, xml
					hint := ""
					if r.requestBodyDropDown.SelectedIndex() == 1 {
						hint = "Enter json"
					} else if r.requestBodyDropDown.SelectedIndex() == 2 {
						hint = "Enter text"
					} else if r.requestBodyDropDown.SelectedIndex() == 3 {
						hint = "Enter xml"
					}

					return r.requestBody.Layout(gtx, theme, hint)
				case 4: // form data
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Start,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return r.formDataParams.WithAddLayout(gtx, "", "", theme)
						}),
					)
				case 5: // binary
					return r.requestBodyBinary.Layout(gtx, theme)
				case 6: // urlencoded
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Start,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return r.urlEncodedParams.WithAddLayout(gtx, "", "", theme)
						}),
					)
				default:
					return layout.Dimensions{}
				}
			})
		}),
	)
}
