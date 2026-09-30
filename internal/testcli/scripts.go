package testcli

import (
	"context"
	"fmt"
	"sync"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/scripting"
)

// lazyScripts starts the script executor the first time a request script
// runs, so a run that uses none never pulls a Docker image or waits for a
// server. It is not shut down at exit: a Docker container is left running,
// as the app leaves it, since the app may be using the same one.
type lazyScripts struct {
	cfg  domain.ScriptingConfig
	once sync.Once
	exec scripting.Executor
	err  error
}

func (l *lazyScripts) Execute(ctx context.Context, script string, params *scripting.ExecParams) (*scripting.ExecResult, error) {
	l.once.Do(func() {
		exec, err := scripting.GetExecutor(l.cfg.Language, l.cfg)
		if err == nil {
			err = exec.Init(l.cfg)
		}
		if err != nil {
			l.err = fmt.Errorf("start the script executor: %w", err)
			return
		}
		l.exec = exec
	})
	if l.err != nil {
		return nil, l.err
	}
	return l.exec.Execute(ctx, script, params)
}
