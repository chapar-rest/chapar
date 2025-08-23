package modals

import (
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
	"github.com/chapar-rest/chapar/ui/widgets/card"
)

type CreateModal struct {
	CreateBtn widget.Clickable
	CloseBtn  widget.Clickable

	Items []*CreateItem
	list  *widget.List
}

type CreateItem struct {
	*widgets.ImageButton
	Key string
}

func NewCreateItem(key string, image paint.ImageOp, title string) *CreateItem {
	return &CreateItem{
		ImageButton: widgets.NewImageButton(image, title),
		Key:         key,
	}
}

func NewCreateModal(items []*CreateItem) *CreateModal {
	return &CreateModal{
		Items:     items,
		CreateBtn: widget.Clickable{},
		CloseBtn:  widget.Clickable{},
		list: &widget.List{
			List: layout.List{
				Axis:      layout.Horizontal,
				Alignment: layout.Start,
			},
		},
	}
}

func (n *CreateModal) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	marginTop := layout.Inset{Top: unit.Dp(90)}

	return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X /= 3
			return marginTop.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return card.Card{
					Title: "Create New",
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

func (n *CreateModal) ItemsLayout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
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
