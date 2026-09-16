package util

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"unicode"
)

// Body kind constants used by egress and the response UI.
const (
	BodyKindJSON = "json"
	BodyKindXML  = "xml"
	BodyKindHTML = "html"
	BodyKindText = "text"
)

// DetectBodyKind chooses a display format from Content-Type, then body sniffing.
func DetectBodyKind(contentType string, body []byte) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch {
	case strings.Contains(ct, "json") || ct == "application/graphql-response+json":
		return BodyKindJSON
	case strings.Contains(ct, "html"):
		return BodyKindHTML
	case strings.Contains(ct, "xml") || strings.Contains(ct, "svg"):
		return BodyKindXML
	}

	trim := bytes.TrimLeftFunc(body, func(r rune) bool { return unicode.IsSpace(r) })
	if len(trim) == 0 {
		return BodyKindText
	}
	if IsJSON(string(body)) {
		return BodyKindJSON
	}
	s := string(trim)
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html") {
		return BodyKindHTML
	}
	if trim[0] == '<' && IsXML(s) {
		if strings.Contains(lower, "<html") {
			return BodyKindHTML
		}
		return BodyKindXML
	}
	return BodyKindText
}

// PrettyBody returns indented text for known structured kinds; otherwise the raw body.
func PrettyBody(body []byte, kind string) (string, error) {
	switch kind {
	case BodyKindJSON:
		return PrettyJSON(body)
	case BodyKindXML:
		return PrettyXML(body)
	case BodyKindHTML:
		return PrettyHTML(body)
	default:
		return string(body), nil
	}
}

// IsXML reports whether s looks like well-formed XML (or HTML-as-XML).
func IsXML(s string) bool {
	dec := xml.NewDecoder(strings.NewReader(s))
	dec.Strict = false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return true
		}
		if err != nil || tok == nil {
			return false
		}
	}
}

// PrettyXML indents XML. On failure it returns the original string with a non-nil error.
func PrettyXML(data []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var out bytes.Buffer
	enc := xml.NewEncoder(&out)
	enc.Indent("", "    ")
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return string(data), err
		}
		if err := enc.EncodeToken(tok); err != nil {
			return string(data), err
		}
	}
	if err := enc.Flush(); err != nil {
		return string(data), err
	}
	return out.String(), nil
}

// PrettyHTML applies a lightweight indent to HTML-ish markup. Failures return raw text.
func PrettyHTML(data []byte) (string, error) {
	// Prefer XML-style pretty when the document parses; otherwise a simple tag indent.
	if pretty, err := PrettyXML(data); err == nil {
		return pretty, nil
	}
	return indentTags(data), nil
}

func indentTags(data []byte) string {
	s := string(data)
	var b strings.Builder
	b.Grow(len(s) + len(s)/8)
	depth := 0
	i := 0
	for i < len(s) {
		if s[i] == '<' {
			j := strings.IndexByte(s[i:], '>')
			if j < 0 {
				b.WriteString(s[i:])
				break
			}
			tag := s[i : i+j+1]
			closing := strings.HasPrefix(tag, "</")
			selfClose := strings.HasSuffix(tag, "/>") || isVoidHTMLTag(tag)
			if closing {
				depth--
				if depth < 0 {
					depth = 0
				}
			}
			b.WriteString(strings.Repeat("    ", depth))
			b.WriteString(tag)
			b.WriteByte('\n')
			if !closing && !selfClose && !strings.HasPrefix(tag, "<!") && !strings.HasPrefix(tag, "<?") {
				depth++
			}
			i += j + 1
			continue
		}
		// text node until next tag
		j := strings.IndexByte(s[i:], '<')
		if j < 0 {
			text := strings.TrimSpace(s[i:])
			if text != "" {
				b.WriteString(strings.Repeat("    ", depth))
				b.WriteString(text)
				b.WriteByte('\n')
			}
			break
		}
		text := strings.TrimSpace(s[i : i+j])
		if text != "" {
			b.WriteString(strings.Repeat("    ", depth))
			b.WriteString(text)
			b.WriteByte('\n')
		}
		i += j
	}
	return strings.TrimRight(b.String(), "\n")
}

func isVoidHTMLTag(tag string) bool {
	name := tag[1:]
	if i := strings.IndexAny(name, " \t\n/>"); i >= 0 {
		name = name[:i]
	}
	name = strings.ToLower(name)
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

// ApplyBodyFormat sets Pretty/BodyKind/IsJSON/JSON on a response-like target.
func ApplyBodyFormat(contentType string, body []byte) (kind, pretty, jsonStr string, isJSON bool) {
	kind = DetectBodyKind(contentType, body)
	pretty, err := PrettyBody(body, kind)
	if err != nil || pretty == "" {
		pretty = string(body)
	}
	if kind == BodyKindJSON {
		isJSON = true
		jsonStr = pretty
	} else if IsJSON(string(body)) {
		// Sniff says structured other kind but body is also JSON — keep jsonpath path.
		isJSON = true
		if js, err := PrettyJSON(body); err == nil {
			jsonStr = js
			kind = BodyKindJSON
			pretty = js
		}
	}
	return kind, pretty, jsonStr, isJSON
}
