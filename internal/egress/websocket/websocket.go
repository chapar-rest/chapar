// Package websocket opens WebSocket connections for WebSocket requests.
//
// Unlike the other protocols a WebSocket request is not one exchange: Dial
// returns a Session that stays open, sends messages when asked, and reports
// everything that happens on the connection as Events.
package websocket

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
)

// Direction says which way an event went.
type Direction string

const (
	DirSent     Direction = "sent"
	DirReceived Direction = "received"
	// DirSystem is the connection itself: opened, closed, failed.
	DirSystem Direction = "system"
)

// Event kinds.
const (
	KindText   = "text"
	KindBinary = "binary"
	KindPing   = "ping"
	KindPong   = "pong"
	KindOpen   = "open"
	KindClose  = "close"
	KindError  = "error"
)

// Event is one thing that happened on a connection.
type Event struct {
	Dir  Direction
	Kind string
	// Data is the message payload, or the ping/pong payload.
	Data []byte
	// Info describes system events: who closed, with what code and reason.
	Info string
	At   time.Time
}

// IsMessage reports whether the event is a text or binary message.
func (e Event) IsMessage() bool { return e.Kind == KindText || e.Kind == KindBinary }

// Options tune a connection. The sender fills them from settings and the
// request.
type Options struct {
	// ConnectTimeout bounds the handshake. Zero means no limit.
	ConnectTimeout time.Duration
	// PingInterval sends a ping this often while connected. Zero never pings.
	PingInterval time.Duration
	// ReadLimit caps one incoming message, in bytes. Zero means 32 MiB.
	ReadLimit int64
	// InsecureSkipVerify skips TLS certificate checks.
	InsecureSkipVerify bool
	// Compression offers permessage-deflate.
	Compression bool
	// UserAgent, when set, is sent unless the request sets its own.
	UserAgent string
	// Cookies and EnvID pick the cookie jar the handshake uses. A nil store
	// sends and keeps no cookies.
	Cookies *cookies.Store
	EnvID   string
}

const defaultReadLimit = 32 << 20

// Handshake is the HTTP upgrade that opened (or failed to open) a
// connection.
type Handshake struct {
	URL             string
	StatusCode      int
	RequestHeaders  map[string]string
	ResponseHeaders map[string]string
	// Subprotocol is the one the server picked, if any.
	Subprotocol string
	Duration    time.Duration
	Timeline    []egress.TimelineStep

	Cookies      []*http.Cookie
	CookieEvents []cookies.Event
	SentCookies  []*http.Cookie
}

// Session is an open connection.
type Session struct {
	conn    *websocket.Conn
	ctx     context.Context
	cancel  context.CancelFunc
	onEvent func(Event)

	// closing is set once the user asked to close, so the server's answering
	// close is not reported as the server closing.
	closing   atomic.Bool
	closeOnce sync.Once
	done      chan struct{}
}

