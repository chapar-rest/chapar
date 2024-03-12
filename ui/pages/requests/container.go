package requests

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
)

const (
	TypeRequest    = "request"
	TypeCollection = "collection"

	TypeMeta = "Type"
)

type Container interface {
	Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions
	SetActiveEnvironment(env *domain.Environment)
	IsDataChanged() bool
	SetDirty(dirty bool)
	SetOnTitleChanged(f func(title string))
	OnClose() bool
	ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...string)
	HidePrompt()
}
