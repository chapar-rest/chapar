package domain

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

const (
	RequestTypeHTTP = "http"
	RequestTypeGRPC = "grpc"

	RequestMethodGET     = "GET"
	RequestMethodPOST    = "POST"
	RequestMethodPUT     = "PUT"
	RequestMethodDELETE  = "DELETE"
	RequestMethodPATCH   = "PATCH"
	RequestMethodHEAD    = "HEAD"
	RequestMethodOPTIONS = "OPTIONS"
	RequestMethodCONNECT = "CONNECT"
	RequestMethodTRACE   = "TRACE"

	RequestBodyTypeNone       = "none"
	RequestBodyTypeJSON       = "json"
	RequestBodyTypeXML        = "xml"
	RequestBodyTypeText       = "text"
	RequestBodyTypeForm       = "form"
	RequestBodyTypeBinary     = "binary"
	RequestBodyTypeUrlEncoded = "urlEncoded"

	PrePostTypeNone      = "none"
	PrePostTypePython    = "python"
	PrePostTypeShell     = "ssh"
	PrePostTypeSSHTunnel = "sshTunnel"
	PrePostTypeK8sTunnel = "k8sTunnel"
)

type Request struct {
	ApiVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	MetaData   RequestMeta `yaml:"metadata"`
	Spec       RequestSpec `yaml:"spec"`
	FilePath   string      `yaml:"-"`

	CollectionName string `yaml:"-"`
	CollectionID   string `yaml:"-"`
}

type RequestMeta struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type RequestSpec struct {
	GRPC *GRPCRequestSpec `yaml:"grpc,omitempty"`
	HTTP *HTTPRequestSpec `yaml:"http,omitempty"`
}

type GRPCRequestSpec struct {
	Host   string `yaml:"host"`
	Method string `yaml:"method"`
}

func (g *GRPCRequestSpec) Clone() *GRPCRequestSpec {
	clone := *g
	return &clone
}

type HTTPRequestSpec struct {
	Method string `yaml:"method"`
	URL    string `yaml:"url"`

	LastUsedEnvironment LastUsedEnvironment `yaml:"lastUsedEnvironment"`

	Request   *HTTPRequest   `yaml:"request"`
	Responses []HTTPResponse `yaml:"responses"`
}

type LastUsedEnvironment struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
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

type Body struct {
	Type string `yaml:"type"`
	// Can be json, xml, or plain text
	Data string `yaml:"data"`

	FormBody   []KeyValue `yaml:"formBody,omitempty"`
	URLEncoded []KeyValue `yaml:"urlEncoded,omitempty"`
}

func (b *Body) Clone() *Body {
	clone := *b
	return &clone
}

type Auth struct {
	Type      string     `yaml:"type"`
	BasicAuth *BasicAuth `yaml:"basicAuth,omitempty"`
	TokenAuth *TokenAuth `yaml:"tokenAuth,omitempty"`
}

func (a *Auth) Clone() *Auth {
	clone := *a
	if a.BasicAuth != nil {
		clone.BasicAuth = a.BasicAuth.Clone()
	}
	if a.TokenAuth != nil {
		clone.TokenAuth = a.TokenAuth.Clone()
	}
	return &clone
}

func (a *BasicAuth) Clone() *BasicAuth {
	clone := *a
	return &clone
}

func (a *TokenAuth) Clone() *TokenAuth {
	clone := *a
	return &clone
}

type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type TokenAuth struct {
	Token string `yaml:"token"`
}

type HTTPResponse struct {
	Headers []KeyValue `yaml:"headers"`
	Body    string     `yaml:"body"`
	Cookies []KeyValue `yaml:"cookies"`
}

type PreRequest struct {
	Type   string `yaml:"type"`
	Script string `yaml:"script"`

	SShTunnel        *SShTunnel        `yaml:"sshTunnel,omitempty"`
	KubernetesTunnel *KubernetesTunnel `yaml:"kubernetesTunnel,omitempty"`
}

type PostRequest struct {
	Type   string `yaml:"type"`
	Script string `yaml:"script"`
}

type KubernetesTunnel struct {
	Target     string `yaml:"target"`
	TargetType string `yaml:"targetType"`

	// The port to be used in the local machine
	LocalPort  int `yaml:"localPort"`
	TargetPort int `yaml:"targetPort"`
}

