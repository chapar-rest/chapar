package envs

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/google/uuid"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/loader"
	"github.com/mirzakhany/chapar/internal/logger"
	"github.com/mirzakhany/chapar/ui/manager"
	"github.com/mirzakhany/chapar/ui/pages/tips"
	"github.com/mirzakhany/chapar/ui/widgets"
)

var (
	menuItems = []string{"Duplicate", "Delete"}
)

type Envs struct {
	newEnvButton widget.Clickable
	searchBox    *widgets.TextField
	treeView     *widgets.TreeView

	split widgets.SplitView
	tabs  *widgets.Tabs

	data []*domain.Environment

	openedTabs []*openedTab

	selectedIndex int

	appManager *manager.Manager
}

type openedTab struct {
	env       *domain.Environment
	tab       *widgets.Tab
	listItem  *widgets.TreeNode
	container *envContainer

	closed bool
}

func New(theme *material.Theme, appManager *manager.Manager) (*Envs, error) {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	data := appManager.GetEnvironments()
	treeViewNodes := make([]*widgets.TreeNode, 0)
	for _, env := range data {
		if env.MetaData.ID == "" {
			env.MetaData.ID = uuid.NewString()
		}

		node := &widgets.TreeNode{
			Text:        env.MetaData.Name,
			Identifier:  env.MetaData.ID,
			MenuOptions: menuItems,
		}
		treeViewNodes = append(treeViewNodes, node)
	}

	e := &Envs{
		appManager: appManager,
		searchBox:  search,
		tabs:       widgets.NewTabs([]*widgets.Tab{}, nil),
		treeView:   widgets.NewTreeView(treeViewNodes),
		split: widgets.SplitView{
			Ratio:         -0.64,
			MinLeftSize:   unit.Dp(250),
			MaxLeftSize:   unit.Dp(800),
			BarWidth:      unit.Dp(2),
			BarColor:      widgets.Gray300,
			BarColorHover: theme.Palette.ContrastBg,
		},
		openedTabs: make([]*openedTab, 0),
	}

	e.treeView.OnDoubleClick(e.onItemDoubleClick)
	e.treeView.SetOnMenuItemClick(func(tr *widgets.TreeNode, item string) {
		if item == "Duplicate" {
			e.duplicateEnv(tr.Identifier)
		}

		if item == "Delete" {
			e.deleteEnv(tr.Identifier)
		}
	})

	e.searchBox.SetOnTextChange(func(text string) {
		if e.data == nil {
			return
		}

		e.treeView.Filter(text)
	})

	return e, nil
}

func (e *Envs) onTitleChanged(id, title string) {
	// find the opened tab and mark it as dirty
	for _, ot := range e.openedTabs {
		if ot.env.MetaData.ID == id {
			// is name changed?
			if ot.env.MetaData.Name != title {
				// Update the tree view item and the tab title
				ot.env.MetaData.Name = title
				ot.tab.Title = title
				ot.listItem.Text = title
			}
		}
	}
}

func (e *Envs) onItemDoubleClick(tr *widgets.TreeNode) {
	// if env is already opened, just switch to it
	for i, ot := range e.openedTabs {
		if ot.env.MetaData.ID == tr.Identifier {
			e.selectedIndex = i
			e.tabs.SetSelected(i)
			return
		}
	}

	for _, env := range e.data {
		if env.MetaData.ID == tr.Identifier {
			tab := &widgets.Tab{Title: env.MetaData.Name, Closable: true, CloseClickable: &widget.Clickable{}}
			tab.SetOnClose(e.onTabClose)
			tab.SetIdentifier(env.MetaData.ID)

			ot := &openedTab{
				env:       env,
				tab:       tab,
				listItem:  tr,
				container: newEnvContainer(env.Clone()),
			}
			ot.container.SetOnTitleChanged(e.onTitleChanged)
			e.openedTabs = append(e.openedTabs, ot)

			i := e.tabs.AddTab(tab)
			e.selectedIndex = i
			e.tabs.SetSelected(i)

			break
		}
	}
}

