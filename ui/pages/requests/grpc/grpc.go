package grpc

import (
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	giox "gioui.org/x/component"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/converter"
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
	r.Breadcrumb.SetOnTitleChanged(f)
}

func (r *Grpc) SetDataChanged(changed bool) {
	r.Breadcrumb.SetDataChanged(changed)
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
		AddressBar: NewAddressBar(theme, req.Spec.GRPC.ServerInfo.Host, req.Spec.GRPC.LasSelectedMethod, req.Spec.GRPC.Methods),
		Request:    NewRequest(req, theme),
		Response:   NewResponse(theme),
	}

	r.setupHooks()

	return r
}

func (r *Grpc) setupHooks() {
	r.AddressBar.SetOnServerAddressChanged(func(url string) {
		r.Req.Spec.GRPC.ServerInfo.Host = url
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.AddressBar.SetOnMethodChanged(func(method string) {
		r.Req.Spec.GRPC.LasSelectedMethod = method
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.AddressBar.SetOnSubmit(func() {
		r.onSubmit(r.Req.MetaData.ID)
	})

	r.Breadcrumb.SetOnSave(func(id string) {
		r.onSave(id)
	})

	r.Request.Body.SetOnChanged(func(data string) {
		r.Req.Spec.GRPC.Body = data
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.Metadata.SetOnChanged(func(items []*widgets.KeyValueItem) {
		data := converter.KeyValueFromWidgetItems(items)
		r.Req.Spec.GRPC.Metadata = data
		r.onDataChanged(r.Req.MetaData.ID, data)
	})

	r.Request.Settings.SetOnChange(func(values map[string]any) {
		r.Req.Spec.GRPC.Settings = convertSettingsToItems(values)
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})
}

func convertSettingsToItems(values map[string]any) domain.Settings {
	return domain.Settings{
		UseSSL:  false,
		Timeout: time.Hour,
	}
}

func (r *Grpc) SetOnDataChanged(f func(id string, data any)) {
	r.onDataChanged = f
}

func (r *Grpc) SetOnSubmit(f func(id string)) {
	r.onSubmit = f
}

func (r *Grpc) SetOnSave(f func(id string)) {
	r.onSave = f
	r.Breadcrumb.SetOnSave(f)
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
