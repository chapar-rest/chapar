package editor

import (
	"strings"
	"unicode/utf8"

	"github.com/yogaui/yoga/render"
)

func f32max(a, b float32) float32 { return max(a, b) }

func clampf(v, lo, hi float32) float32 { return max(lo, min(v, hi)) }

func alpha(c render.Color, a float32) render.Color {
	c.A = a
	return c
}

func normalizeModToken(p string) string { return strings.ToLower(strings.TrimSpace(p)) }

// splitChord splits a chord into its modifier and key tokens, as ui.ParseChord
// does: "Mod+Shift+L", or macOS symbols without separators ("⌘⇧L").
func splitChord(s string) []string {
	// Unicode symbols without separators: ⌘⇧⌥⌃ followed by a key token.
	if strings.ContainsAny(s, "⌘⇧⌥⌃") && !strings.Contains(s, "+") {
		var parts []string
		rest := s
		for len(rest) > 0 {
			r, size := utf8.DecodeRuneInString(rest)
			switch r {
			case '⌘':
				parts = append(parts, "⌘")
				rest = rest[size:]
			case '⇧':
				parts = append(parts, "Shift")
				rest = rest[size:]
			case '⌥':
				parts = append(parts, "Alt")
				rest = rest[size:]
			case '⌃':
				parts = append(parts, "Ctrl")
				rest = rest[size:]
			default:
				parts = append(parts, rest)
				return parts
			}
		}
		return parts
	}
	raw := strings.Split(s, "+")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// seqParts splits a shortcut on the spaces between chords, keeping spaces
// around "+" inside a chord ("Ctrl + K").
func seqParts(s string) []string {
	var out []string
	for _, f := range strings.Fields(s) {
		if n := len(out); n > 0 && (f == "+" || strings.HasPrefix(f, "+") || strings.HasSuffix(out[n-1], "+")) {
			out[n-1] += f
			continue
		}
		out = append(out, f)
	}
	return out
}
