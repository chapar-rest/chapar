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
	Settings          GRPCSettings  `yaml:"settings"`
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
	// ImportPaths are the roots protoc resolves `import` statements against,
	// so a request carries everything its proto files need to parse.
	ImportPaths []string `yaml:"importPaths"`
}

type GRPCSettings struct {
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

	// Deep clone slices to avoid modifying the original
	if len(r.Metadata) > 0 {
		clone.Metadata = make([]KeyValue, len(r.Metadata))
		copy(clone.Metadata, r.Metadata)
	}

	if len(r.Variables) > 0 {
		clone.Variables = make([]Variable, len(r.Variables))
		copy(clone.Variables, r.Variables)
	}

	if len(r.Services) > 0 {
		clone.Services = make([]GRPCService, len(r.Services))
		for i, service := range r.Services {
			clone.Services[i] = service
			if len(service.Methods) > 0 {
				clone.Services[i].Methods = make([]GRPCMethod, len(service.Methods))
				copy(clone.Services[i].Methods, service.Methods)
			}
		}
	}

	// Deep clone ServerInfo.ProtoFiles
	if len(r.ServerInfo.ProtoFiles) > 0 {
		clone.ServerInfo.ProtoFiles = make([]string, len(r.ServerInfo.ProtoFiles))
		copy(clone.ServerInfo.ProtoFiles, r.ServerInfo.ProtoFiles)
	}

	if len(r.ServerInfo.ImportPaths) > 0 {
		clone.ServerInfo.ImportPaths = make([]string, len(r.ServerInfo.ImportPaths))
		copy(clone.ServerInfo.ImportPaths, r.ServerInfo.ImportPaths)
	}

	// Clone Auth
	if r.Auth != (Auth{}) {
		clone.Auth = r.Auth.Clone()
	}

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
				Settings: GRPCSettings{
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

	if !CompareGRPCSettings(a.Settings, b.Settings) {
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

func CompareGRPCSettings(a, b GRPCSettings) bool {
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

	return compareStringSlices(a.ProtoFiles, b.ProtoFiles) &&
		compareStringSlices(a.ImportPaths, b.ImportPaths)
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
