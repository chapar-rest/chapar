package domain

import (
	"net/url"
	"strings"
)

const DefaultRequestName = "New Request"

// RequestUsesAutoName reports whether metadata.name is unset (empty or the default).
func RequestUsesAutoName(name string) bool {
	name = strings.TrimSpace(name)
	return name == "" || name == DefaultRequestName
}

// RequestAutoName returns the generated display label from URL or gRPC method.
func RequestAutoName(r *Request) string {
	if r == nil {
		return DefaultRequestName
	}
	switch r.MetaData.Type {
	case RequestTypeGRPC:
		if r.Spec.GRPC != nil {
			if m := strings.TrimSpace(r.Spec.GRPC.LasSelectedMethod); m != "" {
				return m
			}
		}
		return DefaultRequestName
	case RequestTypeGraphQL:
		if r.Spec.GraphQL != nil {
			if s := urlDisplayFromString(r.Spec.GraphQL.URL); s != "" {
				return s
			}
		}
		return DefaultRequestName
	default:
		if r.Spec.HTTP != nil {
			if s := urlDisplayFromString(r.Spec.HTTP.URL); s != "" {
				return s
			}
		}
		return DefaultRequestName
	}
}

// RequestDisplayName returns the user-facing label: custom name or auto-generated.
func RequestDisplayName(r *Request) string {
	if r == nil {
		return DefaultRequestName
	}
	name := strings.TrimSpace(r.MetaData.Name)
	if RequestUsesAutoName(name) {
		return RequestAutoName(r)
	}
	return name
}

// RequestInfoNameValue is the controlled value for the Info tab name field.
func RequestInfoNameValue(r *Request) string {
	if r == nil || RequestUsesAutoName(r.MetaData.Name) {
		return ""
	}
	return r.MetaData.Name
}

// SetRequestInfoName updates metadata.name from the Info tab field.
func SetRequestInfoName(r *Request, value string) {
	if r == nil {
		return
	}
	value = strings.TrimSpace(value)
	if value == "" {
		r.MetaData.Name = DefaultRequestName
		return
	}
	r.MetaData.Name = value
}

func urlDisplayFromString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return hostPath(u)
	}
	s := raw
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimRight(s, "/")
	if s == "" {
		return ""
	}
	return s
}

func hostPath(u *url.URL) string {
	path := u.EscapedPath()
	if path == "" || path == "/" {
		path = ""
	} else {
		path = strings.TrimSuffix(path, "/")
	}
	host := u.Host
	if path != "" {
		return host + path
	}
	return host
}
