package component

import (
	"io"
	"strings"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/codegen"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type CodeModal struct {
	req *domain.Request

	codeEditor  *widgets.CodeEditor
	CopyButton  widget.Clickable
	CloseButton widget.Clickable
	dropDown    *widgets.DropDown

	copyButtonText string

	updateCode bool
	visible    bool

	lang string
	code string
}

func NewCodeModal(theme *chapartheme.Theme) *CodeModal {
	c := &CodeModal{
		dropDown: widgets.NewDropDown(
			theme,
			widgets.NewDropDownOption("Curl").WithValue("curl"),
			widgets.NewDropDownOption("Python").WithValue("python"),
			widgets.NewDropDownOption("Golang").WithValue("golang"),
			widgets.NewDropDownOption("Axios").WithValue("axios"),
			widgets.NewDropDownOption("Node Fetch").WithValue("node-fetch"),
			widgets.NewDropDownOption("Java OkHTTP").WithValue("java-okhttp"),
			widgets.NewDropDownOption("Ruby Net").WithValue("ruby-net"),
			widgets.NewDropDownOption(".Net").WithValue("dot-net"),
		),
		codeEditor:     widgets.NewCodeEditor("", widgets.CodeLanguagePython, theme),
		copyButtonText: "Copy",
	}

	c.dropDown.MaxWidth = unit.Dp(200)

	c.dropDown.SetOnChanged(c.onLangSelected)

	return c
}

func (c *CodeModal) onLangSelected(lang string) {
	var code string
	switch lang {
	case "curl":
		c.lang = widgets.CodeLanguageShell
		code, _ = codegen.DefaultService.GenerateCurlCommand(c.req.Spec.HTTP)
	case "python":
		c.lang = widgets.CodeLanguagePython
		code, _ = codegen.DefaultService.GeneratePythonRequest(c.req.Spec.HTTP)
	case "golang":
		c.lang = widgets.CodeLanguageGolang
		code, _ = codegen.DefaultService.GenerateGoRequest(c.req.Spec.HTTP)
	case "axios":
		c.lang = widgets.CodeLanguageJavaScript
		code, _ = codegen.DefaultService.GenerateAxiosCommand(c.req.Spec.HTTP)
	case "node-fetch":
		c.lang = widgets.CodeLanguageJavaScript
		code, _ = codegen.DefaultService.GenerateFetchCommand(c.req.Spec.HTTP)
	case "java-okhttp":
		c.lang = widgets.CodeLanguageJava
		code, _ = codegen.DefaultService.GenerateJavaOkHttpCommand(c.req.Spec.HTTP)
	case "ruby-net":
		c.lang = widgets.CodeLanguageRuby
		code, _ = codegen.DefaultService.GenerateRubyNetHttpCommand(c.req.Spec.HTTP)
	}

	c.code = code
	c.updateCode = true
}

func (c *CodeModal) SetVisible(visible bool) {
	c.visible = visible
}

func (c *CodeModal) SetRequest(req *domain.Request) {
	c.req = req
}

func (c *CodeModal) layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        theme.TableBorderColor,
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(2),
	}

	if c.CopyButton.Clicked(gtx) {
		gtx.Execute(clipboard.WriteCmd{
			Data: io.NopCloser(strings.NewReader(c.code)),
		})
		c.copyButtonText = "Copied"

		// Start a goroutine to reset the button text after 900ms
		go func() {
			time.Sleep(900 * time.Millisecond)
			c.copyButtonText = "Copy"
			// Trigger a re-render
			gtx.Execute(op.InvalidateCmd{})
		}()
	}

	if c.CloseButton.Clicked(gtx) {
		c.visible = false
		c.code = ""
		c.codeEditor.SetCode("")
		c.req = nil
		c.dropDown.SetSelected(0)
	}

	return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(600)
			gtx.Constraints.Max.Y = gtx.Dp(600)

			return component.NewModalSheet(component.NewModal()).Layout(gtx, theme.Material(), &component.VisibilityAnimation{}, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return c.dropDown.Layout(gtx, theme)
										}),
										layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return widgets.Button(theme.Material(), &c.CopyButton, widgets.CopyIcon, widgets.IconPositionStart, c.copyButtonText).Layout(gtx, theme)
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return widgets.Button(theme.Material(), &c.CloseButton, widgets.CloseIcon, widgets.IconPositionStart, "Close").Layout(gtx, theme)
								}),
							)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if c.updateCode {
								c.codeEditor.SetLanguage(c.lang)
								c.codeEditor.SetCode(c.code)
								c.updateCode = false
							}

							return c.codeEditor.Layout(gtx, theme, "")
						}),
					)
				})
			})
		})
	})
}

func (c *CodeModal) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if !c.visible {
		return layout.Dimensions{}
	}

	// if code is empty, generate curl command
	if c.code == "" {
		c.onLangSelected("curl")
	}

	ops := op.Record(gtx.Ops)
	dims := c.layout(gtx, theme)
	defer op.Defer(gtx.Ops, ops.Stop())

	return dims
}
