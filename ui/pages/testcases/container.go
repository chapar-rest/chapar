package testcases

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	giox "gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
	"github.com/chapar-rest/chapar/ui/widgets/codeeditor"
)

type container struct {
	*ui.Base

	testCase *domain.TestCase
	yaml     string

	Breadcrumb *component.Breadcrumb
	Actions    *component.Actions

	codeEditor *codeeditor.CodeEditor

	saveButton   widget.Clickable
	runButton    widget.Clickable
	cancelButton widget.Clickable

	// Split view for editor and results
	split        widgets.SplitView
	resultsPanel *ResultsPanel

	// Dirty state tracking
	DataChanged bool

	// Prompt modal for confirmations
	Prompt *widgets.Prompt

	onSave         func(id string)
	onRun          func(id string)
	onTitleChanged func(id, title string)
	onYAMLChanged  func(id, yaml string)
}

func newContainer(base *ui.Base, testCase *domain.TestCase, yaml string) *container {
	c := &container{
		Base:         base,
		testCase:     testCase,
		yaml:         yaml,
		Breadcrumb:   component.NewBreadcrumb(testCase.MetaData.ID, "", "", testCase.MetaData.Name),
		Actions:      component.NewActions(true),
		codeEditor:   codeeditor.NewCodeEditor(yaml, codeeditor.CodeLanguageYAML, base.Theme),
		resultsPanel: NewResultsPanel(base.Theme),
		Prompt:       widgets.NewPrompt("", "", ""),
		split: widgets.SplitView{
			Resize: giox.Resize{
				Ratio: 0.65, // 65% for editor, 35% for results
				Axis:  layout.Horizontal,
			},
			BarWidth: unit.Dp(2),
		},
	}

	c.setupHooks()

	return c
}

func (c *container) setupHooks() {
	c.codeEditor.SetOnChanged(func(text string) {
		c.yaml = text
		if c.onYAMLChanged != nil {
			c.onYAMLChanged(c.testCase.MetaData.ID, text)
		}
	})

	c.Prompt.SetOnSubmit(func(selectedOption string, remember bool) {
		if c.onSave != nil {
			c.onSave(c.testCase.MetaData.ID)
		}
	})

	prefs.AddGlobalConfigChangeListener(func(old, updated domain.GlobalConfig) {
		isChanged := old.Spec.General.UseHorizontalSplit != updated.Spec.General.UseHorizontalSplit
		if isChanged {
			if updated.Spec.General.UseHorizontalSplit {
				c.split.Axis = layout.Horizontal
			} else {
				c.split.Axis = layout.Vertical
			}
		}
	})
}

func (c *container) SetTitle(title string) {
	c.Breadcrumb.SetTitle(title)
}

func (c *container) SetOnSave(onSave func(id string)) {
	c.onSave = onSave
}

func (c *container) SetOnRun(onRun func(id string)) {
	c.onRun = onRun
}

func (c *container) SetOnTitleChanged(onTitleChanged func(id, title string)) {
	c.Breadcrumb.SetOnTitleChanged(func(title string) {
		if c.onTitleChanged != nil {
			c.onTitleChanged(c.testCase.MetaData.ID, title)
		}
	})
}

func (c *container) SetOnYAMLChanged(onYAMLChanged func(id, yaml string)) {
	c.onYAMLChanged = onYAMLChanged
}

func (c *container) GetYAML() string {
	return c.yaml
}

func (c *container) UpdateResult(result *domain.TestResult) {
	c.resultsPanel.UpdateResult(result)
}

func (c *container) ClearResult() {
	c.resultsPanel.ClearResult()
}

func (c *container) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	if c.saveButton.Clicked(gtx) {
		if c.onSave != nil {
			c.onSave(c.testCase.MetaData.ID)
		}
	}

	if c.runButton.Clicked(gtx) {
		if c.onRun != nil {
			c.onRun(c.testCase.MetaData.ID)
		}
	}

	dims := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return c.Prompt.Layout(gtx, th)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(15), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return c.Breadcrumb.Layout(gtx, th)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								border := widget.Border{
									Color:        th.Palette.ContrastFg,
									Width:        unit.Dp(1),
									CornerRadius: unit.Dp(4),
								}

								bt := widgets.Button(th.Material(), &c.saveButton, widgets.SaveIcon, widgets.IconPositionStart, "Save")
								if c.DataChanged {
									bt.Color = th.Palette.ContrastFg
									border.Width = unit.Dp(1)
									border.Color = th.Palette.ContrastFg
								} else {
									bt.Color = widgets.Disabled(th.Palette.ContrastFg)
									border.Color = widgets.Disabled(th.Palette.ContrastFg)
									border.Width = 0
								}

								return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return bt.Layout(gtx, th)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return widgets.Button(th.Material(), &c.runButton, widgets.PlayIcon, widgets.IconPositionStart, "Run Test").Layout(gtx, th)
							}),
						)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return c.split.Layout(gtx, th,
					func(gtx layout.Context) layout.Dimensions {
						return c.layoutEditor(gtx, th)
					},
					func(gtx layout.Context) layout.Dimensions {
						return c.layoutResults(gtx, th)
					},
				)
			}),
		)
	})

	return dims
}

func (c *container) layoutEditor(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Bottom: unit.Dp(5),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(th.Material(), th.TextSize, "Edit the YAML content below:")
				label.Color = th.Fg
				return label.Layout(gtx)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Right: unit.Dp(5),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return c.codeEditor.Layout(gtx, th, "")
			})
		}),
	)
}

func (c *container) layoutResults(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	if !c.resultsPanel.HasResult() {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			label := material.Label(th.Material(), th.TextSize, "No test results yet. Click 'Run Test' to execute.")
			label.Color = th.Fg
			return label.Layout(gtx)
		})
	}

	return layout.Inset{
		Top:   unit.Dp(10),
		Left:  unit.Dp(10),
		Right: unit.Dp(10),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return c.resultsPanel.Layout(gtx, th)
	})
}
