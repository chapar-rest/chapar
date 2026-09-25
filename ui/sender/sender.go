package sender

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	graphqlsvc "github.com/chapar-rest/chapar/internal/egress/graphql"
	grpcsvc "github.com/chapar-rest/chapar/internal/egress/grpc"
	restsvc "github.com/chapar-rest/chapar/internal/egress/rest"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/scripting"
)

// RequestLookup finds a request by id without using internal/state.
type RequestLookup func(id string) *domain.Request

// CollectionLookup finds a collection by id without using internal/state.
type CollectionLookup func(id string) *domain.Collection

// Service sends requests from domain objects and persists env side effects via the repository.
type Service struct {
	rest    *restsvc.Service
	graphql *graphqlsvc.Service
	grpc    *grpcsvc.Service
	repo    repository.RepositoryV2
	lookup  RequestLookup
	colls   CollectionLookup
	onEnv   func(*domain.Environment)
	script  Scripts
	cookies *cookies.Store
	// scriptingOn reports whether scripting is enabled in settings;
	// replaced in tests.
	scriptingOn func() bool
}

// SetCookieStore makes HTTP and GraphQL requests use the cookie jar of the
// environment they are sent with.
func (s *Service) SetCookieStore(store *cookies.Store) {
	s.cookies = store
	s.rest.SetCookieStore(store)
	s.graphql.SetCookieStore(store)
}

// Cookies returns the cookie store, or nil when cookies are not handled.
func (s *Service) Cookies() *cookies.Store {
	return s.cookies
}

// Scripts runs pre/post-request scripts.
type Scripts interface {
	Execute(ctx context.Context, script string, params *scripting.ExecParams) (*scripting.ExecResult, error)
}

// SetExecutor wires the Python scripting executor for pre/post scripts.
func (s *Service) SetExecutor(exec Scripts) {
	s.script = exec
}

// New builds a sender that never reads internal/state.
func New(repo repository.RepositoryV2, lookup RequestLookup, colls CollectionLookup, onEnv func(*domain.Environment)) *Service {
	return &Service{
		rest:    &restsvc.Service{},
		graphql: &graphqlsvc.Service{},
		grpc:    grpcsvc.NewDirect(),
		repo:    repo,
		lookup:  lookup,
		colls:   colls,
		onEnv:   onEnv,
		scriptingOn: func() bool {
			return prefs.GetGlobalConfig().Spec.Scripting.Enabled
		},
	}
}

// Send runs pre-request, the protocol send, and post-request using the given documents.
//
// A pre-request script's changes to the request go into a copy made for
// this send; req itself is never changed.
func (s *Service) Send(req *domain.Request, env *domain.Environment) (*egress.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	var timeline []egress.TimelineStep

	var collection *domain.Collection
	if req.CollectionID != "" && s.colls != nil {
		collection = s.colls(req.CollectionID)
	}

	preStep, preResult, err := s.preRequestTimed(req, env, collection)
	if preStep != nil {
		timeline = append(timeline, *preStep)
	}
	if err != nil {
		return &egress.Response{Timeline: timeline, Error: err}, err
	}
	if preResult != nil && preResult.Skip {
		err := errors.New("the pre-request script skipped this request")
		if preResult.SkipReason != "" {
			err = fmt.Errorf("the pre-request script skipped this request: %s", preResult.SkipReason)
		}
		return &egress.Response{Timeline: timeline, Error: err}, err
	}

	sendReq := req
	if preResult != nil && !preResult.Request.Empty() {
		sendReq = req.Clone()
		sendReq.MetaData.ID = req.MetaData.ID
		scripting.ApplyChanges(sendReq, preResult.Request)
		if preResult.Request.Headers != nil && collection != nil {
			// The script saw the collection headers merged in and returned
			// the full set; merging them again would undo its removals.
			c := *collection
			c.Spec.Headers = nil
			collection = &c
		}
	}

	sendEnv := copyEnv(env)
	var res *egress.Response
	switch req.MetaData.Type {
	case domain.RequestTypeHTTP:
		res, err = s.rest.SendObject(sendReq, sendEnv, collection)
	case domain.RequestTypeGraphQL:
		res, err = s.graphql.SendObject(sendReq, sendEnv, collection)
	case domain.RequestTypeGRPC:
		res, err = s.grpc.SendObject(sendReq, sendEnv, collection)
	default:
		return nil, fmt.Errorf("unknown request type: %s", req.MetaData.Type)
	}
	s.saveCookies(env)
	if err != nil {
		if res == nil {
			res = &egress.Response{Error: err}
		}
		res.Timeline = append(timeline, res.Timeline...)
		return res, err
	}

	timeline = append(timeline, res.Timeline...)

	postStep, postErr := s.postRequestTimed(sendReq, res, env, collection)
	if postStep != nil {
		timeline = append(timeline, *postStep)
	}
	res.Timeline = timeline
	// The response arrived; a failing post-request action must not hide it.
	res.PostRequestError = postErr
	return res, nil
}

