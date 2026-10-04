package sender

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/scripting"
)

func TestConnectWebSocket(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()
		_, _, _ = conn.Read(r.Context())
	}))
	defer srv.Close()

	headers := []scripting.Pair{{"X-Col", "c"}, {"X-Saved", "{{who}}"}, {"X-Sig", "abc"}}
	scripts := &fakeScripts{results: map[scripting.Phase]*scripting.ExecResult{
		scripting.PhasePre: {Request: &scripting.RequestChanges{Headers: &headers}},
	}}
	cols := map[string]*domain.Collection{"col": {Spec: domain.ColSpec{
		Headers: []domain.KeyValue{{Key: "X-Col", Value: "c", Enable: true}},
	}}}
	s := newTestSender(scripts, cols)

	req := domain.NewWebSocketRequest("ws")
	req.CollectionID = "col"
	ws := req.Spec.WebSocket
	ws.URL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/{{path}}"
	ws.Headers = []domain.KeyValue{{Key: "X-Saved", Value: "{{who}}", Enable: true}}
	ws.PreRequest = domain.PreRequest{Type: domain.PrePostTypePython, Script: "pre"}

	env := domain.NewEnvironment("dev")
	env.SetKey("who", "me")
	env.SetKey("path", "socket")

	session, res, err := s.ConnectWebSocket(context.Background(), req, env, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	if res.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if got.Get("X-Saved") != "me" || got.Get("X-Sig") != "abc" || got.Get("X-Col") != "c" {
		t.Fatalf("headers = %v", got)
	}
	// The script saw the handshake as a GET with the collection headers.
	pre := scripts.params[scripting.PhasePre]
	if pre.Req.Method != "GET" || len(pre.Req.Headers) != 2 {
		t.Fatalf("script request = %+v", pre.Req)
	}
	// The saved request is not changed.
	if len(req.Spec.WebSocket.Headers) != 1 || req.Spec.WebSocket.Headers[0].Value != "{{who}}" {
		t.Fatalf("saved headers changed: %+v", req.Spec.WebSocket.Headers)
	}

	if got := s.ResolveText(`{"to":"{{who}}"}`, env); got != `{"to":"me"}` {
		t.Fatalf("ResolveText = %s", got)
	}

	if _, err := s.Send(req, env); !errors.Is(err, ErrWebSocketNotSendable) {
		t.Fatalf("Send = %v, want ErrWebSocketNotSendable", err)
	}
}

func TestConnectWebSocketRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()

	req := domain.NewWebSocketRequest("ws")
	req.Spec.WebSocket.URL = srv.URL
	_, res, err := newTestSender(nil, nil).ConnectWebSocket(context.Background(), req, nil, nil)
	if err == nil || res == nil || res.StatusCode != http.StatusForbidden {
		t.Fatalf("err %v, res %+v; want a 403 handshake", err, res)
	}
}
