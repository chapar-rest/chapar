package requests

import (
	"fmt"
	"image/color"

	"gioui.org/op"

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
	theme *material.Theme

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

	req := &Requests{
		theme:     theme,
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
		openedTabs: make([]*openedTab, 0),
	}
	req.treeView.ParentMenuOptions = []string{"Duplicate", "Rename", "Delete"}
	req.treeView.ChildMenuOptions = []string{"Move", "Duplicate", "Rename", "Delete"}
	req.treeView.OnDoubleClick(req.onItemDoubleClick)
	req.treeView.SetOnMenuItemClick(func(tr *widgets.TreeNode, item string) {
		if item == "Duplicate" {
			req.duplicateReq(tr.Identifier)
		}

		if item == "Delete" {
			req.deleteReq(tr.Identifier)
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

func (r *Requests) onTabClose(t *widgets.Tab) {
	for _, ot := range r.openedTabs {
		if ot.req.MetaData.ID == t.Identifier {
			// can we close the tab?
			if !ot.container.OnClose() {
				return
			}
			ot.closed = true
			break
		}
	}
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
				container: rest.NewRestContainer(r.theme, req.Clone()),
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

func (r *Requests) duplicateReq(identifier string) {
	for _, req := range r.data {
		if req.MetaData.ID == identifier {
			newReq := req.Clone()
			newReq.MetaData.ID = uuid.NewString()
			newReq.MetaData.Name = newReq.MetaData.Name + " (copy)"
			// add copy to file name
			newReq.FilePath = loader.AddSuffixBeforeExt(newReq.FilePath, "-copy")
			r.data = append(r.data, newReq)

			node := &widgets.TreeNode{
				Text:       newReq.MetaData.Name,
				Identifier: newReq.MetaData.ID,
			}
			r.treeView.AddNode(node)
			if err := loader.UpdateRequest(newReq); err != nil {
				fmt.Println("failed to update request", err)
			}
			break
		}
	}
}

func (r *Requests) deleteReq(identifier string) {
	for i, req := range r.data {
		if req.MetaData.ID == identifier {
			r.data = append(r.data[:i], r.data[i+1:]...)
			r.treeView.RemoveNode(identifier)

			if err := loader.DeleteRequest(req); err != nil {
				fmt.Println("failed to delete request", err)
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
			if r.selectedIndex > len(r.openedTabs)-1 {
				return layout.Dimensions{}
			}
			ct := r.openedTabs[r.selectedIndex].container
			r.openedTabs[r.selectedIndex].tab.SetDirty(ct.IsDataChanged())

			return ct.Layout(gtx, theme)
		}),
	)
}

func (r *Requests) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	// update tabs with new items
	tabItems := make([]*widgets.Tab, 0)
	openItems := make([]*openedTab, 0)
	for _, ot := range r.openedTabs {
		if !ot.closed {
			tabItems = append(tabItems, ot.tab)
			openItems = append(openItems, ot)
		}
	}

	r.tabs.SetTabs(tabItems)
	r.openedTabs = openItems
	selectTab := r.tabs.Selected()
	gtx.Execute(op.InvalidateCmd{})

	// is selected tab is closed:
	// if its the last tab and there is another tab before it, select the previous one
	// if its the first tab and there is another tab after it, select the next one
	// if its the only tab, select it

	if selectTab > len(openItems)-1 {
		if len(openItems) > 0 {
			r.tabs.SetSelected(len(openItems) - 1)
		} else {
			selectTab = 0
			r.tabs.SetSelected(0)
		}
	}

	if r.selectedIndex != selectTab {
		r.selectedIndex = selectTab
	}

	return r.split.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return r.list(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return r.container(gtx, theme)
		},
	)
}
