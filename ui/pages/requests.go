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
	theme *material.Theme

	actions *Actions

	textEditor widget.Editor
	searchBox  *widgets.TextField

	split widgets.SplitView

	tabs *widgets.Tabs

	tabsCounter int

	requestsTree *widgets.TreeView

	restContainer *RestContainer
}

func NewRequest(theme *material.Theme) *Request {
	search := widgets.NewTextField(theme, "", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	actions := NewActions(theme)

	tabItems := []*widgets.Tab{
		widgets.NewTab("Request 1", &widget.Clickable{}),
	}

	onTabsChange := func(index int) {
		fmt.Println("selected tab", index)
	}

	req := &Request{
		theme:        theme,
		actions:      actions,
		searchBox:    search,
		tabs:         widgets.NewTabs(tabItems, onTabsChange),
		requestsTree: widgets.NewTreeView(),
		split: widgets.SplitView{
			Ratio:         -0.64,
			MinLeftSize:   420,
			MaxLeftSize:   800,
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
		tabsCounter:   2,
		restContainer: NewRestContainer(theme),
	}

	rq := widgets.NewNode("Request 0", false)
	rq.AddChild(widgets.NewNode("Request 01", false))
	rq.AddChild(widgets.NewNode("Request 02", false))
	rq.AddChild(widgets.NewNode("Request 03", false))
	rq.AddChild(widgets.NewNode("Request 04", false))
	req.requestsTree.AddNode(rq, nil)

	addOne := func() {
		name := fmt.Sprintf("Request %d", req.tabsCounter)
		req.tabs.AddTab(widgets.NewTab(name, &widget.Clickable{}))
		req.requestsTree.AddRootNode(name, true)
		req.tabsCounter++
	}

	for i := 0; i < 10; i++ {
		addOne()
	}

	actions.OnClick(func(target string) {
		if target == "Add" {
			addOne()
		}
	})

	return req
}

func (r *Request) list(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.actions.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.searchBox.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.requestsTree.Layout(r.theme, gtx)
				})
			}),
		)
	})
}

func (r *Request) requestContainer(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.tabs.Layout(r.theme, gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.restContainer.Layout(r.theme, gtx)
		}),
	)
}

func (r *Request) Layout(gtx layout.Context) layout.Dimensions {
	return r.split.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return r.list(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			return r.requestContainer(gtx)
		},
	)
}
