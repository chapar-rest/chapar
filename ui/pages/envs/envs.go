package envs

import (
	"image/color"

	"gioui.org/op"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/google/uuid"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/loader"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Envs struct {
	addEnvButton widget.Clickable
	searchBox    *widgets.TextField
	treeView     *widgets.TreeView

	split        widgets.SplitView
	tabs         *widgets.Tabs
	envContainer *envContainer

	data []*domain.Environment

	openedTabs []*openedTab

	selectedIndex int
}

type openedTab struct {
	env       *domain.Environment
	tab       *widgets.Tab
	listItem  *widgets.TreeViewNode
	container *envContainer

	closed  bool
	isDirty bool
}

func New(theme *material.Theme) (*Envs, error) {
	data, err := loader.ReadEnvironmentsData()
	if err != nil {
		return nil, err
	}

	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	treeView := widgets.NewTreeView()

	e := &Envs{
		data:      data,
		searchBox: search,
		tabs:      widgets.NewTabs([]*widgets.Tab{}, nil),
		treeView:  treeView,
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

	for _, env := range data {
		if env.Meta.ID == "" {
			env.Meta.ID = uuid.NewString()
		}

		node := widgets.NewNode(env.Meta.Name, false)
		node.OnDoubleClick(e.onItemDoubleClick)
		node.SetIdentifier(env.Meta.ID)
		treeView.AddNode(node, nil)
	}

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
		if ot.env.Meta.ID == id {
			// is name changed?
			if ot.env.Meta.Name != title {
				// Update the tree view item and the tab title
				ot.env.Meta.Name = title
				ot.tab.Title = title
				ot.listItem.Text = title
			}
		}
	}
}

func (e *Envs) onItemDoubleClick(tr *widgets.TreeViewNode) {
	// if env is already opened, just switch to it
	for i, ot := range e.openedTabs {
		if ot.env.Meta.ID == tr.Identifier {
			e.selectedIndex = i
			e.tabs.SetSelected(i)
			return
		}
	}

	for _, env := range e.data {
		if env.Meta.ID == tr.Identifier {
			tab := &widgets.Tab{Title: env.Meta.Name, Closable: true, CloseClickable: &widget.Clickable{}}
			tab.SetOnClose(e.onTabClose)
			tab.SetIdentifier(env.Meta.ID)

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
		}
	}
}

func (e *Envs) addNewEmptyEnv() {
	env := &domain.Environment{
		ApiVersion: "v1",
		Kind:       "Environment",
		Meta: domain.EnvMeta{
			ID:   uuid.NewString(),
			Name: "New env",
		},
		Values:   make([]domain.EnvValue, 0),
		FilePath: "",
	}

	treeViewNode := widgets.NewNode(env.Meta.Name, false)
	treeViewNode.OnDoubleClick(e.onItemDoubleClick)
	treeViewNode.SetIdentifier(env.Meta.ID)
	e.treeView.AddNode(treeViewNode, nil)

	tab := &widgets.Tab{Title: env.Meta.Name, Closable: true, CloseClickable: &widget.Clickable{}}
	tab.SetOnClose(e.onTabClose)
	tab.SetIdentifier(env.Meta.ID)

	ot := &openedTab{
		env:       env,
		tab:       tab,
		listItem:  treeViewNode,
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
		if ot.env.Meta.ID == t.Identifier {
			ot.closed = true
			break
		}
	}
}

func (e *Envs) SetData(data []*domain.Environment) {
	e.data = data
}

func (e *Envs) container(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return e.tabs.Layout(gtx, theme)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
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
							if e.addEnvButton.Clicked(gtx) {
								e.addNewEmptyEnv()
							}

							return material.Button(theme, &e.addEnvButton, "Add").Layout(gtx)
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
			if len(openItems) == 0 {
				return layout.Dimensions{}
			}
			return e.container(gtx, theme)
		},
	)
}
