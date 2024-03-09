package requests

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
)

type Container interface {
	Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions
	SetActiveEnvironment(env *domain.Environment)
	IsDataChanged() bool
	SetDirty(dirty bool)
	SetOnTitleChanged(f func(string, string))
	OnClose() bool
}
