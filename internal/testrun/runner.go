package testrun

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/sender"
)

// Sender sends one request with an environment. *sender.Service is one.
type Sender interface {
	Send(req *domain.Request, env *domain.Environment) (*egress.Response, error)
}

// NewSender returns a factory of senders for test runs. Each one keeps
// cookies in memory for its run and never saves the environment.
// scriptingOn decides whether request scripts run; nil follows the
// scripting setting, as the app does.
func NewSender(requests sender.RequestLookup, colls sender.CollectionLookup, scripts sender.Scripts, scriptingOn func() bool) func() Sender {
	return func() Sender {
		s := sender.New(nil, requests, colls, nil)
		s.SetExecutor(scripts)
		s.SetCookieStore(cookies.NewMemoryStore())
		if scriptingOn != nil {
			s.SetScriptingEnabled(scriptingOn)
		}
		return s
	}
}

type Config struct {
	// NewSender returns the sender for one run, see NewSender.
	NewSender func() Sender
	Requests  RequestSource
	// SaveEnv writes an environment back for cases with persistEnv; nil
	// ignores persistEnv.
	SaveEnv func(env *domain.Environment) error
	// Getenv reads OS environment variables; nil means os.Getenv.
	Getenv func(key string) string
}

// Runner runs test cases. It is safe to run several cases at once.
type Runner struct {
	cfg Config
}

func New(cfg Config) *Runner {
	if cfg.Getenv == nil {
		cfg.Getenv = os.Getenv
	}
	return &Runner{cfg: cfg}
}

type Options struct {
	// Env is the environment to run with, with secrets already resolved.
	// It is copied and never changed; nil runs without one.
	Env *domain.Environment
	// Only runs just these steps (by id) of the steps section; setup and
	// teardown still run. Empty runs all.
	Only []string
	// OnEvent reports progress. It is called on the run's goroutine.
	OnEvent func(Event)
}

// Run runs tc and returns its result. Cancelling ctx stops the run after
// the current send; teardown still runs.
func (r *Runner) Run(ctx context.Context, tc *domain.TestCase, o Options) *Run {
	run := &Run{
		ID:       uuid.NewString(),
		CaseID:   tc.ID(),
		CaseName: tc.GetName(),
		Status:   StatusRunning,
		Started:  time.Now(),
	}
	if o.Env != nil {
		run.EnvName = o.Env.GetName()
	}

	x := &execution{
		Runner: r,
		tc:     tc,
		opts:   o,
		run:    run,
		scope:  newScope(o.Env),
		send:   r.cfg.NewSender(),
	}
	for _, v := range tc.Spec.Variables {
		value := v.Value
		if v.From != nil {
			value = r.cfg.Getenv(v.From.OsEnv)
		}
		x.scope.vars[v.Key] = x.scope.expand(value, nil)
	}

	only := map[string]bool{}
	for _, id := range o.Only {
		only[id] = true
	}

	stop := ""
	for _, s := range tc.Spec.Setup {
		stop = x.step(ctx, SectionSetup, s, stop)
		if stop == "" && failed(x.last()) {
			stop = "setup step " + s.ID + " did not pass"
		}
	}
	for _, s := range tc.Spec.Steps {
		if len(only) > 0 && !only[s.ID] && stop == "" {
			x.skip(SectionSteps, s, "not selected")
			continue
		}
		stop = x.step(ctx, SectionSteps, s, stop)
		if stop == "" && failed(x.last()) &&
			!s.ContinueOnFailure && !tc.Spec.Options.ContinueOnFailure {
			stop = "step " + s.ID + " did not pass"
		}
	}
	// Teardown cleans up after whatever ran, so it outlives a cancel.
	teardownCtx := context.WithoutCancel(ctx)
	for _, s := range tc.Spec.Teardown {
		x.step(teardownCtx, SectionTeardown, s, "")
	}

	if tc.Spec.Options.PersistEnv && o.Env != nil && r.cfg.SaveEnv != nil && x.scope.changed() {
		env := copyEnv(o.Env)
		x.scope.applyChanges(env)
		if err := r.cfg.SaveEnv(env); err != nil {
			run.Message = "save environment: " + err.Error()
		}
	}

	run.Duration = time.Since(run.Started)
	run.Status = runStatus(ctx, run)
	if o.OnEvent != nil {
		o.OnEvent(Event{Kind: EventRunFinished, Run: run})
	}
	return run
}