// Dial opens a connection for spec, which must already have its variables
// filled in. onEvent is called for every event, from the session's own
// goroutines; it must not block. When Dial fails the returned Handshake, if
// not nil, describes the server's answer.
func Dial(ctx context.Context, spec *domain.WebSocketRequestSpec, opts Options, onEvent func(Event)) (*Session, *Handshake, error) {
	if spec == nil {
		return nil, nil, errors.New("invalid websocket request")
	}
	if onEvent == nil {
		onEvent = func(Event) {}
	}

	target, err := normalizeURL(spec.URL)
	if err != nil {
		return nil, nil, err
	}

	header := http.Header{}
	for _, h := range spec.Headers {
		if !h.Enable || h.Key == "" {
			continue
		}
		header.Add(h.Key, h.Value)
	}
	applyAuth(header, spec.Auth)
	if opts.UserAgent != "" && header.Get("User-Agent") == "" {
		header.Set("User-Agent", opts.UserAgent)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify}, //nolint:gosec // the user's TLS setting
		},
	}

	// The jar sees the handshake as the http(s) request it is.
	probe, err := http.NewRequest(http.MethodGet, httpURL(target), nil)
	if err != nil {
		return nil, nil, err
	}
	probe.Header = header
	jar, err := opts.Cookies.Attach(opts.EnvID, client, probe)
	if err != nil {
		return nil, nil, fmt.Errorf("cookie jar: %w", err)
	}

	dialCtx := ctx
	if opts.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, opts.ConnectTimeout)
		defer cancel()
	}
	trace := egress.NewHTTPTraceCollector()
	dialCtx = httptrace.WithClientTrace(dialCtx, trace.ClientTrace())

	s := &Session{onEvent: onEvent, done: make(chan struct{})}
	dialOpts := &websocket.DialOptions{
		HTTPClient:   client,
		HTTPHeader:   header,
		Subprotocols: subprotocols(spec.Subprotocols),
		OnPingReceived: func(_ context.Context, payload []byte) bool {
			s.emit(Event{Dir: DirReceived, Kind: KindPing, Data: clone(payload)})
			return true
		},
		OnPongReceived: func(_ context.Context, payload []byte) {
			s.emit(Event{Dir: DirReceived, Kind: KindPong, Data: clone(payload)})
		},
	}
	if opts.Compression {
		dialOpts.CompressionMode = websocket.CompressionNoContextTakeover
	}

	start := time.Now()
	conn, resp, err := websocket.Dial(dialCtx, target, dialOpts)
	end := time.Now()

	hs := &Handshake{
		URL:            target,
		RequestHeaders: flatten(header),
		Duration:       end.Sub(start),
		Timeline:       trace.Steps(end),
	}
	if resp != nil {
		hs.StatusCode = resp.StatusCode
		hs.ResponseHeaders = flatten(resp.Header)
		hs.Cookies = resp.Cookies()
	}
	if jar != nil {
		hs.CookieEvents = jar.Events()
		hs.SentCookies = jar.Sent()
	}
	if err != nil {
		if resp == nil {
			hs = nil
		}
		return nil, hs, err
	}
	hs.Subprotocol = conn.Subprotocol()

	readLimit := opts.ReadLimit
	if readLimit <= 0 {
		readLimit = defaultReadLimit
	}
	conn.SetReadLimit(readLimit)

	s.conn = conn
	s.ctx, s.cancel = context.WithCancel(context.Background())
	info := "Connected to " + target
	if hs.Subprotocol != "" {
		info += " (subprotocol " + hs.Subprotocol + ")"
	}
	s.emit(Event{Dir: DirSystem, Kind: KindOpen, Info: info})

	go s.readLoop()
	if opts.PingInterval > 0 {
		go s.pingLoop(opts.PingInterval)
	}
	return s, hs, nil
}

// Send sends one message. body is the text as written: JSON is checked and
// sent as text, binary is decoded from base64.
func (s *Session) Send(format, body string) error {
	typ := websocket.MessageText
	data := []byte(body)
	switch format {
	case domain.WebSocketFormatJSON:
		if !json.Valid(data) {
			return errors.New("the message is not valid JSON")
		}
	case domain.WebSocketFormatBinary:
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(body), ""))
		if err != nil {
			return fmt.Errorf("a binary message must be base64: %w", err)
		}
		typ, data = websocket.MessageBinary, decoded
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	if err := s.conn.Write(ctx, typ, data); err != nil {
		return err
	}
	kind := KindText
	if typ == websocket.MessageBinary {
		kind = KindBinary
	}
	s.emit(Event{Dir: DirSent, Kind: kind, Data: data})
	return nil
}

// Close closes the connection normally and waits, briefly, for the
// server's close.
func (s *Session) Close() {
	s.closing.Store(true)
	_ = s.conn.Close(websocket.StatusNormalClosure, "")
	s.finish(nil)
}

// Done is closed once the connection is over.
func (s *Session) Done() <-chan struct{} { return s.done }

func (s *Session) readLoop() {
	for {
		typ, data, err := s.conn.Read(s.ctx)
		if err != nil {
			s.finish(err)
			return
		}
		kind := KindText
		if typ == websocket.MessageBinary {
			kind = KindBinary
		}
		s.emit(Event{Dir: DirReceived, Kind: kind, Data: data})
	}
}

func (s *Session) pingLoop(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-t.C:
			s.emit(Event{Dir: DirSent, Kind: KindPing})
			ctx, cancel := context.WithTimeout(s.ctx, every)
			// The pong arrives through OnPongReceived.
			_ = s.conn.Ping(ctx)
			cancel()
		}
	}
}

