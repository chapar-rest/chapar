package grpc

import (
	"fmt"
	"net/http"
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
)

type Response struct {
	copyButton *widgets.FlatButton
	Tabs       *widgets.Tabs

	copyClickable widget.Clickable

	responseCode int
	duration     time.Duration
	responseSize int

	Metadata *component.ValuesTable

	response string
	message  string
	err      error

	onCopyResponse func(gtx layout.Context, dataType, data string)

	isResponseUpdated   bool
	responseIsAvailable bool
	jsonViewer          *widgets.JsonViewer
}

func NewResponse(theme *chapartheme.Theme) *Response {
	r := &Response{
		copyButton: &widgets.FlatButton{
			Text:            "Copy",
			BackgroundColor: theme.Palette.Bg,
			TextColor:       theme.Palette.Fg,
			MinWidth:        unit.Dp(75),
			Icon:            widgets.CopyIcon,
			Clickable:       new(widget.Clickable),
			IconPosition:    widgets.FlatButtonIconEnd,
			SpaceBetween:    unit.Dp(5),
		},
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Body"},
			{Title: "Metadata"},
		}, nil),
		jsonViewer: widgets.NewJsonViewer(),
		Metadata:   component.NewValuesTable("Metadata", nil),
	}
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

func (r *Response) SetStatusParams(code int, duration time.Duration, size int) {
	r.responseCode = code
	r.duration = duration
	r.responseSize = size
}

func (r *Response) SetMetadata(metadata []domain.KeyValue) {
	r.Metadata.SetData(metadata)
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
						return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.LabelStyle{
								Text:     formatStatus(r.responseCode, r.duration, uint64(r.responseSize)),
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
						return btn.Layout(gtx, theme)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if r.Tabs.Selected() == 0 {
					return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if !r.isResponseUpdated {
							r.jsonViewer.SetData(r.response)
							r.isResponseUpdated = true
						}

						return r.jsonViewer.Layout(gtx, theme)
					})
				} else {
					return r.Metadata.Layout(gtx, theme)
				}
			}),
		)
	})
}

func formatStatus(statueCode int, duration time.Duration, size uint64) string {
	return fmt.Sprintf("%d %s, %s, %s", statueCode, http.StatusText(statueCode), duration, humanize.Bytes(size))
}

func (r *Response) handleCopy(gtx layout.Context) {
	if r.onCopyResponse == nil {
		return
	}

	if r.Tabs.Selected() == 1 {
		r.onCopyResponse(gtx, "Metadata", domain.KeyValuesToText(r.Metadata.GetData()))
	} else {
		r.onCopyResponse(gtx, "Response", r.response)
	}
}