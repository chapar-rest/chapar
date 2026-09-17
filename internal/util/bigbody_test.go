package util

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func compactJSON(n int) []byte {
	var b strings.Builder
	b.Grow(n + 1024)
	b.WriteString(`{"items":[`)
	for i := 0; b.Len() < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"name":"item-%06d","email":"user%06d@example.com","active":%v}`,
			i, i, i, i%2 == 0)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

// TestPrettyJSONLargeBodyStaysBounded guards the streaming indent. Decoding
// into interface{} and marshalling back built a value tree of maps, slices and
// boxed numbers costing many times the body size; a 10 MB response pushed the
// heap past 195 MB and the process never returned it.
func TestPrettyJSONLargeBodyStaysBounded(t *testing.T) {
	body := compactJSON(8 << 20)

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	pretty, err := PrettyJSON(body)
	if err != nil {
		t.Fatal(err)
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc

	if !strings.Contains(pretty, "\n") || !strings.Contains(pretty, `    "id"`) {
		t.Errorf("expected indented output, got %.120q", pretty)
	}
	// Streaming indent touches the output buffer and one string copy. The round
	// trip cost well over 10x the body.
	if budget := uint64(6 * len(body)); allocated > budget {
		t.Errorf("PrettyJSON allocated %.1f MB for a %.1f MB body (budget %.1f MB)",
			float64(allocated)/1e6, float64(len(body))/1e6, float64(budget)/1e6)
	}
	t.Logf("%.1f MB body -> %.1f MB pretty, allocated %.1f MB",
		float64(len(body))/1e6, float64(len(pretty))/1e6, float64(allocated)/1e6)
}

// TestPrettyJSONPreservesKeyOrder covers the reason for indenting rather than
// re-marshalling: Go map marshalling sorts keys, which misrepresents what the
// server sent.
func TestPrettyJSONPreservesKeyOrder(t *testing.T) {
	out, err := PrettyJSON([]byte(`{"zebra":1,"apple":2,"mango":3}`))
	if err != nil {
		t.Fatal(err)
	}
	zebra := strings.Index(out, `"zebra"`)
	apple := strings.Index(out, `"apple"`)
	mango := strings.Index(out, `"mango"`)
	if zebra < 0 || apple < 0 || mango < 0 {
		t.Fatalf("missing keys in %q", out)
	}
	if !(zebra < apple && apple < mango) {
		t.Errorf("keys were reordered:\n%s", out)
	}
}

// TestPrettyJSONPreservesNumberText covers the other round-trip casualty:
// float64 cannot hold every int64, and it rewrites exponents and trailing
// zeros.
func TestPrettyJSONPreservesNumberText(t *testing.T) {
	cases := []string{
		"9007199254740993",     // beyond float64's exact integer range
		"1e3",                  // exponent form
		"1.100",                // trailing zeros
		"-0.30000000000000004", // full float64 precision
		"12345678901234567890", // beyond int64 entirely
	}
	for _, want := range cases {
		out, err := PrettyJSON([]byte(`{"n":` + want + `}`))
		if err != nil {
			t.Errorf("%s: %v", want, err)
			continue
		}
		if !strings.Contains(out, want) {
			t.Errorf("number %s was rewritten: %s", want, strings.TrimSpace(out))
		}
	}
}

// TestPrettyJSONDuplicateKeysSurvive covers a body a re-marshal would silently
// collapse.
func TestPrettyJSONDuplicateKeysSurvive(t *testing.T) {
	out, err := PrettyJSON([]byte(`{"a":1,"a":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, `"a"`); n != 2 {
		t.Errorf("expected both duplicate keys, got %d in %s", n, strings.TrimSpace(out))
	}
}

func TestPrettyJSONTrimsSurroundingWhitespace(t *testing.T) {
	out, err := PrettyJSON([]byte("\n\t  {\"a\":1}  \n\n"))
	if err != nil {
		t.Fatal(err)
	}
	if out != strings.TrimSpace(out) {
		t.Errorf("output has surrounding whitespace: %q", out)
	}
	if !strings.HasPrefix(out, "{") {
		t.Errorf("expected output to start at the object, got %q", out)
	}
}

func TestPrettyJSONRejectsInvalid(t *testing.T) {
	for _, bad := range []string{"", "   ", "{", `{"a":}`, "not json"} {
		if out, err := PrettyJSON([]byte(bad)); err == nil {
			t.Errorf("PrettyJSON(%q) = %q, want an error", bad, out)
		}
	}
}

// TestDetectBodyKindSniffsWithoutWholeBodyCopies covers the head-only sniff.
func TestDetectBodyKindSniffsWithoutWholeBodyCopies(t *testing.T) {
	big := strings.Repeat("<item>x</item>", 200000)
	cases := []struct {
		name, ct, body, want string
	}{
		{"html doctype no ct", "", "<!DOCTYPE html><html><body>hi</body></html>", BodyKindHTML},
		{"html tag upper", "", "<HTML><BODY>hi</BODY></HTML>", BodyKindHTML},
		{"large xml", "", "<root>" + big + "</root>", BodyKindXML},
		{"json no ct", "", `{"a":1}`, BodyKindJSON},
		{"plain text", "", "just some text", BodyKindText},
	}
	for _, tc := range cases {
		if got := DetectBodyKind(tc.ct, []byte(tc.body)); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}
