package domain

import (
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

// Assertion and capture targets: the part of a response they read.
const (
	TestTargetStatus   = "status"
	TestTargetHeader   = "header"
	TestTargetCookie   = "cookie"
	TestTargetMetadata = "metadata"
	TestTargetTrailer  = "trailer"
	TestTargetBody     = "body"
	TestTargetText     = "text"
	TestTargetTime     = "time"
	TestTargetSize     = "size"
)

// Assertion operators.
const (
	TestOpExists      = "exists"
	TestOpNotExists   = "notExists"
	TestOpEq          = "eq"
	TestOpNe          = "ne"
	TestOpGt          = "gt"
	TestOpGte         = "gte"
	TestOpLt          = "lt"
	TestOpLte         = "lte"
	TestOpIn          = "in"
	TestOpContains    = "contains"
	TestOpNotContains = "notContains"
	TestOpMatches     = "matches"
	TestOpType        = "type"
	TestOpLength      = "length"
)

// TestCase is a sequence of existing requests whose responses are checked,
// with values passed from one step to the next. See docs/testcases-design.md.
type TestCase struct {
	ApiVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	MetaData   MetaData     `yaml:"metadata"`
	Spec       TestCaseSpec `yaml:"spec"`
}

type TestCaseSpec struct {
	Description string         `yaml:"description,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	Options     TestOptions    `yaml:"options,omitempty"`
	Variables   []TestVariable `yaml:"variables,omitempty"`
	// Setup runs before Steps; a failure there ends the run.
	Setup []TestStep `yaml:"setup,omitempty"`
	Steps []TestStep `yaml:"steps"`
	// Teardown always runs, even after a failure or a cancel.
	Teardown []TestStep `yaml:"teardown,omitempty"`
}

type TestOptions struct {
	ContinueOnFailure bool `yaml:"continueOnFailure,omitempty"`
	// Timeout is the default for each step; zero means no limit.
	Timeout time.Duration `yaml:"timeout,omitempty"`
	// PersistEnv writes the run's env changes back to the real environment.
	PersistEnv bool `yaml:"persistEnv,omitempty"`
}

type TestVariable struct {
	Key   string              `yaml:"key"`
	Value string              `yaml:"value,omitempty"`
	From  *TestVariableSource `yaml:"from,omitempty"`
}

type TestVariableSource struct {
	OsEnv string `yaml:"osEnv,omitempty"`
}

type TestStep struct {
	ID      string             `yaml:"id"`
	Name    string             `yaml:"name,omitempty"`
	Request TestRequestRef     `yaml:"request"`
	With    *TestStepOverrides `yaml:"with,omitempty"`
	// Timeout overrides the case's default when set.
	Timeout           time.Duration `yaml:"timeout,omitempty"`
	Retry             *TestRetry    `yaml:"retry,omitempty"`
	ContinueOnFailure bool          `yaml:"continueOnFailure,omitempty"`
	// Disabled steps are skipped, without failing the run.
	Disabled bool            `yaml:"disabled,omitempty"`
	Assert   []TestAssertion `yaml:"assert,omitempty"`
	Capture  []TestCapture   `yaml:"capture,omitempty"`
}

// TestRequestRef points at the request a step sends. ID is looked up
// first; Ref ("Collection/Request", or "Request" outside a collection)
// is the fallback for hand-written files and the name shown to users.
type TestRequestRef struct {
	ID  string `yaml:"id,omitempty"`
	Ref string `yaml:"ref,omitempty"`
}

type TestStepOverrides struct {
	Variables map[string]string `yaml:"variables,omitempty"`
	Headers   []TestKeyValue    `yaml:"headers,omitempty"`
	Query     []TestKeyValue    `yaml:"query,omitempty"`
	// Body replaces the HTTP body, GraphQL query or gRPC message when set.
	Body string `yaml:"body,omitempty"`
}

type TestKeyValue struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// TestRetry re-sends a step until its assertions pass.
type TestRetry struct {
	Count int           `yaml:"count"`
	Delay time.Duration `yaml:"delay,omitempty"`
}

type TestAssertion struct {
	Target string `yaml:"target"`
	// Key selects a header, cookie, metadata or trailer entry.
	Key string `yaml:"key,omitempty"`
	// Path is a JSONPath into the body.
	Path string `yaml:"path,omitempty"`
	Op   string `yaml:"op"`
	// Value keeps its YAML type: false is a bool, 201 a number, [..] a list.
	Value any `yaml:"value,omitempty"`
}

// TestCapture stores part of a response in a run variable.
type TestCapture struct {
	Var  string `yaml:"var"`
	From string `yaml:"from"`
	Key  string `yaml:"key,omitempty"`
	Path string `yaml:"path,omitempty"`
}

func (t *TestCase) ID() string {
	return t.MetaData.ID
}

func (t *TestCase) GetKind() string {
	return t.Kind
}

func (t *TestCase) GetName() string {
	return t.MetaData.Name
}

func (t *TestCase) SetName(name string) {
	t.MetaData.Name = name
}

func (t *TestCase) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(t)
}

// AllSteps returns setup, steps and teardown in run order.
func (t *TestCase) AllSteps() []TestStep {
	out := make([]TestStep, 0, len(t.Spec.Setup)+len(t.Spec.Steps)+len(t.Spec.Teardown))
	out = append(out, t.Spec.Setup...)
	out = append(out, t.Spec.Steps...)
	return append(out, t.Spec.Teardown...)
}

func NewTestCase(name string) *TestCase {
	return &TestCase{
		ApiVersion: ApiVersion,
		Kind:       KindTestCase,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: name,
		},
		Spec: TestCaseSpec{
			Steps: []TestStep{},
		},
	}
}

// Clone returns a deep copy with a new ID.
func (t *TestCase) Clone() *TestCase {
	c := &TestCase{
		ApiVersion: t.ApiVersion,
		Kind:       t.Kind,
		MetaData:   MetaData{ID: uuid.NewString(), Name: t.MetaData.Name},
		Spec: TestCaseSpec{
			Description: t.Spec.Description,
			Tags:        cloneSlice(t.Spec.Tags),
			Options:     t.Spec.Options,
			Setup:       cloneSteps(t.Spec.Setup),
			Steps:       cloneSteps(t.Spec.Steps),
			Teardown:    cloneSteps(t.Spec.Teardown),
		},
	}
	if t.Spec.Variables != nil {
		c.Spec.Variables = make([]TestVariable, len(t.Spec.Variables))
		for i, v := range t.Spec.Variables {
			c.Spec.Variables[i] = v
			if v.From != nil {
				from := *v.From
				c.Spec.Variables[i].From = &from
			}
		}
	}
	return c
}

func cloneSteps(steps []TestStep) []TestStep {
	if steps == nil {
		return nil
	}
	out := make([]TestStep, len(steps))
	for i, s := range steps {
		out[i] = s
		if s.With != nil {
			w := TestStepOverrides{
				Headers: cloneSlice(s.With.Headers),
				Query:   cloneSlice(s.With.Query),
				Body:    s.With.Body,
			}
			if s.With.Variables != nil {
				w.Variables = make(map[string]string, len(s.With.Variables))
				for k, v := range s.With.Variables {
					w.Variables[k] = v
				}
			}
			out[i].With = &w
		}
		if s.Retry != nil {
			r := *s.Retry
			out[i].Retry = &r
		}
		if s.Assert != nil {
			out[i].Assert = make([]TestAssertion, len(s.Assert))
			for j, a := range s.Assert {
				out[i].Assert[j] = a
				out[i].Assert[j].Value = cloneValue(a.Value)
			}
		}
		out[i].Capture = cloneSlice(s.Capture)
	}
	return out
}

func cloneSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	return append(make([]T, 0, len(s)), s...)
}

// cloneValue deep-copies a value decoded from YAML.
func cloneValue(v any) any {
	switch t := v.(type) {
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = cloneValue(e)
		}
		return out
	case map[any]any:
		out := make(map[any]any, len(t))
		for k, e := range t {
			out[k] = cloneValue(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = cloneValue(e)
		}
		return out
	}
	return v
}

// TestBundle is one file that holds test cases with everything they need
// to run outside the app: the requests they send, the collections those
// belong to, and optionally an environment. `chapar-cli test` runs it without
// a workspace.
type TestBundle struct {
	ApiVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	MetaData   MetaData       `yaml:"metadata"`
	Spec       TestBundleSpec `yaml:"spec"`
}

type TestBundleSpec struct {
	TestCases []*TestCase `yaml:"testCases"`
	// Collections hold only the requests the cases use, with the headers
	// and auth those requests inherit.
	Collections []*Collection `yaml:"collections,omitempty"`
	Requests    []*Request    `yaml:"requests,omitempty"`
	Environment *Environment  `yaml:"environment,omitempty"`
	// SecretsLeftOut names secret environment values that were not
	// exported; they have to be given when the bundle runs.
	SecretsLeftOut []string `yaml:"secretsLeftOut,omitempty"`
}
