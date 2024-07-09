package component

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type FormData struct {
	theme *chapartheme.Theme

	Fields    []*FormDataField
	addButton *widgets.IconButton

	list *widget.List

	onSelectFile func(id string)
	onChanged    func(values []domain.FormField)
}

type FormDataFieldType string

const (
	FormDataFieldTypeText FormDataFieldType = "text"
	FormDataFieldTypeFile FormDataFieldType = "file"
)

type FormDataField struct {
	Type       string
	Identifier string
	Key        string
	Value      string
	Files      []string
	Enable     bool

	typeDropDown *widgets.DropDown
	keyEditor    *widget.Editor
	valueEditor  *widget.Editor
	badgeInput   *widgets.BadgeInput

	activeBool   *widget.Bool
	deleteButton widget.Clickable
	uploadButton widget.Clickable
}

func NewFormData(theme *chapartheme.Theme, fields ...*FormDataField) *FormData {
	f := &FormData{
		theme: theme,
		addButton: &widgets.IconButton{
			Icon:      widgets.PlusIcon,
			Size:      unit.Dp(20),
			Clickable: &widget.Clickable{},
		},
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}

	for _, field := range fields {
		f.addField(field)
	}

	f.addButton.OnClick = func() {
		f.addField(NewFormDataField(FormDataFieldTypeText, "", "", nil))
		if f.onChanged != nil {
			f.onChanged(f.GetValues())
		}
	}

	return f
}

func (f *FormData) SetOnChanged(fn func(values []domain.FormField)) {
	f.onChanged = fn
}

func (f *FormData) SetOnSelectFile(fn func(id string)) {
	f.onSelectFile = fn
}

func (f *FormData) AddFile(id string, file string) {
	for _, field := range f.Fields {
		if field.Identifier == id {
			field.Files = append(field.Files, file)
			field.badgeInput.AddItem(file, getFileName(file))
		}
	}

	if f.onChanged != nil {
		f.onChanged(f.GetValues())
	}
}

func (f *FormData) GetValues() []domain.FormField {
	values := make([]domain.FormField, 0, len(f.Fields))
	for _, field := range f.Fields {
		values = append(values, domain.FormField{
			ID:     field.Identifier,
			Type:   field.Type,
			Key:    field.Key,
			Value:  field.Value,
			Files:  field.Files,
			Enable: field.Enable,
		})
	}
	return values
}

func (f *FormData) SetValues(values []domain.FormField) {
	f.Fields = make([]*FormDataField, 0, len(values))
	for _, field := range values {
		f.addField(&FormDataField{
			Identifier: field.ID,
			Type:       field.Type,
			Key:        field.Key,
			Value:      field.Value,
			Files:      field.Files,
			Enable:     field.Enable,
		})
	}
}

func (f *FormData) addField(field *FormDataField) {
	field.typeDropDown = widgets.NewDropDownWithoutBorder(
		f.theme,
		widgets.NewDropDownOption("File").WithIdentifier("file").WithValue("file"),
		widgets.NewDropDownOption("Text").WithIdentifier("text").WithValue("text"),
	)
	field.typeDropDown.SetSelectedByValue(field.Type)
	field.typeDropDown.MaxWidth = unit.Dp(60)

	field.keyEditor = &widget.Editor{SingleLine: true}
	field.keyEditor.SetText(field.Key)

	field.valueEditor = &widget.Editor{SingleLine: true}
	field.valueEditor.SetText(field.Value)

	field.badgeInput = widgets.NewBadgeInput()
	for _, file := range field.Files {
		field.badgeInput.AddItem(file, getFileName(file))
	}

	field.badgeInput.SetOnChange(func(values map[string]string) {
		field.Files = make([]string, 0, len(values))
		for file := range values {
			field.Files = append(field.Files, file)
		}

		f.triggerChanged()
	})

	field.activeBool = new(widget.Bool)
	field.activeBool.Value = field.Enable

	field.typeDropDown.SetOnChanged(func(selected string) {
		field.Type = selected

		f.triggerChanged()
	})

	f.Fields = append(f.Fields, field)
}

func getFileName(filePath string) string {
	if filePath == "" {
		return ""
	}

	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' {
			return filePath[i+1:]
		}
	}

	return filePath
}

func NewFormDataField(t FormDataFieldType, key, value string, files []string) *FormDataField {
	return &FormDataField{
		Identifier: uuid.NewString(),
		Type:       string(t),
		Key:        key,
		Value:      value,
		Files:      files,
	}
}

