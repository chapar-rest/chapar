package scripting

import (
	"net/url"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/variables"
)

// RequestDataFromDomain builds what a script sees of req: its fields as
// saved, with collection headers merged in, and a resolved copy with env and
// dynamic variables filled in. It changes neither req nor env.
func RequestDataFromDomain(req *domain.Request, env *domain.Environment, collection *domain.Collection) *RequestData {
	if req == nil {
		return &RequestData{}
	}
	merged := withCollection(req, collection)
	out := requestData(merged)
	out.Resolved = requestData(resolve(merged, env))
	return out
}

func withCollection(req *domain.Request, collection *domain.Collection) *domain.Request {
	r := req.Clone()
	if collection == nil {
		return r
	}
	switch {
	case r.Spec.HTTP != nil && r.Spec.HTTP.Request != nil:
		r.Spec.HTTP.Request.Headers = domain.MergeHeaders(collection.Spec.Headers, r.Spec.HTTP.Request.Headers)
	case r.Spec.GRPC != nil:
		r.Spec.GRPC.Metadata = domain.MergeHeaders(collection.Spec.Headers, r.Spec.GRPC.Metadata)
	case r.Spec.GraphQL != nil:
		r.Spec.GraphQL.Headers = domain.MergeHeaders(collection.Spec.Headers, r.Spec.GraphQL.Headers)
	}
	return r
}

// resolve fills variables into a copy of req the way the senders do.
func resolve(req *domain.Request, env *domain.Environment) *domain.Request {
	r := req.Clone()
	vars := variables.GetVariables()
	var e *domain.Environment
	if env != nil {
		e = env.Clone()
		variables.ApplyToEnv(vars, &e.Spec)
	}
	switch {
	case r.Spec.HTTP != nil:
		variables.ApplyToHTTPRequest(vars, r.Spec.HTTP)
		e.ApplyToHTTPRequest(r.Spec.HTTP)
	case r.Spec.GRPC != nil:
		variables.ApplyToGRPCRequest(vars, r.Spec.GRPC)
		e.ApplyToGRPCRequest(r.Spec.GRPC)
	case r.Spec.GraphQL != nil:
		variables.ApplyToGraphQLRequest(vars, r.Spec.GraphQL)
		e.ApplyToGraphQLRequest(r.Spec.GraphQL)
	}
	return r
}

func requestData(req *domain.Request) *RequestData {
	out := &RequestData{}
	switch {
	case req.Spec.HTTP != nil:
		h := req.Spec.HTTP
		out.URL = h.URL
		out.Method = h.Method
		out.Query = ParseQuery(h.URL)
		if h.Request != nil {
			out.Headers = enabledPairs(h.Request.Headers)
			out.PathParams = map[string]string{}
			for _, p := range h.Request.PathParams {
				if p.Enable {
					out.PathParams[p.Key] = p.Value
				}
			}
			out.Body = h.Request.Body.Data
		}
	case req.Spec.GRPC != nil:
		g := req.Spec.GRPC
		out.URL = g.ServerInfo.Address
		out.Method = g.LasSelectedMethod
		out.Headers = enabledPairs(g.Metadata)
		out.Body = g.Body
	case req.Spec.GraphQL != nil:
		g := req.Spec.GraphQL
		out.URL = g.URL
		out.Method = "POST"
		out.Headers = enabledPairs(g.Headers)
		out.GraphQL = &GraphQLData{Query: g.Query, Variables: g.Variables}
	}
	if out.Headers == nil {
		out.Headers = []Pair{}
	}
	return out
}

func enabledPairs(kvs []domain.KeyValue) []Pair {
	out := make([]Pair, 0, len(kvs))
	for _, kv := range kvs {
		if kv.Enable && !kv.Locked {
			out = append(out, Pair{kv.Key, kv.Value})
		}
	}
	return out
}

func keyValues(pairs []Pair) []domain.KeyValue {
	out := make([]domain.KeyValue, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, domain.KeyValue{Key: p[0], Value: p[1], Enable: true})
	}
	return out
}

