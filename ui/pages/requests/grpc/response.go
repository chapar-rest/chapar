package grpc

import (
	"fmt"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dustin/go-humanize"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"
	"github.com/chapar-rest/chapar/ui/widgets"
	"github.com/chapar-rest/chapar/ui/widgets/codeeditor"
)

type Response struct {
	copyButton *widgets.FlatButton
	Tabs       *widgets.Tabs

	copyClickable widget.Clickable

	responseCode int
	status       string
	duration     time.Duration
	responseSize int

	Metadata   *codeeditor.CodeEditor
	Trailers   *codeeditor.CodeEditor
	jsonViewer *codeeditor.CodeEditor

	response string
	message  string
	err      error

	onCopyResponse func(gtx layout.Context, dataType, data string)

	isResponseUpdated   bool
	responseIsAvailable bool
}

func NewResponse(theme *chapartheme.Theme) *Response {
	r := &Response{
		copyButton: &widgets.FlatButton{
			Text:            "Copy",
			BackgroundColor: theme.Material().Palette.Bg,
			TextColor:       theme.Material().Palette.Fg,
			MinWidth:        unit.Dp(75),
			Icon:            widgets.CopyIcon,
			Clickable:       new(widget.Clickable),
			IconPosition:    widgets.FlatButtonIconEnd,
			SpaceBetween:    unit.Dp(5),
		},
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Body"},
			{Title: "Metadata"},
			{Title: "Trailers"},
		}, nil),
		jsonViewer: codeeditor.NewCodeEditor("", codeeditor.CodeLanguageJSON, theme),
		Metadata:   codeeditor.NewCodeEditor("", codeeditor.CodeLanguageProperties, theme),
		Trailers:   codeeditor.NewCodeEditor("", codeeditor.CodeLanguageProperties, theme),
	}

	r.jsonViewer.SetReadOnly(true)
	r.Metadata.SetReadOnly(true)
	r.Trailers.SetReadOnly(true)

	return r
}

func (r *Response) SetOnCopyResponse(f func(gtx layout.Context, dataType, data string)) {
	r.onCopyResponse = f
}

func (r *Response) SetResponse(response string) {
	r.response = response
	r.err = nil
	r.message = ""
	r.isResponseUpdated = false
	r.responseIsAvailable = true
}

func (r *Response) SetStatusParams(code int, status string, duration time.Duration, size int) {
	r.responseCode = code
	r.duration = duration
	r.responseSize = size
	r.status = status
}

func (r *Response) SetMetadata(requestMetadata, responseMetadata []domain.KeyValue) {
	if len(requestMetadata) == 0 && len(responseMetadata) == 0 {
		r.Metadata.SetCode("")
		return
	}

	if len(requestMetadata) == 0 {
		r.Metadata.SetCode(domain.KeyValuesToText(responseMetadata))
		return
	}

	if len(responseMetadata) == 0 {
		r.Metadata.SetCode(domain.KeyValuesToText(requestMetadata))
		return
	}

	txt := fmt.Sprintf("# --- Request Metadata ---\n\n%s\n\n# --- Response Metadata ---\n\n%s", domain.KeyValuesToText(requestMetadata), domain.KeyValuesToText(responseMetadata))
	r.Metadata.SetCode(txt)
}

func (r *Response) SetTrailers(trailers []domain.KeyValue) {
	r.Trailers.SetCode(domain.KeyValuesToText(trailers))
}

func (r *Response) SetMessage(message string) {
	r.message = message
}

func (r *Response) SetError(err error) {
	r.err = err
}

func (r *Response) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if r.err != nil {
		return component.Message(gtx, component.MessageTypeError, theme, r.err.Error())
	}

	if r.message != "" {
		return component.Message(gtx, component.MessageTypeInfo, theme, r.message)
	}

	if !r.responseIsAvailable {
		return component.Message(gtx, component.MessageTypeInfo, theme, "No response available yet ;)")
	}

	if r.copyClickable.Clicked(gtx) {
		r.handleCopy(gtx)
	}

	inset := layout.Inset{Top: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Tabs.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(5), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.LabelStyle{
								Text:     formatStatus(r.responseCode, r.status, r.duration, uint64(r.responseSize)),
								Color:    theme.ResponseStatusColor,
								TextSize: theme.TextSize,
								Shaper:   theme.Shaper,
							}
							l.Font.Typeface = theme.Face
							return l.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := widgets.Button(theme.Material(), &r.copyClickable, widgets.CopyIcon, widgets.IconPositionStart, "Copy")
						btn.Color = theme.ButtonTextColor
						btn.Inset = layout.Inset{
							Top: 4, Bottom: 4,
							Left: 4, Right: 4,
						}
						return btn.Layout(gtx, theme)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(5), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					switch r.Tabs.Selected() {
					case 0:

						if !r.isResponseUpdated {
							r.jsonViewer.SetCode(r.response)
							r.isResponseUpdated = true
						}
						return r.jsonViewer.Layout(gtx, theme, "")

					case 1:
						return r.Metadata.Layout(gtx, theme, "")
					default:
						return r.Trailers.Layout(gtx, theme, "")
					}
				})
			}),
		)
	})
}

func formatStatus(statueCode int, status string, duration time.Duration, size uint64) string {
	return fmt.Sprintf("%d %s, %s, %s", statueCode, status, duration, humanize.Bytes(size))
}

func (r *Response) handleCopy(gtx layout.Context) {
	if r.onCopyResponse == nil {
		return
	}

	switch r.Tabs.Selected() {
	case 0:
		r.onCopyResponse(gtx, "Response", r.response)
	case 1:
		r.onCopyResponse(gtx, "Metadata", r.Metadata.Code())
	default:
		r.onCopyResponse(gtx, "Trailers", r.Trailers.Code())
	}
}
