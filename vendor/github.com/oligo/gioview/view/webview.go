package view

// import (
// 	"log"
// 	"net/url"

// 	"gioui.org/f32"
// 	"gioui.org/layout"

// 	"github.com/oligo/gioview/theme"

// 	"github.com/gioui-plugins/gio-plugins/webviewer"
// 	"github.com/gioui-plugins/gio-plugins/webviewer/webview"
// )

// var (
// 	WebViewID = NewViewID("webview")
// )

// // Experimental webview integration.
// // gio-plugins use an older version which is not compatible with gio v0.5.
// // We have to wait for its upgration.
// type WebView struct {
// 	*BaseView
// 	location string
// 	sysErr   error
// 	loaded   bool

// 	localStorage   [][]webview.StorageData
// 	sessionStorage [][]webview.StorageData
// 	cookieStorage  [][]webview.CookieData
// }

// func (web *WebView) ID() ViewID {
// 	return WebViewID
// }

// func (web *WebView) Title() string {
// 	return web.location
// }

// func (web *WebView) OnNavTo(intent Intent) error {
// 	web.BaseView.OnNavTo(intent)
// 	web.sysErr = nil
// 	if intent.Params == nil || len(intent.Params) < 1 {
// 		log.Println("missing parameter")
// 		return nil
// 	}

// 	web.location = intent.Params["url"].(string)
// 	if _, err := url.Parse(web.location); err != nil {
// 		return err
// 	}

// 	log.Println("going to ", web.location)
// 	return nil
// }

// func (web *WebView) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
// 	return layout.Flex{
// 		Axis: layout.Vertical,
// 	}.Layout(gtx,
// 		layout.Flexed(1, func(gtx C) D {
// 			defer webviewer.WebViewOp{Tag: web}.Push(gtx.Ops).Pop(gtx.Ops)
// 			if !web.loaded {
// 				webviewer.NavigateOp{URL: web.location}.Add(gtx.Ops)
// 				web.loaded = true
// 			}
// 			webviewer.OffsetOp{Point: f32.Point{Y: float32(gtx.Dp(29)),
// 				X: float32(gtx.Dp(220))}}.Add(gtx.Ops)

// 			webviewer.RectOp{Size: f32.Point{X: float32(gtx.Constraints.Max.X), Y: float32(gtx.Constraints.Max.Y)}}.Add(gtx.Ops)
// 			return layout.Dimensions{Size: gtx.Constraints.Max}
// 		}),
// 	)
// }

// func (web *WebView) Update(gtx layout.Context) {
// 	// todo
// }

// func (web *WebView) OnFinish() {
// 	web.location = ""
// }

// func NewWebView() *WebView {
// 	webview.SetDebug(true)
// 	return &WebView{
// 		BaseView: &BaseView{},
// 	}
// }
