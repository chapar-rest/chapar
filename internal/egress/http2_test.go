package egress

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTP2TransportSpeaksHTTP2(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("server saw %s, want HTTP/2", r.Proto)
		}
	})

	h2c := httptest.NewUnstartedServer(handler)
	h2c.Config.Protocols = new(http.Protocols)
	h2c.Config.Protocols.SetUnencryptedHTTP2(true)
	h2c.Start()
	defer h2c.Close()

	tlsSrv := httptest.NewUnstartedServer(handler)
	tlsSrv.EnableHTTP2 = true
	tlsSrv.StartTLS()
	defer tlsSrv.Close()

	client := &http.Client{Transport: NewHTTP2Transport(64<<20, &tls.Config{InsecureSkipVerify: true})}
	for _, url := range []string{h2c.URL, tlsSrv.URL} {
		resp, err := client.Get(url)
		if err != nil {
			t.Fatalf("GET %s: %v", url, err)
		}
		_ = resp.Body.Close()
		if resp.ProtoMajor != 2 {
			t.Errorf("GET %s used %s, want HTTP/2", url, resp.Proto)
		}
	}
}
