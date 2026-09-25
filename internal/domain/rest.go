package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type HTTPRequestSpec struct {
	Method string `yaml:"method"`
	URL    string `yaml:"url"`

	LastUsedEnvironment LastUsedEnvironment `yaml:"lastUsedEnvironment"`

	Request   *HTTPRequest   `yaml:"request"`
	Responses []HTTPResponse `yaml:"responses"`
}

func (h *HTTPRequestSpec) Clone() *HTTPRequestSpec {
	clone := *h

	if h.Request != nil {
		clone.Request = h.Request.Clone()
	}

	return &clone
}

type HTTPRequest struct {
	Headers []KeyValue `yaml:"headers"`

	PathParams  []KeyValue `yaml:"pathParams"`
	QueryParams []KeyValue `yaml:"queryParams"`

	Body Body `yaml:"body"`

	Auth      Auth       `yaml:"auth"`
	Variables []Variable `yaml:"variables"`

	PreRequest  PreRequest  `yaml:"preRequest"`
	PostRequest PostRequest `yaml:"postRequest"`
}

type Body struct {
	Type string `yaml:"type"`
	// Can be json, xml, or plain text
	Data string `yaml:"data"`

	FormData       FormData   `yaml:"formData,omitempty"`
	URLEncoded     []KeyValue `yaml:"urlEncoded,omitempty"`
	BinaryFilePath string     `yaml:"binaryFilePath,omitempty"`
}

func (b *Body) Clone() *Body {
	clone := *b

	// Deep clone FormData
	if len(b.FormData.Fields) > 0 {
		clone.FormData.Fields = make([]FormField, len(b.FormData.Fields))
		for i, field := range b.FormData.Fields {
			clone.FormData.Fields[i] = field
			// Deep clone Files slice
			if len(field.Files) > 0 {
				clone.FormData.Fields[i].Files = make([]string, len(field.Files))
				copy(clone.FormData.Fields[i].Files, field.Files)
			}
		}
	}

	// Deep clone URLEncoded
	if len(b.URLEncoded) > 0 {
		clone.URLEncoded = make([]KeyValue, len(b.URLEncoded))
		copy(clone.URLEncoded, b.URLEncoded)
	}

	return &clone
}

type FormData struct {
	Fields []FormField `yaml:"fields"`
}

const (
	FormFieldTypeText = "text"
	FormFieldTypeFile = "file"
)

type FormField struct {
	ID     string   `yaml:"id"`
	Type   string   `yaml:"type"`
	Key    string   `yaml:"key"`
	Value  string   `yaml:"value"`
	Files  []string `yaml:"files"`
	Enable bool     `yaml:"enable"`
}

type HTTPResponse struct {
	Headers []KeyValue `yaml:"headers"`
	Body    string     `yaml:"body"`
	Cookies []KeyValue `yaml:"cookies"`
}

func (r *HTTPRequest) Clone() *HTTPRequest {
	clone := *r

	// Deep clone slices to avoid modifying the original
	if len(r.Headers) > 0 {
		clone.Headers = make([]KeyValue, len(r.Headers))
		copy(clone.Headers, r.Headers)
	}

	if len(r.QueryParams) > 0 {
		clone.QueryParams = make([]KeyValue, len(r.QueryParams))
		copy(clone.QueryParams, r.QueryParams)
	}

	if len(r.PathParams) > 0 {
		clone.PathParams = make([]KeyValue, len(r.PathParams))
		copy(clone.PathParams, r.PathParams)
	}

	if len(r.Variables) > 0 {
		clone.Variables = make([]Variable, len(r.Variables))
		copy(clone.Variables, r.Variables)
	}

	// Clone Body
	clone.Body = *r.Body.Clone()

	if r.Auth != (Auth{}) {
		clone.Auth = r.Auth.Clone()
	}

	return &clone
}

