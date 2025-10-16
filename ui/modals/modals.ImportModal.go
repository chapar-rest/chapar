package modals

import (
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/assets"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
	"github.com/chapar-rest/chapar/ui/widgets/card"
)

type ImportModal struct {
	CloseBtn widget.Clickable

	Items []*ImportItem
	list  *widget.List
}

type ImportItem struct {
	*widgets.ImageButton
	Key string
}

func NewImportItem(key string, image paint.ImageOp, title string) *ImportItem {
	return &ImportItem{
		ImageButton: widgets.NewImageButton(image, title),
		Key:         key,
	}
}

func NewImportModal() *ImportModal {
	return &ImportModal{
		Items: []*ImportItem{
			NewImportItem("postman", assets.CollectionImage, "Postman"),
			NewImportItem("openapi", assets.HTTPImage, "OpenAPI"),
			NewImportItem("protofile", assets.GRPCImage, "Proto File"),
		},
		CloseBtn: widget.Clickable{},
		list: &widget.List{
			List: layout.List{
				Axis:      layout.Horizontal,
				Alignment: layout.Start,
			},
		},
	}
}

func (n *ImportModal) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	marginTop := layout.Inset{Top: unit.Dp(90)}

	return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X /= 3
			return marginTop.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return card.Card{
					Title: "Import",
					Body: func(gtx layout.Context) layout.Dimensions {
						return n.ItemsLayout(gtx, th)
					},
					Actions: []card.Action{
						{
							Clickable: &n.CloseBtn,
							Label:     "Close",
							Fg:        th.ButtonTextColor,
							Bg:        th.ActionButtonBgColor,
							Float:     card.FloatRight,
						},
					},
				}.Layout(gtx, th.Material())
			})
		})
	})
}

func (n *ImportModal) ItemsLayout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return material.List(th.Material(), n.list).Layout(gtx, len(n.Items), func(gtx layout.Context, index int) layout.Dimensions {
		item := n.Items[index]
		if index > 0 {
			return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return item.Layout(gtx, th)
			})
		}

		return item.Layout(gtx, th)
	})
}
