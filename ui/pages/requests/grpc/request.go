package grpc

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Request struct {
	Tabs *widgets.Tabs

	//Body     *Body
	//Metadata *Metadata
	//Auth     *Auth
}

func NewRequest(req *domain.Request, theme *chapartheme.Theme) *Request {
	r := &Request{
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Body"},
			{Title: "Auth"},
			{Title: "Meta Data"},
		}, nil),
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
				//case "Body":
				//	return r.Body.Layout(gtx, theme)
				//case "Meta Data":
				//	return r.Metadata.Layout(gtx, theme)
				//case "Auth":
				//	return r.Auth.Layout(gtx, theme)

				default:
					return layout.Dimensions{}
				}
			}),
		)
	})
}
