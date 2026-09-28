package testcase

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/ui"
	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/testrun"
	"github.com/chapar-rest/chapar/ui/container"
)

// stepModel is the form state of one step. Durations and assertion values
// are kept as the text the user typed, and parsed when the case is dumped.
type stepModel struct {
	key  string // stable widget key; steps can move
	open bool

	id, name          string
	request           domain.TestRequestRef
	timeout           string
	retryCount        string
	retryDelay        string
	continueOnFailure bool

	asserts  []*assertModel
	captures []*captureModel

	showOverrides bool
	vars          *ui.Table
	headers       *ui.Table
	query         *ui.Table
	body          *ui.Editor
}

type assertModel struct {
	key    string
	target string
	// sel is the key or the path, whichever the target uses.
	sel   string
	op    string
	value string
}

type captureModel struct {
	key  string
	v    string
	from string
	sel  string
}

func newKey() string { return uuid.NewString() }

func loadStep(s domain.TestStep, markDirty func()) *stepModel {
	m := &stepModel{
		key:               newKey(),
		id:                s.ID,
		name:              s.Name,
		request:           s.Request,
		timeout:           durationText(s.Timeout),
		continueOnFailure: s.ContinueOnFailure,
	}
	if s.Retry != nil {
		m.retryCount = strconv.Itoa(s.Retry.Count)
		m.retryDelay = durationText(s.Retry.Delay)
	}
	for _, a := range s.Assert {
		sel := a.Key
		if usesPath(a.Target) {
			sel = a.Path
		}
		m.asserts = append(m.asserts, &assertModel{key: newKey(), target: a.Target, sel: sel, op: a.Op, value: valueText(a.Value)})
	}
	for _, c := range s.Capture {
		sel := c.Key
		if usesPath(c.From) {
			sel = c.Path
		}
		m.captures = append(m.captures, &captureModel{key: newKey(), v: c.Var, from: c.From, sel: sel})
	}

	m.vars = container.NewKVTable("vars", markDirty)
	m.headers = container.NewKVTable("headers", markDirty)
	m.query = container.NewKVTable("query", markDirty)
	body := ""
	if w := s.With; w != nil {
		var vars []domain.KeyValue
		for k, v := range w.Variables {
			vars = append(vars, domain.KeyValue{Key: k, Value: v, Enable: true})
		}
		container.LoadKV(m.vars, sortKV(vars))
		container.LoadKV(m.headers, toKV(w.Headers))
		container.LoadKV(m.query, toKV(w.Query))
		body = w.Body
		m.showOverrides = len(vars)+len(w.Headers)+len(w.Query) > 0 || body != ""
	}
	m.body = ui.NewEditor([]byte(body), highlight.NewJSON(), ui.WithSoftWrap(true))
	return m
}

func (m *stepModel) close() {
	m.body.Close()
}

// dump turns the form back into a step. Text that does not parse is
// reported, and left out of the step.
func (m *stepModel) dump(where string, problems *[]testrun.Problem) domain.TestStep {
	s := domain.TestStep{
		ID:                strings.TrimSpace(m.id),
		Name:              strings.TrimSpace(m.name),
		Request:           m.request,
		ContinueOnFailure: m.continueOnFailure,
	}
	s.Timeout = parseDuration(m.timeout, where+".timeout", problems)
	count, delay := strings.TrimSpace(m.retryCount), strings.TrimSpace(m.retryDelay)
	if count != "" || delay != "" {
		n, err := strconv.Atoi(count)
		if count != "" && err != nil {
			*problems = append(*problems, testrun.Problem{Where: where + ".retry", Message: fmt.Sprintf("retries %q is not a whole number", count)})
		}
		s.Retry = &domain.TestRetry{Count: n, Delay: parseDuration(delay, where+".retry", problems)}
	}

	for _, a := range m.asserts {
		ta := domain.TestAssertion{Target: a.target, Op: a.op, Value: parseValue(a.value)}
		if usesPath(a.target) {
			ta.Path = strings.TrimSpace(a.sel)
		} else if usesKey(a.target) {
			ta.Key = strings.TrimSpace(a.sel)
		}
		if a.op == domain.TestOpExists || a.op == domain.TestOpNotExists {
			ta.Value = nil
		}
		s.Assert = append(s.Assert, ta)
	}
	for _, c := range m.captures {
		tc := domain.TestCapture{Var: strings.TrimSpace(c.v), From: c.from}
		if usesPath(c.from) {
			tc.Path = strings.TrimSpace(c.sel)
		} else if usesKey(c.from) {
			tc.Key = strings.TrimSpace(c.sel)
		}
		s.Capture = append(s.Capture, tc)
	}

	w := &domain.TestStepOverrides{
		Headers: fromKV(container.DumpKV(m.headers)),
		Query:   fromKV(container.DumpKV(m.query)),
		Body:    string(m.body.Bytes()),
	}
	for _, kv := range container.DumpKV(m.vars) {
		if kv.Enable && kv.Key != "" {
			if w.Variables == nil {
				w.Variables = map[string]string{}
			}
			w.Variables[kv.Key] = kv.Value
		}
	}
	if len(w.Variables)+len(w.Headers)+len(w.Query) > 0 || w.Body != "" {
		s.With = w
	}
	return s
}

