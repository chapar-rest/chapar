package mitm

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// HAR 1.2, as described at http://www.softwareishard.com/blog/har-12-spec/.
// Only the fields Chapar has are filled; the rest use the spec's "unknown"
// values (-1) or are left out where they are optional.

type harFile struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Version string     `json:"version"`
	Creator harCreator `json:"creator"`
	Entries []harEntry `json:"entries"`
}

type harCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type harEntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         harRequest  `json:"request"`
	Response        harResponse `json:"response"`
	Cache           struct{}    `json:"cache"`
	Timings         harTimings  `json:"timings"`
	ServerIPAddress string      `json:"serverIPAddress,omitempty"`
	Error           string      `json:"_error,omitempty"`
	App             string      `json:"_app,omitempty"`
}

type harRequest struct {
	Method      string       `json:"method"`
	URL         string       `json:"url"`
	HTTPVersion string       `json:"httpVersion"`
	Cookies     []harCookie  `json:"cookies"`
	Headers     []harNV      `json:"headers"`
	QueryString []harNV      `json:"queryString"`
	PostData    *harPostData `json:"postData,omitempty"`
	HeadersSize int          `json:"headersSize"`
	BodySize    int          `json:"bodySize"`
}

type harResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Cookies     []harCookie `json:"cookies"`
	Headers     []harNV     `json:"headers"`
	Content     harContent  `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int64       `json:"bodySize"`
}

type harNV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
	Encoding string `json:"_encoding,omitempty"`
}

type harContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

type harTimings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// HAR encodes flows as a HAR 1.2 log. Pending flows and tunnels that were
// not decrypted are left out: they have no request or response to show.
// Bodies are the decoded ones Chapar kept; a binary body is base64.
func HAR(flows []*Flow, creatorVersion string) ([]byte, error) {
	entries := make([]harEntry, 0, len(flows))
	for _, f := range flows {
		if f.State == StatePending || f.Tunnel {
			continue
		}
		entries = append(entries, harEntryOf(f))
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].StartedDateTime < entries[j].StartedDateTime
	})
	return json.MarshalIndent(harFile{Log: harLog{
		Version: "1.2",
		Creator: harCreator{Name: "Chapar", Version: creatorVersion},
		Entries: entries,
	}}, "", "  ")
}

func harEntryOf(f *Flow) harEntry {
	ms := float64(f.Duration().Microseconds()) / 1000
	e := harEntry{
		StartedDateTime: f.Start.UTC().Format(time.RFC3339Nano),
		Time:            ms,
		Request: harRequest{
			Method:      f.Method,
			URL:         f.URL,
			HTTPVersion: protoOr(f.Proto),
			Cookies:     requestCookies(f.ReqHeader),
			Headers:     harHeaders(f.ReqHeader),
			QueryString: harQuery(f.URL),
			HeadersSize: -1,
			BodySize:    len(f.ReqBody),
		},
		Response: harResponse{
			Status:      f.Status,
			StatusText:  http.StatusText(f.Status),
			HTTPVersion: protoOr(f.Proto),
			Cookies:     responseCookies(f.RespHeader),
			Headers:     harHeaders(f.RespHeader),
			RedirectURL: f.RespHeader.Get("Location"),
			HeadersSize: -1,
			BodySize:    f.RespSize,
			Content: harContent{
				Size:     len(f.RespBody),
				MimeType: f.RespHeader.Get("Content-Type"),
			},
		},
		Timings: harTimings{Send: 0, Wait: ms, Receive: 0},
		Error:   f.Error,
		App:     f.App,
	}
	if f.State == StateFailed && f.Status == 0 {
		e.Response.BodySize = 0
	}
	if len(f.ReqBody) > 0 {
		text, enc := harText(f.ReqBody)
		e.Request.PostData = &harPostData{
			MimeType: f.ReqHeader.Get("Content-Type"),
			Text:     text,
			Encoding: enc,
		}
	} else if f.ReqStreamed {
		e.Request.BodySize = -1
	}
	switch {
	case len(f.RespBody) > 0:
		e.Response.Content.Text, e.Response.Content.Encoding = harText(f.RespBody)
	case f.RespStreamed:
		e.Response.Content.Comment = "Body was larger than the capture limit and was not kept."
	}
	return e
}

// harText returns body as text, or base64 with its encoding when it
// isn't valid UTF-8.
func harText(b []byte) (text, encoding string) {
	if utf8.Valid(b) {
		return string(b), ""
	}
	return base64.StdEncoding.EncodeToString(b), "base64"
}

func harHeaders(h http.Header) []harNV {
	out := []harNV{}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range h[k] {
			out = append(out, harNV{Name: k, Value: v})
		}
	}
	return out
}

func harQuery(raw string) []harNV {
	out := []harNV{}
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" {
		return out
	}
	for _, pair := range strings.Split(u.RawQuery, "&") {
		if pair == "" {
			continue
		}
		k, v, _ := strings.Cut(pair, "=")
		if dk, err := url.QueryUnescape(k); err == nil {
			k = dk
		}
		if dv, err := url.QueryUnescape(v); err == nil {
			v = dv
		}
		out = append(out, harNV{Name: k, Value: v})
	}
	return out
}

func requestCookies(h http.Header) []harCookie {
	out := []harCookie{}
	r := http.Request{Header: h}
	for _, c := range r.Cookies() {
		out = append(out, harCookie{Name: c.Name, Value: c.Value})
	}
	return out
}

func responseCookies(h http.Header) []harCookie {
	out := []harCookie{}
	r := http.Response{Header: h}
	for _, c := range r.Cookies() {
		out = append(out, harCookie{Name: c.Name, Value: c.Value})
	}
	return out
}

func protoOr(p string) string {
	if p == "" {
		return "HTTP/1.1"
	}
	return p
}
