package environments

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	giox "gioui.org/x/component"
	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/converter"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/pages/tips"
	"github.com/chapar-rest/chapar/ui/widgets"
)

const (
	Duplicate = "Duplicate"
	Delete    = "Delete"
)

type View struct {
	newEnvButton widget.Clickable
	importButton widget.Clickable

	treeViewSearchBox *widgets.TextField
	treeView          *widgets.TreeView

	split     widgets.SplitView
	tabHeader *widgets.Tabs

	// callbacks
	onTitleChanged        func(id, title string) error
	onNewEnv              func() error
	onImportEnv           func() error
	onTabClose            func(id string) error
	onItemsChanged        func(id string, items []domain.KeyValue) error
	onSave                func(id string) error
	onTreeViewNodeClicked func(id string) error
	onTreeViewMenuClicked func(id string, action string) error
	onTabSelected         func(id string)

	// state
	containers    *safemap.Map[*container]
	openTabs      *safemap.Map[*widgets.Tab]
	treeViewNodes *safemap.Map[*widgets.TreeNode]

	tipsView *tips.Tips
}

func NewView(theme *chapartheme.Theme) *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(theme.BorderColor)

	itemsSearchBox := widgets.NewTextField("", "Search...")
	itemsSearchBox.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	itemsSearchBox.SetBorderColor(theme.BorderColor)

	v := &View{
		treeViewSearchBox: search,
		tabHeader:         widgets.NewTabs([]*widgets.Tab{}, nil),
		treeView:          widgets.NewTreeView([]*widgets.TreeNode{}),
		split: widgets.SplitView{
			Resize: giox.Resize{
				Ratio: 0.19,
			},
			BarWidth: unit.Dp(2),
		},
		treeViewNodes: safemap.New[*widgets.TreeNode](),
		openTabs:      safemap.New[*widgets.Tab](),
		containers:    safemap.New[*container](),

		tipsView: tips.New(),
	}

	v.treeViewSearchBox.SetOnTextChange(func(text string) {
		if v.treeViewNodes.Len() == 0 {
			return
		}
		v.treeView.Filter(text)
	})
	return v
}

func (v *View) PopulateTreeView(envs []*domain.Environment) {
	treeViewNodes := make([]*widgets.TreeNode, 0)
	for _, env := range envs {
		if env.MetaData.ID == "" {
			env.MetaData.ID = uuid.NewString()
		}

		node := &widgets.TreeNode{
			Text:        env.MetaData.Name,
			Identifier:  env.MetaData.ID,
			MenuOptions: []string{Duplicate, Delete},
		}

		treeViewNodes = append(treeViewNodes, node)
		v.treeViewNodes.Set(env.MetaData.ID, node)
	}
	v.treeView.SetNodes(treeViewNodes)
}

func (v *View) AddTreeViewNode(env *domain.Environment) {
	if env.MetaData.ID == "" {
		env.MetaData.ID = uuid.NewString()
	}

	node := &widgets.TreeNode{
		Text:        env.MetaData.Name,
		Identifier:  env.MetaData.ID,
		MenuOptions: []string{Duplicate, Delete},
	}
	v.treeView.AddNode(node)
	v.treeViewNodes.Set(env.MetaData.ID, node)
}

func (v *View) UpdateTreeViewNode(env *domain.Environment) {
	if node, ok := v.treeViewNodes.Get(env.MetaData.ID); ok {
		node.Text = env.MetaData.Name
	}
}

func (v *View) RemoveTreeViewNode(id string) {
	if _, ok := v.treeViewNodes.Get(id); !ok {
		return
	}

	v.treeView.RemoveNode(id)
}

func (v *View) SetOnItemsChanged(onItemsChanged func(id string, items []domain.KeyValue) error) {
	v.onItemsChanged = onItemsChanged
}

func (v *View) SetOnTreeViewNodeClicked(onTreeViewNodeClicked func(id string) error) {
	v.onTreeViewNodeClicked = onTreeViewNodeClicked
	v.treeView.OnNodeClick(func(node *widgets.TreeNode) {
		v.handleError(v.onTreeViewNodeClicked(node.Identifier))
	})
}

func (v *View) SetOnTreeViewMenuClicked(onTreeViewMenuClicked func(id string, action string) error) {
	v.onTreeViewMenuClicked = onTreeViewMenuClicked
	v.treeView.SetOnMenuItemClick(func(node *widgets.TreeNode, item string) {
		v.handleError(v.onTreeViewMenuClicked(node.Identifier, item))
	})
}

func (v *View) SetOnSave(onSave func(id string) error) {
	v.onSave = onSave
}

func (v *View) SetOnTitleChanged(onTitleChanged func(id, title string) error) {
	v.onTitleChanged = onTitleChanged
}

func (v *View) SetOnNewEnv(onNewEnv func() error) {
	v.onNewEnv = onNewEnv
}

func (v *View) SetOnImportEnv(onImportEnv func() error) {
	v.onImportEnv = onImportEnv
}

func (v *View) SetOnTabSelected(onTabSelected func(id string)) {
	v.onTabSelected = onTabSelected
}

func (v *View) SetOnTabClose(onTabClose func(id string) error) {
	v.onTabClose = onTabClose
}

func (v *View) UpdateTabTitle(id, title string) {
	if tab, ok := v.openTabs.Get(id); ok {
		tab.Title = title
	}
}

