package envs

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type envContainer struct {
	items *widgets.KeyValue
	title *widgets.EditableLabel
}

func newEnvContainer(title string) *envContainer {
	e := &envContainer{
		items: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),
		title: widgets.NewEditableLabel(title),
	}

	e.title.SetOnChanged(func(text string) {
		e.title.Text = text
	})

	return e
}

func (r *envContainer) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					// TODO : fix the max width
					// should be calculated based on the text length
					gtx.Constraints.Max.X = gtx.Dp(200)
					return r.title.Layout(gtx, theme)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return r.items.Layout(gtx, theme)
			}),
		)
	})
}
