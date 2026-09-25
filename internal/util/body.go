package util

import (
	"bytes"
	"encoding/xml"
	"errors"
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
	if IsJSONBytes(body) {
		return BodyKindJSON
	}
	// Sniffing only needs the opening tag, so prefix-match on the head of the
	// body. Lower-casing the whole thing copied a multi-megabyte body twice
	// more for a decision the first few bytes already make.
	head := trim
	if len(head) > 1024 {
		head = head[:1024]
	}
	lowerHead := strings.ToLower(string(head))
	if strings.HasPrefix(lowerHead, "<!doctype html") || strings.HasPrefix(lowerHead, "<html") {
		return BodyKindHTML
	}
	if trim[0] == '<' && IsXMLBytes(trim) {
		if bytes.Contains(bytes.ToLower(head), []byte("<html")) {
			return BodyKindHTML
		}
		return BodyKindXML
	}
	return BodyKindText
}

// MaxPrettyBytes is the hard ceiling on formatting a body.
//
// A formatted copy is a second full representation of the body, and producing
// it needs a working buffer on top of that: formatting an 86 MB JSON response
// cost ~200 MB in copies and buffers. Past this size the raw body is displayed
// as received, which also lets the viewer read the response's own buffer
// instead of copying it.
const MaxPrettyBytes = 32 << 20

// formattedSampleBytes is how much of the body alreadyFormatted inspects, and
// formattedLineLimit is the average line length at or below which the body is
// taken to be formatted already.
const (
	formattedSampleBytes = 64 << 10
	formattedLineLimit   = 200
)

// ErrBodyNotFormatted reports that a body was deliberately left unformatted —
// it is already laid out in lines, or too large to reformat. Callers should
// display the raw body.
var ErrBodyNotFormatted = errors.New("body left unformatted")

// alreadyFormatted reports whether body is already broken into short lines, in
// which case reformatting it costs a second full copy and changes nothing a
// reader would notice. Large API payloads are commonly served pretty-printed,
// and that is exactly the case where the copy hurts most.
//
// Only the head of the body is sampled, so the check is O(1) in body size.
func alreadyFormatted(body []byte) bool {
	sample := body
	if len(sample) > formattedSampleBytes {
		sample = sample[:formattedSampleBytes]
	}
	newlines := bytes.Count(sample, []byte{'\n'})
	if newlines == 0 {
		return false
	}
	return len(sample)/newlines <= formattedLineLimit
}

// PrettyBody returns indented text for known structured kinds; otherwise the raw body.
//
// A body that is already laid out in lines, or larger than MaxPrettyBytes,
// returns ErrBodyNotFormatted with no text.
func PrettyBody(body []byte, kind string) (string, error) {
	if len(body) > MaxPrettyBytes || alreadyFormatted(body) {
		return "", ErrBodyNotFormatted
	}
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

// IsXMLBytes reports whether data looks like well-formed XML (or HTML-as-XML)
// without copying it into a string.
func IsXMLBytes(data []byte) bool {
	return isXMLReader(bytes.NewReader(data))
}

// IsXML reports whether s looks like well-formed XML (or HTML-as-XML).
func IsXML(s string) bool {
	return isXMLReader(strings.NewReader(s))
}

func isXMLReader(r io.Reader) bool {
	dec := xml.NewDecoder(r)
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
//
// When the body is left unformatted (already line-broken, or over
// MaxPrettyBytes) pretty is returned empty so the caller displays the raw body
// without copying it. jsonStr is still filled for bodies within
// MaxPrettyBytes, since jsonpath extraction needs a string; past that it is
// left empty rather than holding another copy of a very large body.
func ApplyBodyFormat(contentType string, body []byte) (kind, pretty, jsonStr string, isJSON bool) {
	kind = DetectBodyKind(contentType, body)
	if len(body) > MaxPrettyBytes {
		return kind, "", "", kind == BodyKindJSON
	}
	pretty, err := PrettyBody(body, kind)
	if errors.Is(err, ErrBodyNotFormatted) {
		// Show the body as received. jsonStr still needs a string form for
		// jsonpath; for JSON that is the body itself.
		isJSON = kind == BodyKindJSON || IsJSONBytes(body)
		if isJSON {
			jsonStr = string(body)
			kind = BodyKindJSON
		}
		return kind, "", jsonStr, isJSON
	}
	if err != nil || pretty == "" {
		pretty = string(body)
	}
	if kind == BodyKindJSON {
		isJSON = true
		jsonStr = pretty
	} else if IsJSONBytes(body) {
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