func (e *Envs) duplicateEnv(identifier string) {
	for _, env := range e.data {
		if env.MetaData.ID == identifier {
			newEnv := env.Clone()
			newEnv.MetaData.ID = uuid.NewString()
			newEnv.MetaData.Name = newEnv.MetaData.Name + " (copy)"
			// add copy to file name
			newEnv.FilePath = loader.AddSuffixBeforeExt(newEnv.FilePath, "-copy")
			e.data = append(e.data, newEnv)

			node := &widgets.TreeNode{
				Text:        newEnv.MetaData.Name,
				Identifier:  newEnv.MetaData.ID,
				MenuOptions: menuItems,
			}
			e.treeView.AddNode(node)
			if err := loader.UpdateEnvironment(newEnv); err != nil {
				logger.Error(fmt.Sprintf("failed to update environment, err %v", err))
			}
			break
		}
	}
}

func (e *Envs) deleteEnv(identifier string) {
	for i, env := range e.data {
		if env.MetaData.ID == identifier {
			e.data = append(e.data[:i], e.data[i+1:]...)
			e.treeView.RemoveNode(identifier)

			if err := loader.DeleteEnvironment(env); err != nil {
				logger.Error(fmt.Sprintf("failed to delete environment, err %v", err))
			}
			break
		}
	}
}

func (e *Envs) addNewEmptyEnv() {
	env := domain.NewEnvironment("New Environment")
	node := &widgets.TreeNode{
		Text:        env.MetaData.Name,
		Identifier:  env.MetaData.ID,
		MenuOptions: menuItems,
	}
	e.treeView.AddNode(node)

	tab := &widgets.Tab{Title: env.MetaData.Name, Closable: true, CloseClickable: &widget.Clickable{}}
	tab.SetOnClose(e.onTabClose)
	tab.SetIdentifier(env.MetaData.ID)

	ot := &openedTab{
		env:       env,
		tab:       tab,
		listItem:  node,
		container: newEnvContainer(env.Clone()),
	}
	ot.container.SetOnTitleChanged(e.onTitleChanged)
	e.openedTabs = append(e.openedTabs, ot)

	i := e.tabs.AddTab(tab)
	e.selectedIndex = i
	e.tabs.SetSelected(i)
}

func (e *Envs) onTabClose(t *widgets.Tab) {
	for _, ot := range e.openedTabs {
		if ot.env.MetaData.ID == t.Identifier {
			// can we close the tab?
			if !ot.container.OnClose() {
				return
			}
			ot.closed = true
			break
		}
	}
}

func (e *Envs) container(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return e.tabs.Layout(gtx, theme)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(e.openedTabs) == 0 {
				t := tips.New()
				return t.Layout(gtx, theme)
			}

			if e.selectedIndex > len(e.openedTabs)-1 {
				return layout.Dimensions{}
			}
			ct := e.openedTabs[e.selectedIndex].container
			e.openedTabs[e.selectedIndex].tab.SetDirty(ct.IsDataChanged())

			return ct.Layout(gtx, theme)
		}),
	)
}

func (e *Envs) list(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if e.newEnvButton.Clicked(gtx) {
								e.addNewEmptyEnv()
							}

							return material.Button(theme, &e.newEnvButton, "New").Layout(gtx)
						}),
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return e.searchBox.Layout(gtx, theme)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return e.treeView.Layout(gtx, theme)
				})
			}),
		)
	})
}

func (e *Envs) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	// update tabs with new items
	tabItems := make([]*widgets.Tab, 0)
	openItems := make([]*openedTab, 0)
	for _, ot := range e.openedTabs {
		if !ot.closed {
			tabItems = append(tabItems, ot.tab)
			openItems = append(openItems, ot)
		}
	}

	e.tabs.SetTabs(tabItems)
	e.openedTabs = openItems
	selectTab := e.tabs.Selected()
	gtx.Execute(op.InvalidateCmd{})

	// is selected tab is closed:
	// if its the last tab and there is another tab before it, select the previous one
	// if its the first tab and there is another tab after it, select the next one
	// if its the only tab, select it

	if selectTab > len(openItems)-1 {
		if len(openItems) > 0 {
			e.tabs.SetSelected(len(openItems) - 1)
		} else {
			selectTab = 0
			e.tabs.SetSelected(0)
		}
	}

	if e.selectedIndex != selectTab {
		e.selectedIndex = selectTab
	}

	return e.split.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return e.list(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return e.container(gtx, theme)
		},
	)
}
