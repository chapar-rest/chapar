package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	giox "gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
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

func New(req *domain.Request, theme *chapartheme.Theme) *Restful {
	r := &Restful{
		Req:        req,
		Prompt:     widgets.NewPrompt("", "", ""),
		Breadcrumb: component.NewBreadcrumb(req.MetaData.ID, req.CollectionName, req.Spec.HTTP.Method, req.MetaData.Name),
		AddressBar: component.NewAddressBar(theme, req.Spec.HTTP.URL, req.Spec.HTTP.Method),
		split: widgets.SplitView{
			Resize: giox.Resize{
				Ratio: 0.5,
			},
			BarWidth: unit.Dp(2),
		},
		Response: NewResponse(theme),
		Request:  NewRequest(req, theme),
	}
	r.setupHooks()

	return r
}

func (r *Restful) SetPostRequestSetValues(set domain.PostRequestSet) {
	r.Request.PostRequest.SetPostRequestSetValues(set)
}

func (r *Restful) SetPostRequestSetPreview(preview string) {
	r.Request.PostRequest.SetPreview(preview)
}

func (r *Restful) SetOnPostRequestSetChanged(f func(id string, statusCode int, item, from, fromKey string)) {
	r.Request.PostRequest.SetOnPostRequestSetChanged(func(statusCode int, item, from, fromKey string) {
		f(r.Req.MetaData.ID, statusCode, item, from, fromKey)
	})
}

func (r *Restful) SetOnDataChanged(f func(id string, data any)) {
	r.onDataChanged = f
}

func (r *Restful) SetOnSubmit(f func(id string)) {
	r.onSubmit = f
}

func (r *Restful) SetURL(url string) {
	r.AddressBar.SetURL(url)
}

func (r *Restful) SetOnBinaryFileSelect(f func(id string)) {
	r.Request.Body.BinaryFile.SetOnSelectFile(func() {
		f(r.Req.MetaData.ID)
	})
}

func (r *Restful) SetOnFormDataFileSelect(f func(requestId, fieldId string)) {
	r.Request.Body.FormData.SetOnSelectFile(func(fieldId string) {
		f(r.Req.MetaData.ID, fieldId)
	})
}

func (r *Restful) AddFileToFormData(fieldId, filePath string) {
	r.Request.Body.FormData.AddFile(fieldId, filePath)
}

func (r *Restful) SetBinaryBodyFilePath(filePath string) {
	r.Request.Body.BinaryFile.SetFileName(filePath)
}

func (r *Restful) SetDataChanged(changed bool) {
	r.Breadcrumb.SetDataChanged(changed)
}

func (r *Restful) SetOnTitleChanged(f func(title string)) {
	r.Breadcrumb.SetOnTitleChanged(f)
}

func (r *Restful) SetOnCopyResponse(f func(gtx layout.Context, dataType, data string)) {
	r.Response.SetOnCopyResponse(f)
}

func (r *Restful) SetHTTPResponse(detail domain.HTTPResponseDetail) {
	if detail.Error != nil {
		r.Response.SetError(detail.Error)
		return
	}

	r.Response.SetResponse(detail.Response)
	r.Response.SetHeaders(detail.Headers)
	r.Response.SetCookies(detail.Cookies)
	r.Response.SetStatusParams(detail.StatusCode, detail.Duration, detail.Size)
}

func (r *Restful) GetHTTPResponse() *domain.HTTPResponseDetail {
	return &domain.HTTPResponseDetail{
		Response: r.Response.response,
		Headers:  r.Response.responseHeaders.GetData(),
		Cookies:  r.Response.responseCookies.GetData(),
	}
}

func (r *Restful) ShowSendingRequestLoading() {
	r.Response.SetMessage("Sending request...")
}

func (r *Restful) HideSendingRequestLoading() {
	r.Response.SetMessage("")
}

func (r *Restful) SetOnSave(f func(id string)) {
	r.onSave = f
}

func (r *Restful) ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...widgets.Option) {
	r.Prompt.Type = modalType
	r.Prompt.Title = title
	r.Prompt.Content = content
	r.Prompt.SetOptions(options...)
	r.Prompt.WithoutRememberBool()
	r.Prompt.SetOnSubmit(onSubmit)
	r.Prompt.Show()
}

func (r *Restful) HidePrompt() {
	r.Prompt.Hide()
}

func (r *Restful) setupHooks() {
	r.Breadcrumb.SetOnSave(func(id string) {
		r.onSave(id)
	})

	r.AddressBar.SetOnMethodChanged(func(method string) {
		r.Req.Spec.HTTP.Method = method
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
		r.Breadcrumb.SetContainerType(method)
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

	r.Request.Body.BinaryFile.SetOnChanged(func(filePath string) {
		r.Req.Spec.HTTP.Request.Body.BinaryFilePath = filePath
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	r.Request.Auth.SetOnChange(func(auth domain.Auth) {
		r.Req.Spec.HTTP.Request.Auth = auth
		r.onDataChanged(r.Req.MetaData.ID, r.Req)
	})

	// r.Request.PreRequest.SetOnDropDownChanged(func(selected string) {
	//	r.Req.Spec.HTTP.Request.PreRequest.Type = selected
	//	r.onDataChanged(r.Req.MetaData.ID, r.Req)
	// })

	// r.Request.PreRequest.SetOnScriptChanged(func(code string) {
	//	r.Req.Spec.HTTP.Request.PreRequest.Script = code
	//	r.onDataChanged(r.Req.MetaData.ID, r.Req)
	// })

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

func (r *Restful) SetQueryParams(params []domain.KeyValue) {
	r.Request.Params.SetQueryParams(params)
}

func (r *Restful) SetPathParams(params []domain.KeyValue) {
	r.Request.Params.SetPathParams(params)
}

func (r *Restful) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
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
