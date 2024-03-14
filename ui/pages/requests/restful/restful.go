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

	Breadcrumb *component.Breadcrumb
	AddressBar *component.AddressBar
	Response   *Response
	Request    *Request

	dataChanged bool
	split       widgets.SplitView

	onSave func(id string)
}

func (r *Restful) SetOnDataChanged(f func(id string, data any)) {
	//TODO implement me
	panic("implement me")
}

func (r *Restful) SetActiveEnvironment(env *domain.Environment) {
	//TODO implement me
	panic("implement me")
}

func (r *Restful) SetDirty(dirty bool) {
	r.dataChanged = dirty
}

func (r *Restful) SetOnTitleChanged(f func(title string)) {
	r.Breadcrumb.SetOnTitleChanged(f)
}

func (r *Restful) OnClose() bool {
	//TODO implement me
	panic("implement me")
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
	return &Restful{
		Prompt:     widgets.NewPrompt("", "", ""),
		Breadcrumb: component.NewBreadcrumb(req.MetaData.Type, req.CollectionName, req.MetaData.Name),
		AddressBar: component.NewAddressBar(req.Spec.HTTP.URL, req.Spec.HTTP.Method),
		split: widgets.SplitView{
			Ratio:         0,
			BarWidth:      unit.Dp(2),
			BarColor:      widgets.Gray300,
			BarColorHover: theme.Palette.ContrastBg,
		},
		Response: NewResponse(theme),
		Request:  NewRequest(req),
	}
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
