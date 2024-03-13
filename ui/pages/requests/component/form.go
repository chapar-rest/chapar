package component

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/keys"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Form struct {
	Fields []*Field

	onChange func(values map[string]string)
}

type Field struct {
	Label string
	Value string

	Editor *widget.Editor
}

func NewForm(fields []*Field) *Form {
	return &Form{
		Fields: fields,
	}
}

func (f *Form) SetOnChange(onChange func(values map[string]string)) {
	f.onChange = onChange
}

func (f *Form) GetValues() map[string]string {
	values := make(map[string]string)
	for _, field := range f.Fields {
		values[field.Label] = field.Editor.Text()
	}
	return values
}

func (f *Form) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	childs := make([]layout.FlexChild, 0)
	for _, field := range f.Fields {
		field := field
		childs = append(childs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if field.Editor == nil {
				field.Editor = new(widget.Editor)
			}

			keys.OnEditorChange(gtx, field.Editor, func() {
				if f.onChange != nil {
					f.onChange(f.GetValues())
				}
			})

			lb := &widgets.LabeledInput{
				Label:          field.Label,
				SpaceBetween:   5,
				MinEditorWidth: unit.Dp(150),
				MinLabelWidth:  unit.Dp(80),
				Editor:         field.Editor,
			}
			return lb.Layout(gtx, theme)
		}), layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout))
	}

	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx, childs...)
}
