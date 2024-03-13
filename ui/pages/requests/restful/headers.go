package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/converter"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Headers struct {
	values *widgets.KeyValue

	onChange func()
}

func NewHeaders(headers []domain.KeyValue) *Headers {
	return &Headers{
		values: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(headers)...,
		),
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
