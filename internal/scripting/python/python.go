package python

import (
	"bytes"
	"context"
	_ "embed"
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

//go:embed chapar.py
var serverScript string

// Plugin implements the ScriptPlugin interface for Python scripts
type Plugin struct {
	pythonPath       string
	serverScriptPath string
	serverPort       int
	serverProcess    *os.Process
	serverURL        string
	client           *http.Client
	variableStore    scripting.VariableStore

	runnerArgs []string
	debug      bool
}

// New creates a new Python plugin instance
func New(variableStore scripting.VariableStore) *Plugin {
	return &Plugin{
		client:        &http.Client{Timeout: 10 * time.Second},
		variableStore: variableStore,
	}
}

// Initialize starts the Python plugin server process
func (p *Plugin) Initialize(runner scripting.Runner) error {
	p.runnerArgs = runner.Args
	p.debug = runner.Debug

	p.pythonPath = "python" // default system
	if runner.BinPath != "" {
		p.pythonPath = runner.BinPath
	}

	if runner.ScriptPath == "" {
		return errors.New("script path is required")
	} else {
		// Check if the script path is absolute
		if !filepath.IsAbs(runner.ScriptPath) {
			return fmt.Errorf("script path must be absolute: %s", runner.ScriptPath)
		}

		// Set the Python path
		p.serverScriptPath = runner.ScriptPath
	}

	p.serverPort = 8090
	if runner.Port != 0 {
		p.serverPort = runner.Port
	}

	p.serverURL = fmt.Sprintf("http://localhost:%d", p.serverPort)

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
	cmd := exec.Command(p.pythonPath, p.runnerArgs...)

	if p.debug {
		cmd.Stdout = os.Stdout // For debugging
		cmd.Stderr = os.Stderr // For debugging
	}

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
			_ = resp.Body.Close()
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
	return os.WriteFile(p.serverScriptPath, []byte(serverScript), 0755)
}
