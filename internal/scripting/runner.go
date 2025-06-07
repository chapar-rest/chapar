package scripting

import (
	"context"

	"github.com/chapar-rest/chapar/internal/domain"
)

type Executor interface {
	Init(cfg domain.ScriptingConfig) error
	Execute(ctx context.Context, script string, params *ExecParams) (*ExecResult, error)
	Shutdown() error
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
	Req             *RequestData
	SetEnvironments map[string]interface{}
}
