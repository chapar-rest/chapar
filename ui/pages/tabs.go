package pages

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Tabs struct {
	theme *material.Theme

	Items         []string
	SelectedIndex int

	itemButtons []*widgets.Tab

	TabsW *widgets.Tabs
}

func NewTabs(theme *material.Theme, items []string) *Tabs {
	t := &Tabs{
		theme: theme,
	}

	for _, item := range items {
		f := widgets.NewTab(item, &widget.Clickable{})
		f.BackgroundColor = theme.Palette.Bg
		f.IndicatorColor = theme.Palette.ContrastBg
		t.itemButtons = append(t.itemButtons, f)
	}

	onSelectedChange := func(i int) {
		fmt.Println("select tab", items[i])
	}

	t.TabsW = widgets.NewTabs(t.itemButtons, onSelectedChange)
	//t.itemButtons[0].IsSelected = true

	return t
}

func (t *Tabs) Layout(gtx layout.Context) layout.Dimensions {
	//var tabs []layout.FlexChild
	//
	//for _, tb := range t.itemButtons {
	//	tb := tb
	//	tabs = append(tabs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
	//		return tb.Layout(t.theme, gtx)
	//	}))
	//}

	return t.TabsW.Layout(t.theme, gtx)
}
