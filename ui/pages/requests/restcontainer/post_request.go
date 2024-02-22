package restcontainer

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func (r *RestContainer) requestPostReqLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme, theme.TextSize, "Action to do after request").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return r.postRequestDropDown.Layout(gtx, theme)
				}),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			hint := ""
			if r.postRequestDropDown.SelectedIndex() == 1 {
				hint = "Python script"
			} else if r.postRequestDropDown.SelectedIndex() == 2 {
				hint = "SSH script"
			} else {
				return layout.Dimensions{}
			}

			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.postRequestBody.Layout(gtx, theme, hint)
			})
		}),
	)
}
