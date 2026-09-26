package testrun

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/chapar-rest/chapar/internal/domain"
)

// JSON type names accepted by the type operator.
var typeNames = map[string]bool{
	"string": true, "number": true, "boolean": true, "null": true, "array": true, "object": true,
}

// normalize converts a value decoded from YAML or JSON to JSON types:
// numbers become float64 and maps get string keys.
func normalize(v any) any {
	switch t := v.(type) {
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case uint64:
		return float64(t)
	case float32:
		return float64(t)
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = normalize(e)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[fmt.Sprint(k)] = normalize(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = normalize(e)
		}
		return out
	}
	return v
}

// apply checks actual against want with op. found is false when the
// selector matched nothing. On failure it also returns why.
func apply(op string, actual any, found bool, want any) (bool, string) {
	switch op {
	case domain.TestOpExists:
		if !found {
			return false, "not found"
		}
		return true, ""
	case domain.TestOpNotExists:
		if found {
			return false, "found " + show(actual)
		}
		return true, ""
	}
	if !found {
		return false, "not found"
	}

	switch op {
	case domain.TestOpEq:
		return check(equal(actual, want), "got %s, want %s", show(actual), show(want))
	case domain.TestOpNe:
		return check(!equal(actual, want), "got %s, want anything else", show(actual))
	case domain.TestOpGt, domain.TestOpGte, domain.TestOpLt, domain.TestOpLte:
		a, ok := number(actual)
		if !ok {
			return false, show(actual) + " is not a number"
		}
		b, ok := number(want)
		if !ok {
			return false, show(want) + " is not a number"
		}
		var passed bool
		switch op {
		case domain.TestOpGt:
			passed = a > b
		case domain.TestOpGte:
			passed = a >= b
		case domain.TestOpLt:
			passed = a < b
		default:
			passed = a <= b
		}
		return check(passed, "got %s, want %s %s", show(actual), op, show(want))
	case domain.TestOpIn:
		list, ok := want.([]any)
		if !ok {
			return false, "value must be a list"
		}
		for _, w := range list {
			if equal(actual, w) {
				return true, ""
			}
		}
		return false, fmt.Sprintf("got %s, want one of %s", show(actual), show(want))
	case domain.TestOpContains, domain.TestOpNotContains:
		has, ok := contains(actual, want)
		if !ok {
			return false, "cannot look inside " + typeName(actual)
		}
		if op == domain.TestOpContains {
			return check(has, "%s does not contain %s", show(actual), show(want))
		}
		return check(!has, "%s contains %s", show(actual), show(want))
	case domain.TestOpMatches:
		re, err := regexp.Compile(text(want))
		if err != nil {
			return false, "bad regexp: " + err.Error()
		}
		return check(re.MatchString(text(actual)), "%s does not match %s", show(actual), text(want))
	case domain.TestOpType:
		got := typeName(actual)
		return check(got == want, "got %s, want %s", got, show(want))
	case domain.TestOpLength:
		n, ok := length(actual)
		if !ok {
			return false, typeName(actual) + " has no length"
		}
		w, ok := number(want)
		if !ok {
			return false, show(want) + " is not a number"
		}
		return check(float64(n) == w, "length is %d, want %s", n, show(want))
	}
	return false, fmt.Sprintf("unknown op %q", op)
}

func check(passed bool, format string, args ...any) (bool, string) {
	if passed {
		return true, ""
	}
	return false, fmt.Sprintf(format, args...)
}

// equal compares JSON values. A number or bool also equals its text, since
// headers, cookies and metadata are always strings.
func equal(a, b any) bool {
	if as, ok := a.(string); ok {
		if _, ok := b.(string); !ok {
			a, b = b, as
		}
	}
	// Now a is the non-string side, if either is.
	if bs, ok := b.(string); ok {
		switch at := a.(type) {
		case float64:
			n, ok := number(bs)
			return ok && n == at
		case bool:
			v, err := strconv.ParseBool(strings.TrimSpace(bs))
			return err == nil && v == at
		}
	}
	return reflect.DeepEqual(a, b)
}

func contains(actual, want any) (has, ok bool) {
	switch t := actual.(type) {
	case string:
		return strings.Contains(t, text(want)), true
	case []any:
		for _, e := range t {
			if equal(e, want) {
				return true, true
			}
		}
		return false, true
	}
	return false, false
}

func number(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	}
	return 0, false
}

func length(v any) (int, bool) {
	switch t := v.(type) {
	case string:
		return utf8.RuneCountInString(t), true
	case []any:
		return len(t), true
	case map[string]any:
		return len(t), true
	}
	return 0, false
}

func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return fmt.Sprintf("%T", v)
}

// text is v as plain text: strings as they are, anything else as JSON.
func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// show is v for a message: like text, but strings are quoted.
func show(v any) string {
	if s, ok := v.(string); ok {
		return strconv.Quote(s)
	}
	return text(v)
}
