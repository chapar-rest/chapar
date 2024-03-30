package requests

import (
	"time"

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
	SetOnDataChanged(f func(id string, data any))
	SetOnTitleChanged(f func(title string))
	SetDataChanged(changed bool)
	SetOnSave(f func(id string))
	ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...string)
	HidePrompt()
}

type RestContainer interface {
	SetHTTPResponse(response string, headers []domain.KeyValue, cookies []domain.KeyValue, statusCode int, duration time.Duration, size int)
	ShowSendingRequestLoading()
	HideSendingRequestLoading()
	SetQueryParams(params []domain.KeyValue)
	SetURL(url string)
}