func (v *View) UpdateTreeNodeTitle(id, title string) {
	if node, ok := v.treeViewNodes.Get(id); ok {
		node.Text = title
	}
}

func (v *View) SetTabDirty(id string, dirty bool) {
	if tab, ok := v.openTabs.Get(id); ok {
		tab.SetDataChanged(dirty)
		if ct, ok := v.containers.Get(id); ok {
			ct.DataChanged = dirty
		}
	}
}

func (v *View) ShowPrompt(id, title, content, modalType string, onSubmit func(selectedOption string, remember bool) error, options ...widgets.Option) {
	ct, ok := v.containers.Get(id)
	if !ok {
		return
	}

	ct.Prompt.Type = modalType
	ct.Prompt.Title = title
	ct.Prompt.Content = content
	ct.Prompt.SetOptions(options...)
	ct.Prompt.WithoutRememberBool()
	ct.Prompt.SetOnSubmit(onSubmit)
	ct.Prompt.Show()
}

func (v *View) HidePrompt(id string) {
	ct, ok := v.containers.Get(id)
	if !ok {
		return
	}

	ct.Prompt.Hide()
}

func (v *View) OpenTab(env *domain.Environment) {
	tab := &widgets.Tab{
		Title:          env.MetaData.Name,
		Closable:       true,
		CloseClickable: &widget.Clickable{},
		Identifier:     env.MetaData.ID,
	}
	if v.onTabClose != nil {
		tab.SetOnClose(func(tab *widgets.Tab) {
			v.handleError(v.onTabClose(tab.Identifier))
		})
	}
	i := v.tabHeader.AddTab(tab)
	v.openTabs.Set(env.MetaData.ID, tab)
	v.tabHeader.SetSelected(i)
}

func (v *View) OpenContainer(env *domain.Environment) {
	if _, ok := v.containers.Get(env.MetaData.ID); ok {
		return
	}

	ct := newContainer(env.MetaData.ID, env.MetaData.Name, env.Spec.Values)
	ct.Title.SetOnChanged(func(text string) {
		if v.onTitleChanged != nil {
			v.handleError(v.onTitleChanged(env.MetaData.ID, text))
		}
	})

	ct.Items.SetOnChanged(func(items []*widgets.KeyValueItem) {
		if v.onItemsChanged != nil {
			v.handleError(v.onItemsChanged(env.MetaData.ID, converter.KeyValueFromWidgetItems(items)))
		}
	})

	ct.SearchBox.SetOnTextChange(func(text string) {
		if ct.Items == nil {
			return
		}
		ct.Items.Filter(text)
	})

	v.containers.Set(env.MetaData.ID, ct)
}

func (v *View) ReloadContainerData(env *domain.Environment) {
	if ct, ok := v.containers.Get(env.MetaData.ID); ok {
		ct.SetItems(env.Spec.Values)
	}
}

func (v *View) CloseTab(id string) {
	if _, ok := v.openTabs.Get(id); ok {
		v.tabHeader.RemoveTabByID(id)
		v.openTabs.Delete(id)
		v.containers.Delete(id)
	}
}

func (v *View) IsTabOpen(id string) bool {
	_, ok := v.openTabs.Get(id)
	return ok
}

func (v *View) SwitchToTab(id string) {
	if _, ok := v.openTabs.Get(id); ok {
		v.tabHeader.SetSelectedByID(id)
	}
}

func (v *View) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return v.split.Layout(gtx, theme,
		func(gtx layout.Context) layout.Dimensions {
			return v.envList(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return v.containerHolder(gtx, theme)
		},
	)
}

func (v *View) envList(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if v.importButton.Clicked(gtx) {
								if v.onImportEnv != nil {
									v.handleError(v.onImportEnv())
								}
							}
							btn := widgets.Button(theme.Material(), &v.importButton, widgets.UploadIcon, widgets.IconPositionStart, "Import")
							btn.Color = theme.ButtonTextColor
							return btn.Layout(gtx, theme)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if v.newEnvButton.Clicked(gtx) {
								if v.onNewEnv != nil {
									v.handleError(v.onNewEnv())
								}
							}
							btn := widgets.Button(theme.Material(), &v.newEnvButton, widgets.PlusIcon, widgets.IconPositionStart, "New")
							btn.Color = theme.ButtonTextColor
							return btn.Layout(gtx, theme)
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
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeView.Layout(gtx, theme)
				})
			}),
		)
	})
}

func (v *View) containerHolder(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if v.onSave != nil {
		keys.OnSaveCommand(gtx, v, func() {
			v.handleError(v.onSave(v.tabHeader.SelectedTab().GetIdentifier()))
		})
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return v.tabHeader.Layout(gtx, theme)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if v.openTabs.Len() == 0 {
				return v.tipsView.Layout(gtx, theme)
			}

			selectedTab := v.tabHeader.SelectedTab()
			if selectedTab != nil {
				if v.onTabSelected != nil {
					v.onTabSelected(selectedTab.Identifier)
				}

				if ct, ok := v.containers.Get(selectedTab.Identifier); ok {
					if v.onSave != nil {
						if ct.SaveButton.Clicked(gtx) {
							v.handleError(v.onSave(selectedTab.Identifier))
						}
					}

					return ct.Layout(gtx, theme, selectedTab.Identifier)
				}
			}

			return layout.Dimensions{}
		}),
	)
}

func (v *View) handleError(err error) {
	// TODO: Implement this
}
