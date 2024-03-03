package rest

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
	"github.com/mirzakhany/chapar/internal/bus"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/loader"
	"github.com/mirzakhany/chapar/internal/notify"
	"github.com/mirzakhany/chapar/ui/converter"
	"github.com/mirzakhany/chapar/ui/keys"
	"github.com/mirzakhany/chapar/ui/state"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Container struct {
	req   *domain.Request
	title *widgets.EditableLabel

	saveButton *widget.Clickable

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

	copyResponseButton *widgets.FlatButton
	responseTabs       *widgets.Tabs

	// Request
	requestBody         *widgets.CodeEditor
	requestBodyDropDown *widgets.DropDown
	requestBodyBinary   *widgets.TextField
	resultStatus        string
	requestTabs         *widgets.Tabs
	preRequestDropDown  *widgets.DropDown
	authDropDown        *widgets.DropDown
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

	dataChanged bool

	basicAuthUsername *widget.Editor
	basicAuthPassword *widget.Editor
	bearerTokenEditor *widget.Editor

	onTitleChanged func(id, title string)
	onDataChanged  func(id string, values []domain.KeyValue)

	prompt *widgets.Prompt
}

type keyValue struct {
	Key   string
	Value string

	keySelectable   *widget.Selectable
	valueSelectable *widget.Selectable
}

func NewRestContainer(theme *material.Theme, req *domain.Request) *Container {
	r := &Container{
		req:   req,
		title: widgets.NewEditableLabel(""),
		split: widgets.SplitView{
			Ratio:         0,
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
		address:           new(widget.Editor),
		basicAuthPassword: new(widget.Editor),
		basicAuthUsername: new(widget.Editor),
		bearerTokenEditor: new(widget.Editor),

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

		saveButton: new(widget.Clickable),
		prompt:     widgets.NewPrompt("Save", "This request value is changed, do you wanna save it before closing it?\nHint: you always can save the changes with ctrl+s", widgets.ModalTypeWarn, "Yes", "No"),
	}
	r.prompt.WithRememberBool()

	r.requestBody.SetOnChanged(r.onTextBodyChanged)
	r.preRequestBody.SetOnChanged(r.onTextBodyChanged)
	r.postRequestBody.SetOnChanged(r.onTextBodyChanged)
	r.queryParams.SetOnChanged(r.onQueryParamChange)
	r.pathParams.SetOnChanged(r.onKeValuesChanged)
	r.headers.SetOnChanged(r.onKeValuesChanged)
	r.formDataParams.SetOnChanged(r.onKeValuesChanged)
	r.urlEncodedParams.SetOnChanged(r.onKeValuesChanged)

	r.title.SetOnChanged(func(text string) {
		if r.req == nil {
			return
		}

		if r.req.MetaData.Name == text {
			return
		}

		// save changes to the request
		r.req.MetaData.Name = text
		if err := loader.UpdateRequest(r.req); err != nil {
			r.showError(fmt.Sprintf("failed to update request: %s", err))
			return
		}

		if r.onTitleChanged != nil {
			r.onTitleChanged(r.req.MetaData.ID, text)
			bus.Publish(state.RequestsChanged, nil)
		}
	})

	r.copyResponseButton = &widgets.FlatButton{
		Text:            "Copy",
		BackgroundColor: theme.Palette.Bg,
		TextColor:       theme.Palette.Fg,
		MinWidth:        unit.Dp(75),
		Icon:            widgets.CopyIcon,
		Clickable:       new(widget.Clickable),
		IconPosition:    widgets.FlatButtonIconEnd,
		SpaceBetween:    unit.Dp(5),
	}

	r.requestBodyBinary.SetIcon(widgets.UploadIcon, widgets.IconPositionEnd)

	r.sendButton = material.Button(theme, &r.sendClickable, "Send")
	r.requestTabs = widgets.NewTabs([]*widgets.Tab{
		{Title: "Params"},
		{Title: "Body"},
		{Title: "Auth"},
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
	r.methodDropDown.SetOnValueChanged(r.onDropDownChanged)

	r.authDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Basic"),
		widgets.NewDropDownOption("Bearer"),
	)

	r.authDropDown.SetOnValueChanged(r.onDropDownChanged)

	r.preRequestDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Python Script"),
		widgets.NewDropDownOption("SSH Script"),
		widgets.NewDropDownOption("SSH Tunnel"),
		widgets.NewDropDownOption("Kubectl Tunnel"),
	)

	r.preRequestDropDown.SetOnValueChanged(r.onDropDownChanged)

	r.postRequestDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Python Script"),
		widgets.NewDropDownOption("SSH Script"),
	)
	r.postRequestDropDown.SetOnValueChanged(r.onDropDownChanged)

	r.requestBodyDropDown = widgets.NewDropDown(
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("JSON"),
		widgets.NewDropDownOption("Text"),
		widgets.NewDropDownOption("XML"),
		widgets.NewDropDownOption("Form data"),
		widgets.NewDropDownOption("Binary"),
		widgets.NewDropDownOption("Urlencoded"),
	)

	r.requestBodyDropDown.SetOnValueChanged(r.onDropDownChanged)

	r.address.SingleLine = true
	r.address.SetText("https://example.com")

	r.Load(req)

	return r
}

