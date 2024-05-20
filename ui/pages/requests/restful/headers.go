package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/converter"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Headers struct {
	values *widgets.KeyValue

	onChange func(values []domain.KeyValue)
}

func NewHeaders(headers []domain.KeyValue) *Headers {
	return &Headers{
		values: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(headers)...,
		),
	}
}

func (h *Headers) SetHeaders(headers []domain.KeyValue) {
	h.values.SetItems(converter.WidgetItemsFromKeyValue(headers))
}

func (h *Headers) SetOnChange(f func(values []domain.KeyValue)) {
	h.onChange = f

	h.values.SetOnChanged(func(items []*widgets.KeyValueItem) {
		h.onChange(converter.KeyValueFromWidgetItems(h.values.Items))
	})
}

func (h *Headers) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(15), Right: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return h.values.WithAddLayout(gtx, "Headers", "", theme)
			}),
		)
	})
}
