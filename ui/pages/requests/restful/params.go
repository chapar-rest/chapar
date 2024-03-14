package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/converter"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Params struct {
	queryParams *widgets.KeyValue
	urlParams   *widgets.KeyValue

	onChange func(queryParams []domain.KeyValue, urlParams []domain.KeyValue)
}

func NewParams(queryParams []domain.KeyValue, urlParams []domain.KeyValue) *Params {
	return &Params{
		queryParams: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(queryParams)...,
		),
		urlParams: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(urlParams)...,
		),
	}
}

func (p *Params) SetOnChange(f func(queryParams []domain.KeyValue, urlParams []domain.KeyValue)) {
	p.onChange = f

	p.urlParams.SetOnChanged(func(items []*widgets.KeyValueItem) {
		p.onChange(converter.KeyValueFromWidgetItems(p.queryParams.Items), converter.KeyValueFromWidgetItems(p.urlParams.Items))
	})

	p.queryParams.SetOnChanged(func(items []*widgets.KeyValueItem) {
		p.onChange(converter.KeyValueFromWidgetItems(p.queryParams.Items), converter.KeyValueFromWidgetItems(p.urlParams.Items))
	})
}

func (p *Params) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(15), Right: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.queryParams.WithAddLayout(gtx, "Query", "", theme)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(30)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.urlParams.WithAddLayout(gtx, "Path", "path params inside bracket, for example: {id}", theme)
			}),
		)
	})
}
