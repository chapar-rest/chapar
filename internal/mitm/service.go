package mitm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lqqyt2423/go-mitmproxy/cert"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"github.com/sirupsen/logrus"
)

const (
	DefaultListenAddr = "127.0.0.1:9090"
	// DefaultMaxBody is the largest body kept per request or response;
	// larger bodies stream through uncaptured.
	DefaultMaxBody int64 = 10 << 20
)

// Config is how the proxy runs.
type Config struct {
	ListenAddr   string
	MaxFlows     int
	MaxBodyBytes int64
	// IgnoreHosts are passed through without decrypting. "example.com"
	// also covers its subdomains; "*" patterns use path.Match.
	IgnoreHosts []string
}

// Service owns the proxy server, its CA and the captured flows.
type Service struct {
	caDir string
	store *Store
	apps  *appResolver

	mu      sync.Mutex
	ca      *CA
	p       *proxy.Proxy
	addr    string
	cfg     Config
	running bool
	lastErr error
	wake    func()
	conns   map[net.Conn]struct{}
	paused  atomic.Bool
}

// NewService returns a stopped proxy keeping its CA in caDir.
func NewService(caDir string) *Service {
	if os.Getenv("CHAPAR_PROXY_DEBUG") == "" {
		// go-mitmproxy logs every connection through the global logrus.
		logrus.SetOutput(io.Discard)
	}
	s := &Service{caDir: caDir, store: NewStore(DefaultMaxFlows), conns: map[net.Conn]struct{}{}}
	s.apps = newAppResolver()
	return s
}

// SetWake registers fn to run (from any goroutine) when state changes.
func (s *Service) SetWake(fn func()) {
	s.mu.Lock()
	s.wake = fn
	s.mu.Unlock()
	s.store.SetOnChange(fn)
}

// Store holds the captured flows.
func (s *Service) Store() *Store { return s.store }

// CADir is where the CA files live.
func (s *Service) CADir() string { return s.caDir }

// CA returns the CA, loading it from disk on first use. It does not
// create one; see EnsureCA.
func (s *Service) CA() (*CA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ca != nil {
		return s.ca, nil
	}
	if !CAExists(s.caDir) {
		return nil, errors.New("no CA generated")
	}
	ca, err := LoadCA(s.caDir)
	if err != nil {
		return nil, err
	}
	s.ca = ca
	return ca, nil
}

// EnsureCA loads the CA, creating it when there is none.
func (s *Service) EnsureCA() (*CA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ca != nil {
		return s.ca, nil
	}
	ca, err := LoadOrCreateCA(s.caDir)
	if err != nil {
		return nil, err
	}
	s.ca = ca
	return ca, nil
}

// RegenerateCA replaces the CA with a new one. A running proxy signs
// with the new CA from its next connection on.
func (s *Service) RegenerateCA() (*CA, error) {
	ca, err := CreateCA(s.caDir)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.ca = ca
	s.mu.Unlock()
	s.notify()
	return ca, nil
}

// DeleteCA removes the CA files. The proxy must be stopped.
func (s *Service) DeleteCA() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return errors.New("stop the proxy first")
	}
	s.ca = nil
	return RemoveCA(s.caDir)
}

// Running reports whether the proxy is listening.
func (s *Service) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Addr is the address the proxy listens on, or "" when stopped.
func (s *Service) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return ""
	}
	return s.addr
}

// LastError is why the proxy last stopped on its own, if it did.
func (s *Service) LastError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErr
}

// SetPaused stops or resumes recording new flows; traffic keeps
// flowing through the proxy either way.
func (s *Service) SetPaused(v bool) {
	s.paused.Store(v)
	s.notify()
}

// Paused reports whether recording is paused.
func (s *Service) Paused() bool { return s.paused.Load() }

// SetIgnoreHosts changes the passthrough list of a running proxy.
func (s *Service) SetIgnoreHosts(hosts []string) {
	s.mu.Lock()
	s.cfg.IgnoreHosts = append([]string(nil), hosts...)
	s.mu.Unlock()
}

