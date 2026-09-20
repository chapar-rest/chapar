package egress

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/notifications"
	"golang.org/x/sync/errgroup"
)

// Timeline step phase constants.
const (
	TimelinePhaseApp     = "app"
	TimelinePhaseNetwork = "network"
)

// TimelineStep is one timed stage of a request (app pipeline or network).
type TimelineStep struct {
	Name     string
	Phase    string // TimelinePhaseApp | TimelinePhaseNetwork
	Duration time.Duration
	Detail   string
	Err      string
}

type Response struct {
	// http and graphql
	StatusCode      int
	ResponseHeaders map[string]string
	RequestHeaders  map[string]string
	Cookies         []*http.Cookie

	// CookieEvents is what the cookie jar did with each Set-Cookie header,
	// including those on redirect hops. SentCookies are the jar cookies that
	// were attached to the request. Both are empty when no jar was used.
	CookieEvents []cookies.Event
	SentCookies  []*http.Cookie

	// grpc
	RequestMetadata  []domain.KeyValue
	ResponseMetadata []domain.KeyValue
	Trailers         []domain.KeyValue
	Size             int
	Error            error

	StatueCode int
	Status     string

	Body       []byte
	TimePassed time.Duration
	IsJSON     bool
	JSON       string

	// Pretty is a formatted body for display when available; Body stays raw.
	Pretty   string
	BodyKind string // util.BodyKindJSON | XML | HTML | Text

	Timeline []TimelineStep
}

type Sender interface {
	SendRequest(requestID, activeEnvironmentID string) (*Response, error)
}

type Service struct {
	requests     *state.Requests
	environments *state.Environments

	senders map[domain.RequestType]Sender

	scriptExecutor scripting.Executor
}

func New(requests *state.Requests, environments *state.Environments, rest, grpc, graphql Sender, scriptExecutor scripting.Executor) *Service {
	return &Service{
		requests:     requests,
		environments: environments,
		senders: map[domain.RequestType]Sender{
			domain.RequestTypeHTTP:    rest,
			domain.RequestTypeGRPC:    grpc,
			domain.RequestTypeGraphQL: graphql,
		},
		scriptExecutor: scriptExecutor,
	}
}

func (s *Service) SetExecutor(executor scripting.Executor) {
	s.scriptExecutor = executor
}

func (s *Service) Send(id, activeEnvironmentID string) (any, error) {
	req := s.requests.GetRequest(id)
	if req == nil {
		return nil, fmt.Errorf("request with id %s not found", id)
	}

	if err := s.preRequest(req, activeEnvironmentID); err != nil {
		return nil, err
	}

	var res *Response
	var err error

	sender, ok := s.senders[req.MetaData.Type]
	if !ok {
		return nil, fmt.Errorf("unknown request type: %s", req.MetaData.Type)
	}

	res, err = sender.SendRequest(req.MetaData.ID, activeEnvironmentID)
	if err != nil {
		return nil, err
	}

	var activeEnvironment *domain.Environment
	// Get environment if provided
	if activeEnvironmentID != "" {
		activeEnvironment = s.environments.GetEnvironment(activeEnvironmentID)
		if activeEnvironment == nil {
			return nil, fmt.Errorf("environment with id %s not found", activeEnvironmentID)
		}
	}

	if err := s.postRequest(req, res, activeEnvironment); err != nil {
		return nil, err
	}

	return res, err
}

func (s *Service) preRequest(req *domain.Request, activeEnvironmentID string) error {
	preReq := req.Spec.GetPreRequest()
	if !domain.DoablePreRequest(preReq) {
		return nil
	}

	// for now we only support trigger request
	if preReq.TriggerRequest == nil {
		return nil
	}

	_, err := s.Send(preReq.TriggerRequest.RequestID, activeEnvironmentID)
	return err
}

func (s *Service) postRequest(req *domain.Request, res *Response, env *domain.Environment) error {
	postReq := req.Spec.GetPostRequest()
	if !domain.DoablePostRequest(postReq) {
		return nil
	}

	// extract variables if any
	if err := s.extactVariables(req.Spec.GetVariables(), res, env); err != nil {
		return err
	}

	// if any script is provided, execute it
	if postReq.Script != "" {
		if err := s.executeScript(postReq.Script, req, res, env); err != nil {
			return err
		}
	}

	if err := s.handlePostRequestSetEnv(postReq, res, env); err != nil {
		return err
	}

	return nil
}