func (r *Container) SetOnTitleChanged(f func(string, string)) {
	r.onTitleChanged = f
}

func (r *Container) onTextBodyChanged(newText string) {
	r.dataChanged = r.req.Spec.HTTP.Body.Body != newText ||
		r.req.Spec.HTTP.Body.PreRequest.Script != newText ||
		r.req.Spec.HTTP.Body.PostRequest.Script != newText
}

func (r *Container) onDropDownChanged(selected string) {
	r.dataChanged = r.req.Spec.HTTP.Method != selected ||
		r.req.Spec.HTTP.Body.PreRequest.Type != selected ||
		r.req.Spec.HTTP.Body.PostRequest.Type != selected ||
		r.req.Spec.HTTP.Body.BodyType != selected
}

func (r *Container) onKeValuesChanged(items []*widgets.KeyValueItem) {
	r.dataChanged = !(domain.CompareKeyValues(r.req.Spec.HTTP.Body.PathParams, converter.KeyValueFromWidgetItems(items)) ||
		domain.CompareKeyValues(r.req.Spec.HTTP.Body.Headers, converter.KeyValueFromWidgetItems(items)) ||
		domain.CompareKeyValues(r.req.Spec.HTTP.Body.FormBody, converter.KeyValueFromWidgetItems(items)) ||
		domain.CompareKeyValues(r.req.Spec.HTTP.Body.URLEncoded, converter.KeyValueFromWidgetItems(items)))
}

func (r *Container) IsDataChanged() bool {
	return r.dataChanged
}

func (r *Container) Load(e *domain.Request) {
	r.req = e
	r.title.SetText(e.MetaData.Name)

	// format url with query params. it will update the query params list as well
	finalURL, err := updateURLWithQueryParams(e.Spec.HTTP.URL, converter.WidgetItemsFromKeyValue(e.Spec.HTTP.Body.QueryParams))
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}

	r.address.SetText(finalURL)
	r.methodDropDown.SetSelectedByValue(e.Spec.HTTP.Method)
	r.headers.SetItems(converter.WidgetItemsFromKeyValue(e.Spec.HTTP.Body.Headers))
	r.pathParams.SetItems(converter.WidgetItemsFromKeyValue(e.Spec.HTTP.Body.PathParams))
	r.requestBodyDropDown.SetSelectedByValue(e.Spec.HTTP.Body.BodyType)
	r.requestBody.SetCode(e.Spec.HTTP.Body.Body)
	r.preRequestDropDown.SetSelectedByValue(e.Spec.HTTP.Body.PreRequest.Type)
	r.preRequestBody.SetCode(e.Spec.HTTP.Body.PreRequest.Script)
	r.postRequestDropDown.SetSelectedByValue(e.Spec.HTTP.Body.PostRequest.Type)
	r.postRequestBody.SetCode(e.Spec.HTTP.Body.PostRequest.Script)
}

// OnClose is called when the tab is closed
// it returns true if the tab can be closed
func (r *Container) OnClose() bool {
	if r.dataChanged {
		r.showNotSavedWarning()
		return false
	}

	return true
}

func (r *Container) showError(err string) {
	r.prompt.Type = widgets.ModalTypeErr
	r.prompt.Content = err
	r.prompt.SetOptions("I see")
	r.prompt.WithoutRememberBool()
	r.prompt.SetOnSubmit(nil)
	r.prompt.Show()
}

func (r *Container) showNotSavedWarning() {
	r.prompt.Type = widgets.ModalTypeWarn
	r.prompt.Content = "This request value is changed, do you wanna save it before closing it?\nHint: you always can save the changes with ctrl+s"
	r.prompt.SetOptions("Yes", "No")
	r.prompt.WithRememberBool()
	r.prompt.SetOnSubmit(r.onPromptSubmit)
	r.prompt.Show()
}