func (f *FormData) itemLayout(gtx layout.Context, theme *chapartheme.Theme, item *FormDataField) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, f.fieldLayouts(gtx, theme, item)...)
}

func (f *FormData) triggerChanged() {
	if f.onChanged != nil {
		f.onChanged(f.GetValues())
	}
}

func (f *FormData) fieldLayouts(gtx layout.Context, theme *chapartheme.Theme, item *FormDataField) []layout.FlexChild {
	keys.OnEditorChange(gtx, item.keyEditor, func() {
		item.Key = item.keyEditor.Text()
		f.triggerChanged()
	})

	keys.OnEditorChange(gtx, item.valueEditor, func() {
		item.Value = item.valueEditor.Text()
		f.triggerChanged()
	})

	if item.activeBool.Update(gtx) {
		item.Enable = item.activeBool.Value
		f.triggerChanged()
	}

	items := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			ch := material.CheckBox(theme.Material(), item.activeBool, "")
			ch.IconColor = theme.CheckBoxColor
			return ch.Layout(gtx)
		}),
		widgets.DrawLineFlex(theme.TableBorderColor, unit.Dp(35), unit.Dp(1)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return item.typeDropDown.Layout(gtx, theme)
		}),
		widgets.DrawLineFlex(theme.TableBorderColor, unit.Dp(35), unit.Dp(1)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(100))
				return material.Editor(theme.Material(), item.keyEditor, "Key").Layout(gtx)
			})
		}),
		widgets.DrawLineFlex(theme.TableBorderColor, unit.Dp(35), unit.Dp(1)),
	}

	itemType := item.typeDropDown.GetSelected().Identifier
	if itemType == string(FormDataFieldTypeText) {
		items = append(items, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(theme.Material(), item.valueEditor, "Value").Layout(gtx)
			})
		}))
	}

	if itemType == string(FormDataFieldTypeFile) {
		items = append(items, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return item.badgeInput.Layout(gtx, theme)
		}))
	}

	items = append(items, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if itemType == string(FormDataFieldTypeFile) {
					ib := widgets.IconButton{
						Icon:      widgets.UploadIcon,
						Size:      unit.Dp(20),
						Color:     theme.TextColor,
						Clickable: &item.uploadButton,
					}
					return ib.Layout(gtx, theme)
				}
				return layout.Dimensions{}
			}),
			widgets.DrawLineFlex(theme.TableBorderColor, unit.Dp(35), unit.Dp(1)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ib := widgets.IconButton{
					Icon:      widgets.DeleteIcon,
					Size:      unit.Dp(20),
					Color:     theme.TextColor,
					Clickable: &item.deleteButton,
				}
				return ib.Layout(gtx, theme)
			}),
		)
	}))

	return items
}

func (f *FormData) layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        theme.TableBorderColor,
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(1),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if len(f.Fields) == 0 {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, material.Label(theme.Material(), unit.Sp(14), "No items").Layout)
		}

		return material.List(theme.Material(), f.list).Layout(gtx, len(f.Fields), func(gtx layout.Context, i int) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return f.itemLayout(gtx, theme, f.Fields[i])
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// only if it's not the last item
					if i == len(f.Fields)-1 {
						return layout.Dimensions{}
					}
					return widgets.DrawLine(gtx, theme.TableBorderColor, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X))
				}),
			)
		})
	})
}

func (f *FormData) Layout(gtx layout.Context, title, hint string, theme *chapartheme.Theme) layout.Dimensions {
	for i, field := range f.Fields {
		if field.deleteButton.Clicked(gtx) {
			f.Fields = append(f.Fields[:i], f.Fields[i+1:]...)
			f.triggerChanged()
		}

		if field.uploadButton.Clicked(gtx) {
			if f.onSelectFile != nil {
				go f.onSelectFile(field.Identifier)
			}
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme.Material(), theme.TextSize, title).Layout(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Left: unit.Dp(10),
						//	Right: unit.Dp(10),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme.Material(), unit.Sp(10), hint).Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top:    0,
						Bottom: unit.Dp(10),
						Left:   0,
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						f.addButton.BackgroundColor = theme.Palette.Bg
						f.addButton.Color = theme.TextColor
						return f.addButton.Layout(gtx, theme)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return f.layout(gtx, theme)
		}),
	)
}
