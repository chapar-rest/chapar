package grpc

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type ServerInfo struct {
	definitionFrom *widget.Enum

	ReloadButton *widget.Clickable
	FileSelector *component.FileSelector

	IsLoading bool
	onReload  func()

	onChanged func()
}

func NewServerInfo(info domain.ServerInfo) *ServerInfo {
	// For now, we only support one proto file, and we will use the last one
	fileName := ""
	if len(info.ProtoFiles) > 0 {
		fileName = info.ProtoFiles[len(info.ProtoFiles)-1]
	}

	s := &ServerInfo{
		definitionFrom: new(widget.Enum),
		FileSelector:   component.NewFileSelector(fileName),
		ReloadButton:   new(widget.Clickable),
		IsLoading:      false,
	}

	if info.ServerReflection {
		s.definitionFrom.Value = "reflection"
	} else {
		s.definitionFrom.Value = "proto_files"
	}

	return s
}

func (s *ServerInfo) SetOnChanged(f func()) {
	s.onChanged = f
}

func (s *ServerInfo) SetOnReload(f func()) {
	s.onReload = f

	s.FileSelector.SetOnRefresh(f)
}

func (s *ServerInfo) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if s.definitionFrom.Update(gtx) {
		if s.onChanged != nil {
			s.onChanged()
		}
	}

	inset := layout.Inset{Top: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme.Material(), unit.Sp(14), "Server definition from:").Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						r := widgets.RadioButton(theme.Material(), s.definitionFrom, "reflection", "Server reflection")
						r.IconColor = theme.CheckBoxColor
						return r.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if s.definitionFrom.Value != "reflection" {
							return layout.Dimensions{}
						}

						btn := widgets.Button(theme.Material(), s.ReloadButton, widgets.RefreshIcon, widgets.IconPositionStart, "Reload")
						btn.Color = theme.ButtonTextColor
						btn.Inset = layout.Inset{
							Top: 4, Bottom: 4,
							Left: 4, Right: 4,
						}

						if s.ReloadButton.Clicked(gtx) {
							if s.onReload != nil {
								s.onReload()
							}
						}

						return btn.Layout(gtx, theme)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
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

				return layout.Dimensions{}
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if s.definitionFrom.Value == "proto_files" {
					return material.Label(theme.Material(), unit.Sp(13), "If your schema requires additional proto files as dependencies, you can add them in the Proto files tab.").Layout(gtx)
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if s.IsLoading {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Loader(theme.Material())
						l.Color = theme.LoaderColor
						return l.Layout(gtx)
					})
				}
				return layout.Dimensions{}
			}),
		)
	})

}
