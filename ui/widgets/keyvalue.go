package widgets

import (
	"sort"
	"strings"
	"sync"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/google/uuid"
	"github.com/mirzakhany/chapar/ui/chapartheme"
)

type KeyValue struct {
	Items []*KeyValueItem

	filterText    string
	filteredItems []*KeyValueItem

	addButton *IconButton

	mx *sync.Mutex

	list *widget.List

	onChanged func(items []*KeyValueItem)
}

type KeyValueItem struct {
	index int

	Identifier string
	Key        string
	Value      string
	Active     bool

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
		kv.AddItem(NewKeyValueItem("", "", uuid.NewString(), true))
		kv.triggerChanged()
	}

	return kv
}

func NewKeyValueItem(key, value, identifier string, active bool) *KeyValueItem {
	k := &widget.Editor{SingleLine: true}
	k.SetText(key)

	v := &widget.Editor{SingleLine: true}
	v.SetText(value)

	return &KeyValueItem{
		Identifier:   identifier,
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
	for i := range items {
		items[i].index = i
	}
	kv.Items = items
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
	if kv.onChanged != nil {
		kv.onChanged(kv.Items)
	}
}

func (kv *KeyValue) itemLayout(gtx layout.Context, theme *chapartheme.Theme, index int, item *KeyValueItem) layout.Dimensions {
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

	if item.activeBool.Update(gtx) {
		item.Active = item.activeBool.Value
		kv.triggerChanged()
	}

	for {
		event, ok := item.keyEditor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := event.(widget.ChangeEvent); ok {
			item.Key = item.keyEditor.Text()
			kv.triggerChanged()
		}
	}

	for {
		event, ok := item.valueEditor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := event.(widget.ChangeEvent); ok {
			item.Value = item.valueEditor.Text()
			kv.triggerChanged()
		}
	}
	leftPadding := layout.Inset{Left: unit.Dp(8)}

	content := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				ch := material.CheckBox(theme.Material(), item.activeBool, "")
				ch.IconColor = theme.CheckBoxColor
				return ch.Layout(gtx)
			})
		}),
		DrawLineFlex(theme.SeparatorColor, unit.Dp(35), unit.Dp(1)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(.80, func(gtx layout.Context) layout.Dimensions {
					return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(theme.Material(), item.keyEditor, "Key")
						ed.SelectionColor = theme.TextSelectionColor
						return ed.Layout(gtx)
					})
				}),
				DrawLineFlex(theme.SeparatorColor, unit.Dp(35), unit.Dp(1)),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return leftPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(theme.Material(), item.valueEditor, "Value")
						ed.SelectionColor = theme.TextSelectionColor
						return ed.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			ib := IconButton{
				Icon:      DeleteIcon,
				Size:      unit.Dp(20),
				Color:     theme.TextColor,
				Clickable: item.deleteButton,
			}
			return ib.Layout(gtx, theme)
		}),
	)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return content
		}),
		// TODO fix unknown padding issue on right side of the separator, as it should be 0 but it's not.
		DrawLineFlex(theme.SeparatorColor, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X)),
	)
}

func (kv *KeyValue) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	items := kv.Items
	if kv.filterText != "" {
		items = kv.filteredItems
	}

	border := widget.Border{
		Color:        theme.TableBorderColor,
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(1),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if len(items) == 0 {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, material.Label(theme.Material(), unit.Sp(14), "No items").Layout)
		}

		return material.List(theme.Material(), kv.list).Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
			return kv.itemLayout(gtx, theme, i, items[i])
		})
	})
}

func (kv *KeyValue) WithAddLayout(gtx layout.Context, title, hint string, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme.Material(), theme.TextSize, title).Layout(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Left:  unit.Dp(10),
						Right: unit.Dp(10),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme.Material(), unit.Sp(10), hint).Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top:    0,
						Bottom: unit.Dp(10),
						Left:   0,
						Right:  unit.Dp(10),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						kv.addButton.BackgroundColor = theme.Palette.Bg
						kv.addButton.Color = theme.TextColor
						return kv.addButton.Layout(gtx, theme)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return kv.Layout(gtx, theme)
		}),
	)
}
