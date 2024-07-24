package component

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Form struct {
	Fields []*Field

	onChange func(values map[string]string)
}

type Field struct {
	Label string
	Value string

	Editor *widgets.PatternEditor
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
		if field.Editor == nil {
			field.Editor = widgets.NewPatternEditor()
			field.Editor.SetOnChanged(f.onDataChange)
		}

		values[field.Label] = field.Editor.Text()
	}
	return values
}

func (f *Form) onDataChange(_ string) {
	if f.onChange != nil {
		f.onChange(f.GetValues())
	}
}

func (f *Form) SetValues(values map[string]string) {
	for _, field := range f.Fields {
		if field.Editor == nil {
			field.Editor = widgets.NewPatternEditor()
			field.Editor.SetOnChanged(f.onDataChange)
		}

		field.Editor.SetText(values[field.Label])
	}
}

func (f *Form) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	childs := make([]layout.FlexChild, 0)
	for _, field := range f.Fields {
		field := field
		childs = append(childs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if field.Editor == nil {
				field.Editor = widgets.NewPatternEditor()
				field.Editor.SetOnChanged(f.onDataChange)
			}

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