func (s *Service) extactVariables(settings []domain.Variable, response *Response, env *domain.Environment) error {
	if settings == nil || response == nil || env == nil {
		return nil
	}

	fn := func(v domain.Variable) error {
		if !v.Enable {
			return nil
		}

		if v.OnStatusCode != response.StatusCode {
			return nil
		}

		switch v.From {
		case domain.VariableFromBody:
			data, err := jsonpath.Get(response.JSON, v.JsonPath)
			if err != nil {
				return err
			}

			if data == nil {
				return nil
			}

			if result, ok := data.(string); ok {
				env.SetKey(v.TargetEnvVariable, result)
				if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
					return err
				}
			}

		case domain.VariableFromHeader:
			if result, ok := response.ResponseHeaders[v.SourceKey]; ok {
				env.SetKey(v.TargetEnvVariable, result)
				if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
					return err
				}
			}
		case domain.VariableFromCookies:
			for _, c := range response.Cookies {
				if c.Name == v.SourceKey {
					env.SetKey(v.TargetEnvVariable, c.Value)
					if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
						return err
					}
				}
			}

		case domain.VariableFromMetaData:
			for _, item := range response.ResponseMetadata {
				if item.Key == v.SourceKey {
					env.SetKey(v.TargetEnvVariable, item.Value)
					if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
						return err
					}
				}
			}
		case domain.VariableFromTrailers:
			for _, item := range response.Trailers {
				if item.Key == v.SourceKey {
					env.SetKey(v.TargetEnvVariable, item.Value)
					if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	errG := errgroup.Group{}
	for _, v := range settings {
		v := v
		errG.Go(func() error {
			return fn(v)
		})
	}

	return errG.Wait()
}

func (s *Service) handlePostRequestSetEnv(postReq domain.PostRequest, res *Response, env *domain.Environment) error {
	// handle set env if any
	// TODO: we need to give feedback to the user that the post request is not valid
	if postReq.Type != domain.PrePostTypeSetEnv || !postReq.PostRequestSet.IsValid() {
		return nil
	}

	// only handle post request if the status code is the same as the one provided
	if res.StatueCode != postReq.PostRequestSet.StatusCode {
		return nil
	}

	switch postReq.PostRequestSet.From {
	case domain.PostRequestSetFromResponseBody:
		return s.handlePostRequestFromBody(postReq, res, env)
	case domain.PostRequestSetFromResponseHeader:
		return s.handlePostRequestFromHeader(postReq, res, env)
	case domain.PostRequestSetFromResponseCookie:
		return s.handlePostRequestFromCookie(postReq, res, env)
	case domain.PostRequestSetFromResponseMetaData:
		return s.handlePostRequestFromMetaData(postReq, res, env)
	case domain.PostRequestSetFromResponseTrailers:
		return s.handlePostRequestFromTrailers(postReq, res, env)
	}
	return nil
}

func (s *Service) executeScript(script string, request *domain.Request, resp *Response, env *domain.Environment) error {
	if !prefs.GetGlobalConfig().Spec.Scripting.Enabled || s.scriptExecutor == nil {
		logger.Warn("Scripting is disabled, cannot execute script")
		notifications.Send("Scripting is disabled, cannot execute script", notifications.NotificationTypeError, time.Second*3)
		return nil
	}

	params := &scripting.ExecParams{
		Env: env,
		Req: scripting.RequestDataFromDomain(request),
		Res: &scripting.ResponseData{
			StatusCode: resp.StatusCode,
			Headers:    resp.ResponseHeaders,
			Body:       resp.JSON,
		},
	}

	result, err := s.scriptExecutor.Execute(context.Background(), script, params)
	if err != nil {
		return err
	}

	if env != nil {
		changed := false
		for k, v := range result.SetEnvironments {
			if data, ok := v.(string); ok {
				env.SetKey(k, data)
				changed = true
			}
		}

		if changed {
			if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
				return err
			}
		}
	} else if len(result.SetEnvironments) > 0 {
		// let user know that the environment is nil
		logger.Warn("No active environment, cannot set environment variables from script")
	}

	for _, pt := range result.Prints {
		logger.Print(pt)
	}

	return nil
}

func (s *Service) handlePostRequestFromBody(r domain.PostRequest, response *Response, env *domain.Environment) error {
	// handle post request
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseBody {
		return nil
	}

	if response.JSON == "" || !response.IsJSON {
		return nil

	}

	data, err := jsonpath.Get(response.JSON, r.PostRequestSet.FromKey)
	if err != nil {
		return err
	}

	if data == nil {
		return nil
	}

	if result, ok := data.(string); ok {
		if env != nil {
			env.SetKey(r.PostRequestSet.Target, result)
			return s.environments.UpdateEnvironment(env, state.SourceRestService, false)
		}
	}

	return nil
}

func (s *Service) handlePostRequestFromHeader(r domain.PostRequest, response *Response, env *domain.Environment) error {
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseHeader {
		return nil
	}

	if result, ok := response.ResponseHeaders[r.PostRequestSet.FromKey]; ok {
		if env != nil {
			env.SetKey(r.PostRequestSet.Target, result)

			if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) handlePostRequestFromCookie(r domain.PostRequest, response *Response, env *domain.Environment) error {
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseCookie {
		return nil
	}

	for _, c := range response.Cookies {
		if c.Name == r.PostRequestSet.FromKey {
			if env != nil {
				env.SetKey(r.PostRequestSet.Target, c.Value)
				return s.environments.UpdateEnvironment(env, state.SourceRestService, false)
			}
		}
	}
	return nil
}

func (s *Service) handlePostRequestFromMetaData(r domain.PostRequest, res *Response, env *domain.Environment) error {
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseMetaData {
		return nil
	}

	for _, item := range res.ResponseMetadata {
		if item.Key == r.PostRequestSet.FromKey {
			if env != nil {
				env.SetKey(r.PostRequestSet.Target, item.Value)
				return s.environments.UpdateEnvironment(env, state.SourceGRPCService, false)
			}
		}
	}

	return nil
}

func (s *Service) handlePostRequestFromTrailers(r domain.PostRequest, res *Response, env *domain.Environment) error {
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseTrailers {
		return nil
	}

	for _, item := range res.Trailers {
		if item.Key == r.PostRequestSet.FromKey {
			if env != nil {
				env.SetKey(r.PostRequestSet.Target, item.Value)
				return s.environments.UpdateEnvironment(env, state.SourceGRPCService, false)
			}
		}
	}

	return nil
}
