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

	Auth Auth `yaml:"auth"`

	PreRequest  PreRequest  `yaml:"preRequest"`
	PostRequest PostRequest `yaml:"postRequest"`
}

const (
	BodyTypeNone       = "none"
	BodyTypeJSON       = "json"
	BodyTypeText       = "text"
	BodyTypeXML        = "xml"
	BodyTypeFormData   = "formData"
	BodyTypeBinary     = "binary"
	BodyTypeUrlencoded = "urlencoded"
)

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

func CompareHTTPRequestSpecs(a, b *HTTPRequestSpec) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Method != b.Method || a.URL != b.URL {
		return false
	}

	if !CompareHTTPRequests(a.Request, b.Request) {
		return false
	}

	if len(a.Responses) != len(b.Responses) {
		return false
	}

	for i, v := range a.Responses {
		if !CompareHTTPResponses(v, b.Responses[i]) {
			return false
		}
	}

	return true
}

func CompareHTTPRequests(a, b *HTTPRequest) bool {
	if a == nil && b == nil {
		return true
	}

	if b == nil || a == nil {
		return false
	}

	if !CompareBody(a.Body, b.Body) {
		return false
	}

	if !CompareKeyValues(a.Headers, b.Headers) {
		return false
	}

	if !CompareKeyValues(a.PathParams, b.PathParams) {
		return false
	}

	if !CompareKeyValues(a.QueryParams, b.QueryParams) {
		return false
	}

	if !CompareFormData(a.Body.FormData, b.Body.FormData) {
		return false
	}

	if !CompareKeyValues(a.Body.URLEncoded, b.Body.URLEncoded) {
		return false
	}

	if !CompareAuth(a.Auth, b.Auth) {
		return false
	}

	if !ComparePreRequest(a.PreRequest, b.PreRequest) {
		return false
	}

	if !ComparePostRequest(a.PostRequest, b.PostRequest) {
		return false
	}

	return true
}

func CompareFormData(a, b FormData) bool {
	if len(a.Fields) != len(b.Fields) {
		return false
	}

	for i, v := range a.Fields {
		if !CompareFormField(v, b.Fields[i]) {
			return false
		}
	}

	return true
}

func CompareFormField(a, b FormField) bool {
	if a.Type != b.Type || a.Key != b.Key || a.Value != b.Value || a.Enable != b.Enable {
		return false
	}

	if len(a.Files) != len(b.Files) {
		return false
	}

	for i, v := range a.Files {
		if v != b.Files[i] {
			return false
		}
	}

	return true
}

func CompareBody(a, b Body) bool {
	if a.Type != b.Type || a.Data != b.Data || a.BinaryFilePath != b.BinaryFilePath {
		return false
	}

	return true
}

func CompareHTTPResponses(a, b HTTPResponse) bool {
	if IsHTTPResponseEmpty(a) && IsHTTPResponseEmpty(b) {
		return true
	}

	if a.Body != b.Body {
		return false
	}

	if !CompareKeyValues(a.Headers, b.Headers) {
		return false
	}

	if !CompareKeyValues(a.Cookies, b.Cookies) {
		return false
	}

	return true
}

func IsHTTPResponseEmpty(r HTTPResponse) bool {
	if r.Body != "" || len(r.Headers) > 0 || len(r.Cookies) > 0 {
		return false
	}

	return true
}

func (r *Request) SetDefaultValuesForHTTP() {
	if r.Spec.HTTP.Method == "" {
		r.Spec.HTTP.Method = "GET"
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
	// remove / from the beginning
	if len(params) > 0 && params[0] == '/' {
		params = params[1:]
	}

	// separate the path params
	pairs := strings.Split(params, "/")
	if len(params) == 0 {
		return nil
	}

	// find path params, which are surrounded by single curly braces and not in query params
	out := make([]KeyValue, 0)
	for _, p := range pairs {
		if strings.Count(p, "{") == 1 && strings.Count(p, "}") == 1 && !strings.Contains(p, "=") {
			key := strings.Trim(p, "{}")
			kv := KeyValue{ID: uuid.NewString(), Key: key, Value: "", Enable: true}
			out = append(out, kv)
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
	Response   string
	Headers    []KeyValue
	Cookies    []KeyValue
	StatusCode int
	Duration   time.Duration
	Size       int

	Error error
}
