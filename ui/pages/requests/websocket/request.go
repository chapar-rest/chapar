package websocket

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/converter"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Request struct {
	Tabs   *widgets.Tabs
	Prompt *widgets.Prompt

	Body *Body

	Settings *widgets.Settings
	Headers  *widgets.KeyValue
	Params   *widgets.KeyValue

	currentTab  string
	OnTabChange func(title string)
}

func NewRequest(req *domain.Request, theme *chapartheme.Theme) *Request {
	r := &Request{
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Message"},
			{Title: "Query Params"},
			{Title: "Headers"},
			{Title: "Settings"},
		}, nil),

		Body: NewBody(req.Spec.WebSocket.Request.BodyType, req.Spec.WebSocket.Request.Body, theme),
		Params: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(req.Spec.WebSocket.Request.QueryParams)...,
		),
		Headers: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(req.Spec.WebSocket.Request.Headers)...,
		),
		Settings: widgets.NewSettings([]*widgets.SettingItem{
			widgets.NewNumberItem("Hand Shake Timeout Milliseconds", "handShakeTimeoutMilliseconds", "Set how long the handshake request should wait before timing out in milliseconds. zero means unlimited", req.Spec.WebSocket.Request.Settings.HandShakeTimeoutMilliseconds),
			widgets.NewNumberItem("Reconnection Attempts", "reconnectionAttempts", "Maximum reconnection attempts when the connection closes abruptly.", req.Spec.WebSocket.Request.Settings.ReconnectionAttempts),
			widgets.NewNumberItem("Reconnection Interval Milliseconds", "reconnectionIntervalMilliseconds", "Interval between each reconnection attempt in milliseconds.", req.Spec.WebSocket.Request.Settings.ReconnectionIntervalMilliseconds),
			widgets.NewNumberItem("Message size limit in MB", "messageSizeLimitMB", "Maximum allowed message size in MB.", req.Spec.WebSocket.Request.Settings.MessageSizeLimitMB),
		}),
	}

	return r
}

func (r *Request) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Tabs.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Prompt.Layout(gtx, theme)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if r.Tabs.SelectedTab().Title != r.currentTab {
					r.currentTab = r.Tabs.SelectedTab().Title
					if r.OnTabChange != nil {
						r.OnTabChange(r.currentTab)
					}
				}
				switch r.Tabs.SelectedTab().Title {
				case "Body":
					return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.Body.Layout(gtx, theme)
					})
				case "Query Params":
					return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.Params.WithAddLayout(gtx, "Query Params", "", theme)
					})
				case "Headers":
					return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.Headers.WithAddLayout(gtx, "Headers", "", theme)
					})
				case "Settings":
					return r.Settings.Layout(gtx, theme)
				default:
					return layout.Dimensions{}
				}
			}),
		)
	})
}
