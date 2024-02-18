package domain

const (
	KindRequest = "Request"

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

type Request[Spec RequestSpec] struct {
	ApiVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Meta       RequestMeta `yaml:"meta"`
	Spec       Spec        `yaml:"spec"`
	FilePath   string      `yaml:"-"`
}

type RequestMeta struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type RequestSpec interface {
	GRPCRequestSpec | HTTPRequestSpec
}

type GRPCRequestSpec struct {
	Host   string `yaml:"host"`
	Method string `yaml:"method"`
}

type HTTPRequestSpec struct {
	Method string `yaml:"method"`
	URL    string `yaml:"url"`

	LastUsedEnvironment string `yaml:"lastUsedEnvironment"`

	Body      HTTPRequest    `yaml:"body"`
	Responses []HTTPResponse `yaml:"responses"`
}

type HTTPRequest struct {
	Headers []KeyValue `yaml:"headers"`

	PathParams  []KeyValue `yaml:"pathParams"`
	QueryParams []KeyValue `yaml:"queryParams"`

	BodyType   string     `yaml:"bodyType"`
	FormBody   []KeyValue `yaml:"formBody"`
	URLEncoded []KeyValue `yaml:"urlEncoded"`

	// Can be json, xml, or plain text
	Body string `yaml:"body"`

	PreRequest  PreRequest  `yaml:"preRequest"`
	PostRequest PostRequest `yaml:"postRequest"`
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
