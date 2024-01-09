package pages

import (
	"example.com/gio_test/ui/widgets"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Request struct {
	theme *material.Theme

	actions *Actions

	textEditor widget.Editor
	searchBox  *widgets.TextField
}

func NewRequest(theme *material.Theme) *Request {
	search := widgets.NewTextField(theme, "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	return &Request{
		theme:     theme,
		actions:   NewActions(theme),
		searchBox: search,
	}
}

func (r *Request) list(gtx layout.Context) layout.Dimensions {
	return layout.Inset{
		Top:    10,
		Bottom: 0,
		Left:   10,
		Right:  10,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.actions.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.searchBox.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.List(r.theme, &widget.List{
					List: layout.List{
						Axis: layout.Vertical,
					},
				},
				).Layout(gtx, 1, func(gtx layout.Context, index int) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Body1(r.theme, "Request").Layout(gtx)
					})
				})
			}),
		)
	})
}

func (r *Request) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
			return r.list(gtx)
		}),
		widgets.VerticalLine(),
		layout.Flexed(0.8, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.H3(r.theme, "Request").Layout(gtx)
			})
		}),
	)
}
