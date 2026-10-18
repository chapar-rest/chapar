package mitm

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestHAR(t *testing.T) {
	start := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	flows := []*Flow{
		{
			State: StateDone, Start: start, End: start.Add(120 * time.Millisecond),
			Method: "POST", URL: "https://api.example.com/items?a=1&b=x%20y", Proto: "HTTP/2.0",
			ReqHeader: http.Header{"Content-Type": {"application/json"}, "Cookie": {"sid=abc"}},
			ReqBody:   []byte(`{"n":1}`),
			Status:    201, RespHeader: http.Header{"Content-Type": {"application/json"}, "Set-Cookie": {"t=1; Path=/"}},
			RespBody: []byte(`{"id":7}`), RespSize: 8, App: "curl",
		},
		{State: StateDone, Start: start, Tunnel: true, Method: "CONNECT"},
		{State: StatePending, Start: start, Method: "GET"},
		{
			State: StateDone, Start: start.Add(time.Second), End: start.Add(time.Second),
			Method: "GET", URL: "https://cdn.example.com/a.png",
			Status: 200, RespHeader: http.Header{"Content-Type": {"image/png"}},
			RespBody: []byte{0x89, 'P', 'N', 'G', 0xff},
		},
		{State: StateFailed, Start: start.Add(2 * time.Second), Method: "GET", URL: "https://down.example.com/", Error: "dial tcp: refused"},
	}
	out, err := HAR(flows, "test")
	if err != nil {
		t.Fatal(err)
	}
	var h struct {
		Log struct {
			Version string
			Entries []struct {
				Request struct {
					Method      string
					QueryString []harNV
					Cookies     []harCookie
					PostData    *harPostData
				}
				Response struct {
					Status  int
					Cookies []harCookie
					Content harContent
				}
				Error string `json:"_error"`
			}
		}
	}
	if err := json.Unmarshal(out, &h); err != nil {
		t.Fatal(err)
	}
	if h.Log.Version != "1.2" || len(h.Log.Entries) != 3 {
		t.Fatalf("version %q, %d entries", h.Log.Version, len(h.Log.Entries))
	}
	e := h.Log.Entries[0]
	if e.Request.PostData == nil || e.Request.PostData.Text != `{"n":1}` {
		t.Fatalf("postData %+v", e.Request.PostData)
	}
	if len(e.Request.QueryString) != 2 || e.Request.QueryString[1].Value != "x y" {
		t.Fatalf("query %+v", e.Request.QueryString)
	}
	if len(e.Request.Cookies) != 1 || len(e.Response.Cookies) != 1 || e.Response.Content.Text != `{"id":7}` {
		t.Fatalf("cookies/content %+v %+v", e.Request.Cookies, e.Response)
	}
	if img := h.Log.Entries[1].Response.Content; img.Encoding != "base64" {
		t.Fatalf("binary body not base64: %+v", img)
	}
	if h.Log.Entries[2].Error == "" || h.Log.Entries[2].Response.Status != 0 {
		t.Fatal("failed flow lost its error")
	}
}
