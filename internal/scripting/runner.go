package scripting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
)

type Executor interface {
	Init(cfg domain.ScriptingConfig) error
	Execute(ctx context.Context, script string, params *ExecParams) (*ExecResult, error)
	Name() string
	Shutdown() error
}

func GetExecutor(language string, cfg domain.ScriptingConfig) (Executor, error) {
	n := strings.ToLower(language)
	if n == "python" {
		return NewPythonExecutor(cfg), nil
	}

	return nil, fmt.Errorf("unknown scripting executor: %s", language)
}

// Phase is when a script runs.
type Phase string

const (
	PhasePre  Phase = "pre"
	PhasePost Phase = "post"
)

// Protocol is the kind of request a script runs for.
type Protocol string

const (
	ProtocolHTTP    Protocol = "http"
	ProtocolGRPC    Protocol = "grpc"
	ProtocolGraphQL Protocol = "graphql"
)

// ProtocolOf maps a request type to the protocol scripts see.
func ProtocolOf(t domain.RequestType) Protocol {
	switch t {
	case domain.RequestTypeGRPC:
		return ProtocolGRPC
	case domain.RequestTypeGraphQL:
		return ProtocolGraphQL
	}
	return ProtocolHTTP
}

// Pair is one header, metadata entry or query param. It travels as a
// two-element JSON array, so repeated keys and their order survive.
type Pair [2]string

// RequestData is the request a script sees. Fields keep their {{variables}};
// Resolved holds the same request with them filled in.
//
// For gRPC, URL is the server address, Method the full method name and
// Headers the metadata. For GraphQL the document and variables are in
// GraphQL and Body is empty.
type RequestData struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    []Pair            `json:"headers"`
	Body       string            `json:"body"`
	Query      []Pair            `json:"query,omitempty"`
	PathParams map[string]string `json:"path_params,omitempty"`
	GraphQL    *GraphQLData      `json:"graphql,omitempty"`
	Resolved   *RequestData      `json:"resolved,omitempty"`
}

type GraphQLData struct {
	Query     string `json:"query"`
	Variables string `json:"variables"`
}

// RequestChanges is what a pre-request script changed. Nil fields were left
// alone.
type RequestChanges struct {
	URL        *string            `json:"url,omitempty"`
	Method     *string            `json:"method,omitempty"`
	Headers    *[]Pair            `json:"headers,omitempty"`
	Body       *string            `json:"body,omitempty"`
	Query      *[]Pair            `json:"query,omitempty"`
	PathParams *map[string]string `json:"path_params,omitempty"`
	GraphQL    *GraphQLChanges    `json:"graphql,omitempty"`
}

type GraphQLChanges struct {
	Query     *string `json:"query,omitempty"`
	Variables *string `json:"variables,omitempty"`
}

// Empty reports whether the script changed nothing.
func (c *RequestChanges) Empty() bool {
	return c == nil || *c == (RequestChanges{})
}

// ResponseData is the response a post-request script sees. For gRPC,
// StatusCode is the gRPC code and Metadata/Trailers what the server sent.
type ResponseData struct {
	StatusCode int      `json:"status_code"`
	Status     string   `json:"status"`
	Headers    []Pair   `json:"headers"`
	Body       string   `json:"body"`
	ElapsedMS  float64  `json:"elapsed_ms"`
	Size       int      `json:"size"`
	Error      string   `json:"error,omitempty"`
	Cookies    []Cookie `json:"cookies,omitempty"`
	Metadata   []Pair   `json:"metadata,omitempty"`
	Trailers   []Pair   `json:"trailers,omitempty"`
}

type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Expires  string `json:"expires,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"http_only,omitempty"`
}

type ExecParams struct {
	Phase    Phase
	Protocol Protocol
	Req      *RequestData
	// Res is nil in pre-request scripts.
	Res *ResponseData
	Env *domain.Environment
}

// TestResult is one chapar.test() a script ran.
type TestResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// ScriptError is an exception the script raised, or a failure to run it
// (a timeout, a crash).
type ScriptError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Line      int    `json:"line,omitempty"`
	Traceback string `json:"traceback,omitempty"`
}

func (e *ScriptError) Error() string {
	msg := e.Type
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.Line > 0 {
		msg = fmt.Sprintf("line %d: %s", e.Line, msg)
	}
	return msg
}

type ExecResult struct {
	// Request is what a pre-request script changed; nil after post-request.
	Request  *RequestChanges `json:"request"`
	EnvSet   map[string]any  `json:"env_set"`
	EnvUnset []string        `json:"env_unset"`
	Prints   []string        `json:"prints"`
	Tests    []TestResult    `json:"tests"`
	// Skip is set when a pre-request script called chapar.skip().
	Skip       bool         `json:"skip"`
	SkipReason string       `json:"skip_reason"`
	Error      *ScriptError `json:"error"`
}

// ApplyEnv writes the script's environment changes into env and reports
// whether anything changed. Values that are not text are stored as JSON.
func (r *ExecResult) ApplyEnv(env *domain.Environment) bool {
	if env == nil || r == nil {
		return false
	}
	changed := false
	for k, v := range r.EnvSet {
		env.SetKey(k, EnvText(v))
		changed = true
	}
	for _, k := range r.EnvUnset {
		if env.UnsetKey(k) {
			changed = true
		}
	}
	return changed
}

// EnvText is how a value a script set is stored in an environment.
func EnvText(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// Summary describes the prints and tests of a run, one per line, for the
// request timeline.
func (r *ExecResult) Summary() string {
	if r == nil {
		return ""
	}
	var lines []string
	for _, t := range r.Tests {
		if t.Passed {
			lines = append(lines, "✓ "+t.Name)
		} else {
			lines = append(lines, "✗ "+t.Name+": "+t.Error)
		}
	}
	lines = append(lines, r.Prints...)
	return strings.Join(lines, "\n")
}

// FailedTests counts the tests that did not pass.
func (r *ExecResult) FailedTests() int {
	n := 0
	if r != nil {
		for _, t := range r.Tests {
			if !t.Passed {
				n++
			}
		}
	}
	return n
}
