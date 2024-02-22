package restcontainer

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func (r *RestContainer) requestPreReqLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme, theme.TextSize, "Action to do before request").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return r.preRequestDropDown.Layout(gtx, theme)
				}),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			hint := ""
			if r.preRequestDropDown.SelectedIndex() == 1 {
				hint = "Python script"
			} else if r.preRequestDropDown.SelectedIndex() == 2 {
				hint = "SSH script"
			} else {
				return layout.Dimensions{}
			}

			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.preRequestBody.Layout(gtx, theme, hint)
			})
		}),
	)
}
