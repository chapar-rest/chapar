package pages

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gioui.org/text"

	"gioui.org/font"

	"github.com/mirzakhany/chapar/internal/notify"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dustin/go-humanize"
	"github.com/mirzakhany/chapar/internal/rest"
	"github.com/mirzakhany/chapar/ui/widgets"
	"golang.design/x/clipboard"
)

var (
	requestTabsInset = layout.Inset{Left: unit.Dp(5), Top: unit.Dp(3)}
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
	resultLines         []string
	resultList          *widget.List

	copyResponseButton *widgets.FlatButton
	saveResponseButton *widgets.FlatButton
	responseTabs       *widgets.Tabs

	// Request
	requestBody         *widget.Editor
	requestBodyDropDown *widgets.DropDown
	requestBodyBinary   *widgets.TextField
	resultStatus        string
	requestTabs         *widgets.Tabs
	preRequestDropDown  *widgets.DropDown
	preRequestBody      *widget.Editor
	postRequestDropDown *widgets.DropDown
	postRequestBody     *widget.Editor

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
		requestBody:       new(widget.Editor),
		preRequestBody:    new(widget.Editor),
		postRequestBody:   new(widget.Editor),
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
		resultList: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},

		copyResponseButton: widgets.NewFlatButton(theme, "Copy"),
		saveResponseButton: widgets.NewFlatButton(theme, "Save"),

		queryParams: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),

		pathParams: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),

		headers: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),

		formDataParams: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),
		urlEncodedParams: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),

		addressMutex: &sync.Mutex{},
	}
	r.requestBodyBinary.SetIcon(widgets.UploadIcon, widgets.IconPositionEnd)

	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)

	r.queryParams.SetOnChanged(r.onQueryParamChange)

	r.copyResponseButton.SetIcon(widgets.CopyIcon, widgets.FlatButtonIconEnd, 5)
	r.copyResponseButton.SetColor(theme.Palette.Bg, theme.Palette.Fg)
	r.copyResponseButton.MinWidth = unit.Dp(75)
	r.copyResponseButton.OnClicked = r.responseCopy

	r.saveResponseButton.SetIcon(widgets.SaveIcon, widgets.FlatButtonIconEnd, 5)
	r.saveResponseButton.SetColor(theme.Palette.Bg, theme.Palette.Fg)
	r.saveResponseButton.MinWidth = unit.Dp(75)

	r.sendButton = material.Button(theme, &r.sendClickable, "Send")
	r.requestTabs = widgets.NewTabs([]widgets.Tab{
		{Title: "Params"},
		{Title: "Body"},
		{Title: "Headers"},
		{Title: "Pre-req"},
		{Title: "Post-req"},
	}, nil)

	r.responseTabs = widgets.NewTabs([]widgets.Tab{
		{Title: "Body"},
		{Title: "Headers"},
		{Title: "Cookies"},
	}, nil)

	r.methodDropDown = widgets.NewDropDown(theme,
		widgets.NewDropDownOption("GET"),
		widgets.NewDropDownOption("POST"),
		widgets.NewDropDownOption("PUT"),
		widgets.NewDropDownOption("PATCH"),
		widgets.NewDropDownOption("DELETE"),
		widgets.NewDropDownOption("HEAD"),
		widgets.NewDropDownOption("OPTION"),
	)
	r.methodDropDown.SetSize(image.Point{X: 150})

	r.preRequestDropDown = widgets.NewDropDown(theme,
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Python Script"),
		widgets.NewDropDownOption("SSH Script"),
		widgets.NewDropDownOption("SSH Tunnel"),
		widgets.NewDropDownOption("Kubectl Tunnel"),
	)
	r.preRequestDropDown.SetSize(image.Point{X: 230})
	r.preRequestDropDown.SetBorder(widgets.Gray400, unit.Dp(1), unit.Dp(4))

	r.postRequestDropDown = widgets.NewDropDown(theme,
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("Python Script"),
		widgets.NewDropDownOption("SSH Script"),
	)

	r.postRequestDropDown.SetSize(image.Point{X: 230})
	r.postRequestDropDown.SetBorder(widgets.Gray400, unit.Dp(1), unit.Dp(4))

	r.requestBodyDropDown = widgets.NewDropDown(theme,
		widgets.NewDropDownOption("None"),
		widgets.NewDropDownOption("JSON"),
		widgets.NewDropDownOption("Text"),
		widgets.NewDropDownOption("XML"),
		widgets.NewDropDownOption("Form data"),
		widgets.NewDropDownOption("Binary"),
		widgets.NewDropDownOption("Urlencoded"),
	)
	r.requestBodyDropDown.SetSize(image.Point{X: 230})
	r.requestBodyDropDown.SetBorder(widgets.Gray400, unit.Dp(1), unit.Dp(4))

	r.address.SingleLine = true
	r.address.SetText("https://jsonplaceholder.typicode.com/comments")

	return r
}

