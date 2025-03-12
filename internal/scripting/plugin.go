package scripting

import "context"

// RequestData represents the HTTP request data that can be modified by scripts
type RequestData struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// ResponseData represents the HTTP response data that can be accessed by scripts
type ResponseData struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// ScriptPlugin defines the interface all language plugins must implement
type ScriptPlugin interface {
	// Initialize sets up the plugin with the provided configuration
	Initialize(config map[string]interface{}) error

	// ExecutePreRequestScript runs a script before the request is sent
	// and potentially modifies the request data
	ExecutePreRequestScript(ctx context.Context, script string, requestData *RequestData) error

	// ExecutePostResponseScript runs a script after a response is received
	// and can access both request and response data
	ExecutePostResponseScript(ctx context.Context, script string, requestData *RequestData, responseData *ResponseData) error

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
	RegisterPlugin(name string, plugin ScriptPlugin, config map[string]interface{}) error
	GetPlugin(name string) (ScriptPlugin, bool)
	ListPlugins() []PluginInfo
	ShutdownAll() error
}