// LoadGRPCServices reflects or parses proto files for the given request.
func (s *Service) LoadGRPCServices(req *domain.Request, env *domain.Environment) ([]domain.GRPCService, error) {
	return s.grpc.GetServicesFrom(req, env)
}

// GRPCExampleBody returns an example JSON payload for the selected method.
func (s *Service) GRPCExampleBody(req *domain.Request, env *domain.Environment) (string, error) {
	return s.grpc.GetRequestStructFrom(req, env)
}

func (s *Service) preRequestTimed(req *domain.Request, env *domain.Environment, collection *domain.Collection) (*egress.TimelineStep, *scripting.ExecResult, error) {
	preReq := req.Spec.GetPreRequest()
	if !domain.DoablePreRequest(preReq) {
		return nil, nil, nil
	}
	start := time.Now()
	detail := ""
	var (
		result *scripting.ExecResult
		err    error
	)
	switch {
	case preReq.Type == domain.PrePostTypePython && preReq.Script != "":
		detail = "Python pre-request script"
		result, err = s.executeScript(scripting.PhasePre, preReq.Script, req, collection, nil, env)
		if err == nil && result != nil && result.ApplyEnv(env) {
			err = s.persistEnv(env)
		}
		detail = scriptDetail(detail, result)
	case preReq.TriggerRequest != nil && s.lookup != nil:
		triggered := s.lookup(preReq.TriggerRequest.RequestID)
		if triggered == nil {
			err = fmt.Errorf("trigger request %s not found", preReq.TriggerRequest.RequestID)
			detail = fmt.Sprintf("Trigger request %s", preReq.TriggerRequest.RequestID)
		} else {
			detail = fmt.Sprintf("Trigger request %s (%s)", triggered.MetaData.Name, triggered.MetaData.ID)
			var tres *egress.Response
			tres, err = s.Send(triggered, env)
			// Chained requests usually depend on what the trigger extracts.
			if err == nil && tres != nil && tres.PostRequestError != nil {
				err = tres.PostRequestError
			}
		}
	default:
		return nil, nil, nil
	}
	step := &egress.TimelineStep{
		Name:     "Pre-request",
		Phase:    egress.TimelinePhaseApp,
		Duration: time.Since(start),
		Detail:   detail,
	}
	if err != nil {
		step.Err = err.Error()
	} else if n := result.FailedTests(); n > 0 {
		step.Err = testsFailed(n)
	}
	return step, result, err
}