func (r *RestContainer) Submit() {
	method := r.methodDropDown.GetSelected().Text
	address := r.address.Text()
	headers := make(map[string]string)
	for _, h := range r.headers.GetItems() {
		if h.Key == "" || !h.Active || h.Value == "" {
			continue
		}
		headers[h.Key] = h.Value
	}

	body := r.prepareBody()

	r.resultStatus = ""
	r.sendButton.Text = "Cancel"
	r.loading = true
	r.resultUpdated = false
	defer func() {
		r.sendButton.Text = "Send"
		r.loading = false
		r.resultUpdated = false
	}()

	res, err := rest.DoRequest(&rest.Request{
		URL:     address,
		Method:  method,
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		r.result = err.Error()
		return
	}

	dataStr := string(res.Body)
	if rest.IsJSON(dataStr) {
		var data map[string]interface{}
		if err := json.Unmarshal(res.Body, &data); err != nil {
			r.result = err.Error()
			return
		}
		var err error
		dataStr, err = rest.PrettyJSON(res.Body)
		if err != nil {
			r.result = err.Error()
			return
		}
	}

	// format response status
	r.resultStatus = fmt.Sprintf("%d %s, %s, %s", res.StatusCode, http.StatusText(res.StatusCode), res.TimePassed, humanize.Bytes(uint64(len(res.Body))))
	r.responseHeaders = make([]keyValue, 0)
	for k, v := range res.Headers {
		r.responseHeaders = append(r.responseHeaders, keyValue{
			Key:             k,
			Value:           v,
			keySelectable:   &widget.Selectable{},
			valueSelectable: &widget.Selectable{},
		})
	}

	// response cookies
	r.responseCookies = make([]keyValue, 0)
	for _, c := range res.Cookies {
		r.responseCookies = append(r.responseCookies, keyValue{
			Key:             c.Name,
			Value:           c.Value,
			keySelectable:   &widget.Selectable{},
			valueSelectable: &widget.Selectable{},
		})
	}

	r.result = dataStr
}

func (r *RestContainer) prepareBody() []byte {
	switch r.requestBodyDropDown.SelectedIndex() {
	case 0: // none
		return nil
	case 1: // json
		o, err := rest.EncodeJSON(r.requestBody.Text())
		if err != nil {
			return nil
		}
		return o
	case 2, 3: // text, xml
		return []byte(r.requestBody.Text())
	case 4: // form data
		return nil
	case 5: // binary
		return nil
	case 6: // urlencoded
		return nil
	}

	return nil
}

func (r *RestContainer) responseCopy() {
	switch r.responseTabs.Selected() {
	case 0:
		if r.result == "" {
			return
		}
		clipboard.Write(clipboard.FmtText, []byte(r.result))
		notify.Send("Response copied to clipboard", time.Second*3)
	case 1:
		if len(r.responseHeaders) == 0 {
			return
		}

		headers := ""
		for _, h := range r.responseHeaders {
			headers += fmt.Sprintf("%s: %s\n", h.Key, h.Value)
		}

		clipboard.Write(clipboard.FmtText, []byte(headers))
		notify.Send("Response headers copied to clipboard", time.Second*3)
	case 2:
		if len(r.responseCookies) == 0 {
			return
		}

		cookies := ""
		for _, c := range r.responseCookies {
			cookies += fmt.Sprintf("%s: %s\n", c.Key, c.Value)
		}
		clipboard.Write(clipboard.FmtText, []byte(cookies))
		notify.Send("Response cookies copied to clipboard", time.Second*3)
	}
}

func (r *RestContainer) onQueryParamChange(items []widgets.KeyValueItem) {
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

	items := make([]widgets.KeyValueItem, 0)
	for k, v := range queryParams {
		if len(v) > 0 {
			// Add the parameter as a new key-value item
			items = append(items, widgets.NewKeyValueItem(k, v[0], true))
		}
	}

	r.updateQueryParams = true
	r.queryParams.SetItems(items)
}

func (r *RestContainer) requestBar(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        widgets.Gray400,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	for _, ev := range r.address.Events() {
		if _, ok := ev.(widget.ChangeEvent); ok {
			if !r.updateAddress {
				r.addressChanged()
			} else {
				r.updateAddress = false
			}
		}
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
			Spacing:   layout.SpaceEnd,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.methodDropDown.Layout(gtx)
			}),
			widgets.VerticalLine(40.0),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					r.addressMutex.Lock()
					defer r.addressMutex.Unlock()
					return material.Editor(theme, r.address, "https://example.com").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if r.sendClickable.Clicked(gtx) {
						go r.Submit()
					}

					gtx.Constraints.Min.X = gtx.Dp(80)
					return r.sendButton.Layout(gtx)
				})
			}),
		)
	})
}

