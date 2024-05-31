package grpc

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type ServerInfo struct {
	definitionFrom *widget.Enum

	ReloadButton *widget.Clickable
	FileSelector *component.FileSelector
}

func NewServerInfo() *ServerInfo {
	s := &ServerInfo{
		definitionFrom: new(widget.Enum),
		FileSelector:   component.NewFileSelector(""),
		ReloadButton:   new(widget.Clickable),
	}

	s.definitionFrom.Value = "reflection"
	return s
}

func (s *ServerInfo) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme.Material(), unit.Sp(14), "Server definition from:").Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				r := widgets.RadioButton(theme.Material(), s.definitionFrom, "reflection", "Server reflection")
				r.IconColor = theme.CheckBoxColor
				return r.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				r := widgets.RadioButton(theme.Material(), s.definitionFrom, "proto_files", "Proto files")
				r.IconColor = theme.CheckBoxColor
				return r.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if s.definitionFrom.Value == "proto_files" {
					return s.FileSelector.Layout(gtx, theme)
				}

				btn := material.Button(theme.Material(), s.ReloadButton, "Load from server reflection")
				btn.Background = theme.SendButtonBgColor
				btn.Color = theme.ButtonTextColor
				return btn.Layout(gtx)
			}),
		)
	})
}