func (s *Service) postRequestTimed(req *domain.Request, res *egress.Response, env *domain.Environment, collection *domain.Collection) (*egress.TimelineStep, error) {
	if res == nil {
		return nil, nil
	}
	postReq := req.Spec.GetPostRequest()
	hasVars := len(req.Spec.GetVariables()) > 0
	if !domain.DoablePostRequest(postReq) && !hasVars {
		return nil, nil
	}

	start := time.Now()
	var details []string
	result, err := s.postRequest(req, collection, res, env)
	if postReq.Type == domain.PrePostTypePython && postReq.Script != "" {
		details = append(details, scriptDetail("Python post-request script", result))
	}
	if postReq.Type == domain.PrePostTypeSetEnv && postReq.PostRequestSet.IsValid() {
		details = append(details, fmt.Sprintf("Set env %s from %s", postReq.PostRequestSet.Target, postReq.PostRequestSet.From))
	}
	if hasVars {
		details = append(details, "Extract variables")
	}
	if len(details) == 0 {
		details = append(details, "Post-request processing")
	}
	step := &egress.TimelineStep{
		Name:     "Post-request",
		Phase:    egress.TimelinePhaseApp,
		Duration: time.Since(start),
		Detail:   strings.Join(details, "\n"),
	}
	if err != nil {
		step.Err = err.Error()
	} else if n := result.FailedTests(); n > 0 {
		step.Err = testsFailed(n)
	}
	return step, err
}

func scriptDetail(title string, result *scripting.ExecResult) string {
	if result == nil {
		return title
	}
	if summary := result.Summary(); summary != "" {
		return title + "\n" + summary
	}
	return title
}

func testsFailed(n int) string {
	if n == 1 {
		return "1 test failed"
	}
	return fmt.Sprintf("%d tests failed", n)
}

func (s *Service) postRequest(req *domain.Request, collection *domain.Collection, res *egress.Response, env *domain.Environment) (*scripting.ExecResult, error) {
	if res == nil {
		return nil, nil
	}
	postReq := req.Spec.GetPostRequest()
	if !domain.DoablePostRequest(postReq) {
		if env == nil {
			return nil, nil
		}
		extractErr := s.extractVariables(req.Spec, res, env)
		return nil, errors.Join(extractErr, s.persistEnv(env))
	}
	if postReq.Type == domain.PrePostTypePython && postReq.Script != "" {
		result, err := s.executeScript(scripting.PhasePost, postReq.Script, req, collection, res, env)
		if err != nil || env == nil || result == nil {
			return result, err
		}
		if !result.ApplyEnv(env) {
			return result, nil
		}
		return result, s.persistEnv(env)
	}
	if env == nil {
		return nil, nil
	}
	extractErr := s.extractVariables(req.Spec, res, env)
	if postReq.Type == domain.PrePostTypeSetEnv && postReq.PostRequestSet.IsValid() {
		code := res.StatusCode
		if code == 0 {
			code = res.StatueCode
		}
		if code == postReq.PostRequestSet.StatusCode {
			s.applyPostSet(postReq, res, env)
		}
	}
	return nil, errors.Join(extractErr, s.persistEnv(env))
}

func (s *Service) applyPostSet(postReq domain.PostRequest, res *egress.Response, env *domain.Environment) {
	switch postReq.PostRequestSet.From {
	case domain.PostRequestSetFromResponseBody:
		if res.JSON != "" && res.IsJSON {
			data, err := jsonpath.Get(res.JSON, postReq.PostRequestSet.FromKey)
			if err != nil {
				return
			}
			if result, ok := data.(string); ok {
				env.SetKey(postReq.PostRequestSet.Target, result)
			}
		}
	case domain.PostRequestSetFromResponseHeader:
		if result, ok := res.ResponseHeaders[postReq.PostRequestSet.FromKey]; ok {
			env.SetKey(postReq.PostRequestSet.Target, result)
		}
	case domain.PostRequestSetFromResponseCookie:
		for _, c := range res.Cookies {
			if c.Name == postReq.PostRequestSet.FromKey {
				env.SetKey(postReq.PostRequestSet.Target, c.Value)
			}
		}
	case domain.PostRequestSetFromResponseMetaData:
		for _, item := range res.ResponseMetadata {
			if item.Key == postReq.PostRequestSet.FromKey {
				env.SetKey(postReq.PostRequestSet.Target, item.Value)
			}
		}
	case domain.PostRequestSetFromResponseTrailers:
		for _, item := range res.Trailers {
			if item.Key == postReq.PostRequestSet.FromKey {
				env.SetKey(postReq.PostRequestSet.Target, item.Value)
			}
		}
	}
}

