package widgets

import (
	"strconv"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
)

const (
	ItemTypeText    = "text"
	ItemTypeBool    = "bool"
	ItemTypeLNumber = "number"
)

type Settings struct {
	Items []*SettingItem

	list *widget.List

	onChange func(values map[string]any)
}

func NewSettings(items []*SettingItem) *Settings {
	s := &Settings{
		Items: items,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}

	for _, i := range items {
		i.onChange = s.onChanged
	}

	return s
}

func (s *Settings) SetOnChange(f func(values map[string]any)) {
	s.onChange = f
}

func (s *Settings) onChanged() {
	if s.onChange == nil {
		return
	}

	values := make(map[string]any, len(s.Items))
	for _, i := range s.Items {
		switch i.Type {
		case ItemTypeBool:
			values[i.Key] = i.boolState.Value
		case ItemTypeLNumber:
			v, err := strconv.Atoi(i.editor.Text())
			if err != nil {
				continue
			}
			values[i.Key] = v
		default:
			values[i.Key] = i.editor.Text()
		}
	}
	s.onChange(values)
}

type SettingItem struct {
	Title       string
	Key         string
	Description string
	Type        string
	Value       any

	boolState *widget.Bool
	editor    *widget.Editor

	onChange func()
}

func NewTextItem(title, key, description string, value string) *SettingItem {
	i := &SettingItem{
		Title:       title,
		Key:         key,
		Description: description,
		Type:        ItemTypeText,
		Value:       value,
		editor:      &widget.Editor{SingleLine: true, Alignment: text.Middle},
	}
	i.editor.SetText(value)
	return i
}

func NewBoolItem(title, key, description string, value bool) *SettingItem {
	b := &SettingItem{
		Title:       title,
		Key:         key,
		Description: description,
		Type:        ItemTypeBool,
		Value:       value,
		boolState:   new(widget.Bool),
	}

	b.boolState.Value = value
	return b
}

func NewNumberItem(title, key, description string, value int) *SettingItem {
	i := &SettingItem{
		Title:       title,
		Key:         key,
		Description: description,
		Type:        ItemTypeLNumber,
		Value:       value,
		editor:      &widget.Editor{SingleLine: true, Alignment: text.Middle},
	}
	i.editor.SetText(strconv.Itoa(value))
	return i
}

func (i *SettingItem) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(15)}

	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.Label(theme.Material(), unit.Sp(14), i.Title).Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lb := material.Label(theme.Material(), unit.Sp(12), i.Description)
						lb.Color = Disabled(theme.TextColor)
						return lb.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					switch i.Type {
					case ItemTypeText:
						return i.editorLayout(gtx, theme)
					case ItemTypeBool:
						return i.switchLayout(gtx, theme)
					case ItemTypeLNumber:
						return i.editorLayout(gtx, theme)
					default:
						return layout.Dimensions{}
					}
				})
			}),
		)
	})
}

func (i *SettingItem) switchLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	desc := "OFF"
	if i.boolState.Value {
		desc = "ON"
	}

	if i.boolState.Update(gtx) {
		i.onChange()
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			s := material.Switch(theme.Material(), i.boolState, "")
			s.Color.Enabled = theme.SwitchBgColor
			s.Color.Disabled = theme.Palette.Fg
			return s.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme.Material(), unit.Sp(12), desc).Layout(gtx)
		}),
	)
}

func (i *SettingItem) editorLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(50)
	border := widget.Border{
		Color:        theme.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	keys.OnEditorChange(gtx, i.editor, func() {
		i.onChange()
	})

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			editor := material.Editor(theme.Material(), i.editor, "")
			editor.TextSize = unit.Sp(14)
			return editor.Layout(gtx)
		})
	})
}

func (s *Settings) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(15), Right: unit.Dp(20)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return s.list.Layout(gtx, len(s.Items), func(gtx layout.Context, i int) layout.Dimensions {
			return s.Items[i].Layout(gtx, theme)
		})
	})
}
