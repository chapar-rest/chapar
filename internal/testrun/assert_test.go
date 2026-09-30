package testrun

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/scripting"
)

const body = `{
  "data": {
    "id": 42,
    "title": "buy milk",
    "done": false,
    "note": null,
    "tags": ["home", "shop"],
    "owner": {"name": "ana", "age": 30},
    "price": "12.50"
  }
}`

func testResponse() *egress.Response {
	return &egress.Response{
		StatusCode:      201,
		ResponseHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "X-Count": "7"},
		Cookies:         []*http.Cookie{{Name: "session", Value: "abc"}},
		Body:            []byte(body),
		Size:            len(body),
		TimePassed:      120 * time.Millisecond,
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name string
		a    domain.TestAssertion
		pass bool
		msg  string // substring of the failure message
	}{
		{"status eq", domain.TestAssertion{Target: "status", Op: "eq", Value: 201}, true, ""},
		{"status eq wrong", domain.TestAssertion{Target: "status", Op: "eq", Value: 200}, false, "got 201, want 200"},
		{"status in", domain.TestAssertion{Target: "status", Op: "in", Value: []any{200, 201}}, true, ""},
		{"status not in", domain.TestAssertion{Target: "status", Op: "in", Value: []any{200, 204}}, false, "want one of [200,204]"},
		{"status gte", domain.TestAssertion{Target: "status", Op: "gte", Value: 200}, true, ""},
		{"status lt", domain.TestAssertion{Target: "status", Op: "lt", Value: 300}, true, ""},

		{"header any case", domain.TestAssertion{Target: "header", Key: "content-type", Op: "contains", Value: "json"}, true, ""},
		{"header number eq", domain.TestAssertion{Target: "header", Key: "X-Count", Op: "eq", Value: 7}, true, ""},
		{"header number gt", domain.TestAssertion{Target: "header", Key: "x-count", Op: "gt", Value: 5}, true, ""},
		{"header missing", domain.TestAssertion{Target: "header", Key: "X-Nope", Op: "eq", Value: "x"}, false, "not found"},
		{"header notExists", domain.TestAssertion{Target: "header", Key: "X-Nope", Op: "notExists"}, true, ""},
		{"cookie eq", domain.TestAssertion{Target: "cookie", Key: "session", Op: "eq", Value: "abc"}, true, ""},

		{"body number", domain.TestAssertion{Target: "body", Path: "$.data.id", Op: "eq", Value: 42}, true, ""},
		{"body number as text", domain.TestAssertion{Target: "body", Path: "$.data.id", Op: "eq", Value: "42"}, true, ""},
		{"body bool", domain.TestAssertion{Target: "body", Path: "$.data.done", Op: "eq", Value: false}, true, ""},
		{"body bool is not text false", domain.TestAssertion{Target: "body", Path: "$.data.title", Op: "eq", Value: false}, false, ""},
		{"body object", domain.TestAssertion{Target: "body", Path: "$.data.owner", Op: "eq", Value: map[any]any{"name": "ana", "age": 30}}, true, ""},
		{"body array", domain.TestAssertion{Target: "body", Path: "$.data.tags", Op: "eq", Value: []any{"home", "shop"}}, true, ""},
		{"body ne", domain.TestAssertion{Target: "body", Path: "$.data.title", Op: "ne", Value: "sleep"}, true, ""},
		{"null exists", domain.TestAssertion{Target: "body", Path: "$.data.note", Op: "exists"}, true, ""},
		{"null type", domain.TestAssertion{Target: "body", Path: "$.data.note", Op: "type", Value: "null"}, true, ""},
		{"missing exists", domain.TestAssertion{Target: "body", Path: "$.data.nope", Op: "exists"}, false, "not found"},
		{"missing notExists", domain.TestAssertion{Target: "body", Path: "$.data.nope", Op: "notExists"}, true, ""},
		{"missing eq", domain.TestAssertion{Target: "body", Path: "$.data.nope", Op: "eq", Value: 1}, false, "not found"},
		{"numeric string gt", domain.TestAssertion{Target: "body", Path: "$.data.price", Op: "gt", Value: 10}, true, ""},
		{"gt on text", domain.TestAssertion{Target: "body", Path: "$.data.title", Op: "gt", Value: 1}, false, "is not a number"},
		{"array contains", domain.TestAssertion{Target: "body", Path: "$.data.tags", Op: "contains", Value: "shop"}, true, ""},
		{"array notContains", domain.TestAssertion{Target: "body", Path: "$.data.tags", Op: "notContains", Value: "work"}, true, ""},
		{"contains on number", domain.TestAssertion{Target: "body", Path: "$.data.id", Op: "contains", Value: 4}, false, "cannot look inside number"},
		{"matches", domain.TestAssertion{Target: "body", Path: "$.data.title", Op: "matches", Value: "^buy "}, true, ""},
		{"matches number", domain.TestAssertion{Target: "body", Path: "$.data.id", Op: "matches", Value: `^\d+$`}, true, ""},
		{"type array", domain.TestAssertion{Target: "body", Path: "$.data.tags", Op: "type", Value: "array"}, true, ""},
		{"type wrong", domain.TestAssertion{Target: "body", Path: "$.data.id", Op: "type", Value: "string"}, false, "got number, want \"string\""},
		{"length array", domain.TestAssertion{Target: "body", Path: "$.data.tags", Op: "length", Value: 2}, true, ""},
		{"length object", domain.TestAssertion{Target: "body", Path: "$.data.owner", Op: "length", Value: 2}, true, ""},
		{"length string", domain.TestAssertion{Target: "body", Path: "$.data.title", Op: "length", Value: 8}, true, ""},
		{"wildcard", domain.TestAssertion{Target: "body", Path: "$.data.tags[*]", Op: "length", Value: 2}, true, ""},
		{"bad path", domain.TestAssertion{Target: "body", Path: "$[", Op: "exists"}, false, "bad JSONPath"},

		{"text", domain.TestAssertion{Target: "text", Op: "contains", Value: `"buy milk"`}, true, ""},
		{"time", domain.TestAssertion{Target: "time", Op: "lt", Value: 500}, true, ""},
		{"size", domain.TestAssertion{Target: "size", Op: "eq", Value: len(body)}, true, ""},

		{"unknown target", domain.TestAssertion{Target: "nope", Op: "exists"}, false, "unknown target"},
		{"unknown op", domain.TestAssertion{Target: "status", Op: "near", Value: 200}, false, "unknown op"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(testResponse(), []domain.TestAssertion{tt.a})[0]
			if got.Passed != tt.pass {
				t.Fatalf("passed = %v, want %v (message %q, actual %#v)", got.Passed, tt.pass, got.Message, got.Actual)
			}
			if tt.pass && got.Message != "" {
				t.Fatalf("passing assertion has message %q", got.Message)
			}
			if !strings.Contains(got.Message, tt.msg) {
				t.Fatalf("message %q does not contain %q", got.Message, tt.msg)
			}
		})
	}
}

