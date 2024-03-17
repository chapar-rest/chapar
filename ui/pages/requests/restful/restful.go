package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/pages/requests/component"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Restful struct {
	Prompt *widgets.Prompt

	Req *domain.Request

	Breadcrumb *component.Breadcrumb
	AddressBar *component.AddressBar
	Response   *Response
	Request    *Request

	split widgets.SplitView

	onSave        func(id string)
	onDataChanged func(id string, data any)
	onSubmit      func(id string)
}

func (r *Restful) SetOnDataChanged(f func(id string, data any)) {
	r.onDataChanged = f
}

func (r *Restful) SetOnSubmit(f func(id string)) {
	r.onSubmit = f
}

func (r *Restful) SetDataChanged(changed bool) {
	r.Breadcrumb.SetDataChanged(changed)
}

func (r *Restful) SetOnTitleChanged(f func(title string)) {
	r.Breadcrumb.SetOnTitleChanged(f)
}

func (r *Restful) SetOnSave(f func(id string)) {
	r.onSave = f
}

func (r *Restful) ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...string) {
	r.Prompt.Type = modalType
	r.Prompt.Title = title
	r.Prompt.Content = content
	r.Prompt.SetOptions(options...)
	r.Prompt.WithRememberBool()
	r.Prompt.SetOnSubmit(onSubmit)
	r.Prompt.Show()
}

func (r *Restful) HidePrompt() {
	r.Prompt.Hide()
}

func New(req *domain.Request, theme *material.Theme) *Restful {
	r := &Restful{
		Req:        req,
		Prompt:     widgets.NewPrompt("", "", ""),
		Breadcrumb: component.NewBreadcrumb(req.MetaData.ID, req.CollectionName, req.MetaData.Type, req.MetaData.Name),
		AddressBar: component.NewAddressBar(req.Spec.HTTP.URL, req.Spec.HTTP.Method),
		split: widgets.SplitView{
			Ratio:         0.05,
			BarWidth:      unit.Dp(2),
			BarColor:      widgets.Gray300,
			BarColorHover: theme.Palette.ContrastBg,
		},
		Response: NewResponse(theme),
		Request:  NewRequest(req),
	}
	r.setupHooks()

	return r
}

func (r *Restful) setupHooks() {
	r.Breadcrumb.SetOnSave(func(id string) {
		r.onSave(id)
	})

	r.AddressBar.SetOnMethodChanged(func(method string) {
		r.Req.Spec.HTTP.Method = method
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.AddressBar.SetOnURLChanged(func(url string) {
		r.Req.Spec.HTTP.URL = url
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.AddressBar.SetOnSubmit(func() {
		r.onSubmit(r.Req.MetaData.ID)
	})

	r.Request.Params.SetOnChange(func(queryParams []domain.KeyValue, urlParams []domain.KeyValue) {
		r.Req.Spec.HTTP.Request.QueryParams = queryParams
		r.Req.Spec.HTTP.Request.PathParams = urlParams
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.Headers.SetOnChange(func(headers []domain.KeyValue) {
		r.Req.Spec.HTTP.Request.Headers = headers
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.Auth.SetOnChange(func(auth domain.Auth) {
		r.Req.Spec.HTTP.Request.Auth = auth
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.PreRequest.SetOnDropDownChanged(func(selected string) {
		r.Req.Spec.HTTP.Request.PreRequest.Type = selected
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.PreRequest.SetOnScriptChanged(func(code string) {
		r.Req.Spec.HTTP.Request.PreRequest.Script = code
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.PostRequest.SetOnDropDownChanged(func(selected string) {
		r.Req.Spec.HTTP.Request.PostRequest.Type = selected
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.PostRequest.SetOnScriptChanged(func(code string) {
		r.Req.Spec.HTTP.Request.PostRequest.Script = code
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.Body.SetOnChange(func(body domain.Body) {
		r.Req.Spec.HTTP.Request.Body = body
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})
}

func (r *Restful) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Prompt.Layout(gtx, theme)
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
				return r.split.Layout(gtx,
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
