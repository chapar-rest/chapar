package container

import "testing"

func TestFormatJSONBodyKeepsPlaceholders(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"a":1,"b":[true,null]}`, "{\n  \"a\": 1,\n  \"b\": [\n    true,\n    null\n  ]\n}"},
		{`{"id":{{id}},"name":"{{name}}"}`, "{\n  \"id\": {{id}},\n  \"name\": \"{{name}}\"\n}"},
		{`[{{a}}, {{ b }}]`, "[\n  {{a}},\n  {{ b }}\n]"},
		{`{"s":"quote \" {{x}}","n":{{n}}}`, "{\n  \"s\": \"quote \\\" {{x}}\",\n  \"n\": {{n}}\n}"},
		{"  \n", "  \n"},
	}
	for _, tc := range cases {
		got, err := FormatJSONBody(tc.in, "  ")
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("%q:\n got %q\nwant %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatJSONBodyLeavesInvalidBodyAlone(t *testing.T) {
	in := `{"a": 1,`
	got, err := FormatJSONBody(in, "  ")
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
	if got != in {
		t.Fatalf("invalid body changed to %q", got)
	}
}