// Start begins listening. It creates the CA when there is none.
func (s *Service) Start(cfg Config) error {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = DefaultListenAddr
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = DefaultMaxBody
	}
	if _, err := s.EnsureCA(); err != nil {
		return fmt.Errorf("create CA: %w", err)
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("proxy already running")
	}
	s.mu.Unlock()

	// Probe the port first: go-mitmproxy listens inside Start, which
	// blocks, so a busy port would only surface later.
	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return portError(cfg.ListenAddr, err)
	}
	addr := ln.Addr().String()
	ln.Close()

	p, err := proxy.NewProxy(&proxy.Options{
		Addr:              addr,
		StreamLargeBodies: cfg.MaxBodyBytes,
		NewCaFunc:         func() (cert.CA, error) { return caRef{s}, nil },
	})
	if err != nil {
		return err
	}
	p.AddAddon(&captureAddon{svc: s, store: s.store})
	p.AddAddon(&serverAddon{svc: s})
	p.SetShouldInterceptRule(func(req *http.Request) bool {
		return !s.ignored(hostOnly(req.Host))
	})

	s.store.SetMax(cfg.MaxFlows)
	s.mu.Lock()
	s.p = p
	s.addr = addr
	s.cfg = cfg
	s.running = true
	s.lastErr = nil
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		err := p.Start()
		errCh <- err
		s.mu.Lock()
		if s.p == p {
			s.running = false
			s.p = nil
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.lastErr = err
			}
		}
		s.mu.Unlock()
		s.notify()
	}()

	// Surface an immediate failure (the port taken since the probe).
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return portError(addr, err)
		}
	case <-time.After(50 * time.Millisecond):
	}
	s.notify()
	return nil
}

// Stop shuts the proxy down and closes open client connections,
// including tunnels the server no longer tracks.
func (s *Service) Stop() error {
	s.mu.Lock()
	p := s.p
	s.p = nil
	s.running = false
	conns := s.conns
	s.conns = map[net.Conn]struct{}{}
	s.mu.Unlock()
	if p == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := p.Shutdown(ctx)
	if err != nil {
		err = p.Close()
	}
	for c := range conns {
		c.Close()
	}
	s.notify()
	return err
}

func (s *Service) ignored(host string) bool {
	if isCertSiteHost(host) {
		return false
	}
	s.mu.Lock()
	pats := s.cfg.IgnoreHosts
	s.mu.Unlock()
	return MatchHost(pats, host)
}

// MatchHost reports whether host matches one of pats. A plain pattern
// matches the host and its subdomains; "*.example.com" matches only
// subdomains; other wildcards use path.Match.
func MatchHost(pats []string, host string) bool {
	host = strings.ToLower(host)
	for _, p := range pats {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "*.") && !strings.ContainsAny(p[2:], "*?[") {
			if strings.HasSuffix(host, p[1:]) {
				return true
			}
			continue
		}
		if strings.ContainsAny(p, "*?[") {
			if ok, _ := path.Match(p, host); ok {
				return true
			}
			continue
		}
		if host == p || strings.HasSuffix(host, "."+p) {
			return true
		}
	}
	return false
}

func (s *Service) maxBody() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.MaxBodyBytes
}

func (s *Service) notify() {
	s.mu.Lock()
	fn := s.wake
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func (s *Service) resolveApp(id uint64, f *proxy.Flow) {
	addr := clientAddr(f)
	ua := f.Request.Header.Get("User-Agent")
	go func() {
		if app := s.apps.lookup(addr, ua); app != "" {
			s.store.Update(id, func(fl *Flow) { fl.App = app })
		}
	}()
}

func portError(addr string, err error) error {
	if errors.Is(err, syscall.EADDRINUSE) {
		return fmt.Errorf("port %s is already in use; pick another port in proxy settings", portOf(addr, addr))
	}
	return err
}

// caRef lets the running proxy follow RegenerateCA.
type caRef struct{ s *Service }

func (r caRef) GetRootCA() *x509.Certificate {
	ca, _ := r.s.CA()
	if ca == nil {
		return nil
	}
	return ca.GetRootCA()
}

func (r caRef) GetCert(host string) (*tls.Certificate, error) {
	ca, err := r.s.CA()
	if err != nil {
		return nil, err
	}
	return ca.GetCert(host)
}

// serverAddon serves the CA page on direct visits and tracks client
// connections so Stop can close them.
type serverAddon struct {
	proxy.BaseAddon
	svc *Service
}

func (a *serverAddon) ClientConnected(c *proxy.ClientConn) {
	a.svc.mu.Lock()
	a.svc.conns[c.Conn] = struct{}{}
	a.svc.mu.Unlock()
}

func (a *serverAddon) ClientDisconnected(c *proxy.ClientConn) {
	a.svc.mu.Lock()
	delete(a.svc.conns, c.Conn)
	a.svc.mu.Unlock()
}

func (a *serverAddon) AccessProxyServer(req *http.Request, res http.ResponseWriter) {
	a.svc.ServeCertSite(res, req)
}
