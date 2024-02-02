package envs

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type envContainer struct {
	items *widgets.KeyValue
	title *widgets.EditableLabel

	searchBox *widgets.TextField
}

func newEnvContainer(tab *widgets.Tab, treeItem *widgets.TreeViewNode) *envContainer {
	search := widgets.NewTextField("", "Search items")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	container := &envContainer{
		items: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),
		title:     widgets.NewEditableLabel(tab.Title),
		searchBox: search,
	}

	container.title.SetOnChanged(func(text string) {
		tab.Title = text
		treeItem.Text = text
	})

	return container
}

func (r *envContainer) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(15)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return r.title.Layout(gtx, theme)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(200)
							return r.searchBox.Layout(gtx, theme)
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return r.items.WithAddLayout(gtx, "", "*disabled items have no effect on your requests", theme)
			}),
		)
	})
}
