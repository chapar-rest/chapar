package restful

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mirzakhany/chapar/internal/domain"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dustin/go-humanize"
	"github.com/mirzakhany/chapar/ui/pages/requests/component"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Response struct {
	copyButton *widgets.FlatButton
	Tabs       *widgets.Tabs

	responseCode int
	duration     time.Duration
	responseSize int

	responseHeaders *component.ValuesTable
	responseCookies *component.ValuesTable

	response       string
	onCopyResponse func(gtx layout.Context, response string)

	isResponseUpdated bool
	jsonViewer        *widgets.JsonViewer
}

func NewResponse(theme *material.Theme) *Response {
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
			{Title: "Headers"},
			{Title: "Cookies"},
		}, nil),
		jsonViewer:      widgets.NewJsonViewer(),
		responseHeaders: component.NewValuesTable("Headers", nil),
		responseCookies: component.NewValuesTable("Cookies", nil),
	}
	return r
}

func (r *Response) SetOnCopyResponse(f func(gtx layout.Context, response string)) {
	r.onCopyResponse = f
}

func (r *Response) SetResponse(response string) {
	r.response = response
	r.isResponseUpdated = false
}

func (r *Response) SetStatusParams(code int, duration time.Duration, size int) {
	r.responseCode = code
	r.duration = duration
	r.responseSize = size
}

func (r *Response) SetHeaders(headers []domain.KeyValue) {
	r.responseHeaders.SetData(headers)
}

func (r *Response) SetCookies(cookies []domain.KeyValue) {
	r.responseCookies.SetData(cookies)
}

func (r *Response) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if r.response == "" {
		return component.Message(gtx, theme, "No response available yet ;)")
	}

	if r.copyButton.Clickable.Clicked(gtx) {
		r.onCopyResponse(gtx, r.response)
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
								Color:    widgets.LightGreen,
								TextSize: theme.TextSize,
								Shaper:   theme.Shaper,
							}
							l.Font.Typeface = theme.Face
							return l.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return r.copyButton.Layout(gtx, theme)
					}),
				)
			}),
			widgets.DrawLineFlex(widgets.Gray300, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				switch r.Tabs.Selected() {
				case 1:
					return r.responseHeaders.Layout(gtx, theme)
				case 2:
					return r.responseCookies.Layout(gtx, theme)
				default:
					return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if !r.isResponseUpdated {
							r.jsonViewer.SetData(r.response)
							r.isResponseUpdated = true
						}

						return r.jsonViewer.Layout(gtx, theme)
					})
				}
			}),
		)
	})
}

func formatStatus(statueCode int, duration time.Duration, size uint64) string {
	return fmt.Sprintf("%d %s, %s, %s", statueCode, http.StatusText(statueCode), duration, humanize.Bytes(size))
}
