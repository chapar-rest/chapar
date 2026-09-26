package testrun

import (
	"regexp"

	"github.com/chapar-rest/chapar/internal/domain"
)

var placeholder = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

// scope holds the variables of one run.
//
// env is a copy of the selected environment. vars are layered over it:
// the case's variables, then captures and the env changes the requests'
// own scripts and extract rules made. A step's with.variables go on top
// of both, for that step only.
type scope struct {
	env  *domain.Environment
	vars map[string]string

	// set and unset are the env changes requests made during the run,
	// for writing back with persistEnv.
	set   map[string]string
	unset map[string]bool
}

func newScope(env *domain.Environment) *scope {
	if env == nil {
		env = &domain.Environment{ApiVersion: domain.ApiVersion, Kind: domain.KindEnv}
	} else {
		env = copyEnv(env)
	}
	return &scope{env: env, vars: map[string]string{}, set: map[string]string{}, unset: map[string]bool{}}
}

// lookup finds a variable the way a step sees it, without step overrides.
func (s *scope) lookup(name string) (string, bool) {
	if v, ok := s.vars[name]; ok {
		return v, true
	}
	for _, kv := range s.env.Spec.Values {
		if kv.Key == name && kv.Enable {
			return kv.Value, true
		}
	}
	return "", false
}

// expand replaces the {{name}} placeholders it knows in text. Unknown
// names, such as built-ins, are left for the send to fill in.
func (s *scope) expand(text string, extra map[string]string) string {
	return placeholder.ReplaceAllStringFunc(text, func(m string) string {
		name := m[2 : len(m)-2]
		if v, ok := extra[name]; ok {
			return v
		}
		if v, ok := s.lookup(name); ok {
			return v
		}
		return m
	})
}

// expandValue expands the placeholders in the text of an assertion value.
func (s *scope) expandValue(v any, extra map[string]string) any {
	switch t := v.(type) {
	case string:
		return s.expand(t, extra)
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = s.expandValue(e, extra)
		}
		return out
	}
	return v
}

// stepVars expands a step's with.variables. Values are expanded here
// because the send substitutes env values only one level deep.
func (s *scope) stepVars(with map[string]string) map[string]string {
	out := make(map[string]string, len(with))
	for k, v := range with {
		out[k] = s.expand(v, nil)
	}
	return out
}

// stepEnv builds the environment one send uses, with the step's expanded
// variables on top.
func (s *scope) stepEnv(with map[string]string) *domain.Environment {
	env := copyEnv(s.env)
	for k, v := range s.vars {
		setVar(env, k, v)
	}
	for k, v := range with {
		setVar(env, k, v)
	}
	return env
}

// merge keeps the changes a send made to env, taken from before.
func (s *scope) merge(before map[string]string, env *domain.Environment) {
	after := snapshot(env)
	for k, v := range after {
		if old, ok := before[k]; ok && old == v {
			continue
		}
		s.vars[k] = v
		setVar(s.env, k, v)
		s.set[k] = v
		delete(s.unset, k)
	}
	for k := range before {
		if _, ok := after[k]; ok {
			continue
		}
		delete(s.vars, k)
		s.env.UnsetKey(k)
		s.unset[k] = true
		delete(s.set, k)
	}
}

// changed reports whether requests changed the env during the run.
func (s *scope) changed() bool {
	return len(s.set) > 0 || len(s.unset) > 0
}

// applyChanges writes the run's env changes into env.
func (s *scope) applyChanges(env *domain.Environment) {
	for k, v := range s.set {
		setVar(env, k, v)
	}
	for k := range s.unset {
		env.UnsetKey(k)
	}
}

func snapshot(env *domain.Environment) map[string]string {
	out := make(map[string]string, len(env.Spec.Values))
	for _, kv := range env.Spec.Values {
		out[kv.Key] = kv.Value
	}
	return out
}

// copyEnv copies env, keeping its ID: the cookie jar is chosen by it.
func copyEnv(env *domain.Environment) *domain.Environment {
	c := *env
	c.Spec.Values = append([]domain.KeyValue(nil), env.Spec.Values...)
	return &c
}

func setVar(env *domain.Environment, key, value string) {
	for i, kv := range env.Spec.Values {
		if kv.Key == key {
			env.Spec.Values[i].Value = value
			env.Spec.Values[i].Enable = true
			return
		}
	}
	env.Spec.Values = append(env.Spec.Values, domain.KeyValue{Key: key, Value: value, Enable: true})
}
