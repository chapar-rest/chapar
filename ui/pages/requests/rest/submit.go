package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gioui.org/widget"
	"github.com/dustin/go-humanize"
	"github.com/mirzakhany/chapar/internal/rest"
)

func (r *Container) Submit() {
	method := r.methodDropDown.GetSelected().Text
	address := r.address.Text()
	headers := make(map[string]string)
	for _, h := range r.headers.GetItems() {
		if h.Key == "" || !h.Active || h.Value == "" {
			continue
		}
		headers[h.Key] = h.Value
	}

	body, contentType := r.prepareBody()
	headers["Content-Type"] = contentType

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

func (r *Container) prepareBody() ([]byte, string) {
	switch r.requestBodyDropDown.SelectedIndex() {
	case 0: // none
		return nil, ""
	case 1: // json
		return []byte(r.requestBody.Code()), "application/json"
	case 2, 3: // text, xml
		return []byte(r.requestBody.Code()), "application/text"
	case 4: // form data
		return nil, "application/form-data"
	case 5: // binary
		return nil, "application/octet-stream"
	case 6: // urlencoded
		return nil, "application/x-www-form-urlencoded"
	}

	return nil, ""
}
