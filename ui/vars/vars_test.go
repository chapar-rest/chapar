package vars

import (
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga/theme"
)

func testSource() Source {
	env := &domain.Environment{Spec: domain.EnvSpec{Values: []domain.KeyValue{
		{Key: "host", Value: "https://api.example.com", Enable: true},
		{Key: "hostPort", Value: "8080", Enable: true},
		{Key: "disabled", Value: "no", Enable: false},
	}}}
	return Source{
		Env: func() *domain.Environment { return env },
		Extra: func() []Entry {
			return FromVariables([]domain.Variable{
				{TargetEnvVariable: "token", Enable: true},
				{TargetEnvVariable: "ignored", Enable: false},
			})
		},
	}
}

func TestEntriesOrderAndFiltering(t *testing.T) {
	entries := testSource().Entries()
	if len(entries) < 3 {
		t.Fatalf("entries: got %d, want the environment, request and built-in names", len(entries))
	}
	if entries[0].Name != "host" || entries[0].Kind != KindEnv {
		t.Fatalf("first entry: got %+v, want the environment's host", entries[0])
	}
	for _, e := range entries {
		if e.Name == "disabled" || e.Name == "ignored" {
			t.Fatalf("disabled entry %q must not be offered", e.Name)
		}
	}
	if _, ok := testSource().Lookup("randomUUID4"); !ok {
		t.Fatalf("built-in dynamic variables should be in scope")
	}
	if _, ok := testSource().Lookup("nope"); ok {
		t.Fatalf("Lookup invented a variable")
	}
}

func TestSuggestInsidePlaceholder(t *testing.T) {
	s := testSource()
	value := "{{ho"
	items, start, end := s.Suggest(value, len(value))
	if len(items) != 2 {
		t.Fatalf("candidates for {{ho: got %d want 2 (%v)", len(items), items)
	}
	if start != 2 || end != len(value) {
		t.Fatalf("replaced range: got [%d,%d) want [2,%d)", start, end, len(value))
	}
	if items[0].Insert != "host}}" {
		t.Fatalf("insert: got %q want %q", items[0].Insert, "host}}")
	}
	if items[0].Detail != "https://api.example.com" {
		t.Fatalf("detail should show the value, got %q", items[0].Detail)
	}
}

func TestSuggestKeepsExistingClosingBraces(t *testing.T) {
	s := testSource()
	value := "{{ho}}"
	items, _, _ := s.Suggest(value, 4)
	if len(items) == 0 {
		t.Fatal("expected candidates inside an already closed placeholder")
	}
	if items[0].Insert != "host" {
		t.Fatalf("insert: got %q, want no extra braces", items[0].Insert)
	}

	items, _, _ = s.Suggest("{{ho}", 4)
	if len(items) == 0 || items[0].Insert != "host}" {
		t.Fatalf("a single closing brace should be completed, got %v", items)
	}
}

func TestSuggestOutsidePlaceholder(t *testing.T) {
	s := testSource()
	for _, tc := range []struct {
		value string
		caret int
	}{
		{"https://example.com", 19},
		{"{host", 5},          // single brace is a path parameter, not a variable
		{"{{host}}/v1/x", 13}, // past the placeholder
		{"{{a b", 5},          // space ends the name
	} {
		if items, _, _ := s.Suggest(tc.value, tc.caret); len(items) != 0 {
			t.Fatalf("%q at %d should offer nothing, got %v", tc.value, tc.caret, items)
		}
	}
}

func TestSuggestFiltersOnSubstring(t *testing.T) {
	s := testSource()
	value := "{{port"
	items, _, _ := s.Suggest(value, len(value))
	if len(items) != 1 || items[0].Label != "hostPort" {
		t.Fatalf("substring match: got %v want hostPort", items)
	}
}

func TestHighlightMarksUnknownNames(t *testing.T) {
	th := theme.Current()
	s := testSource()
	value := "{{host}}/x/{{nope}}"
	spans := s.Highlight(value)
	if len(spans) != 2 {
		t.Fatalf("spans: got %d want 2 (%v)", len(spans), spans)
	}
	if spans[0].Color != th.Info {
		t.Fatalf("a defined name should use the info color")
	}
	if spans[1].Color != th.Warning {
		t.Fatalf("an undefined name should use the warning color")
	}
	if spans[0].Start != 0 || spans[0].End != 8 {
		t.Fatalf("first span: got [%d,%d) want [0,8)", spans[0].Start, spans[0].End)
	}
}

func TestHighlightURLAddsPathParams(t *testing.T) {
	s := testSource()
	spans := s.HighlightURL("{{host}}/users/{id}")
	if len(spans) != 2 {
		t.Fatalf("spans: got %d want the placeholder and the path param (%v)", len(spans), spans)
	}
	param := spans[1]
	if param.Start != 15 || param.End != 19 {
		t.Fatalf("path param span: got [%d,%d) want [15,19)", param.Start, param.End)
	}
	if param.Color != theme.Current().Success {
		t.Fatalf("a path param should use its own color")
	}
}

func TestHighlightPlainText(t *testing.T) {
	s := testSource()
	if spans := s.Highlight("https://example.com/v1"); len(spans) != 0 {
		t.Fatalf("plain text should have no spans, got %v", spans)
	}
	if spans := s.Highlight("{}"); len(spans) != 0 {
		t.Fatalf("a bare brace pair should have no spans, got %v", spans)
	}
}

func TestFromKeys(t *testing.T) {
	got := FromKeys(KindParam, []domain.KeyValue{
		{Key: "id", Value: "7", Enable: true},
		{Key: "off", Enable: false},
	})
	if len(got) != 1 || got[0].Name != "id" || got[0].Kind != KindParam {
		t.Fatalf("FromKeys: got %v", got)
	}
}

func TestHoverAtExplainsPlaceholders(t *testing.T) {
	s := testSource()
	value := "{{host}}/users/{id}?t={{nope}}"

	card, ok := s.HoverAt(value, 3)
	if !ok {
		t.Fatal("hovering a defined placeholder should say something")
	}
	if card.Title != "host · "+KindEnv {
		t.Fatalf("title: got %q", card.Title)
	}
	if card.Body != "https://api.example.com" {
		t.Fatalf("body: got %q", card.Body)
	}
	if card.Start != 0 || card.End != 8 {
		t.Fatalf("range: got [%d,%d) want [0,8)", card.Start, card.End)
	}

	// A name nothing defines says so rather than showing an empty value.
	card, ok = s.HoverAt(value, 24)
	if !ok || card.Title != "nope · not defined" {
		t.Fatalf("undefined placeholder: %+v ok=%v", card, ok)
	}

	// Path parameters are explained too, and plain text is not.
	card, ok = s.HoverAt(value, 16)
	if !ok || card.Title != "id · "+KindParam {
		t.Fatalf("path param: %+v ok=%v", card, ok)
	}
	if _, ok := s.HoverAt(value, 10); ok {
		t.Fatal("plain text should have no hover card")
	}
}

func TestHoverAtEmptyValueSaysSo(t *testing.T) {
	env := &domain.Environment{Spec: domain.EnvSpec{Values: []domain.KeyValue{
		{Key: "token", Value: "", Enable: true},
	}}}
	s := Source{Env: func() *domain.Environment { return env }}
	card, ok := s.HoverAt("{{token}}", 4)
	if !ok || card.Body != "(empty)" {
		t.Fatalf("empty value: %+v ok=%v", card, ok)
	}
}
