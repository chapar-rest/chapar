package scriptsrv

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/scripting"
)

type fakeExec struct {
	mu       sync.Mutex
	inits    int
	shutdown int
	initErr  error
}

func (f *fakeExec) Init(domain.ScriptingConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inits++
	return f.initErr
}

func (f *fakeExec) Execute(context.Context, string, *scripting.ExecParams) (*scripting.ExecResult, error) {
	return &scripting.ExecResult{Prints: []string{"ok"}}, nil
}

func (f *fakeExec) Name() string { return "Fake" }

func (f *fakeExec) Shutdown() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shutdown++
	return nil
}

func newTestService(exec *fakeExec, probeErr error) *Service {
	return &Service{
		newExecutor: func(domain.ScriptingConfig) (scripting.Executor, error) { return exec, nil },
		probe:       func(int) error { return probeErr },
	}
}

func waitState(t *testing.T, s *Service, want State) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st, _ := s.State(); st == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	st, err := s.State()
	t.Fatalf("state = %v (err %v), want %v", st, err, want)
}

var dockerCfg = domain.ScriptingConfig{Enabled: true, Language: "python", UseDocker: true, DockerImage: "img", Port: 2397}

// Restarting a running executor has to start a fresh one: a Docker executor
// keeps a container it finds running, so the old one is shut down first.
func TestRestartReplacesRunningExecutor(t *testing.T) {
	exec := &fakeExec{}
	s := newTestService(exec, nil)

	s.Restart(dockerCfg)
	waitState(t, s, Running)
	if got := s.Status(); !strings.HasPrefix(got, "Running · Docker container") {
		t.Fatalf("Status = %q", got)
	}

	s.run(dockerCfg)
	exec.mu.Lock()
	defer exec.mu.Unlock()
	if exec.inits != 2 {
		t.Fatalf("inits = %d, want 2", exec.inits)
	}
	// stale container cleared before each start, plus the running one stopped
	if exec.shutdown != 3 {
		t.Fatalf("shutdowns = %d, want 3", exec.shutdown)
	}
}

func TestFailedStartIsReportedAndRestartable(t *testing.T) {
	exec := &fakeExec{initErr: errors.New("docker is not running")}
	s := newTestService(exec, nil)

	s.Restart(dockerCfg)
	waitState(t, s, Failed)
	if got := s.Status(); got != "Failed · docker is not running" {
		t.Fatalf("Status = %q", got)
	}
	if _, err := s.Execute(context.Background(), "print(1)", &scripting.ExecParams{}); err == nil ||
		!strings.Contains(err.Error(), "docker is not running") {
		t.Fatalf("Execute err = %v", err)
	}

	exec.mu.Lock()
	exec.initErr = nil
	exec.mu.Unlock()
	s.run(dockerCfg)
	res, err := s.Execute(context.Background(), "print(1)", &scripting.ExecParams{})
	if err != nil || len(res.Prints) != 1 {
		t.Fatalf("Execute = %v, %v", res, err)
	}
}

func TestDisablingStops(t *testing.T) {
	exec := &fakeExec{}
	s := newTestService(exec, nil)
	s.Restart(dockerCfg)
	waitState(t, s, Running)

	off := dockerCfg
	off.Enabled = false
	s.run(off)
	if st, _ := s.State(); st != Stopped {
		t.Fatalf("state = %v, want Stopped", st)
	}
	if _, err := s.Execute(context.Background(), "", &scripting.ExecParams{}); err == nil {
		t.Fatal("Execute ran on a stopped executor")
	}
}

// Without Docker Chapar starts nothing, so a server that is not listening is
// a failed start, not a running one.
func TestLocalServerMustListen(t *testing.T) {
	s := newTestService(&fakeExec{}, errors.New("nothing is listening on localhost:2397"))
	local := dockerCfg
	local.UseDocker = false
	s.Restart(local)
	waitState(t, s, Failed)
}