// failed reports whether a step that ran stops the run: anything but a
// pass, or a skip of a disabled step.
func failed(s *StepResult) bool {
	return s.Status != StatusPassed && s.Status != StatusSkipped
}

func runStatus(ctx context.Context, run *Run) Status {
	if ctx.Err() != nil {
		return StatusCancelled
	}
	status := StatusPassed
	for _, s := range run.Steps {
		switch s.Status {
		case StatusError, StatusCancelled:
			return StatusError
		case StatusFailed:
			status = StatusFailed
		}
	}
	if status == StatusPassed && run.Message != "" {
		return StatusError
	}
	return status
}

// execution is the state of one Run call.
type execution struct {
	*Runner
	tc    *domain.TestCase
	opts  Options
	run   *Run
	scope *scope
	send  Sender
}

func (x *execution) last() *StepResult {
	return &x.run.Steps[len(x.run.Steps)-1]
}

func (x *execution) emit(kind EventKind, s *StepResult) {
	if x.opts.OnEvent != nil {
		c := *s
		x.opts.OnEvent(Event{Kind: kind, Step: &c})
	}
}

func (x *execution) skip(section string, s domain.TestStep, why string) {
	res := StepResult{Section: section, StepID: s.ID, Name: stepName(s), Status: StatusSkipped, Message: why}
	x.run.Steps = append(x.run.Steps, res)
	x.emit(EventStepFinished, &res)
}

// step runs s unless the run is stopping, and returns the reason the run
// is stopping after it, if any.
func (x *execution) step(ctx context.Context, section string, s domain.TestStep, stop string) string {
	if stop != "" {
		x.skip(section, s, stop)
		return stop
	}
	if s.Disabled {
		x.skip(section, s, "disabled")
		return ""
	}
	if ctx.Err() != nil {
		x.skip(section, s, "run cancelled")
		return "run cancelled"
	}

	res := StepResult{Section: section, StepID: s.ID, Name: stepName(s), Status: StatusRunning}
	x.emit(EventStepStarted, &res)
	x.runStep(ctx, s, &res)
	x.run.Steps = append(x.run.Steps, res)
	x.emit(EventStepFinished, &res)

	if res.Status == StatusCancelled {
		return "run cancelled"
	}
	return ""
}

func stepName(s domain.TestStep) string {
	if s.Name != "" {
		return s.Name
	}
	if s.Request.Ref != "" {
		return s.Request.Ref
	}
	return s.ID
}

func (x *execution) runStep(ctx context.Context, s domain.TestStep, res *StepResult) {
	start := time.Now()
	defer func() { res.Duration = time.Since(start) }()

	req, err := Resolve(x.cfg.Requests, s.Request)
	if err != nil {
		res.Status, res.Message = StatusError, err.Error()
		return
	}
	res.Request = summarizeRequest(req)

	timeout := s.Timeout
	if timeout == 0 {
		timeout = x.tc.Spec.Options.Timeout
	}
	attempts, delay := 1, time.Duration(0)
	if s.Retry != nil {
		attempts, delay = 1+s.Retry.Count, s.Retry.Delay
	}

	var resp *egress.Response
	for i := 0; i < attempts; i++ {
		if i > 0 && !sleep(ctx, delay) {
			res.Status, res.Message = StatusCancelled, "run cancelled"
			return
		}
		res.Attempts = i + 1
		resp = x.attempt(ctx, req, s, timeout, res)
		if res.Status == StatusPassed || res.Status == StatusCancelled {
			break
		}
	}
	if resp == nil || res.Status == StatusError || res.Status == StatusCancelled {
		return
	}

	captures, vars := Capture(resp, s.Capture)
	res.Captures = captures
	for k, v := range vars {
		x.scope.vars[k] = v
	}
	for _, c := range captures {
		if c.Error != "" && res.Status == StatusPassed {
			res.Status, res.Message = StatusFailed, fmt.Sprintf("capture %s: %s", c.Var, c.Error)
		}
	}
}

