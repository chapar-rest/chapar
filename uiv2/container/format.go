package container

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
)

// EditorIndent is one level of indentation as the editor settings ask for it.
func EditorIndent() string {
	ed := prefs.GetGlobalConfig().Spec.Editor
	if ed.Indentation == domain.IndentationTabs {
		return "\t"
	}
	w := ed.TabWidth
	if w <= 0 {
		w = 4
	}
	return strings.Repeat(" ", w)
}

// FormatJSONBody indents a JSON request body, keeping its {{variable}}
// placeholders.
//
// A body may hold a placeholder where a value goes ({"id": {{id}}}), which is
// not JSON until the variable is substituted at send time. Each bare
// placeholder is swapped for a string sentinel before indenting and swapped
// back after, so it survives formatting exactly as written. A general-purpose
// formatter would drop it, silently losing part of the body.
//
// indent is one level of indentation. A body that is not JSON, placeholders
// aside, is returned with the error and should be left as it is.
func FormatJSONBody(body, indent string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return body, nil
	}
	masked, vars := maskPlaceholders(trimmed)
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(masked), "", indent); err != nil {
		return body, fmt.Errorf("body is not valid JSON: %w", err)
	}
	formatted := out.String()
	for i, v := range vars {
		formatted = strings.Replace(formatted, placeholderSentinel(i), v, 1)
	}
	return formatted, nil
}

// maskPlaceholders replaces each {{…}} outside a JSON string with a quoted
// sentinel, returning the masked text and the placeholders in order.
// Placeholders inside strings are already valid JSON and are left alone.
func maskPlaceholders(s string) (string, []string) {
	var (
		b        strings.Builder
		vars     []string
		inString bool
		escaped  bool
	)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			b.WriteByte(ch)
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == '"':
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			b.WriteByte(ch)
			continue
		}
		if ch == '{' && strings.HasPrefix(s[i:], "{{") {
			if end := strings.Index(s[i+2:], "}}"); end >= 0 {
				n := i + 2 + end + 2
				b.WriteString(placeholderSentinel(len(vars)))
				vars = append(vars, s[i:n])
				i = n - 1
				continue
			}
		}
		b.WriteByte(ch)
	}
	return b.String(), vars
}

func placeholderSentinel(i int) string {
	return fmt.Sprintf(`"\u0000chapar-var-%d\u0000"`, i)
}
