// Package testrun runs test cases and checks their responses.
// See docs/testcases-design.md.
package testrun

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/internal/scripting"
)

// Where an assertion came from.
const (
	SourceAssert = "assert"
	SourceScript = "script"
)

// AssertionResult is one assertion checked against a response.
type AssertionResult struct {
	Source   string `json:"source"`
	Target   string `json:"target,omitempty"`
	Key      string `json:"key,omitempty"`
	Path     string `json:"path,omitempty"`
	Op       string `json:"op,omitempty"`
	Expected any    `json:"expected,omitempty"`
	Actual   any    `json:"actual,omitempty"`
	Passed   bool   `json:"passed"`
	// Message says why it failed.
	Message string `json:"message,omitempty"`
}

// CaptureResult is one value a step stored in a run variable.
type CaptureResult struct {
	Var   string `json:"var"`
	Value string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

var errNoResponse = errors.New("no response")

// Evaluate checks asserts against res.
func Evaluate(res *egress.Response, asserts []domain.TestAssertion) []AssertionResult {
	r := &response{res: res}
	out := make([]AssertionResult, 0, len(asserts))
	for _, a := range asserts {
		out = append(out, r.assert(a))
	}
	return out
}

// ScriptAssertions reports the chapar.test() calls of a request's scripts
// as assertions.
func ScriptAssertions(tests []scripting.TestResult) []AssertionResult {
	out := make([]AssertionResult, 0, len(tests))
	for _, t := range tests {
		out = append(out, AssertionResult{
			Source:  SourceScript,
			Path:    t.Name,
			Passed:  t.Passed,
			Message: t.Error,
		})
	}
	return out
}

// Capture reads the values caps store from res. vars holds the ones that
// were read; a capture that fails leaves its variable unset.
func Capture(res *egress.Response, caps []domain.TestCapture) (results []CaptureResult, vars map[string]string) {
	r := &response{res: res}
	vars = make(map[string]string, len(caps))
	for _, c := range caps {
		cr := CaptureResult{Var: c.Var}
		v, found, err := r.lookup(c.From, c.Key, c.Path)
		switch {
		case err != nil:
			cr.Error = err.Error()
		case !found:
			cr.Error = "not found"
		default:
			cr.Value = scripting.EnvText(v)
			vars[c.Var] = cr.Value
		}
		results = append(results, cr)
	}
	return results, vars
}

// response reads the parts of an egress.Response that tests look at,
// decoding the body at most once.
type response struct {
	res     *egress.Response
	decoded bool
	body    any
	bodyErr error
}

func (r *response) assert(a domain.TestAssertion) AssertionResult {
	out := AssertionResult{
		Source:   SourceAssert,
		Target:   a.Target,
		Key:      a.Key,
		Path:     a.Path,
		Op:       a.Op,
		Expected: normalize(a.Value),
	}
	actual, found, err := r.lookup(a.Target, a.Key, a.Path)
	if err != nil {
		out.Message = err.Error()
		return out
	}
	if found {
		out.Actual = actual
	}
	out.Passed, out.Message = apply(a.Op, actual, found, out.Expected)
	return out
}

// lookup returns the value target selects, as a JSON type. found is false
// when the key or path matches nothing.
func (r *response) lookup(target, key, path string) (any, bool, error) {
	res := r.res
	if res == nil {
		return nil, false, errNoResponse
	}
	switch target {
	case domain.TestTargetStatus:
		code := res.StatusCode
		if code == 0 {
			code = res.StatueCode
		}
		return float64(code), true, nil
	case domain.TestTargetHeader:
		for k, v := range res.ResponseHeaders {
			if strings.EqualFold(k, key) {
				return v, true, nil
			}
		}
		return nil, false, nil
	case domain.TestTargetCookie:
		for _, c := range res.Cookies {
			if c.Name == key {
				return c.Value, true, nil
			}
		}
		return nil, false, nil
	case domain.TestTargetMetadata:
		return findKV(res.ResponseMetadata, key)
	case domain.TestTargetTrailer:
		return findKV(res.Trailers, key)
	case domain.TestTargetBody:
		body, err := r.json()
		if err != nil {
			return nil, false, err
		}
		v, found, err := jsonpath.Query(body, path)
		if err != nil {
			return nil, false, fmt.Errorf("bad JSONPath %q: %w", path, err)
		}
		return normalize(v), found, nil
	case domain.TestTargetText:
		return string(res.Body), true, nil
	case domain.TestTargetTime:
		return float64(res.TimePassed.Milliseconds()), true, nil
	case domain.TestTargetSize:
		return float64(responseSize(res)), true, nil
	}
	return nil, false, fmt.Errorf("unknown target %q", target)
}

func (r *response) json() (any, error) {
	if !r.decoded {
		r.decoded = true
		if err := json.Unmarshal(r.res.Body, &r.body); err != nil {
			r.bodyErr = fmt.Errorf("response body is not JSON: %w", err)
		}
	}
	return r.body, r.bodyErr
}

func findKV(kvs []domain.KeyValue, key string) (any, bool, error) {
	for _, kv := range kvs {
		if strings.EqualFold(kv.Key, key) {
			return kv.Value, true, nil
		}
	}
	return nil, false, nil
}

// responseSize is the body size. Not every protocol fills Size.
func responseSize(res *egress.Response) int {
	if res.Size > 0 {
		return res.Size
	}
	return len(res.Body)
}
