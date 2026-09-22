package scripting

import (
	"reflect"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

func httpRequest() *domain.Request {
	r := domain.NewHTTPRequest("r")
	h := r.Spec.HTTP
	h.Method = "POST"
	h.URL = "https://{{host}}/users/{id}?page=1&q=a%20b"
	h.Request.Headers = []domain.KeyValue{
		{Key: "Content-Type", Value: "application/json", Enable: true},
		{Key: "X-Off", Value: "no", Enable: false},
		{Key: "Authorization", Value: "Bearer {{token}}", Enable: true},
	}
	h.Request.PathParams = []domain.KeyValue{{Key: "id", Value: "7", Enable: true}}
	h.Request.Body = domain.Body{Type: domain.RequestBodyTypeJSON, Data: `{"a": 1}`}
	return r
}

func env() *domain.Environment {
	e := domain.NewEnvironment("dev")
	e.SetKey("host", "api.test")
	e.SetKey("token", "t0k")
	return e
}

func TestRequestDataHTTP(t *testing.T) {
	req := httpRequest()
	e := env()
	coll := &domain.Collection{Spec: domain.ColSpec{Headers: []domain.KeyValue{
		{Key: "X-Team", Value: "core", Enable: true},
	}}}

	d := RequestDataFromDomain(req, e, coll)

	if d.URL != req.Spec.HTTP.URL || d.Method != "POST" || d.Body != `{"a": 1}` {
		t.Fatalf("fields = %+v", d)
	}
	wantHeaders := []Pair{{"X-Team", "core"}, {"Content-Type", "application/json"}, {"Authorization", "Bearer {{token}}"}}
	if !reflect.DeepEqual(d.Headers, wantHeaders) {
		t.Errorf("headers = %v, want %v", d.Headers, wantHeaders)
	}
	if want := []Pair{{"page", "1"}, {"q", "a b"}}; !reflect.DeepEqual(d.Query, want) {
		t.Errorf("query = %v, want %v", d.Query, want)
	}
	if d.PathParams["id"] != "7" {
		t.Errorf("path params = %v", d.PathParams)
	}
	if d.Resolved.URL != "https://api.test/users/{id}?page=1&q=a%20b" {
		t.Errorf("resolved url = %q", d.Resolved.URL)
	}
	if got := d.Resolved.Headers[2][1]; got != "Bearer t0k" {
		t.Errorf("resolved auth = %q", got)
	}
	// Building the view changes neither the request nor the environment.
	if len(req.Spec.HTTP.Request.Headers) != 3 || req.Spec.HTTP.Request.Headers[2].Value != "Bearer {{token}}" {
		t.Errorf("request changed: %+v", req.Spec.HTTP.Request.Headers)
	}
	if v := e.Spec.Values[0].Value; v != "api.test" {
		t.Errorf("env changed: %v", v)
	}
}

func TestRequestDataGRPCAndGraphQL(t *testing.T) {
	g := domain.NewGRPCRequest("g")
	g.Spec.GRPC.ServerInfo.Address = "{{host}}:50051"
	g.Spec.GRPC.LasSelectedMethod = "pkg.Svc/Get"
	g.Spec.GRPC.Body = `{"id": 1}`
	g.Spec.GRPC.Metadata = []domain.KeyValue{{Key: "auth", Value: "{{token}}", Enable: true}}
	d := RequestDataFromDomain(g, env(), nil)
	if d.URL != "{{host}}:50051" || d.Method != "pkg.Svc/Get" || d.Body != `{"id": 1}` ||
		!reflect.DeepEqual(d.Headers, []Pair{{"auth", "{{token}}"}}) {
		t.Errorf("grpc = %+v", d)
	}
	if d.Resolved.URL != "api.test:50051" || d.Resolved.Headers[0][1] != "t0k" {
		t.Errorf("grpc resolved = %+v", d.Resolved)
	}

	q := domain.NewGraphQLRequest("q")
	q.Spec.GraphQL.URL = "https://{{host}}/graphql"
	q.Spec.GraphQL.Query = "{ me }"
	q.Spec.GraphQL.Variables = `{"id": 1}`
	d = RequestDataFromDomain(q, nil, nil)
	if d.GraphQL == nil || d.GraphQL.Query != "{ me }" || d.GraphQL.Variables != `{"id": 1}` || d.Method != "POST" {
		t.Errorf("graphql = %+v", d)
	}
}

func ptr[T any](v T) *T { return &v }

func TestApplyChangesHTTP(t *testing.T) {
	req := httpRequest()
	ApplyChanges(req, &RequestChanges{
		Headers:    &[]Pair{{"Content-Type", "application/json"}, {"X-Sig", "abc"}},
		Query:      &[]Pair{{"page", "2"}, {"q", "{{term}} x"}},
		Body:       ptr(`{"a": 2}`),
		Method:     ptr("put"),
		PathParams: &map[string]string{"id": "8"},
	})
	h := req.Spec.HTTP
	if h.URL != "https://{{host}}/users/{id}?page=2&q={{term}}+x" {
		t.Errorf("url = %q", h.URL)
	}
	if h.Method != "PUT" {
		t.Errorf("method = %q", h.Method)
	}
	if len(h.Request.Headers) != 2 || h.Request.Headers[1].Key != "X-Sig" || !h.Request.Headers[1].Enable {
		t.Errorf("headers = %+v", h.Request.Headers)
	}
	if h.Request.Body.Data != `{"a": 2}` || h.Request.Body.Type != domain.RequestBodyTypeJSON {
		t.Errorf("body = %+v", h.Request.Body)
	}
	if len(h.Request.PathParams) != 1 || h.Request.PathParams[0].Value != "8" {
		t.Errorf("path params = %+v", h.Request.PathParams)
	}
}

func TestApplyChangesBodyOnFormRequest(t *testing.T) {
	req := httpRequest()
	req.Spec.HTTP.Request.Body = domain.Body{Type: domain.RequestBodyTypeNone}
	ApplyChanges(req, &RequestChanges{Body: ptr("hello")})
	if b := req.Spec.HTTP.Request.Body; b.Type != domain.RequestBodyTypeText || b.Data != "hello" {
		t.Errorf("body = %+v", b)
	}
}

func TestApplyChangesGRPCAndGraphQL(t *testing.T) {
	g := domain.NewGRPCRequest("g")
	ApplyChanges(g, &RequestChanges{
		URL: ptr("localhost:1"), Method: ptr("a.B/C"), Body: ptr("{}"),
		Headers: &[]Pair{{"k", "v"}},
	})
	s := g.Spec.GRPC
	if s.ServerInfo.Address != "localhost:1" || s.LasSelectedMethod != "a.B/C" || s.Body != "{}" ||
		len(s.Metadata) != 1 || s.Metadata[0].Value != "v" {
		t.Errorf("grpc = %+v", s)
	}

	q := domain.NewGraphQLRequest("q")
	q.Spec.GraphQL.Query = "{ a }"
	ApplyChanges(q, &RequestChanges{GraphQL: &GraphQLChanges{Variables: ptr(`{"x": 1}`)}})
	if q.Spec.GraphQL.Query != "{ a }" || q.Spec.GraphQL.Variables != `{"x": 1}` {
		t.Errorf("graphql = %+v", q.Spec.GraphQL)
	}
}

func TestQueryRoundTrip(t *testing.T) {
	u := "http://x/p?a=1&b=two%20words#frag"
	got := WithQuery(u, ParseQuery(u))
	if got != "http://x/p?a=1&b=two+words#frag" {
		t.Errorf("round trip = %q", got)
	}
	if got := WithQuery("http://x/p?a=1", nil); got != "http://x/p" {
		t.Errorf("clear = %q", got)
	}
}

func TestApplyEnv(t *testing.T) {
	e := env()
	r := &ExecResult{
		EnvSet:   map[string]any{"s": "v", "n": float64(5), "m": map[string]any{"a": true}, "nil": nil},
		EnvUnset: []string{"host", "missing"},
	}
	if !r.ApplyEnv(e) {
		t.Fatal("ApplyEnv reported no change")
	}
	got := map[string]string{}
	for _, kv := range e.Spec.Values {
		got[kv.Key] = kv.Value
	}
	want := map[string]string{"token": "t0k", "s": "v", "n": "5", "m": `{"a":true}`, "nil": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("env = %v, want %v", got, want)
	}
}

func TestScriptErrorAndSummary(t *testing.T) {
	err := &ScriptError{Type: "KeyError", Message: "'x'", Line: 4}
	if err.Error() != "line 4: KeyError: 'x'" {
		t.Errorf("error = %q", err.Error())
	}
	r := &ExecResult{
		Prints: []string{"hi"},
		Tests:  []TestResult{{Name: "a", Passed: true}, {Name: "b", Error: "nope"}},
	}
	if r.Summary() != "✓ a\n✗ b: nope\nhi" || r.FailedTests() != 1 {
		t.Errorf("summary = %q, failed = %d", r.Summary(), r.FailedTests())
	}
}
