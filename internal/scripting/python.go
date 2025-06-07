package scripting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
)

const (
	containerName = "chapar-python-executor"
)

var _ Executor = &PythonExecutor{}

type PythonExecutor struct {
	client *http.Client
	cfg    domain.ScriptingConfig
}

func NewPythonExecutor(cfg domain.ScriptingConfig) *PythonExecutor {
	return &PythonExecutor{
		client: &http.Client{Timeout: 10 * time.Second},
		cfg:    cfg,
	}
}

func (p PythonExecutor) Init(cfg domain.ScriptingConfig) error {
	logger.Info(fmt.Sprintf("Python executor config port: %d", cfg.Port))
	if cfg.UseDocker {
		return p.initWithDocker(cfg)
	}

	return nil
}

func (p PythonExecutor) initWithDocker(cfg domain.ScriptingConfig) error {
	// Check if the container is already running
	running, err := isContainerRunning(containerName)
	if err != nil {
		return err
	}
	if running {
		return nil
	}
	logger.Info("Pulling python executor docker image")
	// Pull the image if not present
	if err := pullImage(cfg.DockerImage); err != nil {
		return err
	}

	ports := []string{
		fmt.Sprintf("%d:%d", cfg.Port, cfg.Port),
	}

	envs := []string{
		fmt.Sprintf("PORT=%d", cfg.Port),
		fmt.Sprintf("DEBUG=true"),
	}

	logger.Info("Starting python executor docker container")
	// Run the container with the specified ports and environment variables
	return runContainer(cfg.DockerImage, containerName, ports, envs)
}

type executeRequestBody struct {
	Script       string                 `json:"script"`
	RequestData  *RequestData           `json:"requestData"`
	ResponseData *ResponseData          `json:"responseData"`
	Variables    map[string]interface{} `json:"variables"`
}

func (p PythonExecutor) Execute(ctx context.Context, script string, params *ExecParams) (*ExecResult, error) {
	// Prepare the request body
	body := executeRequestBody{
		Script:       script,
		RequestData:  params.Req,
		ResponseData: params.Res,
		Variables:    params.Env.GetKeyValues(),
	}

	// Execute the script
	// TODO: update executor to just expose one endpoint for execute
	result, err := p.executeScript("/execute-post-response", body)
	if err != nil {
		return nil, err
	}

	out := &ExecResult{
		SetEnvironments: map[string]interface{}{},
	}
	// Update variables in the store
	if updatedVars, ok := result["set_environments"].(map[string]interface{}); ok {
		out.SetEnvironments = updatedVars
	}

	return out, nil
}

func (p PythonExecutor) Shutdown() error {
	if p.cfg.UseDocker {
		return forceRemoveContainer(containerName)
	}
	return nil
}

// executeScript sends a request to the Python server to execute a script
func (p PythonExecutor) executeScript(endpoint string, requestBody interface{}) (map[string]interface{}, error) {
	// Marshal the request body
	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.serverURL() + endpoint
	// Send the request to the Python server
	resp, err := p.client.Post(url, "application/json", io.NopCloser(bytes.NewReader(data)))
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

func (p PythonExecutor) serverURL() string {
	return fmt.Sprintf("http://localhost:%d", p.cfg.Port)
}
