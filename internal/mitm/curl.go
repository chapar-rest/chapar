package mitm

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"
)

// curlSkipHeaders are set by curl itself or only meant for the proxy hop.
var curlSkipHeaders = map[string]bool{
	"Content-Length":    true,
	"Connection":        true,
	"Proxy-Connection":  true,
	"Keep-Alive":        true,
	"Transfer-Encoding": true,
	"Te":                true,
	"Upgrade":           true,
}

// Curl returns a shell command that sends the flow's request again. Every
// value is single-quoted, so it pastes safely into sh, bash and zsh. A
// binary body is piped in through base64.
func Curl(f *Flow) string {
	head := "curl"
	hasBody := len(f.ReqBody) > 0
	switch {
	case f.Method == http.MethodHead:
		head += " --head"
	case f.Method == http.MethodGet && !hasBody:
	case f.Method == http.MethodPost && hasBody:
	default:
		head += " -X " + shellQuote(f.Method)
	}
	head += " " + shellQuote(f.URL)

	// Each option goes on its own line, which keeps long commands readable.
	var opts []string
	host := ""
	if u, err := url.Parse(f.URL); err == nil {
		host = u.Host
	}
	keys := make([]string, 0, len(f.ReqHeader))
	for k := range f.ReqHeader {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	compressed := false
	for _, k := range keys {
		ck := http.CanonicalHeaderKey(k)
		if curlSkipHeaders[ck] {
			continue
		}
		for _, v := range f.ReqHeader[k] {
			switch {
			case ck == "Host" && strings.EqualFold(v, host):
				continue
			case ck == "Accept-Encoding":
				// --compressed asks for the same and decodes the reply,
				// which a bare header would leave as gzip on the terminal.
				compressed = true
				continue
			}
			opts = append(opts, "-H "+shellQuote(k+": "+v))
		}
	}
	if compressed {
		opts = append(opts, "--compressed")
	}

	prefix := ""
	if hasBody {
		if utf8.Valid(f.ReqBody) {
			opts = append(opts, "--data-raw "+shellQuote(string(f.ReqBody)))
		} else {
			prefix = "printf %s " + shellQuote(base64.StdEncoding.EncodeToString(f.ReqBody)) + " | base64 -d | "
			opts = append(opts, "--data-binary @-")
		}
	}
	return prefix + strings.Join(append([]string{head}, opts...), " \\\n  ")
}

// shellQuote wraps s in single quotes, closing and reopening them around
// each single quote inside.
func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./:=@%+,", r))
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
