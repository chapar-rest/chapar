package widgets

import (
	"strconv"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/keys"
)

const (
	ItemTypeText     = "text"
	ItemTypeFile     = "file"
	ItemTypeBool     = "bool"
	ItemTypeNumber   = "number"
	ItemTypeDropDown = "dropdown"
	ItemTypeHeader   = "header"
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

func (s *Settings) getValues() map[string]any {
	values := make(map[string]any, len(s.Items))
	for _, i := range s.Items {
		switch i.Type {
		case ItemTypeBool:
			values[i.Key] = i.boolState.Value
		case ItemTypeNumber:
			v, err := strconv.Atoi(i.editor.Text())
			if err != nil {
				continue
			}
			values[i.Key] = v
		case ItemTypeFile:
			values[i.Key] = i.FileSelector.GetFilePath()
		case ItemTypeDropDown:
			values[i.Key] = i.dropDown.GetSelected().GetValue()
		case ItemTypeText:
			values[i.Key] = i.editor.Text()
		case ItemTypeHeader:
			// do nothing
		}
	}
	return values
}

func (s *Settings) SetValues(values map[string]any) {
	for _, i := range s.Items {
		if v, ok := values[i.Key]; ok {
			switch i.Type {
			case ItemTypeBool:
				i.boolState.Value = v.(bool)
			case ItemTypeNumber:
				i.editor.SetText(strconv.Itoa(v.(int)))
			case ItemTypeFile:
				i.FileSelector.SetFileName(v.(string))
			case ItemTypeDropDown:
				i.dropDown.SetSelectedByValue(v.(string))
			case ItemTypeText:
				i.editor.SetText(v.(string))
			}
		}
	}
}

func (s *Settings) onChanged() {
	if s.onChange == nil {
		return
	}

	s.onChange(s.getValues())
}

type SettingItem struct {
	Title       string
	Key         string
	Description string
	Type        string
	Value       any
	Default     any

	boolState *widget.Bool
	editor    *widget.Editor

	dropDown *DropDown

	FileSelector *FileSelector

	visible     bool
	visibleWhen func(values map[string]any) bool

	onChange func()

	minWidth      unit.Dp
	textAlignment text.Alignment
}

func NewFileItem(explorer *explorer.Explorer, title, key, description string, value string, extensions ...string) *SettingItem {
	i := &SettingItem{
		Title:        title,
		Key:          key,
		Description:  description,
		Type:         ItemTypeFile,
		Value:        value,
		FileSelector: NewFileSelector(value, explorer, extensions...),
		visible:      true,
	}

	return i
}

func NewTextItem(title, key, description string, value string) *SettingItem {
	i := &SettingItem{
		Title:       title,
		Key:         key,
		Description: description,
		Type:        ItemTypeText,
		Value:       value,
		editor:      &widget.Editor{SingleLine: true, Alignment: text.Middle},
		visible:     true,
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
		visible:     true,
	}

	b.boolState.Value = value
	return b
}

func NewNumberItem(title, key, description string, value int) *SettingItem {
	i := &SettingItem{
		Title:       title,
		Key:         key,
		Description: description,
		Type:        ItemTypeNumber,
		Value:       value,
		editor:      &widget.Editor{SingleLine: true, Alignment: text.Middle},
		visible:     true,
	}
	i.editor.SetText(strconv.Itoa(value))
	return i
}

func NewDropDownItem(title, key, description, value string, options ...*DropDownOption) *SettingItem {
	i := &SettingItem{
		Title:       title,
		Key:         key,
		Description: description,
		Type:        ItemTypeDropDown,
		Value:       value,
		dropDown:    NewDropDown(options...),
		visible:     true,
	}

	i.dropDown.SetSelectedByValue(value)
	i.dropDown.MaxWidth = unit.Dp(150)
	return i
}

func NewHeaderItem(title string) *SettingItem {
	i := &SettingItem{
		Title:   title,
		Type:    ItemTypeHeader,
		visible: true,
	}

	return i
}

func (i *SettingItem) WithDefaultValue(value any) *SettingItem {
	i.Default = value
	return i
}

func (i *SettingItem) MinWidth(w unit.Dp) *SettingItem {
	i.minWidth = w
	return i
}

func (i *SettingItem) TextAlignment(a text.Alignment) *SettingItem {
	i.textAlignment = a
	if i.editor != nil {
		i.editor.Alignment = a
	}
	return i
}

func (i *SettingItem) SetVisibleWhen(f func(values map[string]any) bool) *SettingItem {
	i.visibleWhen = f
	return i
}

func (i *SettingItem) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if !i.visible {
		return layout.Dimensions{}
	}

	inset := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(15)}
	if i.Type == ItemTypeHeader {
		inset = layout.Inset{Top: unit.Dp(5), Bottom: 0}
	}

	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(theme.Material(), unit.Sp(14), i.Title)
							if i.Type == ItemTypeHeader {
								l.Font.Weight = font.Bold
							}
							return l.Layout(gtx)
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
					case ItemTypeNumber:
						return i.editorLayout(gtx, theme)
					case ItemTypeFile:
						return i.fileLayout(gtx, theme)
					case ItemTypeDropDown:
						return i.dropDownLayout(gtx, theme)
					default:
						return layout.Dimensions{}
					}
				})
			}),
		)
	})
}

func (i *SettingItem) fileLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if i.FileSelector.Changed() {
		i.onChange()
	}

	return i.FileSelector.Layout(gtx, theme)
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
	if i.minWidth > 0 {
		gtx.Constraints.Min.X = gtx.Dp(i.minWidth)
	} else {
		gtx.Constraints.Min.X = gtx.Dp(50)
	}

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
			editor.SelectionColor = theme.TextSelectionColor
			editor.TextSize = unit.Sp(14)
			return editor.Layout(gtx)
		})
	})
}

func (i *SettingItem) dropDownLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if i.dropDown.Changed() {
		i.onChange()
	}

	return i.dropDown.Layout(gtx, theme)
}

func (s *Settings) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	// update visibility
	values := s.getValues()
	for _, i := range s.Items {
		if i.visibleWhen != nil {
			i.visible = i.visibleWhen(values)
		}
	}

	inset := layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(15), Right: unit.Dp(20)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		l := material.List(theme.Material(), s.list)
		l.AnchorStrategy = material.Occupy
		return l.Layout(gtx, len(s.Items), func(gtx layout.Context, i int) layout.Dimensions {
			return s.Items[i].Layout(gtx, theme)
		})
	})
}
