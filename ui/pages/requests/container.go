package requests

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
)

const (
	TypeRequest    = "request"
	TypeCollection = "collection"

	TypeMeta = "Type"
)

//type Container interface {
//	Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions
//	SetActiveEnvironment(env *domain.Environment)
//	IsDataChanged() bool
//	SetDataChanged(dirty bool)
//	SetOnTitleChanged(f func(title string))
//	OnClose() bool
//	ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...string)
//	HidePrompt()
//}

type Container interface {
	Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions
	SetOnDataChanged(func(id string, data any))
	SetOnTitleChanged(f func(title string))
	SetDataChanged(changed bool)
	SetOnSave(f func(id string))

	ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...string)
	HidePrompt()
}
