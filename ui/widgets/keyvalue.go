package widgets

import (
	"sync"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type KeyValue struct {
	Items []KeyValueItem

	addButton *IconButton

	mx *sync.Mutex

	list *widget.List
}

type KeyValueItem struct {
	Key   string
	Value string

	keyEditor   *widget.Editor
	valueEditor *widget.Editor

	activeBool   *widget.Bool
	deleteButton *widget.Clickable
}

func NewKeyValue(items ...KeyValueItem) *KeyValue {
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
	}

	kv.addButton.OnClick = func() {
		kv.AddItem(NewKeyValueItem("", "", false))
	}

	return kv
}

func NewKeyValueItem(key, value string, active bool) KeyValueItem {
	k := &widget.Editor{SingleLine: true}
	k.SetText(key)

	v := &widget.Editor{SingleLine: true}
	v.SetText(value)

	return KeyValueItem{
		Key:          key,
		Value:        value,
		keyEditor:    k,
		valueEditor:  v,
		deleteButton: &widget.Clickable{},
		activeBool:   &widget.Bool{Value: active},
	}
}

func (kv *KeyValue) AddItem(item KeyValueItem) {
	kv.mx.Lock()
	defer kv.mx.Unlock()
	kv.Items = append(kv.Items, item)
}

func (kv *KeyValue) SetItems(items []KeyValueItem) {
	kv.mx.Lock()
	defer kv.mx.Unlock()
	kv.Items = items
}

func (kv *KeyValue) SetActive(key string) {
	kv.mx.Lock()
	defer kv.mx.Unlock()
	for i, item := range kv.Items {
		if item.Key == key {
			kv.Items[i].activeBool.Value = true
		} else {
			kv.Items[i].activeBool.Value = false
		}
	}
}

func (kv *KeyValue) GetActive() KeyValueItem {
	for _, item := range kv.Items {
		if item.activeBool.Value {
			return item
		}
	}
	return KeyValueItem{}
}

func (kv *KeyValue) itemLayout(gtx layout.Context, theme *material.Theme, index int) layout.Dimensions {
	if kv.Items[index].deleteButton.Clicked(gtx) {
		kv.mx.Lock()
		defer kv.mx.Unlock()
		kv.Items = append(kv.Items[:index], kv.Items[index+1:]...)
	}

	if index >= len(kv.Items) {
		if len(kv.Items) == 0 {
			return layout.Dimensions{}
		}
		index = len(kv.Items) - 1
	}

	border := widget.Border{
		Color:        Gray300,
		CornerRadius: 0,
		Width:        1,
	}

	leftPadding := layout.Inset{Left: unit.Dp(8)}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.CheckBox(theme, kv.Items[index].activeBool, "").Layout(gtx)
				})
			}),
			DrawLineFlex(Gray300, unit.Dp(35), unit.Dp(1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(100))
				return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Editor(theme, kv.Items[index].keyEditor, "Key").Layout(gtx)
				})
			}),
			DrawLineFlex(Gray300, unit.Dp(35), unit.Dp(1)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Editor(theme, kv.Items[index].valueEditor, "Value").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ib := IconButton{
					Icon:      DeleteIcon,
					Size:      unit.Dp(20),
					Color:     Gray800,
					Clickable: kv.Items[index].deleteButton,
				}

				return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ib.Layout(theme, gtx)
				})
			}),
		)
	})
}

func (kv *KeyValue) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return material.List(theme, kv.list).Layout(gtx, len(kv.Items), func(gtx layout.Context, i int) layout.Dimensions {
		return kv.itemLayout(gtx, theme, i)
	})
}

func (kv *KeyValue) WithAddLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
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
