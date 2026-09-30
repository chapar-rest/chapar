package testcli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
)

// overrides are environment values given on the command line, applied
// over the chosen environment in this order.
type overrides struct {
	file  string   // --env-file: a chapar environment file, or KEY=VALUE lines
	osEnv string   // --os-env: OS variables with this prefix, prefix removed
	vars  []string // --var key=value
}

func (o overrides) empty() bool { return o.file == "" && o.osEnv == "" && len(o.vars) == 0 }

// apply returns a copy of env with the overrides set, and the keys they
// set. A nil env with overrides becomes an environment of its own.
func (o overrides) apply(env *domain.Environment, environ []string) (*domain.Environment, map[string]bool, error) {
	if o.empty() {
		return env, nil, nil
	}
	if env == nil {
		env = domain.NewEnvironment("command line")
	} else {
		cp := *env
		cp.Spec.Values = append([]domain.KeyValue(nil), env.Spec.Values...)
		env = &cp
	}
	set := map[string]bool{}
	put := func(k, v string) {
		env.SetKey(k, v)
		for i := range env.Spec.Values {
			if env.Spec.Values[i].Key == k {
				env.Spec.Values[i].Enable = true
			}
		}
		set[k] = true
	}

	if o.file != "" {
		values, err := readEnvFile(o.file)
		if err != nil {
			return nil, nil, err
		}
		for _, kv := range values {
			put(kv.Key, kv.Value)
		}
	}
	if o.osEnv != "" {
		for _, e := range environ {
			k, v, ok := strings.Cut(e, "=")
			if ok && strings.HasPrefix(k, o.osEnv) && len(k) > len(o.osEnv) {
				put(strings.TrimPrefix(k, o.osEnv), v)
			}
		}
	}
	for _, kv := range o.vars {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return nil, nil, fmt.Errorf("--var %q: want key=value", kv)
		}
		put(k, v)
	}
	return env, set, nil
}

// readEnvFile reads a chapar environment file (as exported from the app)
// or a .env file of KEY=VALUE lines.
func readEnvFile(path string) ([]domain.KeyValue, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var env domain.Environment
	if yaml.Unmarshal(b, &env) == nil && env.Kind == domain.KindEnv {
		var out []domain.KeyValue
		for _, kv := range env.Spec.Values {
			if kv.Enable {
				out = append(out, kv)
			}
		}
		return out, nil
	}

	var out []domain.KeyValue
	sc := bufio.NewScanner(bytes.NewReader(b))
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: want KEY=VALUE", path, n)
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
		out = append(out, domain.KeyValue{Key: k, Value: v, Enable: true})
	}
	return out, sc.Err()
}

// requestChanges returns what a run changed in an environment: the values
// saved differs from sent, and the keys it removed.
func requestChanges(sent, saved *domain.Environment) (set map[string]string, unset []string) {
	before := map[string]string{}
	for _, kv := range sent.Spec.Values {
		before[kv.Key] = kv.Value
	}
	after := map[string]bool{}
	set = map[string]string{}
	for _, kv := range saved.Spec.Values {
		after[kv.Key] = true
		if old, ok := before[kv.Key]; !ok || old != kv.Value {
			set[kv.Key] = kv.Value
		}
	}
	for k := range before {
		if !after[k] {
			unset = append(unset, k)
		}
	}
	sort.Strings(unset)
	return set, unset
}
