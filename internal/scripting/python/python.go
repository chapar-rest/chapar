package python

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/chapar-rest/chapar/internal/scripting"
)

// Plugin implements the ScriptPlugin interface for Python scripts
type Plugin struct {
	pythonPath       string
	serverScriptPath string
	serverPort       int
	serverProcess    *os.Process
	serverURL        string
	client           *http.Client
	variableStore    scripting.VariableStore
}

// New creates a new Python plugin instance
func New(variableStore scripting.VariableStore) *Plugin {
	return &Plugin{
		client:        &http.Client{Timeout: 10 * time.Second},
		variableStore: variableStore,
	}
}

// Initialize starts the Python plugin server process
func (p *Plugin) Initialize(config map[string]interface{}) error {
	// Extract configuration
	pythonPath, ok := config["pythonPath"].(string)
	if !ok {
		pythonPath = "python" // Use system default if not specified
	}
	p.pythonPath = pythonPath

	// Get server script path from config or use default
	serverScriptPath, ok := config["serverScriptPath"].(string)
	if !ok {
		// If not provided, use the default location
		return errors.New("serverScriptPath is required")
	}
	p.serverScriptPath = serverScriptPath

	// Get port from config or use default
	port, ok := config["port"].(int)
	if !ok {
		port = 8090 // Default port
	}
	p.serverPort = port
	p.serverURL = fmt.Sprintf("http://localhost:%d", port)

	// Start the Python server
	return p.startServer()
}

// startServer launches the Python server as a separate process
func (p *Plugin) startServer() error {
	// Ensure script directory exists
	scriptDir := filepath.Dir(p.serverScriptPath)
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return fmt.Errorf("failed to create script directory: %w", err)
	}

	if err := p.createServerScript(); err != nil {
		return fmt.Errorf("failed to create server script: %w", err)
	}

	// Start the Python process
	cmd := exec.Command(
		p.pythonPath,
		p.serverScriptPath,
		"--port", fmt.Sprintf("%d", p.serverPort),
	)
	cmd.Stdout = os.Stdout // For debugging
	cmd.Stderr = os.Stderr // For debugging

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Python server: %w", err)
	}

	p.serverProcess = cmd.Process

	// Wait for the server to become available
	return p.waitForServer()
}

