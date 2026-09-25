package sender

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/scripting"
)

// fakeScripts answers each phase with a fixed result and records params.
type fakeScripts struct {
	results map[scripting.Phase]*scripting.ExecResult
	params  map[scripting.Phase]*scripting.ExecParams
}

func (f *fakeScripts) Execute(_ context.Context, _ string, p *scripting.ExecParams) (*scripting.ExecResult, error) {
	if f.params == nil {
		f.params = map[scripting.Phase]*scripting.ExecParams{}
	}
	f.params[p.Phase] = p
	if r := f.results[p.Phase]; r != nil {
		return r, nil
	}
	return &scripting.ExecResult{}, nil
}

func newTestSender(scripts Scripts, colls map[string]*domain.Collection) *Service {
	s := New(nil, nil, func(id string) *domain.Collection { return colls[id] }, nil)
	s.SetExecutor(scripts)
	s.scriptingOn = func() bool { return true }
	return s
}

func echoServer(t *testing.T) (*httptest.Server, *http.Header) {
	t.Helper()
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Header().Set("X-Reply", "yes")
		_, _ = w.Write([]byte(`{"token": "new"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func scriptedRequest(url string) *domain.Request {
	r := domain.NewHTTPRequest("r")
	r.Spec.HTTP.Method = "GET"
	r.Spec.HTTP.URL = url
	r.Spec.HTTP.Request.Headers = []domain.KeyValue{{Key: "X-Saved", Value: "1", Enable: true}}
	r.Spec.HTTP.Request.PreRequest = domain.PreRequest{Type: domain.PrePostTypePython, Script: "pre"}
	r.Spec.HTTP.Request.PostRequest = domain.PostRequest{Type: domain.PrePostTypePython, Script: "post"}
	return r
}

func TestPreScriptChangesOnlyTheSentCopy(t *testing.T) {
	srv, got := echoServer(t)
	headers := []scripting.Pair{{"X-Saved", "1"}, {"X-Sig", "abc"}}
	scripts := &fakeScripts{results: map[scripting.Phase]*scripting.ExecResult{
		scripting.PhasePre: {
			Request: &scripting.RequestChanges{Headers: &headers},
			EnvSet:  map[string]any{"pre": "set"},
			Prints:  []string{"from pre"},
		},
		scripting.PhasePost: {
			EnvSet: map[string]any{"post": "set"},
			Tests:  []scripting.TestResult{{Name: "status", Error: "nope"}},
		},
	}}
	s := newTestSender(scripts, nil)
	req := scriptedRequest(srv.URL)
	env := domain.NewEnvironment("dev")

	res, err := s.Send(req, env)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got.Get("X-Sig") != "abc" || got.Get("X-Saved") != "1" {
		t.Errorf("server got headers %v", *got)
	}
	if h := req.Spec.HTTP.Request.Headers; len(h) != 1 {
		t.Errorf("saved request changed: %+v", h)
	}
	if env.GetKeyValues()["pre"] != "set" || env.GetKeyValues()["post"] != "set" {
		t.Errorf("env = %v", env.GetKeyValues())
	}

	// The post script sees the request as sent and the response.
	post := scripts.params[scripting.PhasePost]
	if post.Res == nil || post.Res.StatusCode != 200 || post.Res.Body != `{"token": "new"}` {
		t.Errorf("post response = %+v", post.Res)
	}
	if len(post.Req.Headers) != 2 {
		t.Errorf("post request headers = %v", post.Req.Headers)
	}

	pre, last := res.Timeline[0], res.Timeline[len(res.Timeline)-1]
	if !strings.Contains(pre.Detail, "from pre") || pre.Err != "" {
		t.Errorf("pre step = %+v", pre)
	}
	if !strings.Contains(last.Detail, "✗ status: nope") || last.Err != "1 test failed" {
		t.Errorf("post step = %+v", last)
	}
}

func TestPreScriptSeesCollectionHeadersAndCanRemoveThem(t *testing.T) {
	srv, got := echoServer(t)
	colls := map[string]*domain.Collection{"c": {Spec: domain.ColSpec{Headers: []domain.KeyValue{
		{Key: "X-Team", Value: "core", Enable: true},
	}}}}
	headers := []scripting.Pair{{"X-Saved", "1"}}
	scripts := &fakeScripts{results: map[scripting.Phase]*scripting.ExecResult{
		scripting.PhasePre: {Request: &scripting.RequestChanges{Headers: &headers}},
	}}
	s := newTestSender(scripts, colls)
	req := scriptedRequest(srv.URL)
	req.CollectionID = "c"

	if _, err := s.Send(req, nil); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if pre := scripts.params[scripting.PhasePre]; len(pre.Req.Headers) != 2 {
		t.Errorf("pre saw headers %v, want the collection's merged in", pre.Req.Headers)
	}
	if got.Get("X-Team") != "" {
		t.Errorf("removed collection header was sent: %v", *got)
	}
}

func TestPreScriptSkip(t *testing.T) {
	srv, got := echoServer(t)
	scripts := &fakeScripts{results: map[scripting.Phase]*scripting.ExecResult{
		scripting.PhasePre: {Skip: true, SkipReason: "no token"},
	}}
	s := newTestSender(scripts, nil)
	_, err := s.Send(scriptedRequest(srv.URL), nil)
	if err == nil || !strings.Contains(err.Error(), "no token") {
		t.Fatalf("err = %v, want skip", err)
	}
	if *got != nil {
		t.Error("request was sent")
	}
}

func TestPreScriptErrorStopsSend(t *testing.T) {
	srv, got := echoServer(t)
	scripts := &fakeScripts{results: map[scripting.Phase]*scripting.ExecResult{
		scripting.PhasePre: {Error: &scripting.ScriptError{Type: "KeyError", Message: "'x'", Line: 2}},
	}}
	s := newTestSender(scripts, nil)
	res, err := s.Send(scriptedRequest(srv.URL), nil)
	var se *scripting.ScriptError
	if !errors.As(err, &se) || se.Line != 2 {
		t.Fatalf("err = %v, want the script error", err)
	}
	if *got != nil {
		t.Error("request was sent")
	}
	if res.Timeline[0].Err != "line 2: KeyError: 'x'" {
		t.Errorf("step = %+v", res.Timeline[0])
	}
}

func TestScriptingDisabled(t *testing.T) {
	srv, _ := echoServer(t)
	s := newTestSender(&fakeScripts{}, nil)
	s.scriptingOn = func() bool { return false }
	if _, err := s.Send(scriptedRequest(srv.URL), nil); !errors.Is(err, ErrScriptingDisabled) {
		t.Fatalf("err = %v, want ErrScriptingDisabled", err)
	}
}

func TestFailedExtractKeepsResponse(t *testing.T) {
	srv, _ := echoServer(t)
	s := newTestSender(&fakeScripts{}, nil)
	req := domain.NewHTTPRequest("r")
	req.Spec.HTTP.Method = "GET"
	req.Spec.HTTP.URL = srv.URL
	req.Spec.HTTP.Request.Variables = []domain.Variable{
		{TargetEnvVariable: "id", From: domain.VariableFromBody, JsonPath: "$.id", Enable: true},
		{TargetEnvVariable: "token", From: domain.VariableFromBody, JsonPath: "$.token", Enable: true},
	}
	env := domain.NewEnvironment("dev")

	res, err := s.Send(req, env)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.StatusCode != 200 || string(res.Body) != `{"token": "new"}` {
		t.Errorf("response = %d %q", res.StatusCode, res.Body)
	}
	if res.PostRequestError == nil || !strings.Contains(res.PostRequestError.Error(), "extract id from $.id") {
		t.Fatalf("PostRequestError = %v, want the extract error", res.PostRequestError)
	}
	// The failing rule does not block the others.
	if got := env.GetKeyValues()["token"]; got != "new" {
		t.Errorf("token = %v, want new", got)
	}
	if last := res.Timeline[len(res.Timeline)-1]; last.Name != "Post-request" || last.Err == "" {
		t.Errorf("post step = %+v", last)
	}
}
