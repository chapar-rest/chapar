package util

import (
	"strings"
	"testing"
)

func TestDetectBodyKind(t *testing.T) {
	cases := []struct {
		ct   string
		body string
		want string
	}{
		{"application/json", `{"a":1}`, BodyKindJSON},
		{"application/xml", `<root/>`, BodyKindXML},
		{"text/html", `<html></html>`, BodyKindHTML},
		{"", `{"x":true}`, BodyKindJSON},
		{"", `<?xml version="1.0"?><a/>`, BodyKindXML},
		{"text/plain", `hello`, BodyKindText},
	}
	for _, tc := range cases {
		got := DetectBodyKind(tc.ct, []byte(tc.body))
		if got != tc.want {
			t.Fatalf("DetectBodyKind(%q, %q)=%q want %q", tc.ct, tc.body, got, tc.want)
		}
	}
}

func TestPrettyXML(t *testing.T) {
	in := []byte(`<root><child>x</child></root>`)
	out, err := PrettyXML(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n") || !strings.Contains(out, "    <child>") {
		t.Fatalf("expected indented xml, got %q", out)
	}
}

func TestPrettyJSONStillWorks(t *testing.T) {
	out, err := PrettyJSON([]byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n") {
		t.Fatalf("expected pretty json, got %q", out)
	}
}

func TestApplyBodyFormat(t *testing.T) {
	kind, pretty, js, isJSON := ApplyBodyFormat("application/json", []byte(`{"a":1}`))
	if kind != BodyKindJSON || !isJSON || js == "" || pretty == "" {
		t.Fatalf("got kind=%s pretty=%q js=%q isJSON=%v", kind, pretty, js, isJSON)
	}
}
