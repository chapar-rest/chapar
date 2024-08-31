package component

import (
	"strconv"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type PrePostRequest struct {
	dropDown *widgets.DropDown
	script   *widgets.CodeEditor

	dropDownItems []Option

	onScriptChanged   func(script string)
	onDropDownChanged func(selected string)

	triggerRequestForm     *TriggerRequestForm
	onTriggerRequestChange func(collectionID, requestID string)

	setEnvForm          *SetEnvForm
	onSetEnvFormChanged func(statusCode int, item, from, fromKey string)
}

type SetEnvForm struct {
	statusCodeEditor *widgets.LabeledInput
	targetEditor     *widgets.LabeledInput
	fromEditor       *widgets.LabeledInput
	fromDropDown     *widgets.DropDown
	preview          string
}

type TriggerRequestForm struct {
	collectionsDropDown *widgets.DropDown
	requestDropDown     *widgets.DropDown
}

const (
	TypeScript         = "script"
	TypeSetEnv         = "set_env"
	TypeShellScript    = "shell_script"
	TypeK8sTunnel      = "kubectl_tunnel"
	TypeSSHTunnel      = "ssh_tunnel"
	TypeTriggerRequest = "trigger_request"
)

type Option struct {
	Title string
	Value string
	Type  string
	Hint  string
}

func NewPrePostRequest(options []Option, theme *chapartheme.Theme) *PrePostRequest {
	p := &PrePostRequest{
		dropDown:      widgets.NewDropDown(theme),
		script:        widgets.NewCodeEditor("", widgets.CodeLanguagePython, theme),
		dropDownItems: options,
		setEnvForm: &SetEnvForm{
			fromDropDown: widgets.NewDropDown(
				theme,
				widgets.NewDropDownOption("From Response").WithValue(domain.PostRequestSetFromResponseBody),
				widgets.NewDropDownOption("From Header").WithValue(domain.PostRequestSetFromResponseHeader),
				widgets.NewDropDownOption("From Cookie").WithValue(domain.PostRequestSetFromResponseCookie),
			),
			statusCodeEditor: &widgets.LabeledInput{
				Label:          "Status Code",
				SpaceBetween:   5,
				MinEditorWidth: unit.Dp(150),
				MinLabelWidth:  unit.Dp(80),
				Editor:         widgets.NewPatternEditor(),
			},
			targetEditor: &widgets.LabeledInput{
				Label:          "Target Key",
				SpaceBetween:   5,
				MinEditorWidth: unit.Dp(150),
				MinLabelWidth:  unit.Dp(80),
				Editor:         widgets.NewPatternEditor(),
			},
			fromEditor: &widgets.LabeledInput{
				Label:          "Key",
				SpaceBetween:   5,
				MinEditorWidth: unit.Dp(150),
				MinLabelWidth:  unit.Dp(80),
				Editor:         widgets.NewPatternEditor(),
				Hint:           "e.g. name",
			},
		},
		triggerRequestForm: &TriggerRequestForm{
			collectionsDropDown: widgets.NewDropDown(theme),
			requestDropDown:     widgets.NewDropDown(theme),
		},
	}
	p.setEnvForm.fromDropDown.MaxWidth = unit.Dp(150)
	p.triggerRequestForm.requestDropDown.MaxWidth = unit.Dp(150)
	p.triggerRequestForm.collectionsDropDown.MaxWidth = unit.Dp(150)

	opts := make([]*widgets.DropDownOption, 0, len(options))
	for _, o := range options {
		opts = append(opts, widgets.NewDropDownOption(o.Title).WithValue(o.Value))
	}

	p.dropDown.SetOptions(opts...)
	p.dropDown.MaxWidth = unit.Dp(200)
	return p
}

func (p *PrePostRequest) SetCollections(collections []*domain.Collection, selectedID string) {
	opts := make([]*widgets.DropDownOption, 0, len(collections)+1)
	opts = append(opts, widgets.NewDropDownOption("Select Collection").WithValue("none"))
	for _, c := range collections {
		opts = append(opts, widgets.NewDropDownOption(c.MetaData.Name).WithValue(c.MetaData.ID))
	}
	p.triggerRequestForm.collectionsDropDown.SetOptions(opts...)
	p.triggerRequestForm.collectionsDropDown.SetSelectedByValue(selectedID)
}

func (p *PrePostRequest) SetRequests(requests []*domain.Request, selectedID string) {
	opts := make([]*widgets.DropDownOption, 0, len(requests)+1)
	opts = append(opts, widgets.NewDropDownOption("Select Request").WithValue("none"))
	for _, r := range requests {
		opts = append(opts, widgets.NewDropDownOption(r.MetaData.Name).WithValue(r.MetaData.ID))
	}
	p.triggerRequestForm.requestDropDown.SetOptions(opts...)
	p.triggerRequestForm.requestDropDown.SetSelectedByValue(selectedID)
}

func (p *PrePostRequest) SetOnTriggerRequestChanged(f func(collectionID, requestID string)) {
	p.onTriggerRequestChange = f
	p.triggerRequestForm.collectionsDropDown.SetOnChanged(func(selected string) {
		p.onTriggerRequestChange(selected, p.triggerRequestForm.requestDropDown.GetSelected().GetValue())
	})

	p.triggerRequestForm.requestDropDown.SetOnChanged(func(selected string) {
		p.onTriggerRequestChange(p.triggerRequestForm.collectionsDropDown.GetSelected().GetValue(), selected)
	})
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
	p.setEnvForm.statusCodeEditor.SetText(strconv.Itoa(set.StatusCode))
	p.setEnvForm.targetEditor.SetText(set.Target)
	p.setEnvForm.fromEditor.SetText(set.FromKey)
	p.setEnvForm.fromDropDown.SetSelectedByValue(set.From)
}

func (p *PrePostRequest) SetOnPostRequestSetChanged(f func(statusCode int, item, from, fromKey string)) {
	p.onSetEnvFormChanged = f
	p.setEnvForm.fromDropDown.SetOnChanged(func(selected string) {
		statusCode, _ := strconv.Atoi(p.setEnvForm.statusCodeEditor.Text())
		p.onSetEnvFormChanged(statusCode, p.setEnvForm.targetEditor.Text(), selected, p.setEnvForm.fromEditor.Text())
	})
	p.setEnvForm.statusCodeEditor.SetOnChanged(func(text string) {
		p.enforceNumericEditor(p.setEnvForm.statusCodeEditor.Editor)
		p.handleDataChange()
	})
	p.setEnvForm.targetEditor.SetOnChanged(func(_ string) {
		p.handleDataChange()
	})
	p.setEnvForm.fromEditor.SetOnChanged(func(_ string) {
		p.handleDataChange()
	})
}

func (p *PrePostRequest) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
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
				case TypeTriggerRequest:
					return p.TriggerRequestForm(gtx, theme)
				}
				return layout.Dimensions{}
			}),
		)
	})
}

