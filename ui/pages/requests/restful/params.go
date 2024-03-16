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
	pathParams  *widgets.KeyValue

	onChange func(queryParams []domain.KeyValue, urlParams []domain.KeyValue)
}

func NewParams(queryParams []domain.KeyValue, pathParams []domain.KeyValue) *Params {
	return &Params{
		queryParams: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(queryParams)...,
		),
		pathParams: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(pathParams)...,
		),
	}
}

func (p *Params) SetQueryParams(queryParams []domain.KeyValue) {
	p.queryParams.SetItems(converter.WidgetItemsFromKeyValue(queryParams))
}

func (p *Params) SetPathParams(pathParams []domain.KeyValue) {
	p.pathParams.SetItems(converter.WidgetItemsFromKeyValue(pathParams))
}

func (p *Params) SetOnChange(f func(queryParams []domain.KeyValue, pathParams []domain.KeyValue)) {
	p.onChange = f

	p.pathParams.SetOnChanged(func(items []*widgets.KeyValueItem) {
		p.onChange(converter.KeyValueFromWidgetItems(p.queryParams.Items), converter.KeyValueFromWidgetItems(p.pathParams.Items))
	})

	p.queryParams.SetOnChanged(func(items []*widgets.KeyValueItem) {
		p.onChange(converter.KeyValueFromWidgetItems(p.queryParams.Items), converter.KeyValueFromWidgetItems(p.queryParams.Items))
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
				return p.pathParams.WithAddLayout(gtx, "Path", "path params inside bracket, for example: {id}", theme)
			}),
		)
	})
}
