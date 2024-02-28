package rest

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

func (r *Container) authLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return r.authDropDown.Layout(gtx, theme)
					}),
				)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				switch r.authDropDown.SelectedIndex() {
				case 1: // basic auth
					return r.basicAuthLayout(gtx, theme)
				case 2: // bearer token
					return r.bearerToken(gtx, theme)
				default:
					return layout.Dimensions{}
				}
			})
		}),
	)
}

func (r *Container) bearerToken(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lb := &widgets.LabeledInput{
					Label:          "Token",
					SpaceBetween:   5,
					MinEditorWidth: unit.Dp(150),
					MinLabelWidth:  unit.Dp(80),
					Editor:         r.bearerTokenEditor,
				}
				return lb.Layout(gtx, theme)
			})
		}),
	)
}

func (r *Container) basicAuthLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lb := &widgets.LabeledInput{
					Label:          "Username",
					SpaceBetween:   5,
					MinEditorWidth: unit.Dp(150),
					MinLabelWidth:  unit.Dp(80),
					Editor:         r.basicAuthUsername,
				}
				return lb.Layout(gtx, theme)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lb := &widgets.LabeledInput{
					Label:          "Password",
					SpaceBetween:   5,
					MinEditorWidth: unit.Dp(150),
					MinLabelWidth:  unit.Dp(80),
					Editor:         r.basicAuthPassword,
				}
				return lb.Layout(gtx, theme)
			})
		}),
	)
}
