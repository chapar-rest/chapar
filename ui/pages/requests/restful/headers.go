package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Headers struct {
	values *widgets.KeyValue

	onChange func()
}

func NewHeaders() *Headers {
	return &Headers{
		values: widgets.NewKeyValue(),
	}
}

func (h *Headers) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return h.values.WithAddLayout(gtx, "Headers", "", theme)
		}),
	)
}
