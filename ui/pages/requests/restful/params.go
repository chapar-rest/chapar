package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Params struct {
	queryParams *widgets.KeyValue
	urlParams   *widgets.KeyValue

	onChange func()
}

func NewParams() *Params {
	return &Params{
		queryParams: widgets.NewKeyValue(),
		urlParams:   widgets.NewKeyValue(),
	}
}

func (p *Params) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return p.queryParams.WithAddLayout(gtx, "Query", "", theme)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(30)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return p.urlParams.WithAddLayout(gtx, "Path", "path params inside bracket, for example: {id}", theme)
		}),
	)
}