func (m *stepModel) modified() bool { return m.body.Modified() }

func (m *stepModel) markSaved() { m.body.MarkSaved() }

// displayName is how a step is shown: its name, else its request.
func (m *stepModel) displayName() string {
	switch {
	case strings.TrimSpace(m.name) != "":
		return m.name
	case m.request.Ref != "":
		return m.request.Ref
	case m.id != "":
		return m.id
	}
	return "New step"
}

func toKV(kvs []domain.TestKeyValue) []domain.KeyValue {
	out := make([]domain.KeyValue, 0, len(kvs))
	for _, kv := range kvs {
		out = append(out, domain.KeyValue{Key: kv.Key, Value: kv.Value, Enable: true})
	}
	return out
}

// fromKV keeps the enabled rows that have a key.
func fromKV(kvs []domain.KeyValue) []domain.TestKeyValue {
	var out []domain.TestKeyValue
	for _, kv := range kvs {
		if kv.Enable && kv.Key != "" {
			out = append(out, domain.TestKeyValue{Key: kv.Key, Value: kv.Value})
		}
	}
	return out
}

func sortKV(kvs []domain.KeyValue) []domain.KeyValue {
	for i := 1; i < len(kvs); i++ {
		for j := i; j > 0 && kvs[j].Key < kvs[j-1].Key; j-- {
			kvs[j], kvs[j-1] = kvs[j-1], kvs[j]
		}
	}
	return kvs
}

func usesPath(target string) bool { return target == domain.TestTargetBody }

func usesKey(target string) bool {
	switch target {
	case domain.TestTargetHeader, domain.TestTargetCookie, domain.TestTargetMetadata, domain.TestTargetTrailer:
		return true
	}
	return false
}

func durationText(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

func parseDuration(s, where string, problems *[]testrun.Problem) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		*problems = append(*problems, testrun.Problem{Where: where, Message: fmt.Sprintf("%q is not a duration such as 500ms or 10s", s)})
		return 0
	}
	return d
}

// parseValue reads an assertion value as typed: true/false, null and
// numbers keep their type, [a, b] is a list and quotes make a string of
// anything. All else is a string as it is.
func parseValue(s string) any {
	t := strings.TrimSpace(s)
	switch {
	case t == "":
		return nil
	case t == "true":
		return true
	case t == "false":
		return false
	case t == "null":
		return nil
	case strings.HasPrefix(t, "[") || strings.HasPrefix(t, `"`) || strings.HasPrefix(t, "'"):
		var v any
		if err := yaml.Unmarshal([]byte(t), &v); err == nil {
			return v
		}
		return s
	}
	if n, err := strconv.ParseInt(t, 10, 64); err == nil {
		return int(n)
	}
	if f, err := strconv.ParseFloat(t, 64); err == nil {
		return f
	}
	return s
}

// valueText is the text parseValue reads back as v.
func valueText(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		if _, isString := parseValue(t).(string); !isString || strings.TrimSpace(t) != t {
			return strconv.Quote(t)
		}
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = valueText(e)
			if s, ok := e.(string); ok && strings.ContainsAny(s, ",[]") {
				parts[i] = strconv.Quote(s)
			}
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	b, err := json.Marshal(normalizeKeys(v))
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// normalizeKeys converts YAML maps to string-keyed ones for JSON.
func normalizeKeys(v any) any {
	switch t := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[fmt.Sprint(k)] = normalizeKeys(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = normalizeKeys(e)
		}
		return out
	}
	return v
}

// uniqueStepID returns base, or base-2, base-3… when taken.
func uniqueStepID(base string, taken map[string]bool) string {
	base = slug(base)
	if base == "" {
		base = "step"
	}
	id := base
	for n := 2; taken[id]; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}
