package modals

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
	"github.com/chapar-rest/chapar/ui/widgets/card"
)

type InputText struct {
	TextField *widgets.TextField
	AddBtn    widget.Clickable
	CloseBtn  widget.Clickable

	Title string
}

func NewInputText(title, placeholder string) *InputText {
	ed := widgets.NewTextField("", placeholder)
	ed.SetIcon(widgets.FileFolderIcon, widgets.IconPositionStart)
	return &InputText{
		TextField: ed,
		Title:     title,
		AddBtn:    widget.Clickable{},
		CloseBtn:  widget.Clickable{},
	}
}

func (i *InputText) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	marginTop := layout.Inset{Top: unit.Dp(90)}

	return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X /= 3
			return marginTop.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return card.Card{
					Title: i.Title,
					Body: func(gtx layout.Context) layout.Dimensions {
						return i.TextField.Layout(gtx, th)
					},
					Actions: []card.Action{
						{
							Clickable: &i.CloseBtn,
							Label:     "Close",
							Fg:        th.ButtonTextColor,
							Bg:        th.ActionButtonBgColor,
							Float:     card.FloatRight,
						},
						{
							Clickable: &i.AddBtn,
							Label:     "Add",
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