// attempt sends the step's request once and records the outcome in res.
func (x *execution) attempt(ctx context.Context, saved *domain.Request, s domain.TestStep, timeout time.Duration, res *StepResult) *egress.Response {
	req := saved.Clone()
	req.MetaData.ID = saved.MetaData.ID
	applyOverrides(req, s.With)

	var with map[string]string
	if s.With != nil {
		with = x.scope.stepVars(s.With.Variables)
	}
	env := x.scope.stepEnv(with)
	before := snapshot(env)

	resp, err := x.sendWithin(ctx, req, env, timeout)
	res.Assertions, res.Response, res.Message = nil, nil, ""
	switch {
	case errors.Is(err, context.Canceled):
		res.Status, res.Message = StatusCancelled, "run cancelled"
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		res.Status, res.Message = StatusError, fmt.Sprintf("no response within %v", timeout)
		return nil
	}

	// The send finished, so env holds what its scripts and extract rules
	// did, even when it failed.
	x.scope.merge(before, env)

	if resp != nil {
		res.Assertions = ScriptAssertions(resp.ScriptTests)
	}
	if err != nil {
		res.Status, res.Message = StatusError, err.Error()
		return nil
	}

	res.Response = summarizeResponse(resp)
	asserts := make([]domain.TestAssertion, len(s.Assert))
	for i, a := range s.Assert {
		a.Value = x.scope.expandValue(a.Value, with)
		asserts[i] = a
	}
	res.Assertions = append(Evaluate(resp, asserts), res.Assertions...)
	failed := 0
	for _, a := range res.Assertions {
		if !a.Passed {
			failed++
		}
	}
	switch {
	case resp.PostRequestError != nil:
		res.Status, res.Message = StatusError, "post-request: "+resp.PostRequestError.Error()
	case failed > 0:
		res.Status, res.Message = StatusFailed, fmt.Sprintf("%d of %d assertions failed", failed, len(res.Assertions))
	default:
		res.Status = StatusPassed
	}
	return resp
}

// sendWithin sends req, giving up when ctx ends or timeout passes. The
// send cannot be interrupted, so a send given up on finishes on its own;
// it only touches its own copies of the request and env.
func (x *execution) sendWithin(ctx context.Context, req *domain.Request, env *domain.Environment, timeout time.Duration) (*egress.Response, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	type result struct {
		res *egress.Response
		err error
	}
	done := make(chan result, 1)
	go func() {
		res, err := x.send.Send(req, env)
		done <- result{res, err}
	}()
	select {
	case r := <-done:
		return r.res, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sleep waits d and reports whether ctx is still going.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func summarizeRequest(req *domain.Request) *RequestSummary {
	out := &RequestSummary{ID: req.ID(), Name: RefOf(req), Protocol: string(req.MetaData.Type)}
	switch {
	case req.Spec.HTTP != nil:
		out.Method, out.URL = req.Spec.HTTP.Method, req.Spec.HTTP.URL
	case req.Spec.GraphQL != nil:
		out.URL = req.Spec.GraphQL.URL
	case req.Spec.GRPC != nil:
		out.Method, out.URL = req.Spec.GRPC.LasSelectedMethod, req.Spec.GRPC.ServerInfo.Address
	}
	return out
}

func summarizeResponse(res *egress.Response) *ResponseSummary {
	status := res.StatusCode
	if status == 0 {
		status = res.StatueCode
	}
	out := &ResponseSummary{
		Status:  status,
		Size:    responseSize(res),
		Time:    res.TimePassed,
		Headers: res.ResponseHeaders,
		Body:    string(res.Body),
	}
	if len(out.Body) > maxBody {
		out.Body, out.Truncated = out.Body[:maxBody], true
	}
	return out
}
