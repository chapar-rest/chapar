package scripting

import (
	"context"

	"github.com/chapar-rest/chapar/internal/domain"
)

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

type Runner struct {
	BinPath    string
	ScriptPath string
	Args       []string
	Port       int

	Debug bool
}

// ScriptPlugin defines the interface all language plugins must implement
type ScriptPlugin interface {
	// Initialize sets up the plugin with the provided configuration
	Initialize(runner Runner) error

	// ExecutePreRequestScript runs a script before the request is sent
	// and potentially modifies the request data
	ExecutePreRequestScript(ctx context.Context, script string, params *ExecParams) (*ExecResult, error)

	// ExecutePostResponseScript runs a script after a response is received
	// and can access both request and response data
	ExecutePostResponseScript(ctx context.Context, script string, params *ExecParams) (*ExecResult, error)

	// GetInfo returns information about the plugin
	GetInfo() PluginInfo

	// Shutdown gracefully stops the plugin
	Shutdown() error
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Language    string `json:"language"`
	Description string `json:"description"`
}

// VariableStore defines an interface for reading and writing variables
// that can be shared between the app and scripts
type VariableStore interface {
	Get(name string) (interface{}, bool)
	Set(name string, value interface{})
	GetAll() map[string]interface{}
}

// PluginManager handles the registration and retrieval of plugins
type PluginManager interface {
	RegisterPlugin(name string, plugin ScriptPlugin, runner Runner) error
	Init() error
	GetPlugin(name string) (ScriptPlugin, bool)
	ListPlugins() []PluginInfo
	ShutdownAll() error
}