// finish reports how the connection ended, once.
func (s *Session) finish(err error) {
	s.closeOnce.Do(func() {
		var ce websocket.CloseError
		switch {
		case s.closing.Load():
			s.emit(Event{Dir: DirSystem, Kind: KindClose, Info: "Disconnected"})
		case errors.As(err, &ce):
			s.emit(Event{Dir: DirSystem, Kind: KindClose, Info: closeInfo("Server closed the connection", ce)})
		case err == nil || s.ctx.Err() != nil:
			s.emit(Event{Dir: DirSystem, Kind: KindClose, Info: "Disconnected"})
		default:
			s.emit(Event{Dir: DirSystem, Kind: KindError, Info: "Connection lost: " + err.Error()})
		}
		s.cancel()
		_ = s.conn.CloseNow()
		close(s.done)
	})
}

func (s *Session) emit(e Event) {
	if e.At.IsZero() {
		e.At = time.Now()
	}
	s.onEvent(e)
}

func closeInfo(prefix string, ce websocket.CloseError) string {
	info := fmt.Sprintf("%s: %d %s", prefix, int(ce.Code), closeCodeName(ce.Code))
	if ce.Reason != "" {
		info += " — " + ce.Reason
	}
	return info
}

func closeCodeName(code websocket.StatusCode) string {
	switch code {
	case websocket.StatusNormalClosure:
		return "Normal Closure"
	case websocket.StatusGoingAway:
		return "Going Away"
	case websocket.StatusProtocolError:
		return "Protocol Error"
	case websocket.StatusUnsupportedData:
		return "Unsupported Data"
	case websocket.StatusNoStatusRcvd:
		return "No Status"
	case websocket.StatusAbnormalClosure:
		return "Abnormal Closure"
	case websocket.StatusInvalidFramePayloadData:
		return "Invalid Payload"
	case websocket.StatusPolicyViolation:
		return "Policy Violation"
	case websocket.StatusMessageTooBig:
		return "Message Too Big"
	case websocket.StatusMandatoryExtension:
		return "Mandatory Extension"
	case websocket.StatusInternalError:
		return "Internal Error"
	case websocket.StatusServiceRestart:
		return "Service Restart"
	case websocket.StatusTryAgainLater:
		return "Try Again Later"
	case websocket.StatusBadGateway:
		return "Bad Gateway"
	}
	return ""
}

// normalizeURL accepts ws, wss, http and https URLs, and a bare host,
// which gets wss.
func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("the URL is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "wss://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	switch u.Scheme {
	case "ws", "wss":
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported URL scheme %q: use ws:// or wss://", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("the URL has no host")
	}
	return u.String(), nil
}

func httpURL(wsURL string) string {
	if rest, ok := strings.CutPrefix(wsURL, "wss://"); ok {
		return "https://" + rest
	}
	if rest, ok := strings.CutPrefix(wsURL, "ws://"); ok {
		return "http://" + rest
	}
	return wsURL
}

func applyAuth(header http.Header, auth domain.Auth) {
	switch auth.Type {
	case domain.AuthTypeToken:
		if auth.TokenAuth != nil && auth.TokenAuth.Token != "" {
			header.Set("Authorization", "Bearer "+auth.TokenAuth.Token)
		}
	case domain.AuthTypeBasic:
		if auth.BasicAuth != nil && auth.BasicAuth.Username != "" {
			cred := base64.StdEncoding.EncodeToString([]byte(auth.BasicAuth.Username + ":" + auth.BasicAuth.Password))
			header.Set("Authorization", "Basic "+cred)
		}
	case domain.AuthTypeAPIKey:
		if auth.APIKeyAuth != nil && auth.APIKeyAuth.Key != "" && auth.APIKeyAuth.Value != "" {
			header.Set(auth.APIKeyAuth.Key, auth.APIKeyAuth.Value)
		}
	}
}

func subprotocols(in []string) []string {
	var out []string
	for _, p := range in {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func flatten(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[k] = strings.Join(v, ", ")
	}
	return out
}

func clone(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return append([]byte(nil), b...)
}
