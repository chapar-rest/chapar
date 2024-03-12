package component

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type PrePostRequest struct {
	dropDown *widgets.DropDown
	script   *widgets.CodeEditor

	dropDownItems []Option

	onScriptChanged   func(script string)
	onDropDownChanged func(selected string)
}

type Option struct {
	Text     string
	IsScript bool
	Hint     string
}

func NewPrePostRequest(options []Option) *PrePostRequest {
	p := &PrePostRequest{
		dropDown:      widgets.NewDropDown(),
		script:        widgets.NewCodeEditor(""),
		dropDownItems: options,
	}

	opts := make([]*widgets.DropDownOption, 0, len(options))
	for _, o := range options {
		opts = append(opts, widgets.NewDropDownOption(o.Text))
	}

	p.dropDown.SetOptions(opts...)
	return p
}

func (p *PrePostRequest) SetOnScriptChanged(f func(script string)) {
	p.onScriptChanged = f
}

func (p *PrePostRequest) SetOnDropDownChanged(f func(selected string)) {
	p.onDropDownChanged = f
}

func (p *PrePostRequest) SetCode(code string) {
	p.script.SetCode(code)
}

func (p *PrePostRequest) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return p.dropDown.Layout(gtx, theme)
					}),
				)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			selectedIndex := p.dropDown.SelectedIndex()
			selectedItem := p.dropDownItems[selectedIndex]

			if selectedItem.IsScript {
				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return p.script.Layout(gtx, theme, selectedItem.Hint)
				})
			}
			return layout.Dimensions{}
		}),
	)
}
