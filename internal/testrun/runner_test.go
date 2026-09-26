package testrun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/scripting"
)

// todoServer is a small API the runner tests talk to.
type todoServer struct {
	*httptest.Server
	mu       sync.Mutex
	deleted  []string
	echoed   *http.Request
	echoBody string
	flaky    atomic.Int32
}

func newTodoServer(t *testing.T) *todoServer {
	t.Helper()
	s := &todoServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Title string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"data":{"id":"t1","title":%q,"token":"tok-1"}}`, in.Title)
	})
	mux.HandleFunc("GET /todos/{id}", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err != nil || c.Value != "abc" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = fmt.Fprintf(w, `{"data":{"id":%q,"auth":%q}}`, r.PathValue("id"), r.Header.Get("Authorization"))
	})
	mux.HandleFunc("DELETE /todos/{id}", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.deleted = append(s.deleted, r.PathValue("id"))
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/flaky", func(w http.ResponseWriter, r *http.Request) {
		if s.flaky.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.echoed, s.echoBody = r, string(b)
		s.mu.Unlock()
		_, _ = fmt.Fprintf(w, `{"trace":%q,"env":%q}`, r.Header.Get("X-Trace"), r.Header.Get("X-Env"))
	})
	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

// requests is a RequestSource over a fixed list.
type requests []*domain.Request

func (rs requests) RequestByID(id string) *domain.Request {
	for _, r := range rs {
		if r.ID() == id {
			return r
		}
	}
	return nil
}

func (rs requests) AllRequests() []*domain.Request { return rs }

func httpReq(id, collection, name, method, url, body string) *domain.Request {
	r := domain.NewHTTPRequest(name)
	r.MetaData.ID = id
	r.CollectionName = collection
	r.Spec.HTTP.Method = method
	r.Spec.HTTP.URL = url
	if body != "" {
		r.Spec.HTTP.Request.Body = domain.Body{Type: domain.RequestBodyTypeJSON, Data: body}
	}
	return r
}

func todoRequests() requests {
	create := httpReq("create", "Todos", "Create", "POST", "{{base}}/todos", `{"title":"{{title}}"}`)
	// An extract rule, as a request can have in the app.
	create.Spec.HTTP.Request.Variables = []domain.Variable{{
		TargetEnvVariable: "token", From: domain.VariableFromBody, JsonPath: "$.data.token", Enable: true,
	}}
	get := httpReq("get", "Todos", "Get", "GET", "{{base}}/todos/{{todoId}}", "")
	get.Spec.HTTP.Request.Headers = []domain.KeyValue{{Key: "Authorization", Value: "Bearer {{token}}", Enable: true}}
	return requests{
		create,
		get,
		httpReq("delete", "Todos", "Delete", "DELETE", "{{base}}/todos/{{todoId}}", ""),
		httpReq("flaky", "", "Flaky", "GET", "{{base}}/flaky", ""),
		httpReq("slow", "", "Slow", "GET", "{{base}}/slow", ""),
		httpReq("echo", "", "Echo", "POST", "{{base}}/echo?keep=1&page=1", `{}`),
	}
}

func testEnv(base string) *domain.Environment {
	env := domain.NewEnvironment("dev")
	env.Spec.Values = []domain.KeyValue{
		{Key: "base", Value: base, Enable: true},
		{Key: "env", Value: "from-env", Enable: true},
	}
	return env
}

func step(id, requestID string, asserts ...domain.TestAssertion) domain.TestStep {
	return domain.TestStep{ID: id, Request: domain.TestRequestRef{ID: requestID}, Assert: asserts}
}

func statusIs(code int) domain.TestAssertion {
	return domain.TestAssertion{Target: "status", Op: "eq", Value: code}
}

func newRunner(src RequestSource, save func(*domain.Environment) error) *Runner {
	return New(Config{
		NewSender: NewSender(src.RequestByID, nil, nil, nil),
		Requests:  src,
		SaveEnv:   save,
		Getenv:    func(k string) string { return "os-" + k },
	})
}

func statuses(run *Run) string {
	var out []string
	for _, s := range run.Steps {
		out = append(out, s.StepID+":"+string(s.Status))
	}
	return strings.Join(out, " ")
}

func TestRunChainsValues(t *testing.T) {
	srv := newTodoServer(t)
	env := testEnv(srv.URL)
	tc := domain.NewTestCase("todos")
	tc.Spec.Variables = []domain.TestVariable{
		{Key: "title", Value: "milk for {{env}}"},
		{Key: "user", From: &domain.TestVariableSource{OsEnv: "USER"}},
	}
	create := step("create", "create",
		statusIs(201),
		domain.TestAssertion{Target: "body", Path: "$.data.title", Op: "eq", Value: "milk for from-env"},
		domain.TestAssertion{Target: "cookie", Key: "session", Op: "exists"},
	)
	create.Capture = []domain.TestCapture{{Var: "todoId", From: "body", Path: "$.data.id"}}
	tc.Spec.Steps = []domain.TestStep{
		create,
		step("get", "get",
			statusIs(200),
			domain.TestAssertion{Target: "body", Path: "$.data.id", Op: "eq", Value: "{{todoId}}"},
			// The create request's extract rule set token; the jar sent the cookie.
			domain.TestAssertion{Target: "body", Path: "$.data.auth", Op: "eq", Value: "Bearer tok-1"},
		),
	}
	tc.Spec.Teardown = []domain.TestStep{step("delete", "delete", statusIs(204))}

	var events []string
	run := newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{
		Env: env,
		OnEvent: func(e Event) {
			switch e.Kind {
			case EventStepStarted:
				events = append(events, "start "+e.Step.StepID)
			case EventStepFinished:
				events = append(events, "done "+e.Step.StepID)
			case EventRunFinished:
				events = append(events, "run "+string(e.Run.Status))
			}
		},
	})

	if run.Status != StatusPassed {
		for _, s := range run.Steps {
			t.Logf("%s: %s %s %+v", s.StepID, s.Status, s.Message, s.Assertions)
		}
		t.Fatalf("run %s: %s", run.Status, statuses(run))
	}
	if got := strings.Join(events, ", "); got != "start create, done create, start get, done get, start delete, done delete, run passed" {
		t.Errorf("events = %s", got)
	}
	if len(srv.deleted) != 1 || srv.deleted[0] != "t1" {
		t.Errorf("deleted = %v", srv.deleted)
	}
	if len(env.Spec.Values) != 2 {
		t.Errorf("the run changed the caller's env: %+v", env.Spec.Values)
	}
	c := run.Steps[0]
	if c.Request.Name != "Todos/Create" || c.Response.Status != 201 || c.Attempts != 1 || c.Captures[0].Value != "t1" {
		t.Errorf("create result = %+v", c)
	}
}

func TestRunCookiesStayInTheRun(t *testing.T) {
	srv := newTodoServer(t)
	src := todoRequests()
	tc := domain.NewTestCase("no login")
	tc.Spec.Variables = []domain.TestVariable{{Key: "todoId", Value: "t1"}}
	tc.Spec.Steps = []domain.TestStep{step("get", "get", statusIs(401))}

	r := newRunner(src, nil)
	login := domain.NewTestCase("login")
	login.Spec.Steps = []domain.TestStep{step("create", "create", statusIs(201))}
	if run := r.Run(context.Background(), login, Options{Env: testEnv(srv.URL)}); run.Status != StatusPassed {
		t.Fatalf("login run: %s", statuses(run))
	}
	// A new run starts with an empty jar.
	if run := r.Run(context.Background(), tc, Options{Env: testEnv(srv.URL)}); run.Status != StatusPassed {
		t.Fatalf("second run sent the first run's cookie: %s", statuses(run))
	}
}

func TestRunStopsAtFailureButTearsDown(t *testing.T) {
	srv := newTodoServer(t)
	tc := domain.NewTestCase("stop")
	tc.Spec.Variables = []domain.TestVariable{{Key: "todoId", Value: "t9"}}
	tc.Spec.Steps = []domain.TestStep{
		step("create", "create", statusIs(500)),
		step("flaky", "flaky"),
	}
	tc.Spec.Teardown = []domain.TestStep{step("delete", "delete")}

	run := newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{Env: testEnv(srv.URL)})
	if got := statuses(run); got != "create:failed flaky:skipped delete:passed" {
		t.Fatalf("statuses = %s", got)
	}
	if run.Status != StatusFailed || run.Steps[0].Message != "1 of 1 assertions failed" {
		t.Errorf("run = %s, create message %q", run.Status, run.Steps[0].Message)
	}
	if run.Steps[1].Message != "step create did not pass" {
		t.Errorf("skip message = %q", run.Steps[1].Message)
	}

	// flaky has no assertions, so any response passes it.
	tc.Spec.Steps[0].ContinueOnFailure = true
	run = newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{Env: testEnv(srv.URL)})
	if got := statuses(run); got != "create:failed flaky:passed delete:passed" {
		t.Fatalf("with continueOnFailure: %s", got)
	}
}

func TestRunSetupFailureSkipsSteps(t *testing.T) {
	srv := newTodoServer(t)
	tc := domain.NewTestCase("setup")
	tc.Spec.Setup = []domain.TestStep{step("login", "missing")}
	tc.Spec.Steps = []domain.TestStep{step("create", "create")}
	run := newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{Env: testEnv(srv.URL)})
	if got := statuses(run); got != "login:error create:skipped" || run.Status != StatusError {
		t.Fatalf("statuses = %s, run %s", got, run.Status)
	}
	if run.Steps[0].Message != "request missing not found" {
		t.Errorf("message = %q", run.Steps[0].Message)
	}
}

func TestRunRetries(t *testing.T) {
	srv := newTodoServer(t)
	tc := domain.NewTestCase("retry")
	s := step("flaky", "flaky", statusIs(200))
	s.Retry = &domain.TestRetry{Count: 3, Delay: 10 * time.Millisecond}
	tc.Spec.Steps = []domain.TestStep{s}

	run := newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{Env: testEnv(srv.URL)})
	if run.Status != StatusPassed || run.Steps[0].Attempts != 3 {
		t.Fatalf("run %s after %d attempts: %s", run.Status, run.Steps[0].Attempts, run.Steps[0].Message)
	}
}

func TestRunTimeout(t *testing.T) {
	srv := newTodoServer(t)
	tc := domain.NewTestCase("timeout")
	tc.Spec.Options.Timeout = 50 * time.Millisecond
	tc.Spec.Steps = []domain.TestStep{step("slow", "slow")}

	run := newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{Env: testEnv(srv.URL)})
	if s := run.Steps[0]; s.Status != StatusError || s.Message != "no response within 50ms" {
		t.Fatalf("step = %s %q", s.Status, s.Message)
	}
}

func TestRunCancel(t *testing.T) {
	srv := newTodoServer(t)
	tc := domain.NewTestCase("cancel")
	tc.Spec.Variables = []domain.TestVariable{{Key: "todoId", Value: "t5"}}
	tc.Spec.Steps = []domain.TestStep{step("slow", "slow"), step("flaky", "flaky")}
	tc.Spec.Teardown = []domain.TestStep{step("delete", "delete")}

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	run := newRunner(todoRequests(), nil).Run(ctx, tc, Options{Env: testEnv(srv.URL)})
	if got := statuses(run); got != "slow:cancelled flaky:skipped delete:passed" || run.Status != StatusCancelled {
		t.Fatalf("statuses = %s, run %s", got, run.Status)
	}
}

func TestRunOverrides(t *testing.T) {
	srv := newTodoServer(t)
	tc := domain.NewTestCase("overrides")
	s := step("echo", "echo",
		domain.TestAssertion{Target: "body", Path: "$.trace", Op: "eq", Value: "t-from-env"},
		domain.TestAssertion{Target: "body", Path: "$.env", Op: "eq", Value: "{{env}}"},
	)
	s.With = &domain.TestStepOverrides{
		Variables: map[string]string{"env": "step-{{env}}"},
		Headers:   []domain.TestKeyValue{{Key: "X-Trace", Value: "t-from-env"}, {Key: "X-Env", Value: "{{env}}"}},
		Query:     []domain.TestKeyValue{{Key: "page", Value: "2"}, {Key: "q", Value: "a b&{{env}}"}},
		Body:      `{"v":"{{env}}"}`,
	}
	tc.Spec.Steps = []domain.TestStep{s, step("after", "echo",
		domain.TestAssertion{Target: "body", Path: "$.env", Op: "eq", Value: ""},
	)}

	run := newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{Env: testEnv(srv.URL)})
	if run.Status != StatusPassed {
		t.Fatalf("%s: %+v", statuses(run), run.Steps[0].Assertions)
	}
	// The second send is the unmodified request: with.variables did not leak.
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.echoed.URL.RawQuery != "keep=1&page=1" || srv.echoBody != `{}` {
		t.Errorf("second send = %q %q", srv.echoed.URL.RawQuery, srv.echoBody)
	}
}

func TestSetQuery(t *testing.T) {
	tests := []struct{ url, key, value, want string }{
		{"http://x/a", "p", "1", "http://x/a?p=1"},
		{"http://x/a?p=1&q=2&p=3", "p", "9", "http://x/a?p=9&q=2"},
		{"{{base}}/a?q=2#top", "p", "a b", "{{base}}/a?q=2&p=a+b#top"},
		{"http://x/a", "q", "{{v}}&x", "http://x/a?q={{v}}%26x"},
	}
	for _, tt := range tests {
		if got := setQuery(tt.url, tt.key, tt.value); got != tt.want {
			t.Errorf("setQuery(%q, %q, %q) = %q, want %q", tt.url, tt.key, tt.value, got, tt.want)
		}
	}
}

func TestRunPersistEnv(t *testing.T) {
	srv := newTodoServer(t)
	env := testEnv(srv.URL)
	tc := domain.NewTestCase("persist")
	create := step("create", "create", statusIs(201))
	create.Capture = []domain.TestCapture{{Var: "todoId", From: "body", Path: "$.data.id"}}
	tc.Spec.Steps = []domain.TestStep{create}

	var saved *domain.Environment
	save := func(e *domain.Environment) error { saved = e; return nil }

	newRunner(todoRequests(), save).Run(context.Background(), tc, Options{Env: env})
	if saved != nil {
		t.Fatal("saved the env without persistEnv")
	}

	tc.Spec.Options.PersistEnv = true
	newRunner(todoRequests(), save).Run(context.Background(), tc, Options{Env: env})
	if saved == nil || saved.ID() != env.ID() {
		t.Fatalf("saved = %+v", saved)
	}
	got := snapshot(saved)
	// The extract rule's change is kept; captures and case variables are not.
	if got["token"] != "tok-1" || got["base"] != srv.URL || len(got) != 3 {
		t.Errorf("saved values = %v", got)
	}
	if len(env.Spec.Values) != 2 {
		t.Errorf("the caller's env changed: %+v", env.Spec.Values)
	}
}

func TestRunOnly(t *testing.T) {
	srv := newTodoServer(t)
	tc := domain.NewTestCase("only")
	tc.Spec.Steps = []domain.TestStep{step("create", "create"), step("flaky", "flaky")}
	run := newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{Env: testEnv(srv.URL), Only: []string{"create"}})
	if got := statuses(run); got != "create:passed flaky:skipped" {
		t.Fatalf("statuses = %s", got)
	}
}

func TestRunRefs(t *testing.T) {
	srv := newTodoServer(t)
	src := append(todoRequests(), httpReq("dup", "Other", "Get", "GET", "{{base}}/flaky", ""),
		httpReq("dup2", "Other", "Get", "GET", "{{base}}/flaky", ""))
	tc := domain.NewTestCase("refs")
	tc.Spec.Steps = []domain.TestStep{
		{ID: "a", Request: domain.TestRequestRef{Ref: "Todos/Create"}, Assert: []domain.TestAssertion{statusIs(201)}},
		{ID: "b", Request: domain.TestRequestRef{ID: "gone", Ref: "Todos/Create"}},
		{ID: "c", Request: domain.TestRequestRef{Ref: "Other/Get"}},
	}
	tc.Spec.Options.ContinueOnFailure = true
	run := newRunner(src, nil).Run(context.Background(), tc, Options{Env: testEnv(srv.URL)})
	if got := statuses(run); got != "a:passed b:passed c:error" {
		t.Fatalf("statuses = %s", got)
	}
	if msg := run.Steps[2].Message; msg != `2 requests are named "Other/Get"; set request.id` {
		t.Errorf("message = %q", msg)
	}
}

func TestRunWithoutEnv(t *testing.T) {
	srv := newTodoServer(t)
	tc := domain.NewTestCase("no env")
	tc.Spec.Variables = []domain.TestVariable{{Key: "base", Value: srv.URL}}
	tc.Spec.Steps = []domain.TestStep{step("flaky", "flaky")}
	run := newRunner(todoRequests(), nil).Run(context.Background(), tc, Options{})
	if run.Status != StatusPassed {
		t.Fatalf("%s: %s", statuses(run), run.Steps[0].Message)
	}
}

// fakeSender answers every send with the same response.
type fakeSender struct {
	res *egress.Response
	err error
}

func (f fakeSender) Send(*domain.Request, *domain.Environment) (*egress.Response, error) {
	return f.res, f.err
}

func runFake(t *testing.T, f fakeSender, s domain.TestStep) *StepResult {
	t.Helper()
	r := New(Config{NewSender: func() Sender { return f }, Requests: todoRequests()})
	tc := domain.NewTestCase("fake")
	tc.Spec.Steps = []domain.TestStep{s}
	return &r.Run(context.Background(), tc, Options{}).Steps[0]
}

func TestRunScriptTests(t *testing.T) {
	res := &egress.Response{StatusCode: 200, Body: []byte(`{}`), ScriptTests: []scripting.TestResult{
		{Name: "ok", Passed: true}, {Name: "has id", Error: "missing"},
	}}
	got := runFake(t, fakeSender{res: res}, step("s", "flaky", statusIs(200)))
	if got.Status != StatusFailed || got.Message != "1 of 3 assertions failed" {
		t.Fatalf("step = %s %q", got.Status, got.Message)
	}
	if a := got.Assertions[2]; a.Source != SourceScript || a.Path != "has id" || a.Passed {
		t.Errorf("script assertion = %+v", a)
	}
}

func TestRunPostRequestError(t *testing.T) {
	res := &egress.Response{StatusCode: 200, Body: []byte(`{}`), PostRequestError: fmt.Errorf("KeyError")}
	got := runFake(t, fakeSender{res: res}, step("s", "flaky", statusIs(200)))
	if got.Status != StatusError || got.Message != "post-request: KeyError" || got.Response == nil {
		t.Fatalf("step = %s %q", got.Status, got.Message)
	}
}

func TestRunSendError(t *testing.T) {
	err := fmt.Errorf("the pre-request script skipped this request")
	got := runFake(t, fakeSender{res: &egress.Response{Error: err}, err: err}, step("s", "flaky"))
	if got.Status != StatusError || got.Message != err.Error() || got.Response != nil {
		t.Fatalf("step = %s %q", got.Status, got.Message)
	}
}

func TestRunCaptureFailure(t *testing.T) {
	s := step("s", "flaky")
	s.Capture = []domain.TestCapture{{Var: "id", From: "body", Path: "$.id"}}
	got := runFake(t, fakeSender{res: &egress.Response{Body: []byte(`{}`)}}, s)
	if got.Status != StatusFailed || got.Message != "capture id: not found" {
		t.Fatalf("step = %s %q", got.Status, got.Message)
	}
}
