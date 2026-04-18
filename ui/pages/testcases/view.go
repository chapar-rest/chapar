package testcases

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	giox "gioui.org/x/component"
	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/modals"
	"github.com/chapar-rest/chapar/ui/navigator"
	"github.com/chapar-rest/chapar/ui/pages/tips"
	"github.com/chapar-rest/chapar/ui/widgets"
)

var _ navigator.View = &View{}

const (
	Duplicate = "Duplicate"
	Delete    = "Delete"
	Run       = "Run"
)

type View struct {
	*ui.Base

	window *app.Window

	newTestButton widget.Clickable
	importButton  widget.Clickable

	treeViewSearchBox *widgets.TextField
	treeView          *widgets.TreeView

	split     widgets.SplitView
	tabHeader *widgets.Tabs

	// callbacks
	onTitleChanged        func(id, title string)
	onNewTestCase         func()
	onImportTestCase      func()
	onTabClose            func(id string)
	onYAMLChanged         func(id string, yaml string)
	onSave                func(id string)
	onRun                 func(id string)
	onTreeViewNodeClicked func(id string)
	onTreeViewMenuClicked func(id string, action string)
	onTabSelected         func(id string)

	// state
	containers    *safemap.Map[*container]
	openTabs      *safemap.Map[*widgets.Tab]
	treeViewNodes *safemap.Map[*widgets.TreeNode]

	tipsView *tips.Tips
}

func (v *View) OnEnter() {
}

func (v *View) Info() navigator.Info {
	return navigator.Info{
		ID:    "testcases",
		Title: "Tests",
		Icon:  widgets.CheckCircleIcon,
	}
}

func NewView(base *ui.Base) *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(base.Theme.BorderColor)

	v := &View{
		Base:              base,
		window:            base.Window,
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
		tipsView:      tips.New(),
	}

	v.treeViewSearchBox.SetOnTextChange(func(text string) {
		if v.treeViewNodes.Len() == 0 {
			return
		}
		v.treeView.Filter(text)
	})
	return v
}

func (v *View) showError(err error) {
	m := modals.NewError(err)
	v.Base.SetModal(func(gtx layout.Context) layout.Dimensions {
		if m.OKBtn.Clicked(gtx) {
			v.Base.CloseModal()
		}
		return m.Layout(gtx, v.Theme)
	})
}

func (v *View) PopulateTreeView(testCases []*domain.TestCase) {
	treeViewNodes := make([]*widgets.TreeNode, 0)
	for _, tc := range testCases {
		if tc.MetaData.ID == "" {
			tc.MetaData.ID = uuid.NewString()
		}

		node := &widgets.TreeNode{
			Text:        tc.MetaData.Name,
			Identifier:  tc.MetaData.ID,
			MenuOptions: []string{Run, Duplicate, Delete},
		}

		treeViewNodes = append(treeViewNodes, node)
		v.treeViewNodes.Set(tc.MetaData.ID, node)
	}
	v.treeView.SetNodes(treeViewNodes)
}

func (v *View) SetContainerTitle(id, title string) {
	if ct, ok := v.containers.Get(id); ok {
		ct.SetTitle(title)
	}
}

func (v *View) AddTreeViewNode(testCase *domain.TestCase) {
	if testCase.MetaData.ID == "" {
		testCase.MetaData.ID = uuid.NewString()
	}

	node := &widgets.TreeNode{
		Text:        testCase.MetaData.Name,
		Identifier:  testCase.MetaData.ID,
		MenuOptions: []string{Run, Duplicate, Delete},
	}
	v.treeView.AddNode(node)
	v.treeViewNodes.Set(testCase.MetaData.ID, node)
}

func (v *View) UpdateTreeViewNode(testCase *domain.TestCase) {
	if node, ok := v.treeViewNodes.Get(testCase.MetaData.ID); ok {
		node.Text = testCase.MetaData.Name
	}
}

func (v *View) RemoveTreeViewNode(id string) {
	if _, ok := v.treeViewNodes.Get(id); !ok {
		return
	}

	v.treeView.RemoveNode(id)
}

func (v *View) SetOnYAMLChanged(onYAMLChanged func(id string, yaml string)) {
	v.onYAMLChanged = onYAMLChanged
}

func (v *View) SetOnTreeViewNodeClicked(onTreeViewNodeClicked func(id string)) {
	v.onTreeViewNodeClicked = onTreeViewNodeClicked
	v.treeView.OnNodeClick(func(node *widgets.TreeNode) {
		v.onTreeViewNodeClicked(node.Identifier)
	})
}

func (v *View) SetOnTreeViewMenuClicked(onTreeViewMenuClicked func(id string, action string)) {
	v.onTreeViewMenuClicked = onTreeViewMenuClicked
	v.treeView.SetOnMenuItemClick(func(node *widgets.TreeNode, item string) {
		v.onTreeViewMenuClicked(node.Identifier, item)
	})
}

