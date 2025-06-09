package scripting

import (
	"context"
	"fmt"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
)

type Executor interface {
	Init(cfg domain.ScriptingConfig) error
	Execute(ctx context.Context, script string, params *ExecParams) (*ExecResult, error)
	Name() string
	Shutdown() error
}

func GetExecutor(language string, cfg domain.ScriptingConfig) (Executor, error) {
	n := strings.ToLower(language)
	if n == "python" {
		return NewPythonExecutor(cfg), nil
	}

	return nil, fmt.Errorf("unknown scripting executor: %s", language)
}

// RequestData represents the HTTP request data that can be modified by scripts
type RequestData struct {
	// Body is the request body
	Body string `json:"body"`
	// when grpc is used, this field in the grpc method name otherwise it is the http method
	Method string `json:"method"`

	// URL is the request URL in case of grpc it is the server address
	URL string `json:"url"`

	// GRPC related fields
	Metadata map[string]string `json:"metadata"`

	// HTTP related fields
	Headers     map[string]string `json:"headers"`
	QueryParams map[string]string `json:"QueryParams"`
	PathParams  map[string]string `json:"pathParams"`
}

// ResponseData represents the HTTP response data that can be accessed by scripts
type ResponseData struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type ExecParams struct {
	Req *RequestData
	Res *ResponseData
	Env *domain.Environment
}

type ExecResult struct {
	Req             *RequestData           `json:"-"`
	SetEnvironments map[string]interface{} `json:"set_environments"`
	Prints          []string               `json:"prints"`
}

func RequestDataFromDomain(req *domain.Request) *RequestData {
	out := &RequestData{}

	if req == nil {
		return out
	}

	if httpReq := req.Spec.GetHTTP(); httpReq != nil {
		out.Method = httpReq.Method
		out.URL = httpReq.URL
		out.Headers = make(map[string]string)
		for _, header := range httpReq.Request.Headers {
			out.Headers[header.Key] = header.Value
		}
		out.QueryParams = make(map[string]string)
		for _, param := range httpReq.Request.QueryParams {
			out.QueryParams[param.Key] = param.Value
		}
		out.PathParams = make(map[string]string)
		for _, param := range httpReq.Request.PathParams {
			out.PathParams[param.Key] = param.Value
		}
		out.Body = httpReq.Request.Body.Data
	}

	if grpcReq := req.Spec.GetGRPC(); grpcReq != nil {
		out.Method = grpcReq.LasSelectedMethod
		out.URL = grpcReq.ServerInfo.Address
		out.Metadata = make(map[string]string)
		for _, header := range grpcReq.Metadata {
			out.Metadata[header.Key] = header.Value
		}
	}

	return out
}
