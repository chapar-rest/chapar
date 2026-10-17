package mitm

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// startProxy runs a proxy on a free port with a CA in a temp dir.
func startProxy(t *testing.T, cfg Config) *Service {
	t.Helper()
	s := NewService(t.TempDir())
	cfg.ListenAddr = "127.0.0.1:0"
	if err := s.Start(cfg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Stop() })
	return s
}

// client sends through the proxy and trusts only its CA.
func client(t *testing.T, s *Service) *http.Client {
	t.Helper()
	ca, err := s.CA()
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.GetRootCA())
	pu, _ := url.Parse("http://" + s.Addr())
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(pu),
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}
}

// waitFlow waits until the store holds a finished flow matching ok.
func waitFlow(t *testing.T, s *Service, ok func(*Flow) bool) *Flow {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, f := range s.Store().Snapshot() {
			if f.State != StatePending && ok(f) {
				return f
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no matching flow; have %d", s.Store().Len())
	return nil
}

func TestCapturesHTTPSAndDecodesBody(t *testing.T) {
	up := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		gz.Write([]byte(`{"echo":` + string(body) + `}`))
		gz.Close()
	}))
	defer up.Close()

	// The upstream uses httptest's own CA; let the proxy accept it.
	s := startProxy(t, Config{})
	s.mu.Lock()
	s.p.Opts.SslInsecure = true
	s.mu.Unlock()

	// By name, not IP: the proxy picks its certificate from the SNI,
	// which clients don't send for IP addresses.
	target := strings.Replace(up.URL, "127.0.0.1", "localhost", 1)
	resp, err := client(t, s).Post(target+"/api/items?x=1", "application/json", strings.NewReader(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	f := waitFlow(t, s, func(f *Flow) bool { return f.Method == http.MethodPost })
	if f.State != StateDone || f.Status != 200 {
		t.Fatalf("state=%v status=%d err=%q", f.State, f.Status, f.Error)
	}
	if f.Scheme != "https" || f.Path != "/api/items?x=1" || f.Host != "localhost" {
		t.Fatalf("scheme=%q path=%q host=%q", f.Scheme, f.Path, f.Host)
	}
	if string(f.ReqBody) != `{"a":1}` {
		t.Fatalf("req body %q", f.ReqBody)
	}
	if string(f.RespBody) != `{"echo":{"a":1}}` {
		t.Fatalf("resp body %q", f.RespBody)
	}
	if f.Kind != KindJSON {
		t.Fatalf("kind %q", f.Kind)
	}
}

func TestCertSiteThroughProxyAndDirect(t *testing.T) {
	s := startProxy(t, Config{})
	ca, _ := s.CA()

	resp, err := client(t, s).Get("http://" + CertSiteHost + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(page), ca.Fingerprint()) {
		t.Fatalf("status %d, page lacks fingerprint", resp.StatusCode)
	}

	resp, err = client(t, s).Get("http://" + CertSiteHost + "/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	der, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Equal(der, ca.CertDER()) {
		t.Fatal("ca.crt is not the CA in DER")
	}

	// Visiting the proxy's own address serves the same page.
	resp, err = http.Get("http://" + s.Addr() + "/ca.pem")
	if err != nil {
		t.Fatal(err)
	}
	pemBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Equal(pemBody, ca.CertPEM()) {
		t.Fatal("direct ca.pem mismatch")
	}

	for _, f := range s.Store().Snapshot() {
		if f.Host == CertSiteHost {
			t.Fatal("cert site requests should not be captured")
		}
	}
}

func TestIgnoredHostIsTunneled(t *testing.T) {
	up := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	}))
	defer up.Close()

	s := startProxy(t, Config{IgnoreHosts: []string{"127.0.0.1"}})
	pu, _ := url.Parse("http://" + s.Addr())
	pool := x509.NewCertPool()
	pool.AddCert(up.Certificate())
	c := &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(pu),
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}}
	// Trusting only the upstream's own cert proves nothing decrypted it.
	resp, err := c.Get(up.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	c.CloseIdleConnections()

	f := waitFlow(t, s, func(f *Flow) bool { return f.Tunnel })
	if f.Kind != KindTunnel || f.Method != http.MethodConnect {
		t.Fatalf("kind=%q method=%q", f.Kind, f.Method)
	}
}

