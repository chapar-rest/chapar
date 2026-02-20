package testcases

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	giox "gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
	"github.com/chapar-rest/chapar/ui/widgets/codeeditor"
)

type container struct {
	*ui.Base

	testCase *domain.TestCase
	yaml     string

	titleEditor *widgets.EditableLabel
	codeEditor  *codeeditor.CodeEditor

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
		titleEditor:  widgets.NewEditableLabel(testCase.MetaData.Name),
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

	c.titleEditor.SetOnChanged(func(text string) {
		if c.onTitleChanged != nil {
			c.onTitleChanged(c.testCase.MetaData.ID, text)
		}
	})

	c.codeEditor.SetOnChanged(func(text string) {
		c.yaml = text
		if c.onYAMLChanged != nil {
			c.onYAMLChanged(c.testCase.MetaData.ID, text)
		}
	})

	return c
}

func (c *container) SetTitle(title string) {
	c.titleEditor.SetText(title)
}

func (c *container) SetOnSave(onSave func(id string)) {
	c.onSave = onSave
}

func (c *container) SetOnRun(onRun func(id string)) {
	c.onRun = onRun
}

func (c *container) SetOnTitleChanged(onTitleChanged func(id, title string)) {
	c.onTitleChanged = onTitleChanged
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

	dim := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Title section with save button indicator
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:    unit.Dp(10),
				Bottom: unit.Dp(10),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return c.titleEditor.Layout(gtx, th)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if c.DataChanged {
							return widgets.SaveButtonLayout(gtx, th, &c.saveButton)
						}
						return layout.Dimensions{}
					}),
				)
			})
		}),
		// Action buttons
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Bottom: unit.Dp(10),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !c.DataChanged {
							return layout.Dimensions{}
						}
						return widgets.Button(th.Material(), &c.saveButton, widgets.SaveIcon, widgets.IconPositionStart, "Save").Layout(gtx, th)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !c.DataChanged {
							return layout.Dimensions{}
						}
						return layout.Spacer{Width: unit.Dp(10)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return widgets.Button(th.Material(), &c.runButton, widgets.PlayIcon, widgets.IconPositionStart, "Run Test").Layout(gtx, th)
					}),
				)
			})
		}),
		// Split view for editor and results
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

	c.Prompt.Layout(gtx, th)
	return dim
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
			return c.codeEditor.Layout(gtx, th, "")
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
