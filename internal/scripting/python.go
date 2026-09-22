package scripting

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
)

const (
	PythonContainerName = "chapar-python-executor"
	PythonExecutorName  = "Python"

	// executorAPI is the executor API version Chapar speaks; /health lists
	// the versions an executor supports.
	executorAPI = 2
	// ScriptTimeout is how long one script may run before the executor
	// kills it.
	ScriptTimeout = 10 * time.Second

	tokenHeader = "X-Chapar-Token"
	tokenEnv    = "CHAPAR_EXECUTOR_TOKEN"
)

var _ Executor = (*PythonExecutor)(nil)

type PythonExecutor struct {
	client *http.Client
	cfg    domain.ScriptingConfig
	// token authenticates Chapar to the container it started; the executor
	// runs any code it is sent, so nothing else may call it. Empty for a
	// server the user runs themselves.
	token string
}

func NewPythonExecutor(cfg domain.ScriptingConfig) *PythonExecutor {
	return &PythonExecutor{
		// The executor enforces the script timeout; this only has to
		// outlast it.
		client: &http.Client{Timeout: ScriptTimeout + 5*time.Second},
		cfg:    cfg,
	}
}

func (p *PythonExecutor) Name() string {
	return PythonExecutorName
}

func (p *PythonExecutor) Init(cfg domain.ScriptingConfig) error {
	logger.Info(fmt.Sprintf("Python executor config port: %d", cfg.Port))
	if cfg.UseDocker {
		if err := p.initWithDocker(cfg); err != nil {
			return err
		}
	}

	return p.checkHealth(10 * time.Second)
}

type healthResponse struct {
	Version string `json:"version"`
	API     []int  `json:"api"`
}

// checkHealth waits for the executor to answer /health and checks it speaks
// executorAPI.
func (p *PythonExecutor) checkHealth(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		h, err := p.health()
		if err == nil {
			if !slices.Contains(h.API, executorAPI) {
				image := "the script server"
				if p.cfg.UseDocker {
					image = "image " + p.cfg.DockerImage
				}
				return fmt.Errorf("%s is too old (version %q) for this Chapar; update it to a version that supports scripting API v%d",
					image, h.Version, executorAPI)
			}
			return nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return fmt.Errorf("python executor is not answering on port %d: %w", p.cfg.Port, lastErr)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (p *PythonExecutor) health() (*healthResponse, error) {
	resp, err := p.client.Get(p.serverURL() + "/health")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check returned %s", resp.Status)
	}
	var h healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return nil, fmt.Errorf("reading health check: %w", err)
	}
	return &h, nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (p *PythonExecutor) initWithDocker(cfg domain.ScriptingConfig) error {
	// Always start a fresh container: it has to hold this executor's token.
	// Removing it first also frees an outdated image for removal.
	containerExists, err := isContainerExists(PythonContainerName)
	if err != nil {
		return fmt.Errorf("failed to check if container exists: %w", err)
	}
	if containerExists {
		if err := forceRemoveContainer(PythonContainerName); err != nil {
			return fmt.Errorf("failed to remove existing container: %w", err)
		}
	}

	imageExists, isUpToDate, err := isImageUpToDate(cfg.DockerImage)
	if err != nil {
		return fmt.Errorf("failed to check if image exists: %w", err)
	}
	if imageExists && !isUpToDate {
		if err := removeImage(cfg.DockerImage); err != nil {
			return fmt.Errorf("failed to remove outdated image: %w", err)
		}
		imageExists = false
	}
	if !imageExists {
		logger.Info("Pulling python executor docker image")
		if err := pullImage(cfg.DockerImage); err != nil {
			return err
		}
	}

	token, err := newToken()
	if err != nil {
		return fmt.Errorf("failed to create executor token: %w", err)
	}
	p.token = token

	ports := []string{
		fmt.Sprintf("%d:%d", cfg.Port, cfg.Port),
	}
	envs := []string{
		fmt.Sprintf("PORT=%d", cfg.Port),
		tokenEnv + "=" + token,
	}

	logger.Info("Starting python executor docker container")
	if err := runContainer(cfg.DockerImage, PythonContainerName, ports, envs); err != nil {
		return fmt.Errorf("failed to run python executor container: %w", err)
	}

	if err := waitForPort("localhost", strconv.Itoa(cfg.Port), 10*time.Second); err != nil {
		return fmt.Errorf("failed to wait for python executor port: %w", err)
	}

	logger.Info("Python executor docker container started successfully")
	return nil
}

type executeEnvironment struct {
	Name string         `json:"name"`
	Vars map[string]any `json:"vars"`
}

type executeRequestBody struct {
	Phase       Phase              `json:"phase"`
	Protocol    Protocol           `json:"protocol"`
	Script      string             `json:"script"`
	TimeoutMS   int64              `json:"timeout_ms"`
	Environment executeEnvironment `json:"environment"`
	Request     *RequestData       `json:"request"`
	Response    *ResponseData      `json:"response,omitempty"`
}

func (p *PythonExecutor) Execute(ctx context.Context, script string, params *ExecParams) (*ExecResult, error) {
	body := executeRequestBody{
		Phase:     params.Phase,
		Protocol:  params.Protocol,
		Script:    script,
		TimeoutMS: ScriptTimeout.Milliseconds(),
		Request:   params.Req,
		Response:  params.Res,
	}
	if body.Phase == "" {
		body.Phase = PhasePre
		if params.Res != nil {
			body.Phase = PhasePost
		}
	}
	if body.Protocol == "" {
		body.Protocol = ProtocolHTTP
	}
	if params.Env != nil {
		body.Environment = executeEnvironment{Name: params.Env.MetaData.Name, Vars: params.Env.GetKeyValues()}
	}
	return p.executeScript(ctx, body)
}

func (p *PythonExecutor) Shutdown() error {
	if p.cfg.UseDocker {
		return forceRemoveContainer(PythonContainerName)
	}
	return nil
}

// executeScript sends a script to the executor. A script that raised is
// not an error here: the result carries it in Error.
func (p *PythonExecutor) executeScript(ctx context.Context, body executeRequestBody) (*ExecResult, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.serverURL()+"/v2/execute", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set(tokenHeader, p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach the python executor: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, errors.New("the python executor rejected Chapar's token; restart it from Settings > Scripting")
	case http.StatusNotFound:
		return nil, fmt.Errorf("the python executor does not support scripting API v%d; update it", executorAPI)
	default:
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &e) == nil && e.Message != "" {
			return nil, fmt.Errorf("python executor: %s", e.Message)
		}
		return nil, fmt.Errorf("python executor returned %s: %s", resp.Status, bytes.TrimSpace(respBody))
	}

	var result ExecResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (p *PythonExecutor) serverURL() string {
	return fmt.Sprintf("http://localhost:%d", p.cfg.Port)
}
