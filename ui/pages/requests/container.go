package requests

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
)

type Container interface {
	Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions
	IsDataChanged() bool
	SetOnTitleChanged(f func(string, string))
	OnClose() bool
}
