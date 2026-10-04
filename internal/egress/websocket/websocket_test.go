package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"

	"github.com/chapar-rest/chapar/internal/domain"
)

// recorder collects a session's events.
type recorder struct {
	mu     sync.Mutex
	events []Event
	added  chan struct{}
}

func newRecorder() *recorder { return &recorder{added: make(chan struct{}, 100)} }

func (r *recorder) add(e Event) {
	r.mu.Lock()
	r.events = append(r.events, e)
	r.mu.Unlock()
	r.added <- struct{}{}
}

// waitFor returns the first event matching ok, waiting up to a second.
func (r *recorder) waitFor(t *testing.T, ok func(Event) bool) Event {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		r.mu.Lock()
		for _, e := range r.events {
			if ok(e) {
				r.mu.Unlock()
				return e
			}
		}
		r.mu.Unlock()
		select {
		case <-r.added:
		case <-deadline:
			t.Fatalf("no matching event in %+v", r.events)
		}
	}
}

func echoServer(t *testing.T, check func(r *http.Request)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		http.SetCookie(w, &http.Cookie{Name: "seen", Value: "1"})
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"chat"}})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()
		for {
			typ, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			if string(data) == "bye" {
				_ = conn.Close(4001, "asked to leave")
				return
			}
			_ = conn.Write(r.Context(), typ, data)
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func TestDialSendReceive(t *testing.T) {
	var gotAuth, gotHeader string
	url := echoServer(t, func(r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotHeader = r.Header.Get("X-Test")
	})

	rec := newRecorder()
	spec := &domain.WebSocketRequestSpec{
		URL: url,
		Headers: []domain.KeyValue{
			{Key: "X-Test", Value: "yes", Enable: true},
			{Key: "X-Off", Value: "no", Enable: false},
		},
		Auth:         domain.Auth{Type: domain.AuthTypeToken, TokenAuth: &domain.TokenAuth{Token: "t1"}},
		Subprotocols: []string{" chat ", ""},
	}
	s, hs, err := Dial(context.Background(), spec, Options{}, rec.add)
	require.NoError(t, err)
	defer s.Close()

	require.Equal(t, http.StatusSwitchingProtocols, hs.StatusCode)
	require.Equal(t, "chat", hs.Subprotocol)
	require.Equal(t, "Bearer t1", gotAuth)
	require.Equal(t, "yes", gotHeader)
	rec.waitFor(t, func(e Event) bool { return e.Kind == KindOpen })

	require.NoError(t, s.Send(domain.WebSocketFormatJSON, `{"a":1}`))
	got := rec.waitFor(t, func(e Event) bool { return e.Dir == DirReceived && e.Kind == KindText })
	require.Equal(t, `{"a":1}`, string(got.Data))

	require.NoError(t, s.Send(domain.WebSocketFormatBinary, "AAEC/w=="))
	got = rec.waitFor(t, func(e Event) bool { return e.Dir == DirReceived && e.Kind == KindBinary })
	require.Equal(t, []byte{0, 1, 2, 255}, got.Data)

	require.Error(t, s.Send(domain.WebSocketFormatJSON, `{"a":`))
	require.Error(t, s.Send(domain.WebSocketFormatBinary, "not base64!"))
}

func TestServerClose(t *testing.T) {
	rec := newRecorder()
	s, _, err := Dial(context.Background(), &domain.WebSocketRequestSpec{URL: echoServer(t, nil)}, Options{}, rec.add)
	require.NoError(t, err)

	require.NoError(t, s.Send(domain.WebSocketFormatText, "bye"))
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("session did not end")
	}
	got := rec.waitFor(t, func(e Event) bool { return e.Kind == KindClose })
	require.Contains(t, got.Info, "4001")
	require.Contains(t, got.Info, "asked to leave")
}

func TestUserClose(t *testing.T) {
	rec := newRecorder()
	s, _, err := Dial(context.Background(), &domain.WebSocketRequestSpec{URL: echoServer(t, nil)}, Options{}, rec.add)
	require.NoError(t, err)

	s.Close()
	<-s.Done()
	got := rec.waitFor(t, func(e Event) bool { return e.Kind == KindClose })
	require.Equal(t, "Disconnected", got.Info)
}

func TestHandshakeRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, hs, err := Dial(context.Background(), &domain.WebSocketRequestSpec{URL: srv.URL}, Options{}, nil)
	require.Error(t, err)
	require.NotNil(t, hs)
	require.Equal(t, http.StatusUnauthorized, hs.StatusCode)
	require.Equal(t, "Bearer", hs.ResponseHeaders["Www-Authenticate"])
}

func TestNormalizeURL(t *testing.T) {
	for in, want := range map[string]string{
		"wss://a.b/x":       "wss://a.b/x",
		"http://a.b:80/x?q": "ws://a.b:80/x?q",
		"https://a.b":       "wss://a.b",
		"a.b/ws":            "wss://a.b/ws",
	} {
		got, err := normalizeURL(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "ftp://a.b", "ws://"} {
		_, err := normalizeURL(bad)
		require.Error(t, err, bad)
	}
}
