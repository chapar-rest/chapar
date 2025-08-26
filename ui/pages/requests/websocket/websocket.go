package websocket

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	giox "gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type WebSocket struct {
	Prompt *widgets.Prompt
	Req    *domain.Request

	Breadcrumb *component.Breadcrumb
	AddressBar *AddressBar
	Actions    *component.Actions

	split widgets.SplitView

	Request *Request
}

func New(req *domain.Request, theme *chapartheme.Theme) *WebSocket {
	splitAxis := layout.Vertical
	if prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit {
		splitAxis = layout.Horizontal
	}

	r := &WebSocket{
		Req:        req,
		Prompt:     widgets.NewPrompt("", "", ""),
		Breadcrumb: component.NewBreadcrumb(req.MetaData.ID, req.CollectionName, "gRPC", req.MetaData.Name),
		split: widgets.SplitView{
			Resize: giox.Resize{
				Ratio: 0.5,
				Axis:  splitAxis,
			},
			BarWidth: unit.Dp(2),
		},
		AddressBar: NewAddressBar(theme, req.Spec.GRPC.ServerInfo.Address),
		Actions:    component.NewActions(false),
	}

	return r
}

func (r *WebSocket) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if r.Actions.IsDataChanged && r.Actions.SaveButton.Clicked(gtx) {
		r.Actions.IsDataChanged = false
	}

	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Prompt.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(15), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return r.Breadcrumb.Layout(gtx, theme)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return r.Actions.Layout(gtx, theme)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.AddressBar.Layout(gtx, theme)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return r.split.Layout(gtx, theme,
					func(gtx layout.Context) layout.Dimensions {
						return r.Request.Layout(gtx, theme)
					},
					func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme.Material(), unit.Sp(14), "Response view is not implemented yet").Layout(gtx)
					},
				)
			}),
		)
	})
}
