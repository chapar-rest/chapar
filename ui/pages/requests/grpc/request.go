package grpc

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"github.com/chapar-rest/chapar/ui/converter"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Request struct {
	Tabs *widgets.Tabs

	Body     *widgets.CodeEditor
	Metadata *widgets.KeyValue
	Auth     *component.Auth
}

func NewRequest(req *domain.Request, theme *chapartheme.Theme) *Request {
	r := &Request{
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Body"},
			{Title: "Auth"},
			{Title: "Meta Data"},
			{Title: "Proto files"},
			{Title: "Settings"},
		}, nil),
		Body: widgets.NewCodeEditor("", "JSON", theme),
		Metadata: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(req.Spec.GRPC.Metadata)...,
		),
		Auth: component.NewAuth(req.Spec.GRPC.Auth, theme),
	}

	return r
}

func (r *Request) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Tabs.Layout(gtx, theme)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				switch r.Tabs.SelectedTab().Title {
				case "Body":
					return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.Body.Layout(gtx, theme, "JSON")
					})
				case "Meta Data":
					return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.Metadata.WithAddLayout(gtx, "", "", theme)
					})
				case "Auth":
					return r.Auth.Layout(gtx, theme)
				default:
					return layout.Dimensions{}
				}
			}),
		)
	})
}
