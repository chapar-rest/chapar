package grpc

import (
	"gioui.org/layout"
	"gioui.org/unit"
	giox "gioui.org/x/component"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Grpc struct {
	Prompt *widgets.Prompt

	Req *domain.Request

	Breadcrumb *component.Breadcrumb
	AddressBar *AddressBar

	Request  *Request
	Response *Response

	split widgets.SplitView

	onSave        func(id string)
	onDataChanged func(id string, data any)
	onSubmit      func(id string)
}

func (r *Grpc) SetOnTitleChanged(f func(title string)) {
	//TODO implement me
	panic("implement me")
}

func (r *Grpc) SetDataChanged(changed bool) {

}

func New(req *domain.Request, theme *chapartheme.Theme) *Grpc {
	r := &Grpc{
		Req:        req,
		Prompt:     widgets.NewPrompt("", "", ""),
		Breadcrumb: component.NewBreadcrumb(req.MetaData.ID, req.CollectionName, "gRPC", req.MetaData.Name),
		split: widgets.SplitView{
			Resize: giox.Resize{
				Ratio: 0.5,
			},
			BarWidth: unit.Dp(2),
		},
		AddressBar: NewAddressBar(theme, req.Spec.GRPC.Host, req.Spec.GRPC.Method),
		Request:    NewRequest(req, theme),
		Response:   NewResponse(theme),
	}

	return r
}

func (r *Grpc) SetOnDataChanged(f func(id string, data any)) {
	r.onDataChanged = f
}

func (r *Grpc) SetOnSubmit(f func(id string)) {
	r.onSubmit = f
}

func (r *Grpc) SetOnSave(f func(id string)) {
	r.onSave = f
}

func (r *Grpc) HidePrompt() {
	r.Prompt.Hide()
}

func (r *Grpc) ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...widgets.Option) {
	r.Prompt.Type = modalType
	r.Prompt.Title = title
	r.Prompt.Content = content
	r.Prompt.SetOptions(options...)
	r.Prompt.WithoutRememberBool()
	r.Prompt.SetOnSubmit(onSubmit)
	r.Prompt.Show()
}

func (r *Grpc) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Prompt.Layout(gtx, theme)
				// return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(15), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.Breadcrumb.Layout(gtx, theme)
				})
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
						return r.Response.Layout(gtx, theme)
					},
				)
			}),
		)
	})
}
