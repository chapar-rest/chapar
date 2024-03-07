package envs

import (
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/google/uuid"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/converter"
	"github.com/mirzakhany/chapar/ui/keys"
	"github.com/mirzakhany/chapar/ui/pages/tips"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type View struct {
	newEnvButton widget.Clickable

	treeViewSearchBox *widgets.TextField
	treeViewNodes     map[string]*widgets.TreeNode
	treeView          *widgets.TreeView

	split     widgets.SplitView
	tabHeader *widgets.Tabs

	openTabs map[string]*widgets.Tab

	// env container
	items          *widgets.KeyValue
	title          *widgets.EditableLabel
	itemsSearchBox *widgets.TextField
	saveButton     widget.Clickable
	prompt         *widgets.Prompt
	dataChanged    bool

	// callbacks
	onTitleChanged              func(id, title string)
	onNewEnv                    func()
	onTabClose                  func(id string)
	onListFilter                func(filter string)
	onItemsFilter               func(filter string)
	onItemsChanged              func(id string, items []domain.KeyValue)
	onSave                      func(id string)
	onTreeViewNodeClicked       func(id string)
	onTreeViewNodeDoubleClicked func(id string)
	onTreeViewMenuClicked       func(id string, action string)
	onTabSelected               func(id string)
}

func NewView(theme *material.Theme) *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	itemsSearchBox := widgets.NewTextField("", "Search...")
	itemsSearchBox.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	itemsSearchBox.SetBorderColor(widgets.Gray600)

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

		items: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", "", false),
		),
		title:          widgets.NewEditableLabel(""),
		itemsSearchBox: itemsSearchBox,
		prompt:         widgets.NewPrompt("Save", "", widgets.ModalTypeWarn, "Yes", "No"),
	}
	v.prompt.WithRememberBool()

	v.title.SetOnChanged(func(text string) {
		if v.onTitleChanged != nil {
			v.onTitleChanged(v.tabHeader.SelectedTab().GetIdentifier(), text)
		}
	})

	v.treeViewSearchBox.SetOnTextChange(func(text string) {
		if len(v.treeViewNodes) == 0 {
			return
		}
		v.treeView.Filter(text)
	})

	v.itemsSearchBox.SetOnTextChange(func(text string) {
		if v.items == nil {
			return
		}
		v.items.Filter(text)
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
			MenuOptions: menuItems,
		}
		treeViewNodes = append(treeViewNodes, node)

		v.treeViewNodes[env.MetaData.ID] = node
	}

	v.treeView.SetNodes(treeViewNodes)
}

func (v *View) SetItems(items []domain.KeyValue) {
	v.items.SetItems(converter.WidgetItemsFromKeyValue(items))
}

func (v *View) SetOnItemsFilter(onItemsFilter func(filter string)) {
	if v.items == nil {
		return
	}
	v.onItemsFilter = onItemsFilter
	v.itemsSearchBox.SetOnTextChange(onItemsFilter)
}

func (v *View) SetOnListFilter(onListFilter func(filter string)) {
	v.onListFilter = onListFilter
	v.treeViewSearchBox.SetOnTextChange(onListFilter)
}

func (v *View) SetOnItemsChanged(onItemsChanged func(id string, items []domain.KeyValue)) {
	v.onItemsChanged = onItemsChanged
	v.items.SetOnChanged(func(items []*widgets.KeyValueItem) {
		if v.onItemsChanged != nil {
			v.onItemsChanged(v.tabHeader.SelectedTab().GetIdentifier(), converter.KeyValueFromWidgetItems(items))
		}
	})
}

func (v *View) SetOnTreeViewNodeDoubleClicked(onTreeViewNodeDoubleClicked func(id string)) {
	v.onTreeViewNodeDoubleClicked = onTreeViewNodeDoubleClicked
	v.treeView.OnNodeDoubleClick(func(node *widgets.TreeNode) {
		if v.onTreeViewNodeDoubleClicked != nil {
			v.onTreeViewNodeDoubleClicked(node.Identifier)
		}
	})
}

