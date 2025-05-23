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
	RequestBodyTypeFormData   = "formData"
	RequestBodyTypeBinary     = "binary"
	RequestBodyTypeUrlencoded = "urlencoded"

	PrePostTypeNone           = "none"
	PrePostTypeTriggerRequest = "triggerRequest"
	PrePostTypeSetEnv         = "setEnv"
	PrePostTypePython         = "python"
	PrePostTypeShell          = "ssh"
	PrePostTypeSSHTunnel      = "sshTunnel"
	PrePostTypeK8sTunnel      = "k8sTunnel"
)

var RequestMethods = []string{
	RequestMethodGET,
	RequestMethodPOST,
	RequestMethodPUT,
	RequestMethodDELETE,
	RequestMethodPATCH,
	RequestMethodHEAD,
	RequestMethodOPTIONS,
	RequestMethodCONNECT,
	RequestMethodTRACE,
}

type Request struct {
	ApiVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	MetaData   RequestMeta `yaml:"metadata"`
	Spec       RequestSpec `yaml:"spec"`

	CollectionName string `yaml:"-"`
	CollectionID   string `yaml:"-"`
}

type ResponseDetail struct {
	HTTP *HTTPResponseDetail
	GRPC *GRPCResponseDetail
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

func (r *RequestSpec) GetGRPC() *GRPCRequestSpec {
	if r.GRPC != nil {
		return r.GRPC
	}
	return nil
}

func (r *RequestSpec) GetHTTP() *HTTPRequestSpec {
	if r.HTTP != nil {
		return r.HTTP
	}
	return nil
}

func (r *GRPCRequestSpec) GetPreRequest() PreRequest {
	if r != nil {
		return r.PreRequest
	}
	return PreRequest{}
}

func (r *GRPCRequestSpec) GetPostRequest() PostRequest {
	if r != nil {
		return r.PostRequest
	}
	return PostRequest{}
}

func (h *HTTPRequestSpec) GetPreRequest() PreRequest {
	if h != nil {
		return h.Request.PreRequest
	}
	return PreRequest{}
}

func (h *HTTPRequestSpec) GetPostRequest() PostRequest {
	if h != nil {
		return h.Request.PostRequest
	}
	return PostRequest{}
}

func (h *HTTPRequestSpec) RenderParams() {
	for _, p := range h.Request.PathParams {
		if !p.Enable {
			continue
		}

		h.URL = strings.ReplaceAll(h.URL, "{"+p.Key+"}", p.Value)
	}

	for _, p := range h.Request.QueryParams {
		if !p.Enable {
			continue
		}

		h.URL = strings.ReplaceAll(h.URL, "{"+p.Key+"}", p.Value)
	}
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
	TriggerRequest   *TriggerRequest   `yaml:"triggerRequest,omitempty"`
}

type TriggerRequest struct {
	CollectionID string `yaml:"collectionID"`
	RequestID    string `yaml:"requestID"`
}

type PostRequest struct {
	Type           string         `yaml:"type"`
	Script         string         `yaml:"script"`
	PostRequestSet PostRequestSet `yaml:"set"`
}

const (
	PostRequestSetFromResponseHeader   = "responseHeader"
	PostRequestSetFromResponseBody     = "responseBody"
	PostRequestSetFromResponseCookie   = "responseCookie"
	PostRequestSetFromResponseMetaData = "responseMetaData"
	PostRequestSetFromResponseTrailers = "responseTrailers"
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

type VariableFrom string

func (v VariableFrom) String() string {
	return string(v)
}

const (
	VariableFromBody     VariableFrom = "body"
	VariableFromHeader   VariableFrom = "header"
	VariableFromCookies  VariableFrom = "cookies"
	VariableFromMetaData VariableFrom = "metadata"
	VariableFromTrailers VariableFrom = "trailers"
)

type Variable struct {
	ID                string       `yaml:"id"`                // Unique identifier
	TargetEnvVariable string       `yaml:"TargetEnvVariable"` // The environment variable to set
	From              VariableFrom `yaml:"from"`              // Source: "body", "header", "cookie"
	SourceKey         string       `yaml:"sourceKey"`         // For "header" or "cookie", specify the key name
	OnStatusCode      int          `yaml:"onStatusCode"`      // Trigger on a specific status code
	JsonPath          string       `yaml:"jsonPath"`          // JSONPath for extracting value (for "body")
	Enable            bool         `yaml:"enable"`            // Enable or disable the variable
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

	if !CompareTriggerRequest(a.TriggerRequest, b.TriggerRequest) {
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

func CompareTriggerRequest(a, b *TriggerRequest) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.CollectionID != b.CollectionID || a.RequestID != b.RequestID {
		return false
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

func CompareVariable(a, b Variable) bool {
	if a.ID != b.ID || a.TargetEnvVariable != b.TargetEnvVariable || a.From != b.From || a.SourceKey != b.SourceKey || a.OnStatusCode != b.OnStatusCode || a.JsonPath != b.JsonPath || a.Enable != b.Enable {
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

func KeyValueToMap(headers []KeyValue) map[string]string {
	headerMap := make(map[string]string)
	for _, h := range headers {
		headerMap[h.Key] = h.Value
	}
	return headerMap
}

func MapToKeyValue(headers map[string]string) []KeyValue {
	var keyValues []KeyValue
	for k, v := range headers {
		keyValues = append(keyValues, KeyValue{Key: k, Value: v})
	}
	return keyValues
}