// ParseQuery returns the query params of rawURL, unescaped. It works on
// URLs that still hold {{variables}}, which url.Parse may reject.
func ParseQuery(rawURL string) []Pair {
	_, query, _ := splitURL(rawURL)
	if query == "" {
		return nil
	}
	var out []Pair
	for _, part := range strings.Split(query, "&") {
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		out = append(out, Pair{unescape(k), unescape(v)})
	}
	return out
}

// WithQuery returns rawURL with its query replaced by pairs.
func WithQuery(rawURL string, pairs []Pair) string {
	base, _, fragment := splitURL(rawURL)
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, escape(p[0])+"="+escape(p[1]))
	}
	out := base
	if len(parts) > 0 {
		out += "?" + strings.Join(parts, "&")
	}
	if fragment != "" {
		out += "#" + fragment
	}
	return out
}

func splitURL(rawURL string) (base, query, fragment string) {
	base, fragment, _ = strings.Cut(rawURL, "#")
	base, query, _ = strings.Cut(base, "?")
	return base, query, fragment
}

func unescape(s string) string {
	if u, err := url.QueryUnescape(s); err == nil {
		return u
	}
	return s
}

// escape query-escapes s but keeps {{variables}} readable, so the sender can
// still fill them in.
func escape(s string) string {
	e := url.QueryEscape(s)
	e = strings.ReplaceAll(e, "%7B%7B", "{{")
	return strings.ReplaceAll(e, "%7D%7D", "}}")
}

// ApplyChanges writes what a pre-request script changed into req, which
// must be the per-send copy: the saved request is never changed. Headers,
// metadata and path params the script returns replace the enabled ones.
func ApplyChanges(req *domain.Request, ch *RequestChanges) {
	if req == nil || ch.Empty() {
		return
	}
	switch {
	case req.Spec.HTTP != nil:
		applyHTTP(req.Spec.HTTP, ch)
	case req.Spec.GRPC != nil:
		g := req.Spec.GRPC
		if ch.URL != nil {
			g.ServerInfo.Address = *ch.URL
		}
		if ch.Method != nil {
			g.LasSelectedMethod = *ch.Method
		}
		if ch.Headers != nil {
			g.Metadata = keyValues(*ch.Headers)
		}
		if ch.Body != nil {
			g.Body = *ch.Body
		}
	case req.Spec.GraphQL != nil:
		g := req.Spec.GraphQL
		if ch.URL != nil {
			g.URL = *ch.URL
		}
		if ch.Headers != nil {
			g.Headers = keyValues(*ch.Headers)
		}
		if ch.GraphQL != nil {
			if ch.GraphQL.Query != nil {
				g.Query = *ch.GraphQL.Query
			}
			if ch.GraphQL.Variables != nil {
				g.Variables = *ch.GraphQL.Variables
			}
		}
	}
}

func applyHTTP(h *domain.HTTPRequestSpec, ch *RequestChanges) {
	if ch.URL != nil {
		h.URL = *ch.URL
	}
	if ch.Query != nil {
		h.URL = WithQuery(h.URL, *ch.Query)
	}
	if ch.Method != nil {
		h.Method = strings.ToUpper(*ch.Method)
	}
	if h.Request == nil {
		h.Request = &domain.HTTPRequest{}
	}
	r := h.Request
	if ch.Query != nil {
		r.QueryParams = keyValues(*ch.Query)
	}
	if ch.Headers != nil {
		r.Headers = keyValues(*ch.Headers)
	}
	if ch.PathParams != nil {
		r.PathParams = make([]domain.KeyValue, 0, len(*ch.PathParams))
		for k, v := range *ch.PathParams {
			r.PathParams = append(r.PathParams, domain.KeyValue{Key: k, Value: v, Enable: true})
		}
	}
	if ch.Body != nil {
		r.Body.Data = *ch.Body
		switch r.Body.Type {
		case domain.RequestBodyTypeJSON, domain.RequestBodyTypeXML, domain.RequestBodyTypeText:
		default:
			// Form, binary or no body: the script's text replaces it.
			r.Body.Type = domain.RequestBodyTypeText
			if looksLikeJSON(*ch.Body) {
				r.Body.Type = domain.RequestBodyTypeJSON
			}
		}
	}
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}
