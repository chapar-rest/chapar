package scripting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
)

const (
	PythonContainerName = "chapar-python-executor"
	PythonExecutorName  = "Python"
)

var _ Executor = (*PythonExecutor)(nil)

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

func (p *PythonExecutor) Name() string {
	return PythonExecutorName
}

func (p *PythonExecutor) Init(cfg domain.ScriptingConfig) error {
	logger.Info(fmt.Sprintf("Python executor config port: %d", cfg.Port))
	if cfg.UseDocker {
		return p.initWithDocker(cfg)
	}

	return nil
}

func (p *PythonExecutor) initWithDocker(cfg domain.ScriptingConfig) error {
	// Check if the image exists and is up to date
	imageExists, isUptoDate, err := isImageUpToDate(cfg.DockerImage)
	if err != nil {
		return fmt.Errorf("failed to check if image exists: %w", err)
	}

	imageUpdated := false
	if imageExists && !isUptoDate {
		// if the image exists but is not up to date, remove it
		if err := removeImage(cfg.DockerImage); err != nil {
			return fmt.Errorf("failed to remove outdated image: %w", err)
		}

		imageExists = false // Set to false, to pull the latest image
	}

	if !imageExists {
		logger.Info("Pulling python executor docker image")
		// Pull the image if not present or not up to date
		if err := pullImage(cfg.DockerImage); err != nil {
			return err
		}

		// did we update the image?
		if !isUptoDate {
			imageUpdated = true
		}
	}

	// Check if the container is already running when image exists and is up to date
	running, err := isContainerRunning(PythonContainerName)
	if err != nil {
		return err
	}
	if running && imageUpdated {
		// If the container is running and the image was updated, we need to restart the container
		if err := forceRemoveContainer(PythonContainerName); err != nil {
			return fmt.Errorf("failed to remove existing outdated container: %w", err)
		}
	} else if running {
		logger.Info("Python executor docker container is already running")
		return nil
	}

	ports := []string{
		fmt.Sprintf("%d:%d", cfg.Port, cfg.Port),
	}

	envs := []string{
		fmt.Sprintf("PORT=%d", cfg.Port),
		"DEBUG=true",
	}

	logger.Info("Starting python executor docker container")
	// if container with same name already exists, remove it
	containerExists, err := isContainerExists(PythonContainerName)
	if err != nil {
		return fmt.Errorf("failed to check if container exists: %w", err)
	}

	if containerExists {
		if err := forceRemoveContainer(PythonContainerName); err != nil {
			return fmt.Errorf("failed to remove existing container: %w", err)
		}
	}

	// Run the container with the specified ports and environment variables
	if err := runContainer(cfg.DockerImage, PythonContainerName, ports, envs); err != nil {
		return fmt.Errorf("failed to run python executor container: %w", err)
	}

	if err := waitForPort("localhost", strconv.Itoa(cfg.Port), 10*time.Second); err != nil {
		return fmt.Errorf("failed to wait for python executor port: %w", err)
	}

	logger.Info("Python executor docker container started successfully")
	return nil
}

type executeRequestBody struct {
	Script       string                 `json:"script"`
	RequestData  *RequestData           `json:"requestData"`
	ResponseData *ResponseData          `json:"responseData"`
	Variables    map[string]interface{} `json:"variables"`
}

func (p *PythonExecutor) Execute(ctx context.Context, script string, params *ExecParams) (*ExecResult, error) {
	// Prepare the request body
	body := executeRequestBody{
		Script:       script,
		RequestData:  params.Req,
		ResponseData: params.Res,
		Variables:    params.Env.GetKeyValues(),
	}

	// Execute the script
	result, err := p.executeScript("/execute", body)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (p *PythonExecutor) Shutdown() error {
	if p.cfg.UseDocker {
		return forceRemoveContainer(PythonContainerName)
	}
	return nil
}

// executeScript sends a request to the Python server to execute a script
func (p *PythonExecutor) executeScript(endpoint string, requestBody interface{}) (*ExecResult, error) {
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
	var result ExecResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (p *PythonExecutor) serverURL() string {
	return fmt.Sprintf("http://localhost:%d", p.cfg.Port)
}