func TestStopFreesPort(t *testing.T) {
	s := startProxy(t, Config{})
	addr := s.Addr()
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if s.Running() {
		t.Fatal("still running")
	}
	if err := s.Start(Config{ListenAddr: addr}); err != nil {
		t.Fatalf("restart on same port: %v", err)
	}
}

func TestStoreCapAndUpdate(t *testing.T) {
	st := NewStore(3)
	var ids []uint64
	for i := 0; i < 5; i++ {
		ids = append(ids, st.Add(&Flow{Method: "GET"}))
	}
	if st.Len() != 3 {
		t.Fatalf("len %d", st.Len())
	}
	if _, ok := st.Get(ids[0]); ok {
		t.Fatal("oldest flow should be dropped")
	}
	st.Update(ids[4], func(f *Flow) { f.Status = 201 })
	f, _ := st.Get(ids[4])
	if f.Status != 201 {
		t.Fatal("update lost")
	}
	st.Remove(map[uint64]bool{ids[3]: true})
	if _, ok := st.Get(ids[3]); ok || st.Len() != 2 {
		t.Fatal("remove failed")
	}
	if f, ok := st.Get(ids[4]); !ok || f.Status != 201 {
		t.Fatal("index broken after remove")
	}
}

func TestMatchHost(t *testing.T) {
	cases := []struct {
		pats []string
		host string
		want bool
	}{
		{[]string{"apple.com"}, "apple.com", true},
		{[]string{"apple.com"}, "api.apple.com", true},
		{[]string{"apple.com"}, "notapple.com", false},
		{[]string{"*.apple.com"}, "apple.com", false},
		{[]string{"*.apple.com"}, "x.apple.com", true},
		{[]string{"api-*.example.com"}, "api-1.example.com", true},
	}
	for _, c := range cases {
		if got := MatchHost(c.pats, c.host); got != c.want {
			t.Errorf("MatchHost(%v, %q) = %v", c.pats, c.host, got)
		}
	}
}

func TestClassify(t *testing.T) {
	if Classify("application/vnd.api+json; charset=utf-8", "/x") != KindJSON {
		t.Error("+json")
	}
	if Classify("", "/static/app.css") != KindCSS {
		t.Error("css by ext")
	}
	if Classify("image/png", "/") != KindImage {
		t.Error("image")
	}
}

func TestUAProduct(t *testing.T) {
	if uaProduct("curl/8.4.0") != "curl" {
		t.Error("curl")
	}
	if uaProduct("Mozilla/5.0 (Macintosh) AppleWebKit/537.36 Chrome/120 Safari/537.36") != "Chrome" {
		t.Error("chrome")
	}
}

func TestParseScutilProxy(t *testing.T) {
	out := []byte("<dictionary> {\n  HTTPEnable : 1\n  HTTPSEnable : 1\n  HTTPSPort : 9090\n  HTTPSProxy : 127.0.0.1\n}\n")
	sp := parseScutilProxy(out)
	if !sp.Enabled || sp.Host != "127.0.0.1" || sp.Port != 9090 || !sp.PointsTo(9090) {
		t.Fatalf("%+v", sp)
	}
}

func TestCapturesPlainHTTP(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<p>hi</p>")
	}))
	defer up.Close()
	s := startProxy(t, Config{})
	resp, err := client(t, s).Get(up.URL + "/page")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	f := waitFlow(t, s, func(f *Flow) bool { return f.Path == "/page" })
	if f.Scheme != "http" || f.Status != 200 || f.Kind != KindHTML || string(f.RespBody) != "<p>hi</p>" {
		t.Fatalf("%+v", f)
	}
}
