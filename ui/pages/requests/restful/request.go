package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/pages/requests/component"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Request struct {
	Tabs *widgets.Tabs

	PreRequest  *component.PrePostRequest
	PostRequest *component.PrePostRequest

	Body    *Body
	Params  *Params
	Headers *Headers
	Auth    *Auth
}

func NewRequest(req *domain.Request) *Request {
	r := &Request{
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Params"},
			{Title: "Body"},
			{Title: "Auth"},
			{Title: "Headers"},
			{Title: "Pre Request"},
			{Title: "Post Request"},
		}, nil),
		PreRequest: component.NewPrePostRequest([]component.Option{
			{Text: "None"},
			{Text: "Python", IsScript: true, Hint: "Write your pre request python script here"},
			{Text: "Shell Script", IsScript: true, Hint: "Write your pre request shell script here"},
			{Text: "Kubectl tunnel", IsScript: false, Hint: "Run kubectl port-forward command"},
			{Text: "SSH tunnel", IsScript: false, Hint: "Run ssh command"},
		}),
		PostRequest: component.NewPrePostRequest([]component.Option{
			{Text: "None"},
			{Text: "Python", IsScript: true, Hint: "Write your post request python script here"},
			{Text: "Shell Script", IsScript: true, Hint: "Write your post request shell script here"},
		}),

		Body:    NewBody(nil),
		Params:  NewParams(nil, nil),
		Headers: NewHeaders(nil),
		Auth:    NewAuth(nil),
	}

	if req != nil && req.Spec != (domain.RequestSpec{}) && req.Spec.HTTP != nil && req.Spec.HTTP.Request != nil {
		r.Params.SetQueryParams(req.Spec.HTTP.Request.QueryParams)
		r.Params.SetPathParams(req.Spec.HTTP.Request.PathParams)
		r.Headers.SetHeaders(req.Spec.HTTP.Request.Headers)

		if req.Spec.HTTP.Request.Auth != nil {
			r.Auth.SetAuth(req.Spec.HTTP.Request.Auth)
		}
		if req.Spec.HTTP.Request.Body != nil {
			r.Body = NewBody(req.Spec.HTTP.Request.Body)
		}

		if req.Spec.HTTP.Request.PreRequest != (domain.PreRequest{}) {
			r.PreRequest.SetSelectedDropDown(req.Spec.HTTP.Request.PreRequest.Type)
			r.PreRequest.SetCode(req.Spec.HTTP.Request.PreRequest.Script)
		}

		if req.Spec.HTTP.Request.PostRequest != (domain.PostRequest{}) {
			r.PostRequest.SetSelectedDropDown(req.Spec.HTTP.Request.PostRequest.Type)
			r.PostRequest.SetCode(req.Spec.HTTP.Request.PostRequest.Script)
		}
	}

	return r
}

func (r *Request) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Tabs.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				switch r.Tabs.SelectedTab().Title {
				case "Pre Request":
					return r.PreRequest.Layout(gtx, theme)
				case "Post Request":
					return r.PostRequest.Layout(gtx, theme)
				case "Params":
					return r.Params.Layout(gtx, theme)
				case "Headers":
					return r.Headers.Layout(gtx, theme)
				case "Auth":
					return r.Auth.Layout(gtx, theme)
				case "Body":
					return r.Body.Layout(gtx, theme)
				default:
					return layout.Dimensions{}
				}
			}),
		)
	})
}
