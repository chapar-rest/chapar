package mitm

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
)

// captureAddon turns go-mitmproxy flows into store entries.
type captureAddon struct {
	proxy.BaseAddon

	svc   *Service
	store *Store

	// errs holds the error a flow failed with, until the flow finishes.
	errs sync.Map // *proxy.Flow -> error
	// ids maps a tracked go-mitmproxy flow to its store ID until it finishes.
	ids sync.Map // *proxy.Flow -> uint64
}

// Request records the body as soon as it is read, so a request still
// waiting for its response can be inspected, copied and opened in full.
func (a *captureAddon) Request(f *proxy.Flow) {
	v, ok := a.ids.Load(f)
	if !ok || f.Request.Body == nil {
		return
	}
	body := capBody(decodeReq(f.Request), a.svc.maxBody())
	a.store.Update(v.(uint64), func(fl *Flow) { fl.ReqBody = body })
}

func (a *captureAddon) Requestheaders(f *proxy.Flow) {
	req := f.Request
	host := hostOnly(req.URL.Host)

	if req.Method == http.MethodConnect {
		// An intercepted CONNECT is followed by the real requests; only a
		// tunnel passed through without decrypting is worth listing.
		if f.ConnContext.Intercept {
			return
		}
		a.track(f, &Flow{
			Start:      f.StartTime,
			Method:     req.Method,
			URL:        req.URL.Host,
			Scheme:     "https",
			Host:       host,
			Port:       portOf(req.URL.Host, "443"),
			Proto:      req.Proto,
			ReqHeader:  req.Header.Clone(),
			Kind:       KindTunnel,
			Tunnel:     true,
			ClientAddr: clientAddr(f),
		})
		return
	}

	if isCertSiteHost(host) {
		f.Response = a.svc.certSiteResponse(req.Raw())
		return
	}

	scheme := req.URL.Scheme
	kind := Kind("")
	if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		kind = KindWS
		if scheme == "https" {
			scheme = "wss"
		} else {
			scheme = "ws"
		}
	}
	a.track(f, &Flow{
		Start:      f.StartTime,
		Method:     req.Method,
		URL:        req.URL.String(),
		Scheme:     scheme,
		Host:       host,
		Port:       portOf(req.URL.Host, defaultPort(req.URL.Scheme)),
		Path:       req.URL.RequestURI(),
		Proto:      req.Proto,
		ReqHeader:  req.Header.Clone(),
		Kind:       kind,
		ClientAddr: clientAddr(f),
	})
}

func (a *captureAddon) RequestError(f *proxy.Flow, err error) {
	a.errs.Store(f, err)
}

func (a *captureAddon) HTTPConnectError(f *proxy.Flow, err error) {
	a.errs.Store(f, err)
}

// track adds fl to the store and fills in the rest once f finishes.
// Waiting on Done rather than the Response hook also covers streamed
// bodies, which skip that hook, and failed requests.
func (a *captureAddon) track(f *proxy.Flow, fl *Flow) {
	if a.svc.Paused() || (a.svc.ignored(fl.Host) && !fl.Tunnel) {
		return
	}
	id := a.store.Add(fl)
	a.ids.Store(f, id)
	a.svc.resolveApp(id, f)
	go func() {
		<-f.Done()
		a.ids.Delete(f)
		a.finish(id, f)
	}()
}

func (a *captureAddon) finish(id uint64, f *proxy.Flow) {
	var ferr error
	if v, ok := a.errs.LoadAndDelete(f); ok {
		ferr = v.(error)
	}
	maxBody := a.svc.maxBody()
	a.store.Update(id, func(fl *Flow) {
		fl.End = time.Now()
		fl.State = StateDone
		if ferr != nil {
			fl.State = StateFailed
			fl.Error = ferr.Error()
		}
		if f.Request != nil {
			if f.Request.Body != nil {
				fl.ReqBody = capBody(decodeReq(f.Request), maxBody)
			} else if f.Stream {
				fl.ReqStreamed = true
			}
		}
		if r := f.Response; r != nil {
			fl.Status = r.StatusCode
			if r.Header != nil {
				fl.RespHeader = r.Header.Clone()
			}
			if r.Body != nil {
				fl.RespSize = int64(len(r.Body))
				body, err := r.DecodedBody()
				if err != nil {
					body = r.Body
				}
				fl.RespBody = capBody(body, maxBody)
			} else if f.Stream {
				fl.RespStreamed = true
				fl.RespSize = contentLength(r.Header)
			}
			if fl.Kind == "" {
				fl.Kind = Classify(r.Header.Get("Content-Type"), pathOnly(fl.Path))
			}
		} else if fl.Kind == "" {
			fl.Kind = Classify("", pathOnly(fl.Path))
		}
		if fl.Kind == KindWS && fl.Status == 0 && ferr == nil {
			fl.Status = http.StatusSwitchingProtocols
		}
	})
}

func decodeReq(r *proxy.Request) []byte {
	b, err := r.DecodedBody()
	if err != nil {
		return r.Body
	}
	return b
}

func capBody(b []byte, max int64) []byte {
	if max > 0 && int64(len(b)) > max {
		b = b[:max]
	}
	// Own the bytes: go-mitmproxy may reuse or keep its buffers.
	return append([]byte(nil), b...)
}

func contentLength(h http.Header) int64 {
	var n int64
	for _, c := range h.Get("Content-Length") {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

func clientAddr(f *proxy.Flow) string {
	if f.ConnContext == nil || f.ConnContext.ClientConn == nil || f.ConnContext.ClientConn.Conn == nil {
		return ""
	}
	return f.ConnContext.ClientConn.Conn.RemoteAddr().String()
}

func hostOnly(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return strings.ToLower(h)
	}
	return strings.ToLower(strings.Trim(hostport, "[]"))
}

func portOf(hostport, def string) string {
	if _, p, err := net.SplitHostPort(hostport); err == nil {
		return p
	}
	return def
}

func defaultPort(scheme string) string {
	if scheme == "https" {
		return "443"
	}
	return "80"
}

func pathOnly(p string) string {
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		return p[:i]
	}
	return p
}
