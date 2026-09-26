package testrun

import (
	"fmt"
	"regexp"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/jsonpath"
)

// Problem is something wrong with a test case file. Where is a path into
// the spec, such as "steps[1].assert[0]".
type Problem struct {
	Where   string
	Message string
}

func (p Problem) Error() string {
	if p.Where == "" {
		return p.Message
	}
	return p.Where + ": " + p.Message
}

// RequestFinder reports why a step's request cannot be found, or nil.
type RequestFinder func(ref domain.TestRequestRef) error

// selector says which of key and path a target uses.
type selector int

const (
	selectNone selector = iota
	selectKey
	selectPath
)

var targets = map[string]selector{
	domain.TestTargetStatus:   selectNone,
	domain.TestTargetHeader:   selectKey,
	domain.TestTargetCookie:   selectKey,
	domain.TestTargetMetadata: selectKey,
	domain.TestTargetTrailer:  selectKey,
	domain.TestTargetBody:     selectPath,
	domain.TestTargetText:     selectNone,
	domain.TestTargetTime:     selectNone,
	domain.TestTargetSize:     selectNone,
}

// Validate lists the problems in tc. find checks that each step's request
// exists; nil skips that check.
func Validate(tc *domain.TestCase, find RequestFinder) []Problem {
	v := &validator{find: find, ids: map[string]string{}}

	if tc.Kind != domain.KindTestCase {
		v.add("", "kind is %q, want %q", tc.Kind, domain.KindTestCase)
	}
	if tc.MetaData.Name == "" {
		v.add("metadata", "name is required")
	}
	if tc.Spec.Options.Timeout < 0 {
		v.add("options", "timeout cannot be negative")
	}

	vars := map[string]bool{}
	for i, tv := range tc.Spec.Variables {
		where := fmt.Sprintf("variables[%d]", i)
		switch {
		case tv.Key == "":
			v.add(where, "key is required")
		case vars[tv.Key]:
			v.add(where, "variable %q is declared twice", tv.Key)
		}
		vars[tv.Key] = true
		if tv.From != nil {
			if tv.From.OsEnv == "" {
				v.add(where, "from needs osEnv")
			}
			if tv.Value != "" {
				v.add(where, "use either value or from, not both")
			}
		}
	}

	if len(tc.Spec.Steps) == 0 {
		v.add("steps", "a test case needs at least one step")
	}
	v.steps("setup", tc.Spec.Setup)
	v.steps("steps", tc.Spec.Steps)
	v.steps("teardown", tc.Spec.Teardown)
	return v.problems
}

type validator struct {
	find     RequestFinder
	ids      map[string]string // step id -> where it was first used
	problems []Problem
}

func (v *validator) add(where, format string, args ...any) {
	v.problems = append(v.problems, Problem{Where: where, Message: fmt.Sprintf(format, args...)})
}

func (v *validator) steps(section string, steps []domain.TestStep) {
	for i, s := range steps {
		v.step(fmt.Sprintf("%s[%d]", section, i), s)
	}
}

func (v *validator) step(where string, s domain.TestStep) {
	switch first, dup := v.ids[s.ID]; {
	case s.ID == "":
		v.add(where, "id is required")
	case dup:
		v.add(where, "id %q is already used by %s", s.ID, first)
	default:
		v.ids[s.ID] = where
	}

	if s.Request.ID == "" && s.Request.Ref == "" {
		v.add(where+".request", "id or ref is required")
	} else if v.find != nil {
		if err := v.find(s.Request); err != nil {
			v.add(where+".request", "%v", err)
		}
	}

	if s.Timeout < 0 {
		v.add(where, "timeout cannot be negative")
	}
	if s.Retry != nil && (s.Retry.Count < 0 || s.Retry.Delay < 0) {
		v.add(where+".retry", "count and delay cannot be negative")
	}
	if s.With != nil {
		for i, h := range s.With.Headers {
			if h.Key == "" {
				v.add(fmt.Sprintf("%s.with.headers[%d]", where, i), "key is required")
			}
		}
		for i, q := range s.With.Query {
			if q.Key == "" {
				v.add(fmt.Sprintf("%s.with.query[%d]", where, i), "key is required")
			}
		}
	}

	for i, a := range s.Assert {
		at := fmt.Sprintf("%s.assert[%d]", where, i)
		v.selector(at, a.Target, a.Key, a.Path)
		v.value(at, a.Op, a.Value)
	}
	for i, c := range s.Capture {
		at := fmt.Sprintf("%s.capture[%d]", where, i)
		if c.Var == "" {
			v.add(at, "var is required")
		}
		v.selector(at, c.From, c.Key, c.Path)
	}
}

// selector checks that target is known and has the key or path it needs.
func (v *validator) selector(where, target, key, path string) {
	sel, ok := targets[target]
	if !ok {
		v.add(where, "unknown target %q", target)
		return
	}
	switch sel {
	case selectKey:
		if key == "" {
			v.add(where, "%s needs a key", target)
		}
	case selectPath:
		if path == "" {
			v.add(where, "%s needs a path", target)
		} else if err := jsonpath.Valid(path); err != nil {
			v.add(where, "bad JSONPath %q: %v", path, err)
		}
	}
	if key != "" && sel != selectKey {
		v.add(where, "%s does not use a key", target)
	}
	if path != "" && sel != selectPath {
		v.add(where, "%s does not use a path", target)
	}
}

// value checks that value has the shape op needs.
func (v *validator) value(where, op string, value any) {
	value = normalize(value)
	switch op {
	case domain.TestOpExists, domain.TestOpNotExists:
		if value != nil {
			v.add(where, "%s takes no value", op)
		}
	case domain.TestOpEq, domain.TestOpNe, domain.TestOpContains, domain.TestOpNotContains:
		if value == nil {
			v.add(where, "%s needs a value (check for null with op: type, value: null)", op)
		}
	case domain.TestOpGt, domain.TestOpGte, domain.TestOpLt, domain.TestOpLte, domain.TestOpLength:
		if _, ok := number(value); !ok {
			v.add(where, "%s needs a number", op)
		}
	case domain.TestOpIn:
		if _, ok := value.([]any); !ok {
			v.add(where, "in needs a list")
		}
	case domain.TestOpMatches:
		s, ok := value.(string)
		if !ok {
			v.add(where, "matches needs a regexp string")
		} else if _, err := regexp.Compile(s); err != nil {
			v.add(where, "bad regexp: %v", err)
		}
	case domain.TestOpType:
		if s, ok := value.(string); !ok || !typeNames[s] {
			v.add(where, "type must be one of string, number, boolean, null, array, object")
		}
	default:
		v.add(where, "unknown op %q", op)
	}
}
