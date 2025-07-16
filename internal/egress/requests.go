package egress

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/rest"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/notifications"
)

type Service struct {
	requests     *state.Requests
	environments *state.Environments

	rest *rest.Service
	grpc *grpc.Service

	scriptExecutor scripting.Executor
}

func New(requests *state.Requests, environments *state.Environments, rest *rest.Service, grpc *grpc.Service, scriptExecutor scripting.Executor) *Service {
	return &Service{
		requests:       requests,
		environments:   environments,
		rest:           rest,
		grpc:           grpc,
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

	var res any
	var err error
	if req.MetaData.Type == domain.RequestTypeHTTP {
		res, err = s.rest.SendRequest(req.MetaData.ID, activeEnvironmentID)
	} else {
		res, err = s.grpc.Invoke(req.MetaData.ID, activeEnvironmentID)
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
	var preReq domain.PreRequest
	if req.MetaData.Type == domain.RequestTypeHTTP {
		preReq = req.Spec.GetHTTP().GetPreRequest()
	} else {
		preReq = req.Spec.GetGRPC().GetPreRequest()
	}

	if preReq == (domain.PreRequest{}) || preReq.TriggerRequest == nil || preReq.TriggerRequest.RequestID == "none" {
		return nil
	}

	if preReq.Type != domain.PrePostTypeTriggerRequest {
		// TODO: implement other types
		return nil
	}

	_, err := s.Send(preReq.TriggerRequest.RequestID, activeEnvironmentID)
	return err
}

func (s *Service) postRequest(req *domain.Request, res any, env *domain.Environment) error {
	if req.MetaData.Type == domain.RequestTypeHTTP {
		postReq := req.Spec.GetHTTP().GetPostRequest()
		if response, ok := res.(*rest.Response); ok {
			// handle variables
			// TODO handling variables does not seem to be good fit here in post request
			if err := s.handleHTTPVariables(req.Spec.GetHTTP().Request.Variables, response, env); err != nil {
				return err
			}

			return s.handleHTTPPostRequest(postReq, req, response, env)
		} else {
			return fmt.Errorf("response is not of type *rest.Response")
		}
	}

	postReq := req.Spec.GetGRPC().GetPostRequest()
	if response, ok := res.(*grpc.Response); ok {
		if err := s.handleGRPcVariables(req.Spec.GetGRPC().Variables, response, env); err != nil {
			return err
		}

		return s.handleGRPCPostRequest(postReq, response, env)
	}

	return fmt.Errorf("response is not of type *grpc.Response")
}

func (s *Service) handleHTTPVariables(variables []domain.Variable, response *rest.Response, env *domain.Environment) error {
	if variables == nil || response == nil || env == nil {
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
		}

		return nil
	}

	errG := errgroup.Group{}
	for _, v := range variables {
		v := v
		errG.Go(func() error {
			return fn(v)
		})
	}

	return errG.Wait()
}

func (s *Service) handleGRPcVariables(variables []domain.Variable, response *grpc.Response, env *domain.Environment) error {
	if variables == nil || response == nil || env == nil {
		return nil
	}

	fn := func(v domain.Variable) error {
		if !v.Enable {
			return nil
		}

		if v.OnStatusCode != response.StatueCode {
			return nil
		}

		switch v.From {
		case domain.VariableFromBody:
			data, err := jsonpath.Get(response.Body, v.JsonPath)
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

		case domain.VariableFromMetaData:
			for _, item := range response.ResponseMetadata {
				if item.Key == v.SourceKey {
					env.SetKey(v.TargetEnvVariable, item.Value)
					if err := s.environments.UpdateEnvironment(env, state.SourceGRPCService, false); err != nil {
						return err
					}
				}
			}
		case domain.VariableFromTrailers:
			for _, item := range response.Trailers {
				if item.Key == v.SourceKey {
					env.SetKey(v.TargetEnvVariable, item.Value)
					if err := s.environments.UpdateEnvironment(env, state.SourceGRPCService, false); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	errG := errgroup.Group{}
	for _, v := range variables {
		v := v
		errG.Go(func() error {
			return fn(v)
		})
	}

	return errG.Wait()
}

func (s *Service) handleHTTPPostRequest(r domain.PostRequest, request *domain.Request, response *rest.Response, env *domain.Environment) error {
	if r == (domain.PostRequest{}) || response == nil {
		return nil
	}

	// TODO: this is a temporary solution to fix crashes when scripting is disabled. it need to be improved
	if r.Type == domain.PrePostTypePython && prefs.GetGlobalConfig().Spec.Scripting.Enabled {
		return s.handlePostRequestScript(r.Script, request, response, env)
	}

	if r.Type != domain.PrePostTypeSetEnv {
		// TODO: implement other types
		return nil
	}

	if env == nil {
		logger.Warn("No active environment, cannot handle post request set environment")
		return nil
	}

	// only handle post request if the status code is the same as the one provided
	if response.StatusCode != r.PostRequestSet.StatusCode {
		return nil
	}

	switch r.PostRequestSet.From {
	case domain.PostRequestSetFromResponseBody:
		return s.handlePostRequestFromBody(r, response, env)
	case domain.PostRequestSetFromResponseHeader:
		return s.handlePostRequestFromHeader(r, response, env)
	case domain.PostRequestSetFromResponseCookie:
		return s.handlePostRequestFromCookie(r, response, env)
	}

	return nil
}

func (s *Service) handlePostRequestScript(script string, request *domain.Request, resp *rest.Response, env *domain.Environment) error {
	// if script executor is not set, return error
	if s.scriptExecutor == nil {
		logger.Warn("script executor is not enable or not ready yet, cannot handle post request script")
		notifications.Send("Failed to run post request script, check console for logs", notifications.NotificationTypeError, time.Second*3)
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

func (s *Service) handlePostRequestFromBody(r domain.PostRequest, response *rest.Response, env *domain.Environment) error {
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

func (s *Service) handlePostRequestFromHeader(r domain.PostRequest, response *rest.Response, env *domain.Environment) error {
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

func (s *Service) handlePostRequestFromCookie(r domain.PostRequest, response *rest.Response, env *domain.Environment) error {
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

func (s *Service) handleGRPCPostRequest(r domain.PostRequest, res *grpc.Response, env *domain.Environment) error {
	if r == (domain.PostRequest{}) || res == nil || env == nil {
		return nil
	}

	if r.Type != domain.PrePostTypeSetEnv {
		return nil
	}

	// only handle post request if the status code is the same as the one provided
	if res.StatueCode != r.PostRequestSet.StatusCode {
		return nil
	}

	switch r.PostRequestSet.From {
	case domain.PostRequestSetFromResponseBody:
		return s.handleGRPCPostRequestFromBody(r, res, env)
	case domain.PostRequestSetFromResponseMetaData:
		return s.handlePostRequestFromMetaData(r, res, env)
	case domain.PostRequestSetFromResponseTrailers:
		return s.handlePostRequestFromTrailers(r, res, env)
	}

	return nil
}

func (s *Service) handleGRPCPostRequestFromBody(r domain.PostRequest, res *grpc.Response, env *domain.Environment) error {
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseBody {
		return nil
	}

	if res.Body == "" {
		return nil
	}

	data, err := jsonpath.Get(res.Body, r.PostRequestSet.FromKey)
	if err != nil {
		return err
	}

	if data == nil {
		return nil
	}

	if result, ok := data.(string); ok {
		if env != nil {
			env.SetKey(r.PostRequestSet.Target, result)

			if err := s.environments.UpdateEnvironment(env, state.SourceGRPCService, false); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) handlePostRequestFromMetaData(r domain.PostRequest, res *grpc.Response, env *domain.Environment) error {
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

func (s *Service) handlePostRequestFromTrailers(r domain.PostRequest, res *grpc.Response, env *domain.Environment) error {
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
