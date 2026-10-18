package mitm_test

import (
	"net/http"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/mitm"
)

// Open in Chapar runs a captured flow through the curl importer; the
// request it makes must match what was captured.
func TestCurlImportsAsChaparRequest(t *testing.T) {
	f := &mitm.Flow{
		Method: "PUT", URL: "https://api.example.com/items/7?expand=owner",
		ReqHeader: http.Header{
			"Content-Type":    {"application/json"},
			"X-Trace":         {`id="a b"`},
			"Accept-Encoding": {"gzip"},
		},
		ReqBody: []byte(`{"name":"O'Brien"}`),
	}
	req, err := importer.ParseCurl(mitm.Curl(f))
	if err != nil {
		t.Fatalf("%v\n%s", err, mitm.Curl(f))
	}
	h := req.Spec.HTTP
	if h.Method != "PUT" || h.URL != f.URL {
		t.Fatalf("method %q url %q", h.Method, h.URL)
	}
	if h.Request.Body.Type != domain.RequestBodyTypeJSON || h.Request.Body.Data != `{"name":"O'Brien"}` {
		t.Fatalf("body %+v", h.Request.Body)
	}
	found := false
	for _, kv := range h.Request.Headers {
		if kv.Key == "X-Trace" && kv.Value == `id="a b"` {
			found = true
		}
	}
	if !found {
		t.Fatalf("headers %+v", h.Request.Headers)
	}
}
