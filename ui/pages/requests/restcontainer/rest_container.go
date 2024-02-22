package restcontainer

import (
	"fmt"
	"image/color"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/notify"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type RestContainer struct {
	// Request Bar
	methodDropDown *widgets.DropDown
	addressMutex   *sync.Mutex

	updateAddress bool
	address       *widget.Editor
	sendClickable widget.Clickable
	sendButton    material.ButtonStyle

	// Response
	responseHeadersList *widget.List
	responseCookiesList *widget.List
	responseHeaders     []keyValue
	responseCookies     []keyValue
	loading             bool
	resultUpdated       bool
	result              string

	jsonViewer *widgets.JsonViewer

	// copyClickable *widget.Clickable
	// saveClickable      *widget.Clickable
	copyResponseButton *widgets.FlatButton
	// saveResponseButton *widgets.FlatButton
	responseTabs *widgets.Tabs

	// Request
	requestBody         *widgets.CodeEditor
	requestBodyDropDown *widgets.DropDown
	requestBodyBinary   *widgets.TextField
	resultStatus        string
	requestTabs         *widgets.Tabs
	preRequestDropDown  *widgets.DropDown
	preRequestBody      *widgets.CodeEditor
	postRequestDropDown *widgets.DropDown
	postRequestBody     *widgets.CodeEditor

	queryParams       *widgets.KeyValue
	updateQueryParams bool
	formDataParams    *widgets.KeyValue
	urlEncodedParams  *widgets.KeyValue
	pathParams        *widgets.KeyValue
	headers           *widgets.KeyValue

	split widgets.SplitView
}

type keyValue struct {
	Key   string
	Value string

	keySelectable   *widget.Selectable
	valueSelectable *widget.Selectable
}

func NewRestContainer(theme *material.Theme) *RestContainer {
	r := &RestContainer{
		split: widgets.SplitView{
			Ratio:         0,
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
		address:           new(widget.Editor),
		requestBody:       widgets.NewCodeEditor(""),
		preRequestBody:    widgets.NewCodeEditor(""),
		postRequestBody:   widgets.NewCodeEditor(""),
		requestBodyBinary: widgets.NewTextField("", "Select file"),
		responseHeadersList: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},

		responseCookiesList: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		jsonViewer: widgets.NewJsonViewer(),

		queryParams: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", "", false),
		),

		pathParams: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", "", false),
		),

		headers: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", "", false),
		),

		formDataParams: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", "", false),
		),
		urlEncodedParams: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", "", false),
		),

		addressMutex: &sync.Mutex{},
	}

	r.copyResponseButton = &widgets.FlatButton{
		Text:            "Copy",
		BackgroundColor: theme.Palette.Bg,
		TextColor:       theme.Palette.Fg,
		MinWidth:        unit.Dp(75),
		Icon:            widgets.CopyIcon,
		IconPosition:    widgets.FlatButtonIconEnd,
		SpaceBetween:    unit.Dp(5),
	}

	r.requestBodyBinary.SetIcon(widgets.UploadIcon, widgets.IconPositionEnd)

	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)

	r.queryParams.SetOnChanged(r.onQueryParamChange)

	r.sendButton = material.Button(theme, &r.sendClickable, "Send")
	r.requestTabs = widgets.NewTabs([]*widgets.Tab{
		{Title: "Params"},
		{Title: "Body"},
		{Title: "Headers"},
		{Title: "Pre-req"},
		{Title: "Post-req"},
	}, nil)

	r.responseTabs = widgets.NewTabs([]*widgets.Tab{
		{Title: "Body"},
		{Title: "Headers"},
		{Title: "Cookies"},
	}, nil)

	r.methodDropDown = widgets.NewDropDownWithoutBorder(
		widgets.NewDropDownOption("GET"),
		widgets.NewDropDownOption("POST"),
		widgets.NewDropDownOption("PUT"),
		widgets.NewDropDownOption("PATCH"),
		widgets.NewDropDownOption("DELETE"),
		widgets.NewDropDownOption("HEAD"),
		widgets.NewDropDownOption("OPTION"),
	)

	r.preRequestDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Python Script"),
		widgets.NewDropDownOption("SSH Script"),
		widgets.NewDropDownOption("SSH Tunnel"),
		widgets.NewDropDownOption("Kubectl Tunnel"),
	)

	r.postRequestDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Python Script"),
		widgets.NewDropDownOption("SSH Script"),
	)

	r.requestBodyDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("JSON"),
		widgets.NewDropDownOption("Text"),
		widgets.NewDropDownOption("XML"),
		widgets.NewDropDownOption("Form data"),
		widgets.NewDropDownOption("Binary"),
		widgets.NewDropDownOption("Urlencoded"),
	)
	r.address.SingleLine = true
	r.address.SetText("https://jsonplaceholder.typicode.com/comments")

	return r
}

