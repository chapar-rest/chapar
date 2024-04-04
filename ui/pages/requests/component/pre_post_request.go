package component

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/keys"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type PrePostRequest struct {
	dropDown *widgets.DropDown
	script   *widgets.CodeEditor

	dropDownItems []Option

	onScriptChanged   func(script string)
	onDropDownChanged func(selected string)

	setEnvForm          *SetEnvForm
	onSetEnvFormChanged func(item, from, fromKey string)
}

type SetEnvForm struct {
	// SetEnvForm
	targetEditor widget.Editor
	fromEditor   widget.Editor
	fromDropDown *widgets.DropDown
	preview      string
}

const (
	TypeScript      = "script"
	TypeSetEnv      = "set_env"
	TypeShellScript = "shell_script"
	TypeK8sTunnel   = "kubectl_tunnel"
	TypeSSHTunnel   = "ssh_tunnel"
)

type Option struct {
	Title string
	Value string
	Type  string
	Hint  string
}

func NewPrePostRequest(options []Option) *PrePostRequest {
	p := &PrePostRequest{
		dropDown:      widgets.NewDropDown(),
		script:        widgets.NewCodeEditor("", "Python"),
		dropDownItems: options,
		setEnvForm: &SetEnvForm{
			fromDropDown: widgets.NewDropDown(
				widgets.NewDropDownOption("From Response").WithValue(domain.PostRequestSetFromResponseBody),
				widgets.NewDropDownOption("From Header").WithValue(domain.PostRequestSetFromResponseHeader),
				widgets.NewDropDownOption("From Cookie").WithValue(domain.PostRequestSetFromResponseCookie),
			),
		},
	}

	opts := make([]*widgets.DropDownOption, 0, len(options))
	for _, o := range options {
		opts = append(opts, widgets.NewDropDownOption(o.Title).WithValue(o.Value))
	}

	p.dropDown.SetOptions(opts...)
	return p
}

func (p *PrePostRequest) SetOnScriptChanged(f func(script string)) {
	p.onScriptChanged = f
	p.script.SetOnChanged(p.onScriptChanged)
}

func (p *PrePostRequest) SetOnDropDownChanged(f func(selected string)) {
	p.onDropDownChanged = f
	p.dropDown.SetOnChanged(p.onDropDownChanged)
}

func (p *PrePostRequest) SetSelectedDropDown(selected string) {
	p.dropDown.SetSelectedByValue(selected)
}

func (p *PrePostRequest) SetCode(code string) {
	p.script.SetCode(code)
}

func (p *PrePostRequest) SetPreview(preview string) {
	p.setEnvForm.preview = preview
}

func (p *PrePostRequest) SetPostRequestSetValues(set domain.PostRequestSet) {
	p.setEnvForm.targetEditor.SetText(set.Target)
	p.setEnvForm.fromEditor.SetText(set.FromKey)
	p.setEnvForm.fromDropDown.SetSelectedByValue(set.From)
}

func (p *PrePostRequest) SetOnPostRequestSetChanged(f func(item, from, fromKey string)) {
	p.onSetEnvFormChanged = f
	p.setEnvForm.fromDropDown.SetOnChanged(func(selected string) {
		p.onSetEnvFormChanged(p.setEnvForm.targetEditor.Text(), selected, p.setEnvForm.fromEditor.Text())
	})
}

func (p *PrePostRequest) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(15), Right: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return p.dropDown.Layout(gtx, theme)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				selectedIndex := p.dropDown.SelectedIndex()
				selectedItem := p.dropDownItems[selectedIndex]

				switch selectedItem.Type {
				case TypeScript:
					return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return p.script.Layout(gtx, theme, selectedItem.Hint)
					})

				case TypeSetEnv:
					return p.SetEnvForm(gtx, theme)
				}
				return layout.Dimensions{}
			}),
		)
	})
}

func (p *PrePostRequest) SetEnvForm(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	topButtonInset := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(4)}

	keys.OnEditorChange(gtx, &p.setEnvForm.targetEditor, func() {
		if p.onSetEnvFormChanged != nil {
			p.onSetEnvFormChanged(p.setEnvForm.targetEditor.Text(), p.setEnvForm.fromDropDown.GetSelected().Value, p.setEnvForm.fromEditor.Text())
		}
	})

	keys.OnEditorChange(gtx, &p.setEnvForm.fromEditor, func() {
		if p.onSetEnvFormChanged != nil {
			p.onSetEnvFormChanged(p.setEnvForm.targetEditor.Text(), p.setEnvForm.fromDropDown.GetSelected().Value, p.setEnvForm.fromEditor.Text())
		}
	})

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lb := &widgets.LabeledInput{
					Label:          "Target Key",
					SpaceBetween:   5,
					MinEditorWidth: unit.Dp(150),
					MinLabelWidth:  unit.Dp(80),
					Editor:         &p.setEnvForm.targetEditor,
				}
				return lb.Layout(gtx, theme)
			})
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(85)
					return material.Label(theme, theme.TextSize, "From").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(170))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(165))
					return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return p.setEnvForm.fromDropDown.Layout(gtx, theme)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lb := &widgets.LabeledInput{
					Label:          "From",
					SpaceBetween:   5,
					MinEditorWidth: unit.Dp(150),
					MinLabelWidth:  unit.Dp(80),
					Editor:         &p.setEnvForm.fromEditor,
				}
				return lb.Layout(gtx, theme)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(85)
					return material.Label(theme, theme.TextSize, "Preview").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme, theme.TextSize, p.setEnvForm.preview).Layout(gtx)
				}),
			)
		}),
	)
}