func TestEvaluateNonJSONBody(t *testing.T) {
	res := testResponse()
	res.Body = []byte("<html>")
	got := Evaluate(res, []domain.TestAssertion{
		{Target: "body", Path: "$.a", Op: "exists"},
		{Target: "text", Op: "contains", Value: "html"},
	})
	if got[0].Passed || !strings.Contains(got[0].Message, "not JSON") {
		t.Fatalf("body on HTML: %+v", got[0])
	}
	if !got[1].Passed {
		t.Fatalf("text on HTML: %+v", got[1])
	}
}

func TestEvaluateGRPC(t *testing.T) {
	res := &egress.Response{
		StatueCode:       5,
		ResponseMetadata: []domain.KeyValue{{Key: "x-request-id", Value: "r1"}},
		Trailers:         []domain.KeyValue{{Key: "grpc-message", Value: "not found"}},
		Body:             []byte(`{}`),
	}
	got := Evaluate(res, []domain.TestAssertion{
		{Target: "status", Op: "eq", Value: 5},
		{Target: "metadata", Key: "X-Request-Id", Op: "eq", Value: "r1"},
		{Target: "trailer", Key: "grpc-message", Op: "contains", Value: "not"},
	})
	for i, r := range got {
		if !r.Passed {
			t.Errorf("assertion %d failed: %s", i, r.Message)
		}
	}
}

func TestEvaluateNoResponse(t *testing.T) {
	got := Evaluate(nil, []domain.TestAssertion{{Target: "status", Op: "eq", Value: 200}})
	if got[0].Passed || got[0].Message != "no response" {
		t.Fatalf("got %+v", got[0])
	}
}

func TestCapture(t *testing.T) {
	results, vars := Capture(testResponse(), []domain.TestCapture{
		{Var: "id", From: "body", Path: "$.data.id"},
		{Var: "owner", From: "body", Path: "$.data.owner"},
		{Var: "title", From: "body", Path: "$.data.title"},
		{Var: "sid", From: "cookie", Key: "session"},
		{Var: "code", From: "status"},
		{Var: "gone", From: "body", Path: "$.data.nope"},
	})

	want := map[string]string{
		"id":    "42",
		"owner": `{"age":30,"name":"ana"}`,
		"title": "buy milk",
		"sid":   "abc",
		"code":  "201",
	}
	for k, v := range want {
		if vars[k] != v {
			t.Errorf("%s = %q, want %q", k, vars[k], v)
		}
	}
	if _, ok := vars["gone"]; ok {
		t.Error("a missing value was captured")
	}
	if last := results[len(results)-1]; last.Error != "not found" {
		t.Errorf("missing capture result = %+v", last)
	}
}

func TestScriptAssertions(t *testing.T) {
	got := ScriptAssertions([]scripting.TestResult{
		{Name: "has id", Passed: true},
		{Name: "is done", Passed: false, Error: "expected True"},
	})
	if len(got) != 2 || got[0].Source != SourceScript || !got[0].Passed || got[1].Passed || got[1].Message != "expected True" {
		t.Fatalf("got %+v", got)
	}
}