type SShTunnel struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	KeyPath  string `yaml:"keyPath"`

	// The port to be used in the local machine
	LocalPort  int `yaml:"localPort"`
	TargetPort int `yaml:"targetPort"`

	Flags []string `yaml:"flags"`
}

func (r *Request) Clone() *Request {
	clone := *r
	clone.MetaData.ID = uuid.NewString()
	clone.Spec = *r.Spec.Clone()
	return &clone
}

func (r *HTTPRequest) Clone() *HTTPRequest {
	clone := *r
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

func NewRequest(name string) *Request {
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

func CompareRequests(a, b *Request) bool {
	if b == nil || a == nil {
		return false
	}

	if a.MetaData.ID != b.MetaData.ID || a.MetaData.Name != b.MetaData.Name || a.MetaData.Type != b.MetaData.Type {
		return false
	}

	if !CompareGRPCRequestSpecs(a.Spec.GRPC, b.Spec.GRPC) {
		return false
	}

	if !CompareHTTPRequestSpecs(a.Spec.HTTP, b.Spec.HTTP) {
		return false
	}

	return true
}

func CompareGRPCRequestSpecs(a, b *GRPCRequestSpec) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Host != b.Host || a.Method != b.Method {
		return false
	}
	return true
}

func CompareHTTPRequestSpecs(a, b *HTTPRequestSpec) bool {
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

	if !CompareKeyValues(a.Body.FormBody, b.Body.FormBody) {
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

func CompareBody(a, b Body) bool {
	if a.Type != b.Type || a.Data != b.Data {
		return false
	}

	return true
}

func CompareAuth(a, b Auth) bool {

	if a.Type != b.Type {
		return false
	}

	if !CompareBasicAuth(a.BasicAuth, b.BasicAuth) {
		return false
	}

	if !CompareTokenAuth(a.TokenAuth, b.TokenAuth) {
		return false
	}

	return true
}

func CompareBasicAuth(a, b *BasicAuth) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Username != b.Username || a.Password != b.Password {
		return false
	}

	return true
}

func CompareTokenAuth(a, b *TokenAuth) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Token != b.Token {
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

func ComparePreRequest(a, b PreRequest) bool {
	if a.Type != b.Type || a.Script != b.Script {
		return false
	}

	if !CompareSShTunnel(a.SShTunnel, b.SShTunnel) {
		return false
	}

	if !CompareKubernetesTunnel(a.KubernetesTunnel, b.KubernetesTunnel) {
		return false
	}

	return true
}

func CompareSShTunnel(a, b *SShTunnel) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Host != b.Host || a.Port != b.Port || a.User != b.User || a.Password != b.Password || a.KeyPath != b.KeyPath || a.LocalPort != b.LocalPort || a.TargetPort != b.TargetPort {
		return false
	}

	if len(a.Flags) != len(b.Flags) {
		return false
	}

	for i, v := range a.Flags {
		if v != b.Flags[i] {
			return false
		}
	}

	return true
}

func CompareKubernetesTunnel(a, b *KubernetesTunnel) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Target != b.Target || a.TargetType != b.TargetType || a.LocalPort != b.LocalPort || a.TargetPort != b.TargetPort {
		return false
	}

	return true
}

func ComparePostRequest(a, b PostRequest) bool {
	if a.Type != b.Type || a.Script != b.Script {
		return false
	}

	return true
}

func Clone[T any](org *T) (*T, error) {
	origJSON, err := json.Marshal(org)
	if err != nil {
		return nil, err
	}

	var clone T
	if err = json.Unmarshal(origJSON, &clone); err != nil {
		return nil, err
	}

	return &clone, nil
}

func IsHTTPResponseEmpty(r HTTPResponse) bool {
	if r.Body != "" || len(r.Headers) > 0 || len(r.Cookies) > 0 {
		return false
	}

	return true
}

func (r *Request) SetDefaultValues() {
	if r.MetaData.Type == "" {
		r.MetaData.Type = KindRequest
	}

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

func ParseQueryParams(params string, oldParams []KeyValue) []KeyValue {
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
		if len(pair) == 2 {
			out = append(out, KeyValue{Key: pair[0], Value: pair[1], Enable: true})
		} else {
			out = append(out, KeyValue{Key: pair[0], Value: "", Enable: true})
		}
	}

	return out
}
