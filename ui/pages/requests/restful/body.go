package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/converter"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Body struct {
	DropDown *widgets.DropDown

	body domain.Body

	FormData   *component.FormData
	urlencoded *widgets.KeyValue
	script     *widgets.CodeEditor
	BinaryFile *component.FileSelector

	onChange func(body domain.Body)
}

func NewBody(body domain.Body, theme *chapartheme.Theme) *Body {
	b := &Body{
		body: body,
		DropDown: widgets.NewDropDown(
			theme,
			widgets.NewDropDownOption("None").WithValue(domain.BodyTypeNone),
			widgets.NewDropDownOption("JSON").WithValue(domain.BodyTypeJSON),
			widgets.NewDropDownOption("Text").WithValue(domain.BodyTypeText),
			widgets.NewDropDownOption("XML").WithValue(domain.BodyTypeXML),
			widgets.NewDropDownOption("Form data").WithValue(domain.BodyTypeFormData),
			widgets.NewDropDownOption("Binary").WithValue(domain.BodyTypeBinary),
			widgets.NewDropDownOption("Urlencoded").WithValue(domain.BodyTypeUrlencoded),
		),
		FormData: component.NewFormData(theme),
		// formData:   widgets.NewKeyValue(),
		urlencoded: widgets.NewKeyValue(),
		script:     widgets.NewCodeEditor("", "JSON", theme),
		BinaryFile: component.NewFileSelector(""),
	}

	b.FormData.SetValues(body.FormData.Fields)

	if body.Type == domain.BodyTypeBinary {
		b.BinaryFile.SetFileName(body.BinaryFilePath)
	}

	b.script.SetCode(body.Data)
	b.DropDown.SetSelectedByValue(body.Type)
	b.DropDown.MaxWidth = unit.Dp(150)

	return b
}

func (b *Body) SetOnChange(f func(body domain.Body)) {
	b.onChange = f

	b.DropDown.SetOnChanged(func(selected string) {
		b.body.Type = selected
		b.onChange(b.body)

		if selected == domain.BodyTypeJSON || selected == domain.BodyTypeXML {
			b.script.SetLanguage(selected)
		}
	})

	b.script.SetOnChanged(func(script string) {
		b.body.Data = script
		b.onChange(b.body)
	})

	b.FormData.SetOnChanged(func(fields []domain.FormField) {
		b.body.FormData = domain.FormData{Fields: fields}
		b.onChange(b.body)
	})

	b.urlencoded.SetOnChanged(func(items []*widgets.KeyValueItem) {
		b.body.URLEncoded = converter.KeyValueFromWidgetItems(b.urlencoded.Items)
		b.onChange(b.body)
	})

	b.BinaryFile.SetOnChanged(func(filePath string) {
		b.body.BinaryFilePath = filePath
		b.onChange(b.body)
	})
}

func (b *Body) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(15), Right: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return b.DropDown.Layout(gtx, theme)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					switch b.DropDown.GetSelected().Value {
					case domain.BodyTypeJSON:
						return b.script.Layout(gtx, theme, "JSON")
					case domain.BodyTypeText:
						return b.script.Layout(gtx, theme, "Text")
					case domain.BodyTypeXML:
						return b.script.Layout(gtx, theme, "XML")
					case domain.BodyTypeFormData:
						return b.FormData.Layout(gtx, "Form data", "Add form data", theme)
					case domain.BodyTypeBinary:
						return b.BinaryFile.Layout(gtx, theme)
					case domain.BodyTypeUrlencoded:
						return b.urlencoded.WithAddLayout(gtx, "Urlencoded", "Add urlencoded", theme)
					default:
						return layout.Dimensions{}
					}
				})
			}),
		)
	})
}