func (p *PrePostRequest) handleDataChange() {
	if p.onSetEnvFormChanged != nil {
		statusCode, _ := strconv.Atoi(p.setEnvForm.statusCodeEditor.Text())

		p.onSetEnvFormChanged(statusCode, p.setEnvForm.targetEditor.Text(), p.setEnvForm.fromDropDown.GetSelected().Value, p.setEnvForm.fromEditor.Text())
	}
}

func (p *PrePostRequest) enforceNumericEditor(editor *widgets.PatternEditor) {
	if _, err := strconv.Atoi(editor.Text()); err != nil {
		editor.SetText("0")
	}
}

func (p *PrePostRequest) SetEnvForm(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	topButtonInset := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(4)}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return p.setEnvForm.targetEditor.Layout(gtx, theme)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return p.setEnvForm.statusCodeEditor.Layout(gtx, theme)
			})
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(85)
					return material.Label(theme.Material(), theme.TextSize, "From").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					p.setEnvForm.fromDropDown.MinWidth = unit.Dp(162)
					return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return p.setEnvForm.fromDropDown.Layout(gtx, theme)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := "Key"
				hint := "e.g. name"
				if p.setEnvForm.fromDropDown.GetSelected().Value == domain.PostRequestSetFromResponseBody {
					label = "JSON Path"
					hint = "e.g. $.data[0].name"
				}

				p.setEnvForm.fromEditor.SetHint(hint)
				p.setEnvForm.fromEditor.SetLabel(label)
				return p.setEnvForm.fromEditor.Layout(gtx, theme)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme.Material(), theme.TextSize, "Preview:").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme.Material(), theme.TextSize, p.setEnvForm.preview).Layout(gtx)
			})
		}),
	)
}

func (p *PrePostRequest) TriggerRequestForm(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	topButtonInset := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(4)}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(85)
					return material.Label(theme.Material(), theme.TextSize, "Collection").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					p.triggerRequestForm.collectionsDropDown.MinWidth = unit.Dp(162)
					return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return p.triggerRequestForm.collectionsDropDown.Layout(gtx, theme)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(85)
					return material.Label(theme.Material(), theme.TextSize, "Request").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					p.triggerRequestForm.requestDropDown.MinWidth = unit.Dp(162)
					return topButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return p.triggerRequestForm.requestDropDown.Layout(gtx, theme)
					})
				}),
			)
		}),
	)
}
