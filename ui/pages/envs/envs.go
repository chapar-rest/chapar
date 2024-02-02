package envs

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Envs struct {
	addEnvButton widget.Clickable
	searchBox    *widgets.TextField
	envsList     *widgets.TreeView

	split        widgets.SplitView
	tabs         *widgets.Tabs
	envContainer *envContainer
}

func New(theme *material.Theme) *Envs {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)

	tabItems := []widgets.Tab{
		{Title: "Production", Closable: true, CloseClickable: &widget.Clickable{}},
		{Title: "Staging", Closable: true, CloseClickable: &widget.Clickable{}},
		{Title: "Dev", Closable: true, CloseClickable: &widget.Clickable{}},
	}

	onTabsChange := func(index int) {

	}

	treeView := widgets.NewTreeView()
	pr := widgets.NewNode("Production", false)
	treeView.AddNode(pr, nil)

	container := &envContainer{
		items: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),
		title: widgets.NewEditableLabel("Production"),
	}

	container.title.SetOnChanged(func(text string) {
		tabItems[0].Title = text
		pr.Text = text
	})

	e := &Envs{
		searchBox: search,
		tabs:      widgets.NewTabs(tabItems, onTabsChange),
		envsList:  treeView,
		split: widgets.SplitView{
			Ratio:         -0.64,
			MinLeftSize:   unit.Dp(250),
			MaxLeftSize:   unit.Dp(800),
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
		envContainer: container,
	}

	return e
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
			return e.container(gtx, theme)
		},
	)
}
