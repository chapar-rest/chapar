package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	giox "gioui.org/x/component"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type RestfulV2 struct {
	*ui.Base
	request *domain.Request

	Breadcrumb *component.Breadcrumb
	AddressBar *component.AddressBar
	Actions    *component.Actions

	split widgets.SplitView

	SaveButton widget.Clickable
	CodeButton widget.Clickable
}

// Close implements container.Container.
func (r *RestfulV2) Close() error {
	panic("unimplemented")
}

// DataChanged implements container.Container.
func (r *RestfulV2) DataChanged() bool {
	panic("unimplemented")
}

func NewV2(base *ui.Base, request *domain.Request) *RestfulV2 {
	req := *request // make a copy

	splitAxis := layout.Vertical
	if prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit {
		splitAxis = layout.Horizontal
	}

	return &RestfulV2{
		Base:       base,
		request:    &req,
		Breadcrumb: component.NewBreadcrumb(req.MetaData.ID, req.CollectionName, req.Spec.HTTP.Method, req.MetaData.Name),
		AddressBar: component.NewAddressBar(req.Spec.HTTP.URL, req.Spec.HTTP.Method),
		Actions:    component.NewActions(true),

		split: widgets.SplitView{
			Resize: giox.Resize{
				Ratio: 0.5,
				Axis:  splitAxis,
			},
			BarWidth: unit.Dp(2),
		},

		SaveButton: widget.Clickable{},
		CodeButton: widget.Clickable{},
	}
}

func (r *RestfulV2) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Dimensions{}
}
