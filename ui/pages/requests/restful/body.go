package restful

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Body struct {
	DropDown *widgets.DropDown

	body *domain.Body

	values *widgets.KeyValue
	script *widgets.CodeEditor
}

func NewBody(body *domain.Body) *Body {
	if body == nil {
		body = &domain.Body{}
	}

	b := &Body{
		body: body,
		DropDown: widgets.NewDropDown(
			widgets.NewDropDownOption("None"),
			widgets.NewDropDownOption("JSON"),
			widgets.NewDropDownOption("Text"),
			widgets.NewDropDownOption("XML"),
			widgets.NewDropDownOption("Form data"),
			widgets.NewDropDownOption("Binary"),
			widgets.NewDropDownOption("Urlencoded"),
		),
		values: widgets.NewKeyValue(),
		script: widgets.NewCodeEditor(body.Data),
	}

	b.DropDown.SetSelectedByValue(body.Type)
	return b
}

func (b *Body) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
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
					switch b.DropDown.GetSelected().Text {
					case "JSON":
						return b.script.Layout(gtx, theme, "JSON")
					case "Text":
						return b.script.Layout(gtx, theme, "Text")
					case "XML":
						return b.script.Layout(gtx, theme, "XML")
					case "Form data":
						return b.values.WithAddLayout(gtx, "Form data", "Add form data", theme)
					case "Binary":
						return b.script.Layout(gtx, theme, "Binary")
					case "Urlencoded":
						return b.values.WithAddLayout(gtx, "Urlencoded", "Add urlencoded", theme)
					default:
						return layout.Dimensions{}
					}
				})
			}),
		)
	})
}
