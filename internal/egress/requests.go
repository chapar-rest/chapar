package egress

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/internal/rest"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/state"
)

type ScriptRunner interface {
	// ExecutePreRequestScript runs a script before the request is sent
	// and potentially modifies the request data
	ExecutePreRequestScript(ctx context.Context, script string, params *scripting.ExecParams) (*scripting.ExecResult, error)

	// ExecutePostResponseScript runs a script after a response is received
	// and can access both request and response data
	ExecutePostResponseScript(ctx context.Context, script string, params *scripting.ExecParams) (*scripting.ExecResult, error)
}

type Service struct {
	requests     *state.Requests
	environments *state.Environments

	rest *rest.Service
	grpc *grpc.Service

	scriptPluginManager scripting.PluginManager
}

func New(requests *state.Requests, environments *state.Environments, rest *rest.Service, grpc *grpc.Service, scriptPluginManager scripting.PluginManager) *Service {
	return &Service{
		requests:            requests,
		environments:        environments,
		rest:                rest,
		grpc:                grpc,
		scriptPluginManager: scriptPluginManager,
	}
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

			return s.handleHTTPPostRequest(postReq, response, env)
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

func (s *Service) handleHTTPPostRequest(r domain.PostRequest, response *rest.Response, env *domain.Environment) error {
	if r == (domain.PostRequest{}) || response == nil || env == nil {
		return nil
	}

	if r.Type == domain.PrePostTypePython {
		return s.handlePythonPostRequest(r.Script, response, env)
	}

	if r.Type != domain.PrePostTypeSetEnv {
		// TODO: implement other types
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

func (s *Service) handlePythonPreRequest(script string, request *domain.Request, env *domain.Environment) error {
	params := &scripting.ExecParams{
		Env: env,
	}

	if request.MetaData.Type == domain.RequestTypeHTTP {
		params.Req = &scripting.RequestData{
			Method:      request.Spec.HTTP.Method,
			URL:         request.Spec.HTTP.URL,
			Headers:     domain.KeyValueToMap(request.Spec.HTTP.Request.Headers),
			QueryParams: domain.KeyValueToMap(request.Spec.HTTP.Request.QueryParams),
			PathParams:  domain.KeyValueToMap(request.Spec.HTTP.Request.PathParams),
			Body:        request.Spec.HTTP.Request.Body.Data,
		}
	} else {
		params.Req = &scripting.RequestData{
			Method:   request.Spec.GRPC.LasSelectedMethod,
			URL:      request.Spec.GRPC.ServerInfo.Address,
			Metadata: domain.KeyValueToMap(request.Spec.GRPC.Metadata),
		}
	}

	runner, _ := s.scriptPluginManager.GetPlugin("python")

	result, err := runner.ExecutePreRequestScript(context.Background(), script, params)
	if err != nil {
		return err
	}

	for k, v := range result.SetEnvironments {
		if data, ok := v.(string); ok {
			if env != nil {
				env.SetKey(k, data)
				return s.environments.UpdateEnvironment(env, state.SourceRestService, false)
			}
		}
	}

	updateRequestFromScriptResult(request, result)

	return nil
}

func updateRequestFromScriptResult(req *domain.Request, result *scripting.ExecResult) {
	if req.MetaData.Type == domain.RequestTypeHTTP {
		req.Spec.HTTP.Method = result.Req.Method
		req.Spec.HTTP.URL = result.Req.URL
		req.Spec.HTTP.Request.Body.Data = result.Req.Body
		req.Spec.HTTP.Request.Headers = domain.MapToKeyValue(result.Req.Headers)
		req.Spec.HTTP.Request.QueryParams = domain.MapToKeyValue(result.Req.QueryParams)
		req.Spec.HTTP.Request.PathParams = domain.MapToKeyValue(result.Req.PathParams)
	} else {
		req.Spec.GRPC.LasSelectedMethod = result.Req.Method
		req.Spec.GRPC.ServerInfo.Address = result.Req.URL
		req.Spec.GRPC.Metadata = domain.MapToKeyValue(result.Req.Metadata)
	}
}

func (s *Service) handlePythonPostRequest(script string, response *rest.Response, env *domain.Environment) error {
	params := &scripting.ExecParams{
		Env: env,
		Res: &scripting.ResponseData{
			StatusCode: response.StatusCode,
			Headers:    response.ResponseHeaders,
			Body:       response.JSON,
		},
	}

	runner, _ := s.scriptPluginManager.GetPlugin("python")

	result, err := runner.ExecutePostResponseScript(context.Background(), script, params)
	if err != nil {
		return err
	}

	for k, v := range result.SetEnvironments {
		if data, ok := v.(string); ok {
			if env != nil {
				env.SetKey(k, data)
				return s.environments.UpdateEnvironment(env, state.SourceRestService, false)
			}
		}
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
