package widgets

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
)

type Tabs struct {
	Items []*Tab

	SelectedIndex int

	onSelectedChange func(int)
}

func NewTabs(Items []*Tab, onSelectedChange func(int)) *Tabs {
	t := &Tabs{
		Items:            Items,
		SelectedIndex:    0,
		onSelectedChange: onSelectedChange,
	}

	t.Items[0].IsSelected = true
	return t
}

func (t *Tabs) Layout(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	var items []layout.FlexChild
	for i := range t.Items {
		i := i
		items = append(items, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if t.Items[i].clickable.Clicked(gtx) {
				t.changeSelected(i)
			}
			return t.Items[i].Layout(theme, gtx)
		}))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx, items...)
		}),
		HorizontalFullLine(),
	)
}

func (t *Tabs) changeSelected(index int) {
	for i, item := range t.Items {
		if i == index {
			t.SelectedIndex = i
			t.Items[i].IsSelected = true
			go t.onSelectedChange(i)
			continue
		} else {
			item.IsSelected = false
		}
	}
}
