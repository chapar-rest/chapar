package domain

import (
	"encoding/json"

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

type LastUsedEnvironment struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

const (
	AuthTypeNone   = "none"
	AuthTypeBasic  = "basic"
	AuthTypeToken  = "token"
	AuthTypeAPIKey = "apiKey"
)

type Auth struct {
	Type       string      `yaml:"type"`
	BasicAuth  *BasicAuth  `yaml:"basicAuth,omitempty"`
	TokenAuth  *TokenAuth  `yaml:"tokenAuth,omitempty"`
	APIKeyAuth *APIKeyAuth `yaml:"apiKey,omitempty"`
}

type APIKeyAuth struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

func (a *Auth) Clone() Auth {
	clone := *a
	if a.BasicAuth != nil {
		clone.BasicAuth = a.BasicAuth.Clone()
	}

	if a.TokenAuth != nil {
		clone.TokenAuth = a.TokenAuth.Clone()
	}

	if a.APIKeyAuth != nil {
		clone.APIKeyAuth = a.APIKeyAuth.Clone()
	}

	return clone
}

func (a *BasicAuth) Clone() *BasicAuth {
	return &BasicAuth{
		Username: a.Username,
		Password: a.Password,
	}
}

func (a *TokenAuth) Clone() *TokenAuth {
	return &TokenAuth{
		Token: a.Token,
	}
}

func (a *APIKeyAuth) Clone() *APIKeyAuth {
	return &APIKeyAuth{
		Key:   a.Key,
		Value: a.Value,
	}
}

type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type TokenAuth struct {
	Token string `yaml:"token"`
}

type PreRequest struct {
	Type   string `yaml:"type"`
	Script string `yaml:"script"`

	SShTunnel        *SShTunnel        `yaml:"sshTunnel,omitempty"`
	KubernetesTunnel *KubernetesTunnel `yaml:"kubernetesTunnel,omitempty"`
}

const (
	PostRequestTypeNone         = "none"
	PostRequestTypeSetEnv       = "setEnv"
	PostRequestTypePythonScript = "pythonScript"
	PostRequestTypeK8sTunnel    = "k8sTunnel"
	PostRequestTypeSSHTunnel    = "sshTunnel"
	PostRequestTypeShellScript  = "shellScript"
)

type PostRequest struct {
	Type           string         `yaml:"type"`
	Script         string         `yaml:"script"`
	PostRequestSet PostRequestSet `yaml:"set"`
}

const (
	PostRequestSetFromResponseHeader = "responseHeader"
	PostRequestSetFromResponseBody   = "responseBody"
	PostRequestSetFromResponseCookie = "responseCookie"
)

type PostRequestSet struct {
	Target     string `yaml:"target"`
	StatusCode int    `yaml:"statusCode"`
	// From can be response header, response body or cookies
	From    string `yaml:"from"`
	FromKey string `yaml:"fromKey"`
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

	if !CompareAPIKey(a.APIKeyAuth, b.APIKeyAuth) {
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

func CompareAPIKey(a, b *APIKeyAuth) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Key != b.Key || a.Value != b.Value {
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

	if !ComparePostRequestSet(a.PostRequestSet, b.PostRequestSet) {
		return false
	}

	return true
}

func ComparePostRequestSet(a, b PostRequestSet) bool {
	if a.Target != b.Target || a.From != b.From || a.FromKey != b.FromKey || a.StatusCode != b.StatusCode {
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

func (r *Request) SetDefaultValues() {
	if r.MetaData.Type == "" {
		r.MetaData.Type = KindRequest
	}

	if r.MetaData.ID == "" {
		r.MetaData.ID = uuid.NewString()
	}

	if r.MetaData.Type == RequestTypeHTTP {
		r.SetDefaultValuesForHTTP()
		return
	}

	if r.MetaData.Type == RequestTypeGRPC {
		r.SetDefaultValuesForGRPC()
	}
}
