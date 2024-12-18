package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/converter"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Body struct {
	DropDown *widgets.DropDown

	body domain.Body

	FormData   *component.FormData
	urlencoded *widgets.KeyValue
	script     *widgets.CodeEditor
	BinaryFile *widgets.FileSelector

	onChange func(body domain.Body)
}

func NewBody(body domain.Body, theme *chapartheme.Theme, explorer *explorer.Explorer) *Body {
	b := &Body{
		body: body,
		DropDown: widgets.NewDropDown(
			theme,
			widgets.NewDropDownOption("None").WithValue(domain.RequestBodyTypeNone),
			widgets.NewDropDownOption("JSON").WithValue(domain.RequestBodyTypeJSON),
			widgets.NewDropDownOption("Text").WithValue(domain.RequestBodyTypeText),
			widgets.NewDropDownOption("XML").WithValue(domain.RequestBodyTypeXML),
			widgets.NewDropDownOption("Form data").WithValue(domain.RequestBodyTypeFormData),
			widgets.NewDropDownOption("Binary").WithValue(domain.RequestBodyTypeBinary),
			widgets.NewDropDownOption("Urlencoded").WithValue(domain.RequestBodyTypeUrlencoded),
		),
		FormData:   component.NewFormData(theme),
		urlencoded: widgets.NewKeyValue(),
		script:     widgets.NewCodeEditor("", widgets.CodeLanguageJSON, theme),
		BinaryFile: widgets.NewFileSelector("", explorer),
	}

	b.FormData.SetValues(body.FormData.Fields)

	if body.Type == domain.RequestBodyTypeBinary {
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

		if selected == domain.RequestBodyTypeJSON || selected == domain.RequestBodyTypeXML {
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
					case domain.RequestBodyTypeJSON:
						return b.script.Layout(gtx, theme, "JSON")
					case domain.RequestBodyTypeText:
						return b.script.Layout(gtx, theme, "Text")
					case domain.RequestBodyTypeXML:
						return b.script.Layout(gtx, theme, "XML")
					case domain.RequestBodyTypeFormData:
						return b.FormData.Layout(gtx, "Form data", "Add form data", theme)
					case domain.RequestBodyTypeBinary:
						return b.BinaryFile.Layout(gtx, theme)
					case domain.RequestBodyTypeUrlencoded:
						return b.urlencoded.WithAddLayout(gtx, "Urlencoded", "Add urlencoded", theme)
					default:
						return layout.Dimensions{}
					}
				})
			}),
		)
	})
}