func (r *Container) onPromptSubmit(selectedOption string, remember bool) {
	if selectedOption == "Yes" {
		r.save()
		r.prompt.Hide()
	}
}

func (r *Container) save() {
	if r.dataChanged {
		r.populateItems()
		if err := loader.UpdateRequest(r.req); err != nil {
			r.showError(fmt.Sprintf("failed to update request: %s", err))
		} else {
			r.dataChanged = false
			bus.Publish(state.RequestsChanged, nil)
		}
	}
}

func (r *Container) populateItems() {
	r.req.Spec.HTTP.Body.Headers = converter.KeyValueFromWidgetItems(r.headers.GetItems())
	r.req.Spec.HTTP.Body.QueryParams = converter.KeyValueFromWidgetItems(r.queryParams.GetItems())
	r.req.Spec.HTTP.Body.PathParams = converter.KeyValueFromWidgetItems(r.pathParams.GetItems())
	r.req.Spec.HTTP.Body.BodyType = r.requestBodyDropDown.GetSelected().Text
	r.req.Spec.HTTP.Body.Body = r.requestBody.Code()
	r.req.Spec.HTTP.Body.PreRequest.Type = r.preRequestDropDown.GetSelected().Text
	r.req.Spec.HTTP.Body.PreRequest.Script = r.preRequestBody.Code()
	r.req.Spec.HTTP.Body.PostRequest.Type = r.postRequestDropDown.GetSelected().Text
	r.req.Spec.HTTP.Body.PostRequest.Script = r.postRequestBody.Code()
	r.req.Spec.HTTP.Method = r.methodDropDown.GetSelected().Text
	r.req.Spec.HTTP.URL = r.address.Text()
}

func (r *Container) copyResponseToClipboard(gtx layout.Context) {
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

func (r *Container) onQueryParamChange(items []*widgets.KeyValueItem) {
	if r.updateQueryParams {
		r.updateQueryParams = false
		return
	}

	if !domain.CompareKeyValues(r.req.Spec.HTTP.Body.QueryParams, converter.KeyValueFromWidgetItems(items)) {
		r.dataChanged = true
	}

	addr := r.address.Text()
	if addr == "" {
		return
	}

	finalURL, err := updateURLWithQueryParams(addr, items)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}

	r.addressMutex.Lock()
	r.updateAddress = true

	_, coll := r.address.CaretPos()
	r.address.SetText(finalURL)
	r.address.SetCaret(coll, coll+1)

	r.addressMutex.Unlock()
}

func updateURLWithQueryParams(addr string, params []*widgets.KeyValueItem) (string, error) {
	// Parse the existing URL
	parsedURL, err := url.Parse(addr)
	if err != nil {
		return "", err
	}

	// Parse the query parameters from the URL
	queryParams := parsedURL.Query()

	// Iterate over the items and update the query parameters
	for _, item := range params {
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
		for _, item := range params {
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
	return parsedURL.String(), nil
}

func (r *Container) addressChanged() {
	newAddress := r.address.Text()
	r.dataChanged = newAddress != r.req.Spec.HTTP.URL

	// Parse the existing URL
	parsedURL, err := url.Parse(newAddress)
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

func (r *Container) paramsLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
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

func (r *Container) requestBodyFormDataLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.queryParams.WithAddLayout(gtx, "Query", "", theme)
		}),
	)
}

func (r *Container) requestLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
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
					return r.authLayout(gtx, theme)
				case 3:
					return r.headers.WithAddLayout(gtx, "Headers", "", theme)
				case 4:
					return r.requestPreReqLayout(gtx, theme)
				case 5:
					return r.requestPostReqLayout(gtx, theme)
				}
				return layout.Dimensions{}
			})
		}),
	)
}

func (r *Container) messageLayout(gtx layout.Context, theme *material.Theme, message string) layout.Dimensions {
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

func (r *Container) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	keys.OnSaveCommand(gtx, r, r.save)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.prompt.Layout(gtx, theme)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:    unit.Dp(15),
				Bottom: unit.Dp(15),
				Left:   unit.Dp(5),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return r.title.Layout(gtx, theme)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if r.dataChanged {
							if r.saveButton.Clicked(gtx) {
								r.save()
							}
							return widgets.SaveButtonLayout(gtx, theme, r.saveButton)
						} else {
							return layout.Dimensions{}
						}
					}),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