func (v *View) SetOnSave(onSave func(id string)) {
	v.onSave = onSave
}

func (v *View) SetOnRun(onRun func(id string)) {
	v.onRun = onRun
}

func (v *View) SetOnTitleChanged(onTitleChanged func(id, title string)) {
	v.onTitleChanged = onTitleChanged
}

func (v *View) SetOnNewTestCase(onNewTestCase func()) {
	v.onNewTestCase = onNewTestCase
}

func (v *View) SetOnImportTestCase(onImportTestCase func()) {
	v.onImportTestCase = onImportTestCase
}

func (v *View) SetOnTabClose(onTabClose func(id string)) {
	v.onTabClose = onTabClose
}

func (v *View) SetOnTabSelected(onTabSelected func(id string)) {
	v.onTabSelected = onTabSelected
}

func (v *View) OpenTab(testCase *domain.TestCase, yaml string) {
	tab := &widgets.Tab{
		Title:          testCase.MetaData.Name,
		Closable:       true,
		CloseClickable: &widget.Clickable{},
		Identifier:     testCase.MetaData.ID,
	}
	if v.onTabClose != nil {
		tab.SetOnClose(func(tab *widgets.Tab) {
			v.onTabClose(tab.Identifier)
		})
	}
	i := v.tabHeader.AddTab(tab)
	v.openTabs.Set(testCase.MetaData.ID, tab)
	v.tabHeader.SetSelected(i)
}

func (v *View) OpenContainer(testCase *domain.TestCase, yaml string) {
	if _, ok := v.containers.Get(testCase.MetaData.ID); ok {
		return
	}

	ct := newContainer(v.Base, testCase, yaml)
	ct.SetOnSave(func(id string) {
		if v.onSave != nil {
			v.onSave(id)
		}
	})
	ct.SetOnRun(func(id string) {
		if v.onRun != nil {
			v.onRun(id)
		}
	})
	ct.SetOnTitleChanged(func(id, title string) {
		if v.onTitleChanged != nil {
			v.onTitleChanged(id, title)
		}
	})
	ct.SetOnYAMLChanged(func(id, yaml string) {
		if v.onYAMLChanged != nil {
			v.onYAMLChanged(id, yaml)
		}
	})

	v.containers.Set(testCase.MetaData.ID, ct)
	v.window.Invalidate()
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

func (v *View) UpdateTabTitle(id, title string) {
	if tab, ok := v.openTabs.Get(id); ok {
		tab.Title = title
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

func (v *View) ShowPrompt(id, title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...widgets.Option) {
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

func (v *View) UpdateTestResult(id string, result *domain.TestResult) {
	if ct, ok := v.containers.Get(id); ok {
		ct.UpdateResult(result)
		v.window.Invalidate()
	}
}

func (v *View) ClearTestResult(id string) {
	if ct, ok := v.containers.Get(id); ok {
		ct.ClearResult()
		v.window.Invalidate()
	}
}

func (v *View) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return v.split.Layout(gtx, th,
		func(gtx layout.Context) layout.Dimensions {
			return v.testCaseList(gtx, th)
		},
		func(gtx layout.Context) layout.Dimensions {
			return v.containerHolder(gtx, th)
		},
	)
}

func (v *View) testCaseList(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if v.importButton.Clicked(gtx) {
								if v.onImportTestCase != nil {
									v.onImportTestCase()
								}
							}
							btn := widgets.Button(th.Material(), &v.importButton, widgets.UploadIcon, widgets.IconPositionStart, "Import")
							btn.Color = th.ButtonTextColor
							return btn.Layout(gtx, th)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if v.newTestButton.Clicked(gtx) {
								if v.onNewTestCase != nil {
									v.onNewTestCase()
								}
							}
							btn := widgets.Button(th.Material(), &v.newTestButton, widgets.PlusIcon, widgets.IconPositionStart, "New")
							btn.Color = th.ButtonTextColor
							return btn.Layout(gtx, th)
						}),
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeViewSearchBox.Layout(gtx, th)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeView.Layout(gtx, th)
				})
			}),
		)
	})
}

func (v *View) containerHolder(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	if v.onSave != nil {
		keys.OnSaveCommand(gtx, v, func() {
			selectedTab := v.tabHeader.SelectedTab()
			if selectedTab != nil {
				v.onSave(selectedTab.GetIdentifier())
			}
		})
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return v.tabHeader.Layout(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if v.openTabs.Len() == 0 {
				return v.tipsView.Layout(gtx, th)
			}

			selectedTab := v.tabHeader.SelectedTab()
			if selectedTab != nil {
				if v.onTabSelected != nil {
					v.onTabSelected(selectedTab.Identifier)
				}

				if ct, ok := v.containers.Get(selectedTab.Identifier); ok {
					return ct.Layout(gtx, th)
				}
			}

			return layout.Dimensions{}
		}),
	)
}