func (s *Service) extractVariables(spec domain.RequestSpec, res *egress.Response, env *domain.Environment) error {
	settings := spec.GetVariables()
	var errs []error
	code := res.StatusCode
	if code == 0 {
		code = res.StatueCode
	}
	for _, v := range settings {
		if !v.Enable {
			continue
		}
		if v.OnStatusCode != 0 && v.OnStatusCode != code {
			continue
		}
		switch v.From {
		case domain.VariableFromBody:
			data, err := jsonpath.Get(res.JSON, v.JsonPath)
			if err != nil {
				// Keep going so one bad path does not block the other rules.
				errs = append(errs, fmt.Errorf("extract %s from %s: %w", v.TargetEnvVariable, v.JsonPath, err))
				continue
			}
			if result, ok := data.(string); ok {
				env.SetKey(v.TargetEnvVariable, result)
			}
		case domain.VariableFromHeader:
			if result, ok := res.ResponseHeaders[v.SourceKey]; ok {
				env.SetKey(v.TargetEnvVariable, result)
			}
		case domain.VariableFromCookies:
			for _, c := range res.Cookies {
				if c.Name == v.SourceKey {
					env.SetKey(v.TargetEnvVariable, c.Value)
				}
			}
		case domain.VariableFromMetaData:
			for _, item := range res.ResponseMetadata {
				if item.Key == v.SourceKey {
					env.SetKey(v.TargetEnvVariable, item.Value)
				}
			}
		case domain.VariableFromTrailers:
			for _, item := range res.Trailers {
				if item.Key == v.SourceKey {
					env.SetKey(v.TargetEnvVariable, item.Value)
				}
			}
		}
	}
	return errors.Join(errs...)
}

// ErrScriptingDisabled is reported when a request has a script but
// scripting is turned off in settings.
var ErrScriptingDisabled = errors.New("scripting is disabled; turn it on in Settings > Scripting")

// executeScript runs a pre- or post-request script. It does not touch env:
// the caller applies result's env changes. A script that raised returns its
// result (prints, tests) along with the error.
func (s *Service) executeScript(phase scripting.Phase, script string, req *domain.Request, collection *domain.Collection, resp *egress.Response, env *domain.Environment) (*scripting.ExecResult, error) {
	if !s.scriptingOn() || s.script == nil {
		logger.Warn("Scripting is disabled, cannot execute script")
		return nil, ErrScriptingDisabled
	}
	params := &scripting.ExecParams{
		Phase:    phase,
		Protocol: scripting.ProtocolOf(req.MetaData.Type),
		Env:      env,
		Req:      scripting.RequestDataFromDomain(req, env, collection),
		Res:      resp.ScriptData(),
	}
	result, err := s.script.Execute(context.Background(), script, params)
	if err != nil {
		return nil, err
	}
	if summary := result.Summary(); summary != "" {
		logger.Print(summary)
	}
	if result.Error != nil {
		if result.Error.Traceback != "" {
			logger.Print(result.Error.Traceback)
		}
		return result, result.Error
	}
	return result, nil
}

func (s *Service) persistEnv(env *domain.Environment) error {
	if env == nil || s.repo == nil {
		return nil
	}
	if err := s.repo.UpdateEnvironment(env); err != nil {
		return err
	}
	if s.onEnv != nil {
		s.onEnv(env)
	}
	return nil
}

func copyEnv(e *domain.Environment) *domain.Environment {
	if e == nil {
		return nil
	}
	return &domain.Environment{
		ApiVersion: e.ApiVersion,
		Kind:       e.Kind,
		MetaData:   e.MetaData,
		Spec:       e.Spec.Clone(),
	}
}

func (s *Service) saveCookies(env *domain.Environment) {
	if s.cookies == nil {
		return
	}
	envID := ""
	if env != nil {
		envID = env.ID()
	}
	if err := s.cookies.Save(envID); err != nil {
		logger.Error(fmt.Sprintf("failed to save cookies: %v", err))
	}
}
