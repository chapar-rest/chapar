package requests

import (
	"fmt"
	"image/color"

	"github.com/google/uuid"

	"github.com/mirzakhany/chapar/internal/loader"

	"github.com/mirzakhany/chapar/internal/domain"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/pages/requests/rest"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Requests struct {
	addRequestButton widget.Clickable
	importButton     widget.Clickable
	searchBox        *widgets.TextField

	treeView *widgets.TreeView

	split widgets.SplitView
	tabs  *widgets.Tabs

	data       []*domain.Request
	openedTabs []*openedTab

	selectedIndex int
}

type openedTab struct {
	req       *domain.Request
	tab       *widgets.Tab
	listItem  *widgets.TreeNode
	container *rest.Container

	closed bool
}

func New(theme *material.Theme) (*Requests, error) {
	data, err := loader.ReadRequestsData()
	if err != nil {
		return nil, err
	}

	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	treeViewNodes := make([]*widgets.TreeNode, 0)
	for _, req := range data {
		if req.MetaData.ID == "" {
			req.MetaData.ID = uuid.NewString()
		}

		node := &widgets.TreeNode{
			Text:       req.MetaData.Name,
			Identifier: req.MetaData.ID,
		}
		treeViewNodes = append(treeViewNodes, node)
	}

	//tabItems := []*widgets.Tab{
	//	{Title: "Register user", Closable: true, CloseClickable: &widget.Clickable{}},
	//	{Title: "Delete user", Closable: true, CloseClickable: &widget.Clickable{}},
	//	{Title: "Update user", Closable: true, CloseClickable: &widget.Clickable{}},
	//}

	//onTabsChange := func(index int) {
	//	fmt.Println("selected tab", index)
	//}

	req := &Requests{
		data:      data,
		searchBox: search,
		tabs:      widgets.NewTabs([]*widgets.Tab{}, nil),
		treeView:  widgets.NewTreeView(treeViewNodes),
		split: widgets.SplitView{
			Ratio:         -0.64,
			MinLeftSize:   unit.Dp(250),
			MaxLeftSize:   unit.Dp(800),
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
	}
	//
	//req.treeView = widgets.NewTreeView([]*widgets.TreeNode{
	//	{
	//		Text:       "Users",
	//		Identifier: "users",
	//		Children: []*widgets.TreeNode{
	//			{
	//				Text:       "Register user",
	//				Identifier: "register_user",
	//			},
	//			{
	//				Text:       "Delete user",
	//				Identifier: "delete_user",
	//			},
	//			{
	//				Text:       "Update user",
	//				Identifier: "update_user",
	//			},
	//		},
	//	},
	//	{
	//		Text:       "Posts",
	//		Identifier: "posts",
	//	},
	//})

	req.treeView.ParentMenuOptions = []string{"Duplicate", "Rename", "Delete"}
	req.treeView.ChildMenuOptions = []string{"Move", "Duplicate", "Rename", "Delete"}
	req.treeView.OnDoubleClick(req.onItemDoubleClick)
	req.treeView.SetOnMenuItemClick(func(tr *widgets.TreeNode, item string) {
		if item == "Duplicate" {
			req.duplicateEnv(tr.Identifier)
		}

		if item == "Delete" {
			req.deleteEnv(tr.Identifier)
		}
	})

	req.searchBox.SetOnTextChange(func(text string) {
		if req.data == nil {
			return
		}

		req.treeView.Filter(text)
	})

	return req, nil
}

func (r *Requests) onTitleChanged(id, title string) {
	// find the opened tab and mark it as dirty
	for _, ot := range r.openedTabs {
		if ot.req.MetaData.ID == id {
			// is name changed?
			if ot.req.MetaData.Name != title {
				// Update the tree view item and the tab title
				ot.req.MetaData.Name = title
				ot.tab.Title = title
				ot.listItem.Text = title
			}
		}
	}
}

func (r *Requests) onItemDoubleClick(tr *widgets.TreeNode) {
	// if request is already opened, just switch to it
	for i, ot := range r.openedTabs {
		if ot.req.MetaData.ID == tr.Identifier {
			r.selectedIndex = i
			r.tabs.SetSelected(i)
			return
		}
	}

	for _, req := range r.data {
		if req.MetaData.ID == tr.Identifier {
			tab := &widgets.Tab{Title: req.MetaData.Name, Closable: true, CloseClickable: &widget.Clickable{}}
			tab.SetOnClose(r.onTabClose)
			tab.SetIdentifier(req.MetaData.ID)

			ot := &openedTab{
				req:       req,
				tab:       tab,
				listItem:  tr,
				container: newEnvContainer(req.Clone()),
			}
			ot.container.SetOnTitleChanged(r.onTitleChanged)
			r.openedTabs = append(r.openedTabs, ot)

			i := r.tabs.AddTab(tab)
			r.selectedIndex = i
			r.tabs.SetSelected(i)

			break
		}
	}
}

func (r *Requests) duplicateEnv(identifier string) {
	for _, env := range e.data {
		if env.MetaData.ID == identifier {
			newEnv := env.Clone()
			newEnv.MetaData.ID = uuid.NewString()
			newEnv.MetaData.Name = newEnv.MetaData.Name + " (copy)"
			// add copy to file name
			newEnv.FilePath = loader.AddSuffixBeforeExt(newEnv.FilePath, "-copy")
			e.data = append(e.data, newEnv)

			node := &widgets.TreeNode{
				Text:       newEnv.MetaData.Name,
				Identifier: newEnv.MetaData.ID,
			}
			e.treeView.AddNode(node)
			if err := loader.UpdateEnvironment(newEnv); err != nil {
				fmt.Println("failed to update environment", err)
			}
			break
		}
	}
}

func (r *Requests) deleteEnv(identifier string) {
	for i, env := range e.data {
		if env.MetaData.ID == identifier {
			e.data = append(e.data[:i], e.data[i+1:]...)
			e.treeView.RemoveNode(identifier)

			if err := loader.DeleteEnvironment(env); err != nil {
				fmt.Println("failed to delete environment", err)
			}
			break
		}
	}
}

func (r *Requests) list(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceStart}.Layout(gtx,
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
				return layout.Inset{Top: unit.Dp(10), Right: 0}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
