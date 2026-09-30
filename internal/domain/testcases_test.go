package domain

import (
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

const testCaseYAML = `apiVersion: v1
kind: TestCase
metadata:
  id: tc-1
  name: Create and fetch todo
spec:
  description: Creates a todo and reads it back
  tags:
  - smoke
  options:
    timeout: 30s
    persistEnv: true
  variables:
  - key: title
    value: buy milk
  - key: token
    from:
      osEnv: API_TOKEN
  steps:
  - id: create
    name: Create todo
    request:
      id: req-1
      ref: Todos/Create
    with:
      variables:
        title: '{{title}} 2'
      headers:
      - key: X-Trace
        value: test
      body: '{"title":"{{title}}"}'
    timeout: 10s
    retry:
      count: 3
      delay: 500ms
    assert:
    - target: status
      op: in
      value:
      - 200
      - 201
    - target: body
      path: $.data.done
      op: eq
      value: false
    - target: body
      path: $.data.tags
      op: eq
      value:
        a: 1
    capture:
    - var: todoId
      from: body
      path: $.data.id
  teardown:
  - id: delete
    request:
      ref: Todos/Delete
`

func TestTestCaseYAMLRoundTrip(t *testing.T) {
	var tc TestCase
	if err := yaml.Unmarshal([]byte(testCaseYAML), &tc); err != nil {
		t.Fatal(err)
	}

	if tc.Spec.Options.Timeout != 30*time.Second {
		t.Fatalf("timeout = %v", tc.Spec.Options.Timeout)
	}
	step := tc.Spec.Steps[0]
	if step.Timeout != 10*time.Second || step.Retry.Delay != 500*time.Millisecond {
		t.Fatalf("step durations = %v, %v", step.Timeout, step.Retry.Delay)
	}
	if v, ok := step.Assert[1].Value.(bool); !ok || v {
		t.Fatalf("false decoded as %#v", step.Assert[1].Value)
	}
	if !reflect.DeepEqual(step.Assert[0].Value, []any{200, 201}) {
		t.Fatalf("list decoded as %#v", step.Assert[0].Value)
	}
	if got := len(tc.AllSteps()); got != 2 {
		t.Fatalf("AllSteps = %d, want 2", got)
	}

	out, err := tc.MarshalYaml()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != testCaseYAML {
		t.Fatalf("round trip changed the file:\n%s", out)
	}
}

func TestTestCaseCloneIsDeep(t *testing.T) {
	var tc TestCase
	if err := yaml.Unmarshal([]byte(testCaseYAML), &tc); err != nil {
		t.Fatal(err)
	}
	c := tc.Clone()
	if c.ID() == tc.ID() {
		t.Fatal("clone kept the id")
	}

	c.Spec.Steps[0].With.Variables["title"] = "changed"
	c.Spec.Steps[0].With.Headers[0].Value = "changed"
	c.Spec.Steps[0].Retry.Count = 9
	c.Spec.Steps[0].Assert[0].Value.([]any)[0] = 999
	c.Spec.Steps[0].Assert[2].Value.(map[any]any)["a"] = 2
	c.Spec.Variables[1].From.OsEnv = "changed"
	c.Spec.Tags[0] = "changed"

	out, err := tc.MarshalYaml()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != testCaseYAML {
		t.Fatalf("editing the clone changed the original:\n%s", out)
	}
}

func TestNewTestCase(t *testing.T) {
	tc := NewTestCase("Smoke")
	if tc.Kind != KindTestCase || tc.ApiVersion != ApiVersion || tc.ID() == "" || tc.GetName() != "Smoke" {
		t.Fatalf("unexpected test case: %+v", tc)
	}
}