func (r *RestContainer) copyResponseToClipboard(gtx layout.Context) {
	switch r.responseTabs.Selected() {
	case 0:
		if r.result == "" {
			return
		}

		gtx.Execute(clipboard.WriteCmd{
			Data: io.NopCloser(strings.NewReader(r.result)),
		})
		notify.Send("Response copied to clipboard", time.Second*3)
	case 1:
		if len(r.responseHeaders) == 0 {
			return
		}

		headers := ""
		for _, h := range r.responseHeaders {
			headers += fmt.Sprintf("%s: %s\n", h.Key, h.Value)
		}

		gtx.Execute(clipboard.WriteCmd{
			Data: io.NopCloser(strings.NewReader(headers)),
		})
		notify.Send("Response headers copied to clipboard", time.Second*3)
	case 2:
		if len(r.responseCookies) == 0 {
			return
		}

		cookies := ""
		for _, c := range r.responseCookies {
			cookies += fmt.Sprintf("%s: %s\n", c.Key, c.Value)
		}

		gtx.Execute(clipboard.WriteCmd{
			Data: io.NopCloser(strings.NewReader(cookies)),
		})
		notify.Send("Response cookies copied to clipboard", time.Second*3)
	}
}

func (r *RestContainer) onQueryParamChange(items []*widgets.KeyValueItem) {
	if r.updateQueryParams {
		r.updateQueryParams = false
		return
	}

	addr := r.address.Text()
	if addr == "" {
		return
	}

	// Parse the existing URL
	parsedURL, err := url.Parse(addr)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}

	// Parse the query parameters from the URL
	queryParams := parsedURL.Query()

	// Iterate over the items and update the query parameters
	for _, item := range items {
		if item.Active && item.Key != "" && item.Value != "" {
			// Set the parameter only if both key and value are non-empty
			queryParams.Set(item.Key, item.Value)
		} else {
			// Remove the parameter if the item is not active or key/value is empty
			queryParams.Del(item.Key)
		}
	}

	// delete items that are not exit in items
	for k := range queryParams {
		found := false
		for _, item := range items {
			if item.Active && item.Key == k {
				found = true
				break
			}
		}
		if !found {
			queryParams.Del(k)
		}
	}

	parsedURL.RawQuery = queryParams.Encode()
	finalURL := parsedURL.String()
	r.addressMutex.Lock()
	r.updateAddress = true

	_, coll := r.address.CaretPos()
	r.address.SetText(finalURL)
	r.address.SetCaret(coll, coll+1)

	r.addressMutex.Unlock()
}

func (r *RestContainer) addressChanged() {
	// Parse the existing URL
	parsedURL, err := url.Parse(r.address.Text())
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}

	// Parse the query parameters from the URL
	queryParams := parsedURL.Query()

	items := make([]*widgets.KeyValueItem, 0)
	for k, v := range queryParams {
		if len(v) > 0 {
			// Add the parameter as a new key-value item
			items = append(items, widgets.NewKeyValueItem(k, v[0], "", true))
		}
	}

	r.updateQueryParams = true
	r.queryParams.SetItems(items)
}

func (r *RestContainer) paramsLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.queryParams.WithAddLayout(gtx, "Query", "", theme)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.pathParams.WithAddLayout(gtx, "Path", "path params inside bracket, for example: {id}", theme)
		}),
	)
}

func (r *RestContainer) requestBodyFormDataLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.queryParams.WithAddLayout(gtx, "Query", "", theme)
		}),
	)
}

func (r *RestContainer) requestLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.requestTabs.Layout(gtx, theme)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				switch r.requestTabs.Selected() {
				case 0:
					return r.paramsLayout(gtx, theme)
				case 1:
					return r.requestBodyLayout(gtx, theme)
				case 2:
					return r.headers.WithAddLayout(gtx, "Headers", "", theme)
				case 3:
					return r.requestPreReqLayout(gtx, theme)
				case 4:
					return r.requestPostReqLayout(gtx, theme)
				}
				return layout.Dimensions{}
			})
		}),
	)
}

func (r *RestContainer) messageLayout(gtx layout.Context, theme *material.Theme, message string) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		l := material.LabelStyle{
			Text:     message,
			Color:    widgets.Gray600,
			TextSize: theme.TextSize,
			Shaper:   theme.Shaper,
		}
		l.Font.Typeface = theme.Face
		return l.Layout(gtx)
	})
}

func (r *RestContainer) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Left:   unit.Dp(10),
				Top:    unit.Dp(10),
				Bottom: unit.Dp(5),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme, theme.TextSize, "Create user").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.requestBar(gtx, theme)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return r.requestLayout(gtx, theme)
				},
				func(gtx layout.Context) layout.Dimensions {
					if r.loading {
						return material.Label(theme, theme.TextSize, "Loading...").Layout(gtx)
					} else {
						// update only once
						if !r.resultUpdated {
							r.jsonViewer.SetData(r.result)
							r.resultUpdated = true
						}
					}

					return r.responseLayout(gtx, theme)
				},
			)
		}),
	)
}
