package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/pages/requests/component"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Request struct {
	Tabs *widgets.Tabs

	PreRequest  *component.PrePostRequest
	PostRequest *component.PrePostRequest
	Params      *Params
	Headers     *Headers
	Auth        *Auth
}

func NewRequest() *Request {
	return &Request{
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
		Params:  NewParams(),
		Headers: NewHeaders(),
		Auth:    NewAuth(),
	}
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
				default:
					return layout.Dimensions{}
				}
			}),
		)
	})
}
