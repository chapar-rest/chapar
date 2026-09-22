package scripting

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

// fakeServer serves /health with api and records the last execute body.
func fakeServer(t *testing.T, api []int, execute http.HandlerFunc) (*PythonExecutor, *executeRequestBody) {
	t.Helper()
	var got executeRequestBody
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "version": "9.9.9", "api": api})
	})
	mux.HandleFunc("/v2/execute", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		execute(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	_, portStr, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	port, _ := strconv.Atoi(portStr)
	return NewPythonExecutor(domain.ScriptingConfig{Port: port}), &got
}

func TestExecuteSendsV2Body(t *testing.T) {
	p, got := fakeServer(t, []int{1, 2}, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(tokenHeader) != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"request": {"headers": [["a", "b"]]}, "env_set": {"x": 1},
			"env_unset": [], "prints": ["hi"], "tests": [], "skip": false, "skip_reason": "", "error": null}`))
	})
	if err := p.Init(p.cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	p.token = "tok"

	e := domain.NewEnvironment("dev")
	e.SetKey("k", "v")
	res, err := p.Execute(context.Background(), "print(1)", &ExecParams{
		Phase: PhasePre, Protocol: ProtocolGRPC, Env: e,
		Req: &RequestData{URL: "localhost:1"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got.Phase != PhasePre || got.Protocol != ProtocolGRPC || got.Script != "print(1)" ||
		got.Environment.Name != "dev" || got.Environment.Vars["k"] != "v" || got.Request.URL != "localhost:1" ||
		got.TimeoutMS != ScriptTimeout.Milliseconds() || got.Response != nil {
		t.Errorf("sent %+v", got)
	}
	if res.Request == nil || (*res.Request.Headers)[0] != (Pair{"a", "b"}) || res.EnvSet["x"] != float64(1) ||
		res.Prints[0] != "hi" || res.Error != nil {
		t.Errorf("result %+v", res)
	}
}

func TestExecuteScriptErrorIsInResult(t *testing.T) {
	p, _ := fakeServer(t, []int{2}, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"request": null, "env_set": {}, "env_unset": [], "prints": ["before"],
			"tests": [], "skip": false, "skip_reason": "",
			"error": {"type": "KeyError", "message": "'x'", "line": 3, "traceback": "tb"}}`))
	})
	res, err := p.Execute(context.Background(), "", &ExecParams{Req: &RequestData{}})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Error == nil || res.Error.Line != 3 || res.Prints[0] != "before" {
		t.Errorf("result %+v", res)
	}
}

func TestExecuteReportsExecutorErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   string
	}{
		{http.StatusUnauthorized, "", "rejected Chapar's token"},
		{http.StatusBadRequest, `{"message": "phase must be one of pre, post"}`, "phase must be"},
		{http.StatusBadGateway, "<html>boom</html>", "502"},
	} {
		p, _ := fakeServer(t, []int{2}, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		})
		_, err := p.Execute(context.Background(), "", &ExecParams{Req: &RequestData{}})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("status %d: err = %v, want %q", tc.status, err, tc.want)
		}
	}
}

func TestInitRejectsOldExecutor(t *testing.T) {
	p, _ := fakeServer(t, []int{1}, nil)
	err := p.Init(p.cfg)
	if err == nil || !strings.Contains(err.Error(), "too old") {
		t.Fatalf("Init err = %v, want too old", err)
	}
}

func TestPhaseDefaultsFromResponse(t *testing.T) {
	p, got := fakeServer(t, []int{2}, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	if _, err := p.Execute(context.Background(), "", &ExecParams{Req: &RequestData{}, Res: &ResponseData{StatusCode: 200}}); err != nil {
		t.Fatal(err)
	}
	if got.Phase != PhasePost || got.Protocol != ProtocolHTTP || got.Response.StatusCode != 200 {
		t.Errorf("sent %+v", got)
	}
}
