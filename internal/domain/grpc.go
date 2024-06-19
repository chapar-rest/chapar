package domain

import (
	"github.com/google/uuid"
)

type GRPCRequestSpec struct {
	LasSelectedMethod string        `yaml:"lastSelectedMethod"`
	Metadata          []KeyValue    `yaml:"metadata"`
	Auth              Auth          `yaml:"auth"`
	ServerInfo        ServerInfo    `yaml:"serverInfo"`
	Settings          Settings      `yaml:"settings"`
	Body              string        `yaml:"body"`
	Services          []GRPCService `yaml:"services"`
}

type GRPCService struct {
	Name    string `yaml:"name"`
	Methods []GRPCMethod
}

type ServerInfo struct {
	Address string `yaml:"address"`

	ServerReflection bool        `yaml:"serverReflection"`
	ProtoFiles       []ProtoFile `yaml:"protoFiles"`
}

type Settings struct {
	UseSSL              bool   `yaml:"useSSL"`
	TimeoutMilliseconds int    `yaml:"timeoutMilliseconds"`
	NameOverride        string `yaml:"nameOverride"`
}

type ProtoFile struct {
	Path string `yaml:"path"`
}

type GRPCMethod struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

func (g *GRPCRequestSpec) Clone() *GRPCRequestSpec {
	clone := *g
	return &clone
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

	return true
}

func (r *Request) SetDefaultValuesForGRPC() {
	if r.Spec.GRPC.ServerInfo.Address == "" {
		r.Spec.GRPC.ServerInfo.Address = "localhost:8090"
	}
}

func CompareSettings(a, b Settings) bool {
	if a.UseSSL != b.UseSSL || a.TimeoutMilliseconds != b.TimeoutMilliseconds || a.NameOverride != b.NameOverride {
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
		if v.Path != b.ProtoFiles[i].Path {
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
		if v.Name != b[i].Name || v.Type != b[i].Type {
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
