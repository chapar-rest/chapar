// Package scriptsrv owns the scripting executor that runs pre/post-request
// scripts: it starts, stops and restarts it, and reports its state.
package scriptsrv

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/scripting"
)

type State int

const (
	Stopped State = iota
	Starting
	Running
	Failed
)

// Service runs one executor at a time. Starting one can take long (a Docker
// image pull), so Restart works in the background and reports progress
// through Status and the wake function.
type Service struct {
	// newExecutor and probe are replaced in tests.
	newExecutor func(domain.ScriptingConfig) (scripting.Executor, error)
	probe       func(port int) error

	// ops serializes starts and stops, so a restart never races a start.
	ops sync.Mutex

	mu    sync.Mutex
	state State
	err   error
	cfg   domain.ScriptingConfig
	exec  scripting.Executor
	wake  func()
}

func New() *Service {
	return &Service{
		newExecutor: func(cfg domain.ScriptingConfig) (scripting.Executor, error) {
			return scripting.GetExecutor(cfg.Language, cfg)
		},
		probe: probePort,
	}
}

// SetWake sets the function that requests a repaint. Safe from any goroutine.
func (s *Service) SetWake(fn func()) {
	s.mu.Lock()
	s.wake = fn
	s.mu.Unlock()
}

// Restart stops the executor and, when cfg enables scripting, starts it
// again with cfg, even when cfg is what it already runs. It returns at once;
// the work runs in the background.
func (s *Service) Restart(cfg domain.ScriptingConfig) {
	go s.run(cfg)
}

// Shutdown stops the executor. It is for app exit, so it does not wait out
// a start in progress (a Docker pull can take minutes); it leaves that one.
func (s *Service) Shutdown() {
	if !s.ops.TryLock() {
		logger.Warn("scripting: executor still starting at exit; leaving it")
		return
	}
	defer s.ops.Unlock()
	s.stopLocked()
}

func (s *Service) run(cfg domain.ScriptingConfig) {
	s.ops.Lock()
	defer s.ops.Unlock()

	s.stopLocked()
	if !cfg.Enabled {
		return
	}

	s.set(Starting, nil, cfg, nil)
	exec, err := s.newExecutor(cfg)
	if err == nil {
		// A Docker executor that finds its container already running keeps
		// it, so clear it out first: restarting has to start a fresh one.
		if err = exec.Shutdown(); err != nil {
			logger.Warn(fmt.Sprintf("scripting: removing old executor: %v", err))
		}
		logger.Info(fmt.Sprintf("Starting %s script executor", exec.Name()))
		err = exec.Init(cfg)
	}
	if err == nil && !cfg.UseDocker {
		// Without Docker Chapar starts nothing; the server must already be
		// listening.
		err = s.probe(cfg.Port)
	}
	if err != nil {
		logger.Error(fmt.Sprintf("scripting: %v", err))
		s.set(Failed, err, cfg, nil)
		return
	}
	logger.Info(fmt.Sprintf("%s script executor is running", exec.Name()))
	s.set(Running, nil, cfg, exec)
}

// stopLocked shuts the running executor down. The caller holds s.ops.
func (s *Service) stopLocked() {
	s.mu.Lock()
	exec := s.exec
	s.mu.Unlock()
	if exec != nil {
		if err := exec.Shutdown(); err != nil {
			logger.Error(fmt.Sprintf("scripting: stopping %s executor: %v", exec.Name(), err))
		}
	}
	s.set(Stopped, nil, domain.ScriptingConfig{}, nil)
}

func (s *Service) set(state State, err error, cfg domain.ScriptingConfig, exec scripting.Executor) {
	s.mu.Lock()
	s.state, s.err, s.cfg, s.exec = state, err, cfg, exec
	wake := s.wake
	s.mu.Unlock()
	if wake != nil {
		wake()
	}
}

// State reports what the executor is doing and, when it failed, why.
func (s *Service) State() (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state, s.err
}

// Status describes the executor's state for the settings page.
func (s *Service) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.state {
	case Starting:
		if s.cfg.UseDocker {
			return "Starting · pulling and running " + s.cfg.DockerImage + "…"
		}
		return "Starting…"
	case Running:
		if s.cfg.UseDocker {
			return fmt.Sprintf("Running · Docker container %s on port %d", scripting.PythonContainerName, s.cfg.Port)
		}
		return fmt.Sprintf("Running · server on port %d", s.cfg.Port)
	case Failed:
		return "Failed · " + s.err.Error()
	}
	return "Stopped"
}

// Execute runs script on the running executor.
func (s *Service) Execute(ctx context.Context, script string, params *scripting.ExecParams) (*scripting.ExecResult, error) {
	s.mu.Lock()
	exec, state, err := s.exec, s.state, s.err
	s.mu.Unlock()
	if exec == nil {
		switch state {
		case Starting:
			return nil, errors.New("script executor is still starting")
		case Failed:
			return nil, fmt.Errorf("script executor failed to start: %w; restart it from Settings > Scripting", err)
		}
		return nil, errors.New("script executor is not running; restart it from Settings > Scripting")
	}
	return exec.Execute(ctx, script, params)
}

func probePort(port int) error {
	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("nothing is listening on %s; start the script server first", addr)
	}
	return conn.Close()
}
