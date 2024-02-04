package envs

import (
	"fmt"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/loader"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Envs struct {
	addEnvButton widget.Clickable
	searchBox    *widgets.TextField
	envsList     *widgets.TreeView

	split        widgets.SplitView
	tabs         *widgets.Tabs
	envContainer *envContainer

	data []*domain.Environment

	openedEnvs []*domain.Environment

	selectedIndex int
}

func New(theme *material.Theme) (*Envs, error) {
	data, err := loader.ReadEnvironmentsData()
	if err != nil {
		return nil, err
	}

	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)

	treeView := widgets.NewTreeView()

	e := &Envs{
		data:      data,
		searchBox: search,
		tabs:      widgets.NewTabs([]widgets.Tab{}, nil),
		envsList:  treeView,
		split: widgets.SplitView{
			Ratio:         -0.64,
			MinLeftSize:   unit.Dp(250),
			MaxLeftSize:   unit.Dp(800),
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
		envContainer: newEnvContainer(),
		openedEnvs:   make([]*domain.Environment, 0),
	}

	for _, env := range data {
		node := widgets.NewNode(env.Meta.Name, false)
		node.OnDoubleClick(e.onItemDoubleClick)
		treeView.AddNode(node, nil)
	}

	return e, nil
}

func (e *Envs) onItemDoubleClick(tr *widgets.TreeViewNode) {
	// if env is already opened, just switch to it
	for i, openedEnv := range e.openedEnvs {
		if openedEnv.Meta.Name == tr.Text {
			e.tabs.SetSelected(i)
			e.envContainer.Load(openedEnv)
			return
		}
	}

	for _, env := range e.data {
		if env.Meta.Name == tr.Text {
			e.openedEnvs = append(e.openedEnvs, env)
			tab := widgets.Tab{Title: env.Meta.Name, Closable: true, CloseClickable: &widget.Clickable{}}
			tab.SetOnClose(e.onTabClose)
			i := e.tabs.AddTab(tab)
			e.tabs.SetSelected(i)
			e.envContainer.Load(env)
		}
	}
}

func (e *Envs) onTabClose(t *widgets.Tab) {
	fmt.Println("tab closed", t.Title)
	fmt.Println("selected tab", e.tabs.Selected())
	fmt.Println("opened envs", len(e.openedEnvs))
	fmt.Println("index", e.selectedIndex)

	for i, env := range e.openedEnvs {
		if env.Meta.Name == t.Title {
			// if it's the last tab in the list, remove it and select the previous one
			if i == len(e.openedEnvs)-1 && i > 0 {
				e.tabs.SetSelected(i - 1)
				e.envContainer.Load(e.openedEnvs[i-1])
			} else {
				// if it's not the last tab, select the next one
				e.tabs.SetSelected(i + 1)
				e.envContainer.Load(e.openedEnvs[i+1])
			}

			e.openedEnvs = append(e.openedEnvs[:i], e.openedEnvs[i+1:]...)
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
			return e.envContainer.Layout(gtx, theme)
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
								e.envsList.AddNode(widgets.NewNode("New env", false), nil)
								i := e.tabs.AddTab(widgets.Tab{Title: "New env", Closable: true, CloseClickable: &widget.Clickable{}})
								e.tabs.SetSelected(i)
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
					return e.envsList.Layout(gtx, theme)
				})
			}),
		)
	})
}

func (e *Envs) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return e.split.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return e.list(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			if len(e.openedEnvs) == 0 {
				return layout.Dimensions{}
			}

			if e.selectedIndex != e.tabs.Selected() {
				e.selectedIndex = e.tabs.Selected()
				e.envContainer.Load(e.openedEnvs[e.selectedIndex])
			}

			return e.container(gtx, theme)
		},
	)
}
