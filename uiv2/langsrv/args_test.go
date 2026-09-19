package langsrv

import (
	"slices"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	cases := map[string][]string{
		"--stdio":                     {"--stdio"},
		"  serve   -rpc.trace ":       {"serve", "-rpc.trace"},
		`-c "echo hi >&2; exit 2"`:    {"-c", "echo hi >&2; exit 2"},
		`-data '/tmp/my ws' --x=a\ b`: {"-data", "/tmp/my ws", "--x=a b"},
		`'' ""`:                       {"", ""},
		`'it'\''s'`:                   {"it's"},
	}
	for in, want := range cases {
		if got := SplitArgs(in); !slices.Equal(got, want) {
			t.Errorf("SplitArgs(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinArgsRoundTrips(t *testing.T) {
	for _, args := range [][]string{
		{"--stdio"},
		{"-c", "echo 'pyright: fatal' >&2; exit 2"},
		{"-data", "/tmp/my ws", ""},
		{`back\slash`, `"quoted"`},
	} {
		if got := SplitArgs(JoinArgs(args)); !slices.Equal(got, args) {
			t.Errorf("SplitArgs(JoinArgs(%q)) = %q (joined %q)", args, got, JoinArgs(args))
		}
	}
}
