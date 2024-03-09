package requests

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/keys"
	"github.com/mirzakhany/chapar/ui/pages/tips"
	"github.com/mirzakhany/chapar/ui/widgets"
)

var (
	collectionMenuItems = []string{"Add Request", "View", "Delete"}
	requestMenuItems    = []string{"View", "Duplicate", "Delete"}
)

type View struct {
	// add menu
	newRequestButton     widget.Clickable
	newMenuContextArea   component.ContextArea
	newMenu              component.MenuState
	menuInit             bool
	newHttpRequestButton widget.Clickable
	newGrpcRequestButton widget.Clickable
	newCollectionButton  widget.Clickable

	treeViewSearchBox *widgets.TextField
	treeView          *widgets.TreeView

	split     widgets.SplitView
	tabHeader *widgets.Tabs

	// callbacks
	onTitleChanged              func(id, title string)
	onNewRequest                func()
	onNewCollection             func()
	onTabClose                  func(id string)
	onTreeViewNodeDoubleClicked func(id string)
	onTreeViewMenuClicked       func(id string, action string)
	onTabSelected               func(id string)
	onSave                      func(id string)

	// state
	containers    map[string]Container
	openTabs      map[string]*widgets.Tab
	treeViewNodes map[string]*widgets.TreeNode
}

func NewView(theme *material.Theme) *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	v := &View{
		treeViewSearchBox: search,
		tabHeader:         widgets.NewTabs([]*widgets.Tab{}, nil),
		treeView:          widgets.NewTreeView([]*widgets.TreeNode{}),
		split: widgets.SplitView{
			Ratio:         -0.64,
			MinLeftSize:   unit.Dp(250),
			MaxLeftSize:   unit.Dp(800),
			BarWidth:      unit.Dp(2),
			BarColor:      widgets.Gray300,
			BarColorHover: theme.Palette.ContrastBg,
		},
		treeViewNodes: make(map[string]*widgets.TreeNode),
		openTabs:      make(map[string]*widgets.Tab),
		newMenuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
	}

	v.treeViewSearchBox.SetOnTextChange(func(text string) {
		if len(v.treeViewNodes) == 0 {
			return
		}
		v.treeView.Filter(text)
	})

	return v
}

func (v *View) AddRequestTreeViewNode(req *domain.Request) {
	node := &widgets.TreeNode{
		Text:        req.MetaData.Name,
		Identifier:  req.MetaData.ID,
		MenuOptions: requestMenuItems,
	}

	v.treeView.AddNode(node)
	v.treeViewNodes[req.MetaData.ID] = node
}

func (v *View) AddCollectionTreeViewNode(collection *domain.Collection) {
	node := &widgets.TreeNode{
		Text:        collection.MetaData.Name,
		Identifier:  collection.MetaData.ID,
		Children:    make([]*widgets.TreeNode, 0),
		MenuOptions: collectionMenuItems,
	}

	v.treeView.AddNode(node)
	v.treeViewNodes[collection.MetaData.ID] = node
}

func (v *View) RemoveTreeViewNode(id string) {
	if _, ok := v.treeViewNodes[id]; !ok {
		return
	}

	v.treeView.RemoveNode(id)
	delete(v.treeViewNodes, id)
}

func (v *View) SetOnNewRequest() {

}

func (v *View) SetOnTitleChanged(onTitleChanged func(id, title string)) {
	v.onTitleChanged = onTitleChanged
}

func (v *View) SetOnTreeViewNodeDoubleClicked(onTreeViewNodeDoubleClicked func(id string)) {
	v.onTreeViewNodeDoubleClicked = onTreeViewNodeDoubleClicked
	v.treeView.OnNodeDoubleClick(func(node *widgets.TreeNode) {
		v.onTreeViewNodeDoubleClicked(node.Identifier)
	})
}

func (v *View) SetOnTreeViewMenuClicked(onTreeViewMenuClicked func(id string, action string)) {
	v.onTreeViewMenuClicked = onTreeViewMenuClicked
	v.treeView.SetOnMenuItemClick(func(node *widgets.TreeNode, item string) {
		v.onTreeViewMenuClicked(node.Identifier, item)
	})
}

func (v *View) SetOnTabSelected(onTabSelected func(id string)) {
	v.onTabSelected = onTabSelected
}

func (v *View) SetOnSave(onSave func(id string)) {
	v.onSave = onSave
}

func (v *View) SetOnTabClose(onTabClose func(id string)) {
	v.onTabClose = onTabClose
}

func (v *View) UpdateTabTitle(id, title string) {
	if tab, ok := v.openTabs[id]; ok {
		tab.Title = title
	}
}

