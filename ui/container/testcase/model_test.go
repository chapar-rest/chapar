package testcase

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/testrun"
)

// TestMain loads the text engine editors need.
func TestMain(m *testing.M) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		panic(err)
	}
	yoga.SetResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
	os.Exit(m.Run())
}

func TestValueTextRoundTrip(t *testing.T) {
	for _, v := range []any{
		nil, true, false, 201, 1.5, "buy milk", "201", "true", " padded", `say "hi"`,
		[]any{200, 201}, []any{"a", "b,c"}, "{{todoId}}",
	} {
		text := valueText(v)
		if got := parseValue(text); !reflect.DeepEqual(got, v) {
			t.Errorf("%#v -> %q -> %#v", v, text, got)
		}
	}
}

func TestParseValue(t *testing.T) {
	tests := map[string]any{
		"":          nil,
		"201":       201,
		"-3":        -3,
		"0.5":       0.5,
		"false":     false,
		"yes":       "yes",
		"[200,201]": []any{200, 201},
		`"201"`:     "201",
		"[broken":   "[broken",
		"$.data.id": "$.data.id",
	}
	for in, want := range tests {
		if got := parseValue(in); !reflect.DeepEqual(got, want) {
			t.Errorf("parseValue(%q) = %#v, want %#v", in, got, want)
		}
	}
}

func TestStepRoundTrip(t *testing.T) {
	step := domain.TestStep{
		ID:      "create",
		Name:    "Create",
		Request: domain.TestRequestRef{ID: "r1", Ref: "Todos/Create"},
		With: &domain.TestStepOverrides{
			Variables: map[string]string{"a": "1", "b": "2"},
			Headers:   []domain.TestKeyValue{{Key: "X-Trace", Value: "t"}},
			Body:      `{"x":1}`,
		},
		Timeout:           10 * time.Second,
		Retry:             &domain.TestRetry{Count: 3, Delay: 500 * time.Millisecond},
		ContinueOnFailure: true,
		Assert: []domain.TestAssertion{
			{Target: "status", Op: "in", Value: []any{200, 201}},
			{Target: "header", Key: "content-type", Op: "contains", Value: "json"},
			{Target: "body", Path: "$.data.id", Op: "exists"},
			{Target: "body", Path: "$.data.done", Op: "eq", Value: false},
		},
		Capture: []domain.TestCapture{
			{Var: "id", From: "body", Path: "$.data.id"},
			{Var: "sid", From: "cookie", Key: "session"},
		},
	}
	m := loadStep(step, func() {})
	defer m.close()

	var problems []testrun.Problem
	got := m.dump("steps[0]", &problems)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if !reflect.DeepEqual(got, step) {
		t.Fatalf("round trip changed the step:\n got %+v\nwant %+v", got, step)
	}
}

func TestStepDumpReportsBadText(t *testing.T) {
	m := loadStep(domain.TestStep{ID: "a"}, func() {})
	defer m.close()
	m.timeout, m.retryCount = "soon", "three"

	var problems []testrun.Problem
	m.dump("steps[0]", &problems)
	if len(problems) != 2 || problems[0].Where != "steps[0].timeout" || problems[1].Where != "steps[0].retry" {
		t.Fatalf("problems = %v", problems)
	}
}

func TestUniqueStepID(t *testing.T) {
	taken := map[string]bool{"create-todo": true}
	if got := uniqueStepID("Todos / Create Todo", taken); got != "todos-create-todo" {
		t.Errorf("got %q", got)
	}
	if got := uniqueStepID("Create todo", taken); got != "create-todo-2" {
		t.Errorf("got %q", got)
	}
	if got := uniqueStepID("?!", nil); got != "step" {
		t.Errorf("got %q", got)
	}
}