func (r *RestContainer) requestBodyLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, theme.TextSize, "Request body").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.requestBodyDropDown.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				switch r.requestBodyDropDown.SelectedIndex() {
				case 1, 2, 3: // json, text, xml
					hint := ""
					if r.requestBodyDropDown.SelectedIndex() == 1 {
						hint = "Enter valid json"
					} else if r.requestBodyDropDown.SelectedIndex() == 2 {
						hint = "Enter text"
					} else if r.requestBodyDropDown.SelectedIndex() == 3 {
						hint = "Enter valid xml"
					}
					return material.Editor(theme, r.requestBody, hint).Layout(gtx)
				case 4: // form data
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Start,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return r.formDataParams.WithAddLayout(gtx, "", "", theme)
							})
						}),
					)
				case 5: // binary
					return r.requestBodyBinary.Layout(gtx, theme)
				case 6: // urlencoded
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Start,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return r.urlEncodedParams.WithAddLayout(gtx, "", "", theme)
							})
						}),
					)
				default:
					return layout.Dimensions{}
				}
			})
		}),
	)
}

func (r *RestContainer) requestPostReqLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, theme.TextSize, "Action to do after request").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.postRequestDropDown.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(theme, r.postRequestBody, "").Layout(gtx)
			})
		}),
	)
}

func (r *RestContainer) requestPreReqLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, theme.TextSize, "Action to do before request").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.preRequestDropDown.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(theme, r.preRequestBody, "").Layout(gtx)
			})
		}),
	)
}

func (r *RestContainer) paramsLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.queryParams.WithAddLayout(gtx, "Query", "", theme)
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.pathParams.WithAddLayout(gtx, "Path", "path params inside bracket, for example: {id}", theme)
			})
		}),
	)
}

func (r *RestContainer) requestBodyFormDataLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.queryParams.WithAddLayout(gtx, "Query", "", theme)
			})
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
			switch r.requestTabs.Selected() {
			case 0:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.paramsLayout(gtx, theme)
				})
			case 1:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.requestBodyLayout(gtx, theme)
				})
			case 2:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return requestTabsInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.headers.WithAddLayout(gtx, "Headers", "", theme)
					})
				})
			case 3:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.requestPreReqLayout(gtx, theme)
				})

			case 4:
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.requestPostReqLayout(gtx, theme)
				})
			}

			return layout.Dimensions{}
		}),
	)
}

func (r *RestContainer) responseKeyValue(gtx layout.Context, theme *material.Theme, state *widget.List, itemType string, items []keyValue) layout.Dimensions {
	if len(items) == 0 {
		return r.messageLayout(gtx, theme, fmt.Sprintf("No %s available", itemType))
	}

	return material.List(theme, state).Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(theme, theme.TextSize, items[i].Key+":")
					l.Font.Weight = font.Bold
					l.State = items[i].keySelectable
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(theme, theme.TextSize, items[i].Value)
					l.State = items[i].valueSelectable
					return l.Layout(gtx)
				})
			}),
		)
	})
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

func (r *RestContainer) responseLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if r.result == "" {
		return r.messageLayout(gtx, theme, "No response available yet ;)")
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.responseTabs.Layout(gtx, theme)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.LabelStyle{
							Text:     r.resultStatus,
							Color:    widgets.LightGreen,
							TextSize: theme.TextSize,
							Shaper:   theme.Shaper,
						}
						l.Font.Typeface = theme.Face
						return l.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return r.copyResponseButton.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return r.saveResponseButton.Layout(gtx)
				}),
			)
		}),
		widgets.DrawLineFlex(widgets.Gray300, unit.Dp(1), unit.Dp(gtx.Constraints.Max.Y)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			switch r.responseTabs.Selected() {
			case 1:
				return r.responseKeyValue(gtx, theme, r.responseHeadersList, "headers", r.responseHeaders)
			case 2:
				return r.responseKeyValue(gtx, theme, r.responseCookiesList, "cookies", r.responseCookies)
			default:
				return material.List(theme, r.resultList).Layout(gtx, len(r.resultLines), func(gtx layout.Context, i int) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								l := material.Label(theme, theme.TextSize, fmt.Sprintf("%d", i+1))
								l.Font.Weight = font.Medium
								l.Color = widgets.Gray800
								l.Alignment = text.End
								return l.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return material.Label(theme, theme.TextSize, r.resultLines[i]).Layout(gtx)
							})
						}),
					)
				})
				//return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				//	return material.Editor(theme, r.responseBody, "").Layout(gtx)
				//})
			}
		}),
	)
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
							r.prepareLines()
							r.resultUpdated = true
						}
					}

					return r.responseLayout(gtx, theme)
				},
			)
		}),
	)
}

func (r *RestContainer) prepareLines() {
	r.resultLines = strings.Split(r.result, "\n")
}
