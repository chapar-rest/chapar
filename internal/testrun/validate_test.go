package testrun

import (
	"errors"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

func validCase() *domain.TestCase {
	tc := domain.NewTestCase("Todos")
	tc.Spec.Variables = []domain.TestVariable{{Key: "title", Value: "milk"}}
	tc.Spec.Steps = []domain.TestStep{{
		ID:      "create",
		Request: domain.TestRequestRef{ID: "req-1"},
		Assert: []domain.TestAssertion{
			{Target: "status", Op: "in", Value: []any{200, 201}},
			{Target: "header", Key: "content-type", Op: "contains", Value: "json"},
			{Target: "body", Path: "$.data.id", Op: "exists"},
			{Target: "body", Path: "$.data.id", Op: "type", Value: "number"},
			{Target: "body", Path: "$.data.title", Op: "matches", Value: "^buy"},
			{Target: "time", Op: "lt", Value: 500},
		},
		Capture: []domain.TestCapture{{Var: "id", From: "body", Path: "$.data.id"}},
	}}
	tc.Spec.Teardown = []domain.TestStep{{ID: "delete", Request: domain.TestRequestRef{Ref: "Todos/Delete"}}}
	return tc
}

func TestValidateValid(t *testing.T) {
	if got := Validate(validCase(), nil); len(got) != 0 {
		t.Fatalf("unexpected problems: %v", got)
	}
}

func TestValidateProblems(t *testing.T) {
	tests := []struct {
		name  string
		edit  func(tc *domain.TestCase)
		where string
		msg   string
	}{
		{"kind", func(tc *domain.TestCase) { tc.Kind = "Request" }, "", "kind is"},
		{"name", func(tc *domain.TestCase) { tc.MetaData.Name = "" }, "metadata", "name is required"},
		{"no steps", func(tc *domain.TestCase) { tc.Spec.Steps = nil }, "steps", "at least one step"},
		{"variable key", func(tc *domain.TestCase) { tc.Spec.Variables[0].Key = "" }, "variables[0]", "key is required"},
		{"variable twice", func(tc *domain.TestCase) {
			tc.Spec.Variables = append(tc.Spec.Variables, domain.TestVariable{Key: "title"})
		}, "variables[1]", "declared twice"},
		{"variable value and from", func(tc *domain.TestCase) {
			tc.Spec.Variables[0].From = &domain.TestVariableSource{OsEnv: "X"}
		}, "variables[0]", "not both"},
		{"step id", func(tc *domain.TestCase) { tc.Spec.Steps[0].ID = "" }, "steps[0]", "id is required"},
		{"step id reused across sections", func(tc *domain.TestCase) { tc.Spec.Teardown[0].ID = "create" }, "teardown[0]", "already used by steps[0]"},
		{"no request", func(tc *domain.TestCase) { tc.Spec.Steps[0].Request = domain.TestRequestRef{} }, "steps[0].request", "id or ref"},
		{"negative retry", func(tc *domain.TestCase) {
			tc.Spec.Steps[0].Retry = &domain.TestRetry{Count: -1}
		}, "steps[0].retry", "negative"},
		{"unknown target", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[0].Target = "code" }, "steps[0].assert[0]", "unknown target"},
		{"header without key", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[1].Key = "" }, "steps[0].assert[1]", "needs a key"},
		{"body without path", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[2].Path = "" }, "steps[0].assert[2]", "needs a path"},
		{"bad path", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[2].Path = "$[" }, "steps[0].assert[2]", "bad JSONPath"},
		{"stray key", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[0].Key = "x" }, "steps[0].assert[0]", "does not use a key"},
		{"unknown op", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[0].Op = "near" }, "steps[0].assert[0]", "unknown op"},
		{"in without list", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[0].Value = 200 }, "steps[0].assert[0]", "needs a list"},
		{"exists with value", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[2].Value = true }, "steps[0].assert[2]", "takes no value"},
		{"eq without value", func(tc *domain.TestCase) {
			tc.Spec.Steps[0].Assert[2].Op = "eq"
		}, "steps[0].assert[2]", "needs a value"},
		{"bad type", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[3].Value = "int" }, "steps[0].assert[3]", "type must be"},
		{"bad regexp", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[4].Value = "(" }, "steps[0].assert[4]", "bad regexp"},
		{"lt on text", func(tc *domain.TestCase) { tc.Spec.Steps[0].Assert[5].Value = "fast" }, "steps[0].assert[5]", "needs a number"},
		{"capture var", func(tc *domain.TestCase) { tc.Spec.Steps[0].Capture[0].Var = "" }, "steps[0].capture[0]", "var is required"},
		{"capture from", func(tc *domain.TestCase) { tc.Spec.Steps[0].Capture[0].From = "json" }, "steps[0].capture[0]", "unknown target"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := validCase()
			tt.edit(tc)
			problems := Validate(tc, nil)
			for _, p := range problems {
				if p.Where == tt.where && strings.Contains(p.Message, tt.msg) {
					return
				}
			}
			t.Fatalf("no problem at %q containing %q; got %v", tt.where, tt.msg, problems)
		})
	}
}

func TestValidateFindsRequests(t *testing.T) {
	find := func(ref domain.TestRequestRef) error {
		if ref.ID == "req-1" {
			return nil
		}
		return errors.New("request " + ref.Ref + " not found")
	}
	problems := Validate(validCase(), find)
	if len(problems) != 1 || problems[0].Error() != "teardown[0].request: request Todos/Delete not found" {
		t.Fatalf("got %v", problems)
	}
}