func (v *View) UpdateTreeNodeTitle(id, title string) {
	if node, ok := v.treeViewNodes[id]; ok {
		node.Text = title
	}
}

func (v *View) SetTabDirty(id string, dirty bool) {
	if tab, ok := v.openTabs[id]; ok {
		tab.SetDirty(dirty)
		if ct, ok := v.containers[id]; ok {
			ct.SetDirty(dirty)
		}
	}
}

func (v *View) CloseTab(id string) {
	if _, ok := v.openTabs[id]; ok {
		v.tabHeader.RemoveTabByID(id)
		delete(v.openTabs, id)
		delete(v.containers, id)
	}
}

func (v *View) IsTabOpen(id string) bool {
	_, ok := v.openTabs[id]
	return ok
}

func (v *View) PopulateTreeView(requests []*domain.Request, collections []*domain.Collection) {
	treeViewNodes := make([]*widgets.TreeNode, 0)
	for _, cl := range collections {
		parentNode := &widgets.TreeNode{
			Text:        cl.MetaData.Name,
			Identifier:  cl.MetaData.ID,
			Children:    make([]*widgets.TreeNode, 0),
			MenuOptions: collectionMenuItems,
		}

		for _, req := range cl.Spec.Requests {
			node := &widgets.TreeNode{
				Text:        req.MetaData.Name,
				Identifier:  req.MetaData.ID,
				MenuOptions: requestMenuItems,
			}
			parentNode.AddChildNode(node)
			v.treeViewNodes[req.MetaData.ID] = node
		}

		treeViewNodes = append(treeViewNodes, parentNode)
		v.treeViewNodes[cl.MetaData.ID] = parentNode
	}

	for _, req := range requests {
		node := &widgets.TreeNode{
			Text:        req.MetaData.Name,
			Identifier:  req.MetaData.ID,
			MenuOptions: requestMenuItems,
		}
		treeViewNodes = append(treeViewNodes, node)
	}

	v.treeView.SetNodes(treeViewNodes)
}

func (v *View) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return v.split.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return v.requestList(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return v.containerHolder(gtx, theme)
		},
	)
}

func (v *View) requestList(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if !v.menuInit {
		v.menuInit = true
		v.newMenu = component.MenuState{
			Options: []func(gtx layout.Context) layout.Dimensions{
				component.MenuItem(theme, &v.newHttpRequestButton, "HTTP Request").Layout,
				component.MenuItem(theme, &v.newGrpcRequestButton, "GRPC Request").Layout,
				component.Divider(theme).Layout,
				component.MenuItem(theme, &v.newCollectionButton, "Collection").Layout,
			},
		}
	}

	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = 0
							return layout.Stack{}.Layout(gtx,
								layout.Stacked(func(gtx layout.Context) layout.Dimensions {
									if v.newHttpRequestButton.Clicked(gtx) {
										// v.addNewEmptyReq("")
									}

									if v.newCollectionButton.Clicked(gtx) {
										// r.addEmptyCollection()
									}

									return material.Button(theme, &v.newRequestButton, "New").Layout(gtx)
								}),
								layout.Expanded(func(gtx layout.Context) layout.Dimensions {
									return v.newMenuContextArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										offset := layout.Inset{
											Top:  unit.Dp(float32(80)/gtx.Metric.PxPerDp + 1),
											Left: unit.Dp(4),
										}
										return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = 0
											m := component.Menu(theme, &v.newMenu)
											m.SurfaceStyle.Fill = widgets.Gray300
											return m.Layout(gtx)
										})
									})
								}),
							)

						}),
						// layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
						// layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						//	return material.Button(theme, &v.importButton, "Import").Layout(gtx)
						// }),
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeViewSearchBox.Layout(gtx, theme)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Right: 0}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeView.Layout(gtx, theme)
				})
			}),
		)
	})
}

func (v *View) containerHolder(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if v.onSave != nil {
		keys.OnSaveCommand(gtx, v, func() {
			v.onSave(v.tabHeader.SelectedTab().GetIdentifier())
		})
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return v.tabHeader.Layout(gtx, theme)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(v.openTabs) == 0 {
				t := tips.New()
				return t.Layout(gtx, theme)
			}

			selectedTab := v.tabHeader.SelectedTab()
			if selectedTab != nil {
				if v.onTabSelected != nil {
					v.onTabSelected(selectedTab.Identifier)
					gtx.Execute(op.InvalidateCmd{})
				}

				if ct, ok := v.containers[selectedTab.Identifier]; ok {
					// if v.onSave != nil {
					//	if ct.SaveButton.Clicked(gtx) {
					//		v.onSave(selectedTab.Identifier)
					//	}
					// }

					return ct.Layout(gtx, theme)
				}
			}

			return layout.Dimensions{}
		}),
	)
}
