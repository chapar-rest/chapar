package modals

import (
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/outlay"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
	"github.com/chapar-rest/chapar/ui/widgets/card"
)

type CreateModal struct {
	CreateBtn widget.Clickable
	CloseBtn  widget.Clickable

	Items []*CreateItem
	flow  outlay.FlowWrap
}

type CreateItem struct {
	Clickable widget.Clickable
	Icon      widgets.Icon
	Title     string
	Key       string
}

func NewCreateItem(key string, icon widgets.Icon, title string) *CreateItem {
	return &CreateItem{
		Clickable: widget.Clickable{},
		Icon:      icon,
		Title:     title,
		Key:       key,
	}
}

func NewCreateModal(items []*CreateItem) *CreateModal {
	return &CreateModal{
		Items:     items,
		CreateBtn: widget.Clickable{},
		CloseBtn:  widget.Clickable{},
		flow: outlay.FlowWrap{
			Axis:      layout.Horizontal,
			Alignment: layout.Start,
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
						return n.flow.Layout(gtx, len(n.Items), func(gtx layout.Context, i int) layout.Dimensions {
							item := n.Items[i]
							return layout.Inset{Left: unit.Dp(5), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return item.Layout(gtx, th)
							})
						})
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

func (c *CreateItem) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        th.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	padding := layout.UniformInset(unit.Dp(8))

	gtx.Constraints.Min.X = gtx.Dp(85)
	gtx.Constraints.Min.Y = gtx.Dp(85)

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Clickable(gtx, &c.Clickable, func(gtx layout.Context) layout.Dimensions {
			return padding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Middle,
					Spacing:   layout.SpaceAround,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(48)
						gtx.Constraints.Max.X = gtx.Dp(48)
						return c.Icon.Layout(gtx, th.ContrastFg)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lb := material.Label(th.Material(), unit.Sp(14), c.Title)
						lb.Alignment = text.Middle
						return lb.Layout(gtx)
					}),
				)
			})
		})
	})
}
