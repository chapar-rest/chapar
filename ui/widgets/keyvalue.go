package widgets

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type KeyValue struct {
	Items []*KeyValueItem

	filterText    string
	filteredItems []*KeyValueItem

	addButton *IconButton

	mx *sync.Mutex

	list *widget.List

	initialLoadComplete bool
	onChanged           func(items []*KeyValueItem)
}

type KeyValueItem struct {
	index  int
	Key    string
	Value  string
	Active bool

	keyEditor   *widget.Editor
	valueEditor *widget.Editor

	activeBool   *widget.Bool
	deleteButton *widget.Clickable
}

func NewKeyValue(items ...*KeyValueItem) *KeyValue {
	kv := &KeyValue{
		mx: &sync.Mutex{},
		addButton: &IconButton{
			Icon:      PlusIcon,
			Size:      unit.Dp(20),
			Clickable: &widget.Clickable{},
			Color:     Gray800,
		},
		Items: items,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},

		filteredItems: make([]*KeyValueItem, 0),
	}

	kv.addButton.OnClick = func() {
		kv.AddItem(NewKeyValueItem("", "", false))

		kv.triggerChanged()
	}

	return kv
}

func NewKeyValueItem(key, value string, active bool) *KeyValueItem {
	k := &widget.Editor{SingleLine: true}
	k.SetText(key)

	v := &widget.Editor{SingleLine: true}
	v.SetText(value)

	return &KeyValueItem{
		Key:          key,
		Value:        value,
		Active:       active,
		keyEditor:    k,
		valueEditor:  v,
		deleteButton: &widget.Clickable{},
		activeBool:   &widget.Bool{Value: active},
	}
}

func (kv *KeyValue) Filter(text string) {
	kv.mx.Lock()
	defer kv.mx.Unlock()

	kv.filterText = text

	if text == "" {
		kv.filteredItems = []*KeyValueItem{}
		return
	}

	var items []*KeyValueItem
	for _, item := range kv.Items {
		if strings.Contains(item.Key, text) || strings.Contains(item.Value, text) {
			items = append(items, item)
		}
	}
	kv.filteredItems = items
}

func (kv *KeyValue) SetOnChanged(onChanged func(items []*KeyValueItem)) {
	kv.onChanged = onChanged
}

func (kv *KeyValue) AddItem(item *KeyValueItem) {
	kv.mx.Lock()
	defer kv.mx.Unlock()
	item.index = len(kv.Items)
	kv.Items = append(kv.Items, item)
}

func (kv *KeyValue) SetItems(items []*KeyValueItem) {
	kv.mx.Lock()
	defer kv.mx.Unlock()
	kv.initialLoadComplete = false

	for i := range items {
		items[i].index = i
	}
	kv.Items = items

	kv.initialLoadComplete = true
}

func (kv *KeyValue) GetItems() []*KeyValueItem {
	kv.mx.Lock()
	defer kv.mx.Unlock()

	sort.Slice(kv.Items, func(i, j int) bool {
		return kv.Items[i].index < kv.Items[j].index
	})

	return kv.Items
}

func (kv *KeyValue) triggerChanged() {
	if kv.onChanged != nil && kv.initialLoadComplete { // Check flag before triggering
		kv.onChanged(kv.Items)
	}
}

func (kv *KeyValue) itemLayout(gtx layout.Context, theme *material.Theme, index int, item *KeyValueItem) layout.Dimensions {
	if index < 0 || index >= len(kv.Items) {
		// Index is out of range, return zero dimensions.
		return layout.Dimensions{}
	}

	if item.deleteButton.Clicked(gtx) {
		kv.mx.Lock()
		kv.Items = append(kv.Items[:index], kv.Items[index+1:]...)
		kv.mx.Unlock()

		kv.triggerChanged()

		return layout.Dimensions{}
	}

	border := widget.Border{
		Color:        Gray300,
		CornerRadius: 0,
		Width:        1,
	}

	if item.activeBool.Update(gtx) {
		item.Active = item.activeBool.Value
		kv.triggerChanged()
	}

	for _, ev := range item.keyEditor.Events() {
		if _, ok := ev.(widget.ChangeEvent); ok {
			item.Key = item.keyEditor.Text()
			fmt.Println("key changed")
			kv.triggerChanged()
		}
	}

	for _, ev := range item.valueEditor.Events() {
		if _, ok := ev.(widget.ChangeEvent); ok {
			item.Value = item.valueEditor.Text()
			fmt.Println("value changed")
			kv.triggerChanged()
		}
	}

	leftPadding := layout.Inset{Left: unit.Dp(8)}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.CheckBox(theme, item.activeBool, "").Layout(gtx)
				})
			}),
			DrawLineFlex(Gray300, unit.Dp(35), unit.Dp(1)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(.80, func(gtx layout.Context) layout.Dimensions {
						return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.Editor(theme, item.keyEditor, "Key").Layout(gtx)
						})
					}),
					DrawLineFlex(Gray300, unit.Dp(35), unit.Dp(1)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.Editor(theme, item.valueEditor, "Value").Layout(gtx)
						})
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ib := IconButton{
					Icon:      DeleteIcon,
					Size:      unit.Dp(20),
					Color:     Gray800,
					Clickable: item.deleteButton,
				}
				return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ib.Layout(theme, gtx)
				})
			}),
		)
	})
}

func (kv *KeyValue) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	items := kv.Items
	if kv.filterText != "" {
		items = kv.filteredItems
	}

	if len(items) == 0 {
		return layout.Center.Layout(gtx, material.Label(theme, unit.Sp(14), "No items").Layout)
	}

	return material.List(theme, kv.list).Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		return kv.itemLayout(gtx, theme, i, items[i])
	})
}

func (kv *KeyValue) WithAddLayout(gtx layout.Context, title, hint string, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme, theme.TextSize, title).Layout(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Left:  unit.Dp(10),
						Right: unit.Dp(10),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, unit.Sp(10), hint).Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top:    0,
						Bottom: unit.Dp(10),
						Left:   0,
						Right:  unit.Dp(10),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return kv.addButton.Layout(theme, gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return kv.Layout(gtx, theme)
		}),
	)
}
