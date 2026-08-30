package sender

import (
	"context"
	"fmt"

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
	proto   func() []*domain.ProtoFile
	onEnv   func(*domain.Environment)
	script  scripting.Executor
}

// SetExecutor wires the Python scripting executor for pre/post scripts.
func (s *Service) SetExecutor(exec scripting.Executor) {
	s.script = exec
}

// New builds a sender that never reads internal/state.
func New(repo repository.RepositoryV2, lookup RequestLookup, colls CollectionLookup, proto func() []*domain.ProtoFile, onEnv func(*domain.Environment)) *Service {
	return &Service{
		rest:    &restsvc.Service{},
		graphql: &graphqlsvc.Service{},
		grpc:    grpcsvc.NewDirect(),
		repo:    repo,
		lookup:  lookup,
		colls:   colls,
		proto:   proto,
		onEnv:   onEnv,
	}
}

// Send runs pre-request, the protocol send, and post-request using the given documents.
func (s *Service) Send(req *domain.Request, env *domain.Environment) (*egress.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if err := s.preRequest(req, env); err != nil {
		return nil, err
	}

	var collection *domain.Collection
	if req.CollectionID != "" && s.colls != nil {
		collection = s.colls(req.CollectionID)
	}

	sendEnv := copyEnv(env)
	var (
		res *egress.Response
		err error
	)
	switch req.MetaData.Type {
	case domain.RequestTypeHTTP:
		res, err = s.rest.SendObject(req, sendEnv, collection)
	case domain.RequestTypeGraphQL:
		res, err = s.graphql.SendObject(req, sendEnv, collection)
	case domain.RequestTypeGRPC:
		var protoFiles []*domain.ProtoFile
		if s.proto != nil {
			protoFiles = s.proto()
		}
		res, err = s.grpc.SendObject(req, sendEnv, collection, protoFiles)
	default:
		return nil, fmt.Errorf("unknown request type: %s", req.MetaData.Type)
	}
	if err != nil {
		return nil, err
	}

	if err := s.postRequest(req, res, env); err != nil {
		return nil, err
	}
	return res, nil
}

// LoadGRPCServices reflects or parses proto files for the given request.
func (s *Service) LoadGRPCServices(req *domain.Request, env *domain.Environment) ([]domain.GRPCService, error) {
	var protoFiles []*domain.ProtoFile
	if s.proto != nil {
		protoFiles = s.proto()
	}
	return s.grpc.GetServicesFrom(req, env, protoFiles)
}

// GRPCExampleBody returns an example JSON payload for the selected method.
func (s *Service) GRPCExampleBody(req *domain.Request, env *domain.Environment) (string, error) {
	var protoFiles []*domain.ProtoFile
	if s.proto != nil {
		protoFiles = s.proto()
	}
	return s.grpc.GetRequestStructFrom(req, env, protoFiles)
}

func (s *Service) preRequest(req *domain.Request, env *domain.Environment) error {
	preReq := req.Spec.GetPreRequest()
	if !domain.DoablePreRequest(preReq) {
		return nil
	}
	if preReq.Type == domain.PrePostTypePython && preReq.Script != "" {
		return s.executeScript(preReq.Script, req, nil, env)
	}
	if preReq.TriggerRequest == nil || s.lookup == nil {
		return nil
	}
	triggered := s.lookup(preReq.TriggerRequest.RequestID)
	if triggered == nil {
		return fmt.Errorf("trigger request %s not found", preReq.TriggerRequest.RequestID)
	}
	_, err := s.Send(triggered, env)
	return err
}

func (s *Service) postRequest(req *domain.Request, res *egress.Response, env *domain.Environment) error {
	if res == nil {
		return nil
	}
	postReq := req.Spec.GetPostRequest()
	if !domain.DoablePostRequest(postReq) {
		return nil
	}
	if postReq.Type == domain.PrePostTypePython && postReq.Script != "" {
		if err := s.executeScript(postReq.Script, req, res, env); err != nil {
			return err
		}
		if env == nil {
			return nil
		}
		return s.persistEnv(env)
	}
	if env == nil {
		return nil
	}
	if err := s.extractVariables(req.Spec, res, env); err != nil {
		return err
	}
	if postReq.Type == domain.PrePostTypeSetEnv && postReq.PostRequestSet.IsValid() {
		code := res.StatusCode
		if code == 0 {
			code = res.StatueCode
		}
		if code == postReq.PostRequestSet.StatusCode {
			s.applyPostSet(postReq, res, env)
		}
	}
	return s.persistEnv(env)
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
				return err
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
	return nil
}

func (s *Service) executeScript(script string, request *domain.Request, resp *egress.Response, env *domain.Environment) error {
	if !prefs.GetGlobalConfig().Spec.Scripting.Enabled || s.script == nil {
		logger.Warn("Scripting is disabled, cannot execute script")
		return nil
	}
	params := &scripting.ExecParams{
		Env: env,
		Req: scripting.RequestDataFromDomain(request),
	}
	if resp != nil {
		params.Res = &scripting.ResponseData{
			StatusCode: resp.StatusCode,
			Headers:    resp.ResponseHeaders,
			Body:       resp.JSON,
		}
	}
	result, err := s.script.Execute(context.Background(), script, params)
	if err != nil {
		return err
	}
	if env != nil {
		for k, v := range result.SetEnvironments {
			if data, ok := v.(string); ok {
				env.SetKey(k, data)
			}
		}
	}
	for _, pt := range result.Prints {
		logger.Print(pt)
	}
	return nil
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
