package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type GRPCRequestSpec struct {
	LasSelectedMethod string        `yaml:"lastSelectedMethod"`
	Metadata          []KeyValue    `yaml:"metadata"`
	Auth              Auth          `yaml:"auth"`
	ServerInfo        ServerInfo    `yaml:"serverInfo"`
	Settings          Settings      `yaml:"settings"`
	Body              string        `yaml:"body"`
	Services          []GRPCService `yaml:"services"`
	Variables         []Variable    `yaml:"variables"`

	PreRequest  PreRequest  `yaml:"preRequest"`
	PostRequest PostRequest `yaml:"postRequest"`
}

type GRPCService struct {
	Name    string `yaml:"name"`
	Methods []GRPCMethod
}

type ServerInfo struct {
	Address string `yaml:"address"`

	ServerReflection bool     `yaml:"serverReflection"`
	ProtoFiles       []string `yaml:"protoFiles"`
}

type Settings struct {
	Insecure            bool   `yaml:"insecure"`
	TimeoutMilliseconds int    `yaml:"timeoutMilliseconds"`
	NameOverride        string `yaml:"nameOverride"`

	RootCertFile   string `yaml:"rootCertFile"`
	ClientCertFile string `yaml:"clientCertFile"`
	ClientKeyFile  string `yaml:"clientKeyFile"`
}

type GRPCMethod struct {
	FullName          string `yaml:"fullName"`
	Name              string `yaml:"name"`
	IsStreamingClient bool   `yaml:"IsStreamingClient"`
	IsStreamingServer bool   `yaml:"IsStreamingServer"`
}

type GRPCResponseDetail struct {
	Response         string
	RequestMetadata  []KeyValue
	ResponseMetadata []KeyValue
	Trailers         []KeyValue
	StatusCode       int
	Duration         time.Duration
	Size             int
	Error            error

	StatueCode int
	Status     string
}

func (r *GRPCRequestSpec) Clone() *GRPCRequestSpec {
	clone := *r
	return &clone
}

func (r *GRPCRequestSpec) HasMethod(method string) bool {
	for _, srv := range r.Services {
		for _, m := range srv.Methods {
			if m.FullName == method {
				return true
			}
		}
	}

	return false
}

func NewGRPCRequest(name string) *Request {
	return &Request{
		ApiVersion: ApiVersion,
		Kind:       KindRequest,
		MetaData: RequestMeta{
			ID:   uuid.NewString(),
			Name: name,
			Type: RequestTypeGRPC,
		},
		Spec: RequestSpec{
			GRPC: &GRPCRequestSpec{
				LasSelectedMethod: "",
				ServerInfo: ServerInfo{
					Address: "localhost:8090",
				},
				Settings: Settings{
					Insecure: true,
				},
			},
		},
	}
}

func CompareGRPCRequestSpecs(a, b *GRPCRequestSpec) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Body != b.Body {
		return false
	}

	if !CompareKeyValues(a.Metadata, b.Metadata) {
		return false
	}

	if !CompareAuth(a.Auth, b.Auth) {
		return false
	}

	if !CompareServerInfo(a.ServerInfo, b.ServerInfo) {
		return false
	}

	if !CompareSettings(a.Settings, b.Settings) {
		return false
	}

	if a.LasSelectedMethod != b.LasSelectedMethod {
		return false
	}

	if !CompareGRPCServices(a.Services, b.Services) {
		return false
	}

	if !ComparePreRequest(a.PreRequest, b.PreRequest) {
		return false
	}

	if !ComparePostRequest(a.PostRequest, b.PostRequest) {
		return false
	}

	if !CompareVariables(a.Variables, b.Variables) {
		return false
	}

	return true
}

func (r *Request) SetDefaultValuesForGRPC() {
	if r.Spec.GRPC.ServerInfo.Address == "" {
		r.Spec.GRPC.ServerInfo.Address = "localhost:8090"
	}
}

func CompareSettings(a, b Settings) bool {
	if a.Insecure != b.Insecure ||
		a.TimeoutMilliseconds != b.TimeoutMilliseconds ||
		a.NameOverride != b.NameOverride ||
		a.RootCertFile != b.RootCertFile ||
		a.ClientCertFile != b.ClientCertFile ||
		a.ClientKeyFile != b.ClientKeyFile {
		return false
	}

	return true
}

func CompareServerInfo(a, b ServerInfo) bool {
	if a.Address != b.Address || a.ServerReflection != b.ServerReflection {
		return false
	}

	if len(a.ProtoFiles) != len(b.ProtoFiles) {
		return false
	}

	for i, v := range a.ProtoFiles {
		if v != b.ProtoFiles[i] {
			return false
		}
	}

	return true
}

func CompareGRPCMethods(a, b []GRPCMethod) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v.Name != b[i].Name || v.FullName != b[i].FullName || v.IsStreamingClient != b[i].IsStreamingClient || v.IsStreamingServer != b[i].IsStreamingServer {
			return false
		}
	}

	return true
}

func CompareGRPCServices(a, b []GRPCService) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v.Name != b[i].Name || !CompareGRPCMethods(v.Methods, b[i].Methods) {
			return false
		}
	}

	return true
}

func MetadataToKeyValue(md metadata.MD) []KeyValue {
	headers := make([]KeyValue, 0, len(md))
	for k, v := range md {
		headers = append(headers, KeyValue{
			Key:   k,
			Value: strings.Join(v, ","),
		})
	}

	return headers
}
