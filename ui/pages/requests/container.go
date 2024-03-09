package requests

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
)

const (
	ContainerTypeRequest    = "request"
	ContainerTypeCollection = "collection"
)

type Container interface {
	Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions
	SetActiveEnvironment(env *domain.Environment)
	IsDataChanged() bool
	SetDirty(dirty bool)
	SetOnTitleChanged(f func(string, string, string))
	OnClose() bool
}
