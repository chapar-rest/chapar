package domain

import "github.com/google/uuid"

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

	Request   HTTPRequest    `yaml:"request"`
	Responses []HTTPResponse `yaml:"responses"`
}

type LastUsedEnvironment struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

func (h *HTTPRequestSpec) Clone() *HTTPRequestSpec {
	clone := *h
	return &clone
}

type HTTPRequest struct {
	Headers []KeyValue `yaml:"headers"`

	PathParams  []KeyValue `yaml:"pathParams"`
	QueryParams []KeyValue `yaml:"queryParams"`

	Body *Body `yaml:"body"`

	Auth *Auth `yaml:"auth"`

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

type Auth struct {
	Type      string     `yaml:"type"`
	BasicAuth *BasicAuth `yaml:"basicAuth,omitempty"`
	TokenAuth *TokenAuth `yaml:"tokenAuth,omitempty"`
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
				Request: HTTPRequest{
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

	if a.Spec.GRPC != nil && b.Spec.GRPC != nil {
		return CompareGRPCRequestSpecs(a.Spec.GRPC, b.Spec.GRPC)
	}

	if a.Spec.HTTP != nil && b.Spec.HTTP != nil {
		return CompareHTTPRequestSpecs(a.Spec.HTTP, b.Spec.HTTP)
	}

	return false
}

func CompareGRPCRequestSpecs(a, b *GRPCRequestSpec) bool {
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

func CompareHTTPRequests(a, b HTTPRequest) bool {
	if a.Body != b.Body || a.Body.Type != b.Body.Type {
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

	return true
}

func CompareHTTPResponses(a, b HTTPResponse) bool {
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
