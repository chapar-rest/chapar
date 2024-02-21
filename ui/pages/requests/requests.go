package requests

import (
	"fmt"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Requests struct {
	addRequestButton widget.Clickable
	importButton     widget.Clickable
	searchBox        *widgets.TextField
	// requestsTree     *widgets.TreeView

	treeView *widgets.TreeView

	split         widgets.SplitView
	tabs          *widgets.Tabs
	restContainer *RestContainer
}

func New(theme *material.Theme) *Requests {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	tabItems := []*widgets.Tab{
		{Title: "Register user", Closable: true, CloseClickable: &widget.Clickable{}},
		{Title: "Delete user", Closable: true, CloseClickable: &widget.Clickable{}},
		{Title: "Update user", Closable: true, CloseClickable: &widget.Clickable{}},
	}

	onTabsChange := func(index int) {
		fmt.Println("selected tab", index)
	}

	req := &Requests{
		searchBox: search,
		tabs:      widgets.NewTabs(tabItems, onTabsChange),
		//requestsTree: widgets.NewTreeView(),
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

	req.treeView = widgets.NewTreeViewV2([]*widgets.TreeNode{
		{
			Text:       "Users",
			Identifier: "users",
			Children: []*widgets.TreeNode{
				{
					Text:       "Register user",
					Identifier: "register_user",
				},
				{
					Text:       "Delete user",
					Identifier: "delete_user",
				},
				{
					Text:       "Update user",
					Identifier: "update_user",
				},
			},
		},
		{
			Text:       "Posts",
			Identifier: "posts",
		},
	})

	req.treeView.ParentMenuOptions = []string{"Delete", "-", "Duplicate"}
	req.treeView.ChildMenuOptions = []string{"Delete", "Duplicate", "Move"}

	//rq := widgets.NewNode("Users", false)
	//rq.AddChild(widgets.NewNode("Register user", false))
	//rq.AddChild(widgets.NewNode("Delete user", false))
	//rq.AddChild(widgets.NewNode("Update user", false))
	//req.requestsTree.AddNode(rq, nil)

	return req
}

func (r *Requests) list(gtx layout.Context, theme *material.Theme) layout.Dimensions {
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
					return r.searchBox.Layout(gtx, theme)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.treeView.Layout(gtx, theme)
				})
			}),
		)
	})
}

func (r *Requests) container(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.tabs.Layout(gtx, theme)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.restContainer.Layout(gtx, theme)
		}),
	)
}

func (r *Requests) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return r.split.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return r.list(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return r.container(gtx, theme)
		},
	)
}
