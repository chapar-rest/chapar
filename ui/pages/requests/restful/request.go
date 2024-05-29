package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Request struct {
	Tabs *widgets.Tabs

	PreRequest  *component.PrePostRequest
	PostRequest *component.PrePostRequest

	Body    *Body
	Params  *Params
	Headers *Headers
	Auth    *component.Auth
}

func NewRequest(req *domain.Request, theme *chapartheme.Theme) *Request {
	r := &Request{
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Params"},
			{Title: "Body"},
			{Title: "Auth"},
			{Title: "Headers"},
			//	{Title: "Pre Request"},
			{Title: "Post Request"},
		}, nil),
		// PreRequest: component.NewPrePostRequest([]component.Option{
		//	{Title: "None", Value: domain.PostRequestTypeNone},
		//	{Title: "Python", Value: domain.PostRequestTypePythonScript, Type: component.TypeScript, Hint: "Write your pre request python script here"},
		//	{Title: "Shell Script", Value: domain.PostRequestTypeSSHTunnel, Type: component.TypeScript, Hint: "Write your pre request shell script here"},
		//	{Title: "Kubectl tunnel", Value: domain.PostRequestTypeK8sTunnel, Type: component.TypeScript, Hint: "Run kubectl port-forward command"},
		//	{Title: "SSH tunnel", Value: domain.PostRequestTypeSSHTunnel, Type: component.TypeScript, Hint: "Run ssh command"},
		// }, theme),
		PostRequest: component.NewPrePostRequest([]component.Option{
			{Title: "None", Value: domain.PostRequestTypeNone},
			{Title: "Set Environment Variable", Value: domain.PostRequestTypeSetEnv, Type: component.TypeSetEnv, Hint: "Set environment variable"},
			//	{Title: "Python", Value: domain.PostRequestTypePythonScript, Type: component.TypeScript, Hint: "Write your post request python script here"},
			//	{Title: "Shell Script", Value: domain.PostRequestTypeShellScript, Type: component.TypeScript, Hint: "Write your post request shell script here"},
		}, theme),

		Body:    NewBody(req.Spec.HTTP.Request.Body, theme),
		Params:  NewParams(nil, nil),
		Headers: NewHeaders(nil),
		Auth:    component.NewAuth(req.Spec.HTTP.Request.Auth, theme),
	}

	if req.Spec != (domain.RequestSpec{}) && req.Spec.HTTP != nil && req.Spec.HTTP.Request != nil {
		r.Params.SetQueryParams(req.Spec.HTTP.Request.QueryParams)
		r.Params.SetPathParams(req.Spec.HTTP.Request.PathParams)
		r.Headers.SetHeaders(req.Spec.HTTP.Request.Headers)

		// if req.Spec.HTTP.Request.PreRequest != (domain.PreRequest{}) {
		//	r.PreRequest.SetSelectedDropDown(req.Spec.HTTP.Request.PreRequest.Type)
		//	r.PreRequest.SetCode(req.Spec.HTTP.Request.PreRequest.Script)
		// }

		if req.Spec.HTTP.Request.PostRequest != (domain.PostRequest{}) {
			r.PostRequest.SetSelectedDropDown(req.Spec.HTTP.Request.PostRequest.Type)
			//	r.PostRequest.SetCode(req.Spec.HTTP.Request.PostRequest.Script)
		}

		if req.Spec.HTTP.Request.PostRequest.PostRequestSet != (domain.PostRequestSet{}) {
			r.PostRequest.SetPostRequestSetValues(req.Spec.HTTP.Request.PostRequest.PostRequestSet)
		}
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
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				switch r.Tabs.SelectedTab().Title {
				// case "Pre Request":
				//	return r.PreRequest.Layout(gtx, theme)
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
