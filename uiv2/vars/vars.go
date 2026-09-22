// Package vars offers the template variables an input can complete, and paints
// the placeholders that name them.
//
// Chapar substitutes {{name}} with an environment value or a built-in dynamic
// value before a request goes out, and {name} in a URL with a path parameter.
// Both read better when the input marks them up, and typing one is much easier
// when the field can finish the name.
package vars

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/variables"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// Kinds of variable, painted as the right-hand hint of a suggestion.
const (
	KindEnv      = "env"
	KindBuiltin  = "dynamic"
	KindResponse = "from response"
	KindParam    = "path param"
)

// Entry is one variable a field can complete.
type Entry struct {
	Name  string
	Value string
	Kind  string
}

// Source is where a field's variable names come from: the active environment,
// the built-in dynamic variables, and whatever the open request adds — the
// names it extracts from a response, say.
type Source struct {
	// Env returns the environment in scope, or nil when there is none.
	Env func() *domain.Environment
	// Extra adds request-scoped names. It may be nil.
	Extra func() []Entry
}

// maxDetailRunes truncates a long value in the suggestion's hint.
const maxDetailRunes = 48

// Entries lists every variable in scope, environment values first, then the
// request's own names, then the built-in dynamic ones.
func (s Source) Entries() []Entry {
	var entries []Entry
	if s.Env != nil {
		if env := s.Env(); env != nil {
			for _, kv := range env.Spec.Values {
				if kv.Enable && kv.Key != "" {
					entries = append(entries, Entry{Name: kv.Key, Value: kv.Value, Kind: KindEnv})
				}
			}
		}
	}
	if s.Extra != nil {
		entries = append(entries, s.Extra()...)
	}
	builtins := make([]Entry, 0, len(variables.GetVariables()))
	for name := range variables.GetVariables() {
		builtins = append(builtins, Entry{Name: name, Kind: KindBuiltin})
	}
	sort.Slice(builtins, func(i, j int) bool { return builtins[i].Name < builtins[j].Name })
	return dedupe(append(entries, builtins...))
}

// dedupe keeps the first entry for each name, so an environment value wins over
// a built-in of the same name — which is also how the request sender resolves it.
func dedupe(entries []Entry) []Entry {
	seen := make(map[string]struct{}, len(entries))
	out := entries[:0]
	for _, e := range entries {
		if _, dup := seen[e.Name]; dup || e.Name == "" {
			continue
		}
		seen[e.Name] = struct{}{}
		out = append(out, e)
	}
	return out
}

// Lookup returns the value behind a name and whether anything defines it.
func (s Source) Lookup(name string) (string, bool) {
	for _, e := range s.Entries() {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

// Suggest is a ui.SuggestFunc: it completes the name being typed inside a
// {{...}} placeholder. Anywhere else it offers nothing.
func (s Source) Suggest(value string, caret int) ([]ui.Suggestion, int, int) {
	start, partial, ok := templateAtCaret(value, caret)
	if !ok {
		return nil, 0, 0
	}
	closing := missingClose(value, caret)
	lower := strings.ToLower(partial)
	var items []ui.Suggestion
	for _, e := range s.Entries() {
		if lower != "" && !strings.Contains(strings.ToLower(e.Name), lower) {
			continue
		}
		items = append(items, ui.Suggestion{
			Label:  e.Name,
			Detail: detail(e),
			Insert: e.Name + closing,
		})
	}
	return items, start, caret
}

// detail is the hint painted next to a suggestion: the value it stands for, or
// what kind of variable it is when there is no value to show yet.
func detail(e Entry) string {
	if e.Value == "" {
		return e.Kind
	}
	v := e.Value
	if utf8.RuneCountInString(v) > maxDetailRunes {
		v = string([]rune(v)[:maxDetailRunes]) + "…"
	}
	return v
}

var (
	// envPattern matches a {{name}} placeholder.
	envPattern = regexp.MustCompile(`\{\{[A-Za-z0-9_$-]*\}\}`)
	// paramPattern matches a {name} path parameter.
	paramPattern = regexp.MustCompile(`\{[A-Za-z0-9_$-]+\}`)
	// namePattern is the run of characters a variable name may be made of.
	namePattern = regexp.MustCompile(`^[A-Za-z0-9_$-]*$`)
)

// Highlight is a ui.TextField highlighter for {{name}} placeholders: a name
// something in scope defines is painted in the info color, one nothing defines
// in the warning color, so a typo shows up before the request is sent.
func (s Source) Highlight(value string) []ui.TextSpan {
	th := theme.Current()
	var spans []ui.TextSpan
	for _, loc := range envPattern.FindAllStringIndex(value, -1) {
		col := th.Warning
		if name := value[loc[0]+2 : loc[1]-2]; name != "" {
			if _, ok := s.Lookup(name); ok {
				col = th.Info
			}
		}
		spans = append(spans, ui.TextSpan{Start: loc[0], End: loc[1], Color: col})
	}
	return spans
}

// HighlightURL is Highlight plus the {name} path parameters a URL carries. It
// suits an address bar; a body or a header value should use Highlight, where a
// lone brace is ordinary text.
func (s Source) HighlightURL(value string) []ui.TextSpan {
	spans := s.Highlight(value)
	th := theme.Current()
	for _, loc := range paramPattern.FindAllStringIndex(value, -1) {
		// {{name}} also matches as the path parameter {name}; the placeholder
		// it sits inside already covers it.
		if covered(spans, loc[0]) {
			continue
		}
		spans = append(spans, ui.TextSpan{Start: loc[0], End: loc[1], Color: th.Success})
	}
	return spans
}

// covered reports whether a span already contains the byte at off.
func covered(spans []ui.TextSpan, off int) bool {
	for _, sp := range spans {
		if off >= sp.Start && off < sp.End {
			return true
		}
	}
	return false
}

// templateAtCaret reports the {{...}} placeholder the caret is typing a name in.
// start is the byte offset just after the opening braces, partial the name so
// far.
func templateAtCaret(value string, caret int) (start int, partial string, ok bool) {
	if caret < 2 || caret > len(value) {
		return 0, "", false
	}
	open := strings.LastIndex(value[:caret], "{{")
	if open < 0 {
		return 0, "", false
	}
	partial = value[open+2 : caret]
	if !namePattern.MatchString(partial) {
		return 0, "", false
	}
	return open + 2, partial, true
}

// missingClose returns the closing braces a completed name still needs, so a
// name typed into an existing {{}} pair is not doubled up.
func missingClose(value string, caret int) string {
	rest := value[caret:]
	switch {
	case strings.HasPrefix(rest, "}}"):
		return ""
	case strings.HasPrefix(rest, "}"):
		return "}"
	default:
		return "}}"
	}
}

// FromVariables turns a request's response-extraction rules into entries, so the
// names it writes back into the environment can be completed before they exist.
func FromVariables(list []domain.Variable) []Entry {
	var out []Entry
	for _, v := range list {
		if v.Enable && v.TargetEnvVariable != "" {
			out = append(out, Entry{Name: v.TargetEnvVariable, Kind: KindResponse})
		}
	}
	return out
}

// FromKeys turns key/value pairs into entries of the given kind.
func FromKeys(kind string, items []domain.KeyValue) []Entry {
	var out []Entry
	for _, kv := range items {
		if kv.Enable && kv.Key != "" {
			out = append(out, Entry{Name: kv.Key, Value: kv.Value, Kind: kind})
		}
	}
	return out
}