func (r *RequestSpec) Clone() *RequestSpec {
	clone := *r
	if r.GRPC != nil {
		clone.GRPC = r.GRPC.Clone()
	}
	if r.HTTP != nil {
		clone.HTTP = r.HTTP.Clone()
	}
	return &clone
}

func NewHTTPRequest(name string) *Request {
	return &Request{
		ApiVersion: ApiVersion,
		Kind:       KindRequest,
		MetaData: RequestMeta{
			ID:   uuid.NewString(),
			Name: name,
			Type: RequestTypeHTTP,
		},
		Spec: RequestSpec{
			HTTP: &HTTPRequestSpec{
				Method: RequestMethodGET,
				URL:    "https://example.com",
				Request: &HTTPRequest{
					Headers: []KeyValue{
						{Key: "Content-Type", Value: "application/json"},
					},
				},
			},
		},
	}
}

func (r *Request) SetDefaultValuesForHTTP() {
	if r.Spec.HTTP.Method == "" {
		r.Spec.HTTP.Method = RequestMethodGET
	}

	if r.Spec.HTTP.URL == "" {
		r.Spec.HTTP.URL = "https://example.com"
	}

	if r.Spec.HTTP.Request.Auth == (Auth{}) {
		r.Spec.HTTP.Request.Auth = Auth{
			Type: "None",
		}
	}

	if r.Spec.HTTP.Request.PostRequest == (PostRequest{}) {
		r.Spec.HTTP.Request.PostRequest = PostRequest{
			Type: "None",
		}
	}

	if r.Spec.HTTP.Request.PreRequest == (PreRequest{}) {
		r.Spec.HTTP.Request.PreRequest = PreRequest{
			Type: "None",
		}
	}
}

func ParseQueryParams(params string) []KeyValue {
	// remove ? from the beginning
	if len(params) > 0 && params[0] == '?' {
		params = params[1:]
	}

	// separate the query params
	pairs := strings.Split(params, "&")

	if len(params) == 0 {
		return nil
	}

	out := make([]KeyValue, 0, len(pairs))
	for _, p := range pairs {
		pair := strings.Split(p, "=")
		if len(pair) != 2 {
			continue
		}
		kv := KeyValue{ID: uuid.NewString(), Key: pair[0], Value: pair[1], Enable: true}
		out = append(out, kv)
	}

	return out
}

func ParsePathParams(params string) []KeyValue {
	// Remove leading slash
	params = strings.TrimPrefix(params, "/")

	out := make([]KeyValue, 0)
	i := 0

	for i < len(params) {
		// Look for an opening brace
		if params[i] == '{' {
			// Skip double-opening braces
			if i+1 < len(params) && params[i+1] == '{' {
				i += 2
				continue
			}

			s := i
			valid := false
			for j := i + 1; j < len(params); j++ {
				if params[j] == '}' {
					// Found a valid closing brace
					valid = true
					key := params[s+1 : j]
					if key != "" {
						kv := KeyValue{
							ID:     uuid.NewString(),
							Key:    key,
							Value:  "",
							Enable: true,
						}
						out = append(out, kv)
					}
					i = j // Move index to the closing brace
					break
				} else if params[j] == '{' {
					// Found another opening brace before closing the current one; invalid nesting
					valid = false
					break
				}
			}
			if !valid {
				// Skip invalid key by moving to the next character after the opening brace
				i = s + 1
			}
		} else {
			// Move to the next character if not a brace
			i++
		}
	}

	return out
}

func EncodeQueryParams(params []KeyValue) string {
	if len(params) == 0 {
		return ""
	}

	out := make([]string, 0, len(params))
	for _, p := range params {
		if !p.Enable {
			continue
		}

		if p.Key == "" || p.Value == "" {
			continue
		}

		out = append(out, p.Key+"="+p.Value)
	}

	return strings.Join(out, "&")
}

type HTTPResponseDetail struct {
	Response        string
	ResponseHeaders []KeyValue
	RequestHeaders  []KeyValue
	Cookies         []KeyValue
	StatusCode      int
	Duration        time.Duration
	Size            int

	Error error
}