func (v *View) SetOnTreeViewMenuClicked(onTreeViewMenuClicked func(id string, action string)) {
	v.onTreeViewMenuClicked = onTreeViewMenuClicked
	v.treeView.SetOnMenuItemClick(func(node *widgets.TreeNode, item string) {
		if v.onTreeViewMenuClicked != nil {
			v.onTreeViewMenuClicked(node.Identifier, item)
		}
	})
}

func (v *View) SetOnSave(onSave func(id string)) {
	v.onSave = onSave
}

func (v *View) SetOnTitleChanged(onTitleChanged func(id, title string)) {
	v.title.SetOnChanged(func(text string) {
		onTitleChanged(v.tabHeader.SelectedTab().GetIdentifier(), text)
	})
}

func (v *View) SetOnNewEnv(onNewEnv func()) {
	v.onNewEnv = onNewEnv
}

func (v *View) SetOnTabSelected(onTabSelected func(id string)) {
	v.onTabSelected = onTabSelected
}

func (v *View) SetOnTabClose(onTabClose func(id string)) {
	v.onTabClose = onTabClose
}

func (v *View) LoadEnv(env *domain.Environment) {
	//v.activeID = env.MetaData.ID
	v.title.SetText(env.MetaData.Name)
	v.items.SetItems(converter.WidgetItemsFromKeyValue(env.Spec.Values))
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
		v.dataChanged = dirty
	}
}

func (v *View) AddNewEnv(env *domain.Environment) {
	node := &widgets.TreeNode{
		Text:        env.MetaData.Name,
		Identifier:  env.MetaData.ID,
		MenuOptions: menuItems,
	}
	v.treeView.AddNode(node)
	v.OpenTab(env)
}

func (v *View) ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...string) {
	v.prompt.Type = modalType
	v.prompt.Title = title
	v.prompt.Content = content
	v.prompt.SetOptions(options...)
	v.prompt.WithRememberBool()
	v.prompt.SetOnSubmit(onSubmit)
	v.prompt.Show()
}

func (v *View) HidePrompt() {
	v.prompt.Hide()
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
			v.onTabClose(tab.Identifier)
		})
	}
	i := v.tabHeader.AddTab(tab)
	v.openTabs[env.MetaData.ID] = tab
	v.LoadEnv(env)
	v.tabHeader.SetSelected(i)
}

func (v *View) CloseTab(id string) {
	if _, ok := v.openTabs[id]; ok {
		v.tabHeader.RemoveTabByID(id)
		delete(v.openTabs, id)
	}
}

func (v *View) IsEnvTabOpen(id string) bool {
	_, ok := v.openTabs[id]
	return ok
}

func (v *View) SwitchToTab(env *domain.Environment) {
	if _, ok := v.openTabs[env.MetaData.ID]; ok {
		v.tabHeader.SetSelectedByID(env.MetaData.ID)
		v.title.SetText(env.MetaData.Name)
		v.items.SetItems(converter.WidgetItemsFromKeyValue(env.Spec.Values))
	}
}

func (v *View) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return v.split.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return v.envList(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return v.containerHolder(gtx, theme)
		},
	)
}

func (v *View) envList(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if v.newEnvButton.Clicked(gtx) {
								if v.onNewEnv != nil {
									v.onNewEnv()
								}
							}
							return material.Button(theme, &v.newEnvButton, "New").Layout(gtx)
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
			}

			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return v.prompt.Layout(gtx, theme)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Top:    unit.Dp(5),
							Bottom: unit.Dp(15),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return v.title.Layout(gtx, theme)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if v.dataChanged {
												if v.saveButton.Clicked(gtx) {
													if v.onSave != nil {
														v.onSave(v.tabHeader.SelectedTab().GetIdentifier())
													}
												}
												return widgets.SaveButtonLayout(gtx, theme, &v.saveButton)
											} else {
												return layout.Dimensions{}
											}
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Max.X = gtx.Dp(200)
									return v.itemsSearchBox.Layout(gtx, theme)
								}),
							)
						})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return v.items.WithAddLayout(gtx, "", "Disabled items have no effect on your requests", theme)
					}),
				)
			})
		}),
	)
}
