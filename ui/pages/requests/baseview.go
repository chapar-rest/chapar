package requests

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	giox "gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/modals"
	"github.com/chapar-rest/chapar/ui/navigator"
	"github.com/chapar-rest/chapar/ui/pages/requests/collections"
	"github.com/chapar-rest/chapar/ui/pages/requests/container"
	"github.com/chapar-rest/chapar/ui/pages/requests/restful"
	"github.com/chapar-rest/chapar/ui/pages/tips"
	"github.com/chapar-rest/chapar/ui/widgets"
)

var _ navigator.View = &BaseView{}

type BaseView struct {
	*ui.Base

	// add menu
	newRequestButton widget.Clickable
	importButton     widget.Clickable

	treeViewSearchBox *widgets.TextField
	treeView          *widgets.TreeView

	containers *safemap.Map[container.Container]

	split     widgets.SplitView
	tabHeader *widgets.Tabs
	tipsView  *tips.Tips
}

func NewBaseView(base *ui.Base) *BaseView {
	searchBox := widgets.NewTextField("", "Search...")
	searchBox.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)

	v := &BaseView{
		Base:              base,
		treeViewSearchBox: searchBox,
		tabHeader:         widgets.NewTabs([]*widgets.Tab{}, nil),
		treeView:          widgets.NewTreeView([]*widgets.TreeNode{}),
		split: widgets.SplitView{
			// Ratio:       -0.64,
			Resize: giox.Resize{
				Ratio: 0.19,
			},
			BarWidth: unit.Dp(2),
		},
		containers: safemap.New[container.Container](),
		tipsView:   tips.New(),
	}

	v.tabHeader.SetMaxTitleWidth(20)

	v.treeViewSearchBox.SetOnTextChange(func(text string) {
		v.treeView.Filter(text)
	})

	return v
}

func (v *BaseView) OnEnter() {
	v.reloadData()
}

func (v *BaseView) Info() navigator.Info {
	return navigator.Info{
		ID:    "requests",
		Title: "Requests",
		Icon:  widgets.SwapHoriz,
	}
}

func (v *BaseView) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	if id, ok := v.treeView.NodeClicked(); ok {
		v.OpenItem(id)
	}

	selectedTab := v.tabHeader.SelectedTab()
	if id, ok := v.tabHeader.TabClosed(); ok {
		v.containers.Delete(id)
		v.tabHeader.RemoveTabByID(id)

		if selectedTab != nil {
			v.containers.Delete(selectedTab.Identifier)

			if len(v.containers.Keys()) > 0 {
				v.tabHeader.SetSelected(len(v.containers.Keys()) - 1)
				selectedTab = v.tabHeader.SelectedTab()
			}
		}
	}

	return v.split.Layout(gtx, th,
		func(gtx layout.Context) layout.Dimensions {
			return v.list(gtx, th)
		},
		func(gtx layout.Context) layout.Dimensions {
			if selectedTab == nil {
				return v.tabs(gtx, th, nil)
			}

			if ct, ok := v.containers.Get(selectedTab.Identifier); ok {
				v.tabHeader.SetDataChanged(selectedTab.Identifier, ct.DataChanged())
				return v.tabs(gtx, th, ct)
			} else {
				return v.tabs(gtx, th, nil)
			}
		},
	)
}

func (v *BaseView) list(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if v.newRequestButton.Clicked(gtx) {
		v.showCreateNewModal()
	}

	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							//if v.importButton.Clicked(gtx) {
							//	if v.onImport != nil {
							//		v.onImport()
							//	}
							//}
							btn := widgets.Button(theme.Material(), &v.importButton, widgets.UploadIcon, widgets.IconPositionStart, "Import")
							btn.Color = theme.ButtonTextColor
							return btn.Layout(gtx, theme)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							newBtn := widgets.Button(theme.Material(), &v.newRequestButton, widgets.PlusIcon, widgets.IconPositionStart, "New")
							newBtn.Color = theme.ButtonTextColor
							return newBtn.Layout(gtx, theme)
						}),
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

func (v *BaseView) tabs(gtx layout.Context, theme *chapartheme.Theme, ct container.Container) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return v.tabHeader.Layout(gtx, theme)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if ct != nil {
				return ct.Layout(gtx, theme)
			} else {
				return v.tipsView.Layout(gtx, theme)
			}
		}),
	)
}

func (v *BaseView) showCreateNewModal() {
	items := []*modals.CreateItem{newGRPCRequest, newHTTPRequest, newHCollection}

	m := modals.NewCreateModal(items)
	v.SetModal(func(gtx layout.Context) layout.Dimensions {
		if m.CloseBtn.Clicked(gtx) {
			v.Base.CloseModal()
		}

		for _, item := range items {
			if item.Clickable.Clicked(gtx) {
				v.Base.CloseModal()
				v.CreateNew(item.Key)
			}
		}

		return m.Layout(gtx, v.Theme)
	})
}

func (v *BaseView) AddCollectionTreeViewNode(collection *domain.Collection) {
	node := &widgets.TreeNode{
		Text:        collection.MetaData.Name,
		Identifier:  collection.MetaData.ID,
		Children:    make([]*widgets.TreeNode, 0),
		MenuOptions: []string{MenuAddHTTPRequest, MenuAddGRPCRequest, MenuDuplicate, MenuView, MenuDelete},
		Meta:        safemap.New[string](),
	}

	node.Meta.Set(TypeMeta, TypeCollection)
	v.treeView.AddNode(node)
}

func (v *BaseView) AddRequestTreeViewNode(request *domain.Request) {
	node := &widgets.TreeNode{
		Text:        request.MetaData.Name,
		Identifier:  request.MetaData.ID,
		MenuOptions: []string{MenuView, MenuDuplicate, MenuDelete},
		Meta:        safemap.New[string](),
	}

	node.Meta.Set(TypeMeta, TypeRequest)
	v.treeView.AddNode(node)
}

func (v *BaseView) OpenCollectionContainer(collection *domain.Collection) {
	if _, ok := v.containers.Get(collection.MetaData.ID); ok {
		return
	}

	ct := collections.NewV2(v.Base, collection)
	v.containers.Set(collection.MetaData.ID, ct)
}

func (v *BaseView) OpenRequestContainer(request *domain.Request) {
	if _, ok := v.containers.Get(request.MetaData.ID); ok {
		return
	}

	ct := restful.NewV2(v.Base, request)
	v.containers.Set(request.MetaData.ID, ct)
}

func (v *BaseView) reloadData() {
	cols, err := v.Repository.LoadCollections()
	if err != nil {
		v.ShowError(fmt.Errorf("failed to load collections: %w", err))
		return
	}

	reqs, err := v.Repository.LoadRequests()
	if err != nil {
		v.ShowError(fmt.Errorf("failed to load requests: %w", err))
		return
	}

	v.PopulateTreeView(reqs, cols)
	v.Window.Invalidate()
}
