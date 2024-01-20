package pages

import (
	"fmt"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Request struct {
	addRequestButton widget.Clickable
	importButton     widget.Clickable

	split  widgets.SplitView
	tabsV2 *widgets.TabsV2

	searchBox     *widgets.TextField
	requestsTree  *widgets.TreeView
	restContainer *RestContainer
}

func NewRequest(theme *material.Theme) *Request {
	search := widgets.NewTextField(theme, "", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	tabV2Items := []widgets.TabV2{
		{Title: "Register user", Closable: true, CloseClickable: &widget.Clickable{}},
	}

	onTabsChange := func(index int) {
		fmt.Println("selected tab", index)
	}

	req := &Request{
		searchBox:    search,
		tabsV2:       widgets.NewTabsV2(tabV2Items, onTabsChange),
		requestsTree: widgets.NewTreeView(),
		split: widgets.SplitView{
			Ratio:         -0.64,
			MinLeftSize:   unit.Dp(250),
			MaxLeftSize:   unit.Dp(800),
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
		restContainer: NewRestContainer(theme),
	}

	rq := widgets.NewNode("Users", false)
	rq.AddChild(widgets.NewNode("Register user", false))
	req.requestsTree.AddNode(rq, nil)

	return req
}

func (r *Request) list(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Button(theme, &r.addRequestButton, "Add").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Button(theme, &r.importButton, "Import").Layout(gtx)
						}),
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.searchBox.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.requestsTree.Layout(theme, gtx)
				})
			}),
		)
	})
}

func (r *Request) requestContainer(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			//return r.tabs.Layout(r.theme, gtx)
			return r.tabsV2.Layout(theme, gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.restContainer.Layout(gtx, theme)
		}),
	)
}

func (r *Request) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return r.split.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return r.list(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return r.requestContainer(gtx, theme)
		},
	)
}
