package mitm

import (
	_ "embed"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
)

// CertSiteHost is the made-up host that serves the CA download page to
// any client sending its traffic through the proxy.
const CertSiteHost = "chapar.mitm"

//go:embed certsite.html
var certSiteHTML string

var certSiteTmpl = template.Must(template.New("certsite").Parse(certSiteHTML))

func isCertSiteHost(host string) bool { return host == CertSiteHost }

type certSitePage struct {
	Name        string
	Fingerprint string
	Expires     string
	Platform    string
	ProxyAddr   string
}

// ServeCertSite serves the CA download page: "/" is the page, and
// "/ca.pem", "/ca.crt" and "/ca.cer" are the certificate.
func (s *Service) ServeCertSite(w http.ResponseWriter, r *http.Request) {
	ca, err := s.CA()
	if err != nil {
		http.Error(w, "Chapar has no CA yet: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	switch strings.TrimSuffix(r.URL.Path, "/") {
	case "", "/index.html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = certSiteTmpl.Execute(w, certSitePage{
			Name:        ca.Name(),
			Fingerprint: ca.Fingerprint(),
			Expires:     ca.NotAfter().Format("2 Jan 2006"),
			Platform:    platformFromUA(r.UserAgent()),
			ProxyAddr:   firstOr(s.ClientAddrs(), ""),
		})
	case "/ca.pem":
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Header().Set("Content-Disposition", `attachment; filename="chapar-ca.pem"`)
		_, _ = w.Write(ca.CertPEM())
	case "/ca.crt", "/ca.cer":
		// DER with this type makes iOS offer a profile install and
		// Android hand the file to its certificate installer.
		name := "chapar-ca" + r.URL.Path[strings.LastIndex(r.URL.Path, "."):]
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		_, _ = w.Write(ca.CertDER())
	default:
		http.NotFound(w, r)
	}
}

// certSiteResponse answers a proxied request for CertSiteHost without
// sending it anywhere.
func (s *Service) certSiteResponse(r *http.Request) *proxy.Response {
	rec := httptest.NewRecorder()
	s.ServeCertSite(rec, r)
	return &proxy.Response{
		StatusCode: rec.Code,
		Header:     rec.Header(),
		Body:       rec.Body.Bytes(),
	}
}

func firstOr(list []string, def string) string {
	if len(list) == 0 {
		return def
	}
	return list[0]
}

func platformFromUA(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		return "ios"
	case strings.Contains(ua, "android"):
		return "android"
	case strings.Contains(ua, "windows"):
		return "windows"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		return "macos"
	case strings.Contains(ua, "linux") || strings.Contains(ua, "x11"):
		return "linux"
	}
	return ""
}
