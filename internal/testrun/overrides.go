package testrun

import (
	"net/url"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
)

// applyOverrides applies a step's with block to req, a copy of the saved
// request. Variables are not applied here; they go into the step's env.
func applyOverrides(req *domain.Request, with *domain.TestStepOverrides) {
	if with == nil {
		return
	}
	switch req.MetaData.Type {
	case domain.RequestTypeHTTP:
		spec := req.Spec.HTTP
		if spec == nil {
			return
		}
		if spec.Request == nil {
			spec.Request = &domain.HTTPRequest{}
		}
		for _, h := range with.Headers {
			setHeader(&spec.Request.Headers, h.Key, h.Value)
		}
		for _, q := range with.Query {
			spec.URL = setQuery(spec.URL, q.Key, q.Value)
			setParam(&spec.Request.QueryParams, q.Key, q.Value)
		}
		if with.Body != "" {
			spec.Request.Body.Data = with.Body
		}
	case domain.RequestTypeGraphQL:
		spec := req.Spec.GraphQL
		if spec == nil {
			return
		}
		for _, h := range with.Headers {
			setHeader(&spec.Headers, h.Key, h.Value)
		}
		for _, q := range with.Query {
			spec.URL = setQuery(spec.URL, q.Key, q.Value)
		}
		if with.Body != "" {
			spec.Query = with.Body
		}
	case domain.RequestTypeGRPC:
		spec := req.Spec.GRPC
		if spec == nil {
			return
		}
		for _, h := range with.Headers {
			setHeader(&spec.Metadata, h.Key, h.Value)
		}
		if with.Body != "" {
			spec.Body = with.Body
		}
	}
}

// setHeader sets a header, matching its name in any case.
func setHeader(kvs *[]domain.KeyValue, key, value string) {
	for i, kv := range *kvs {
		if strings.EqualFold(kv.Key, key) {
			(*kvs)[i].Value = value
			(*kvs)[i].Enable = true
			return
		}
	}
	*kvs = append(*kvs, domain.KeyValue{Key: key, Value: value, Enable: true})
}

func setParam(kvs *[]domain.KeyValue, key, value string) {
	for i, kv := range *kvs {
		if kv.Key == key {
			(*kvs)[i].Value = value
			(*kvs)[i].Enable = true
			return
		}
	}
	*kvs = append(*kvs, domain.KeyValue{Key: key, Value: value, Enable: true})
}

// setQuery sets a query parameter in rawURL, replacing the first one with
// that name and dropping any others. It edits the text rather than
// parsing the URL, which may hold {{placeholders}}.
func setQuery(rawURL, key, value string) string {
	base, fragment, hasFragment := strings.Cut(rawURL, "#")
	path, query, _ := strings.Cut(base, "?")

	pair := escapeQuery(key) + "=" + escapeQuery(value)
	var parts []string
	replaced := false
	if query != "" {
		for _, p := range strings.Split(query, "&") {
			name, _, _ := strings.Cut(p, "=")
			if unescaped, err := url.QueryUnescape(name); err == nil {
				name = unescaped
			}
			if name != key {
				parts = append(parts, p)
				continue
			}
			if !replaced {
				parts = append(parts, pair)
				replaced = true
			}
		}
	}
	if !replaced {
		parts = append(parts, pair)
	}

	out := path + "?" + strings.Join(parts, "&")
	if hasFragment {
		out += "#" + fragment
	}
	return out
}

// escapeQuery escapes s for a query string, leaving {{placeholders}} as
// they are so the send can still fill them in.
func escapeQuery(s string) string {
	var b strings.Builder
	last := 0
	for _, m := range placeholder.FindAllStringIndex(s, -1) {
		b.WriteString(url.QueryEscape(s[last:m[0]]))
		b.WriteString(s[m[0]:m[1]])
		last = m[1]
	}
	b.WriteString(url.QueryEscape(s[last:]))
	return b.String()
}
