package websocket

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets/codeeditor"
)

type Body struct {
	Type string `yaml:"type"`
	Data string `yaml:"data"`

	Editor        *codeeditor.CodeEditor
	SendClickable widget.Clickable
}

func NewBody(bodyType, bodyData string, theme *chapartheme.Theme) *Body {
	b := &Body{
		Type:   bodyType,
		Data:   bodyData,
		Editor: codeeditor.NewCodeEditor(bodyData, bodyType, theme),
	}
	return b
}

func (b *Body) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return b.Editor.Layout(gtx, th, "message")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th.Material(), &b.SendClickable, "Send")
			btn.Background = th.ActionButtonBgColor
			btn.Color = th.ButtonTextColor
			return btn.Layout(gtx)
		}),
	)
}