// waitForServer polls the server until it's ready
func (p *Plugin) waitForServer() error {
	for i := 0; i < 20; i++ { // Try for up to 10 seconds
		resp, err := p.client.Get(p.serverURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil // Server is up and running
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return errors.New("timeout waiting for Python server to start")
}

// ExecutePreRequestScript executes a Python script before a request is sent
func (p *Plugin) ExecutePreRequestScript(ctx context.Context, script string, params *scripting.ExecParams) (*scripting.ExecResult, error) {
	type requestBody struct {
		Script      string                 `json:"script"`
		RequestData *scripting.RequestData `json:"requestData"`
		Variables   map[string]interface{} `json:"variables"`
	}

	// Prepare the request body
	body := requestBody{
		Script:      script,
		RequestData: params.Req,
		Variables:   params.Env.GetKeyValues(),
	}

	// Execute the script
	result, err := p.executeScript("/execute-pre-request", body)
	if err != nil {
		return nil, err
	}

	out := &scripting.ExecResult{
		SetEnvironments: map[string]interface{}{},
	}

	if params.Req != nil {
		out.Req = &scripting.RequestData{}

		// Update the request data with the result
		if updatedReqData, ok := result["requestData"].(map[string]interface{}); ok {
			if method, ok := updatedReqData["method"].(string); ok {
				out.Req.Method = method
			}
			if url, ok := updatedReqData["url"].(string); ok {
				out.Req.URL = url
			}
			if headers, ok := updatedReqData["headers"].(map[string]interface{}); ok {
				for k, v := range headers {
					if strVal, ok := v.(string); ok {
						out.Req.Headers[k] = strVal
					}
				}
			}
			if body, ok := updatedReqData["body"].(string); ok {
				out.Req.Body = body
			}
		}
	}

	// Update variables in the store
	if updatedVars, ok := result["set_environments"].(map[string]interface{}); ok {
		out.SetEnvironments = updatedVars
	}

	return out, nil
}

// ExecutePostResponseScript executes a Python script after a response is received
func (p *Plugin) ExecutePostResponseScript(
	ctx context.Context,
	script string,
	params *scripting.ExecParams,
) (*scripting.ExecResult, error) {
	type requestBody struct {
		Script       string                  `json:"script"`
		RequestData  *scripting.RequestData  `json:"requestData"`
		ResponseData *scripting.ResponseData `json:"responseData"`
		Variables    map[string]interface{}  `json:"variables"`
	}

	// Prepare the request body
	body := requestBody{
		Script:       script,
		RequestData:  params.Req,
		ResponseData: params.Res,
		Variables:    params.Env.GetKeyValues(),
	}

	// Execute the script
	result, err := p.executeScript("/execute-post-response", body)
	if err != nil {
		return nil, err
	}

	out := &scripting.ExecResult{
		SetEnvironments: map[string]interface{}{},
	}
	// Update variables in the store
	if updatedVars, ok := result["set_environments"].(map[string]interface{}); ok {
		out.SetEnvironments = updatedVars
	}

	return out, nil
}

// executeScript sends a request to the Python server to execute a script
func (p *Plugin) executeScript(endpoint string, requestBody interface{}) (map[string]interface{}, error) {
	// Marshal the request body
	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send the request to the Python server
	resp, err := p.client.Post(p.serverURL+endpoint, "application/json", io.NopCloser(io.NopCloser(io.ReadCloser(io.NopCloser(bytes.NewReader(data))))))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Python server: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python server returned error: %s", string(respBody))
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetInfo returns information about the plugin
func (p *Plugin) GetInfo() scripting.PluginInfo {
	return scripting.PluginInfo{
		Name:        "python",
		Version:     "1.0.0",
		Language:    "Python",
		Description: "Executes Python scripts for request customization",
	}
}

// Shutdown terminates the Python server process
func (p *Plugin) Shutdown() error {
	if p.serverProcess != nil {
		// Try to gracefully terminate
		if err := p.serverProcess.Signal(os.Interrupt); err != nil {
			// Force kill if graceful termination fails
			return p.serverProcess.Kill()
		}
	}
	return nil
}

// createServerScript writes the Python server script to disk
func (p *Plugin) createServerScript() error {
	content := `
#!/usr/bin/env python3
import types
import sys
import json
import argparse
import traceback
from flask import Flask, request, jsonify

def create_chapar_module():
    """
    Dynamically create the chapar module with full implementation
    """
    # Create a new module object
    chapar_module = types.ModuleType('chapar')
    chapar_module.__file__ = '<dynamic>'
    chapar_module.__doc__ = """
    chapar module - Interface for interacting with the Chapar application
    """

    # Store environments internally
    environments = {}
    set_environments = {}

    # Environment variable methods
    def get_env(name):
        value = environments.get(name)
        return value

    def set_env(name, value):
        set_environments[name] = value

    def log(message):
        print(f"CHAPAR_LOG: {message}")

    # Assign methods to the module
    chapar_module.get_env = get_env
    chapar_module.set_env = set_env
    chapar_module.log = log
    chapar_module.onResponse = None
    chapar_module._environments = environments
    chapar_module._set_environments = set_environments

    # Register the module in sys.modules
    sys.modules['chapar'] = chapar_module
    return chapar_module


# Create the chapar module
chapar = create_chapar_module()
app = Flask(__name__)


@app.route("/health")
def health_check():
    return jsonify({"status": "ok"})


@app.route("/execute-pre-request", methods=["POST"])
def execute_pre_request():
    try:
        data = request.json
        script = data.get("script", "")
        request_data = data.get("requestData", {})
        environments = data.get("environments", {})

        # Update chapar module environments
        chapar._environments.clear()
        chapar._set_environments.clear()
        chapar._environments.update(environments)

        # Prepare execution environment
        globals_dict = {
            "__builtins__": __builtins__,
            "chapar": chapar  # Make chapar available in globals
        }

        locals_dict = {
            "request": request_data,
            "chapar": chapar,  # Also make it available in locals
            "print": print
        }

        # Now execute the actual script
        exec(script, globals_dict, locals_dict)

        # Return the potentially modified data
        return jsonify({
            "requestData": locals_dict["request"],
            "set_environments": chapar._set_environments
        })
    except Exception as e:
        return jsonify({"error": str(e), "traceback": traceback.format_exc()}), 400


@app.route("/execute-post-response", methods=["POST"])
def execute_post_response():
    try:
        data = request.json
        script = data.get("script", "")
        request_data = data.get("requestData", {})
        response_data = data.get("responseData", {})
        environments = data.get("environments", {})

        # Update chapar module environments
        chapar._environments.clear()
        chapar._set_environments.clear()
        chapar._environments.update(environments)

        # Create response object
        response_obj = type("ResponseObject", (), {
            "status_code": response_data.get("statusCode"),
            "headers": response_data.get("headers", {}),
            "text": response_data.get("body", ""),
            "json": lambda self=None: json.loads(response_data.get("body", "{}")),
        })()

        # Prepare execution environment
        globals_dict = {
            "__builtins__": __builtins__,
            "chapar": chapar  # Make chapar available in globals
        }

        locals_dict = {
            "request": request_data,
            "response": response_obj,
            "chapar": chapar,  # Also make it available in locals
            "print": print
        }

        # Reset any callbacks
        chapar.onResponse = None

        # Execute the script
        exec(script, globals_dict, locals_dict)

        # If onResponse was set, call it
        if chapar.onResponse is not None and callable(chapar.onResponse):
            chapar.onResponse(response_obj)

        # Return the potentially modified data
        return jsonify({
            "environments": chapar._environments,
            "set_environments": chapar._set_environments,
        })
    except Exception as e:
        return jsonify({"error": str(e), "traceback": traceback.format_exc()}), 400


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=8090)
    args = parser.parse_args()
    app.run(host="127.0.0.1", port=args.port, debug=False)
`
	return os.WriteFile(p.serverScriptPath, []byte(content), 0755)
}
