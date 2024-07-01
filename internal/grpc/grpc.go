package grpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/internal/variables"
)

var (
	ErrRequestNotFound = errors.New("request not found")
)

type Service struct {
	requests     *state.Requests
	environments *state.Environments

	protoFiles *safemap.Map[*protoregistry.Files]
}

type Response struct {
	Body       string
	Metadata   []domain.KeyValue
	Trailers   []domain.KeyValue
	Cookies    []*http.Cookie
	TimePassed time.Duration
	Size       int
	Error      error

	StatueCode int
	Status     string
}

func NewService(requests *state.Requests, envs *state.Environments) *Service {
	return &Service{
		requests:     requests,
		environments: envs,
		protoFiles:   safemap.New[*protoregistry.Files](),
	}
}

func (s *Service) Dial(address string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	return grpc.NewClient(address, opts...)
}

func (s *Service) GetRequestStruct(id, environmentID string) (string, error) {
	req := s.requests.GetRequest(id)
	if req == nil {
		return "", ErrRequestNotFound
	}

	method := req.Spec.GRPC.LasSelectedMethod
	// get the method descriptor
	md, err := s.getMethodDesc(id, environmentID, method)
	if err != nil {
		return "", err
	}

	request := dynamicpb.NewMessage(md.Input())
	reqJSON, err := (protojson.MarshalOptions{
		Indent:          "  ",
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}).Marshal(request)
	if err != nil {
		return "", err
	}

	return string(reqJSON), nil
}

func (s *Service) Invoke(id, activeEnvironmentID string) (*Response, error) {
	req := s.requests.GetRequest(id)
	if req == nil {
		return nil, ErrRequestNotFound
	}

	r := req.Clone()

	var activeEnvironment = s.getActiveEnvironment(activeEnvironmentID)

	return s.invoke(id, r.Spec.GRPC, activeEnvironment)
}

func (s *Service) invoke(id string, req *domain.GRPCRequestSpec, env *domain.Environment) (*Response, error) {
	vars := variables.GetVariables()
	variables.ApplyToEnv(vars, &env.Spec)
	variables.ApplyToGRPCRequest(vars, req)
	env.ApplyToGRPCRequest(req)

	method := req.LasSelectedMethod
	rawJSON := []byte(req.Body)

	conn, err := s.Dial(req.ServerInfo.Address)
	if err != nil {
		return nil, err
	}

	// get the method descriptor
	md, err := s.getMethodDesc(id, env.MetaData.ID, method)
	if err != nil {
		return nil, err
	}

	// create the message
	request := dynamicpb.NewMessage(md.Input())
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, request); err != nil {
		return nil, err
	}

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.New(nil))
	for _, item := range req.Metadata {
		ctx = metadata.AppendToOutgoingContext(ctx, item.Key, item.Value)
	}

	if authHeaders := s.prepareAuth(req); authHeaders != nil {
		ctx = metadata.NewOutgoingContext(ctx, *authHeaders)
	}

	var respHeaders, respTrailers metadata.MD
	resp := dynamicpb.NewMessage(md.Output())

	start := time.Now()
	respErr := conn.Invoke(ctx, method, request, resp, grpc.Header(&respHeaders), grpc.Trailer(&respTrailers))
	elapsed := time.Since(start)

	out := &Response{
		TimePassed: elapsed,
		Metadata:   domain.MetadataToKeyValue(respHeaders),
		Trailers:   domain.MetadataToKeyValue(respTrailers),
		Error:      respErr,
		StatueCode: int(status.Code(respErr)),
		Status:     status.Code(respErr).String(),
		Size:       len(resp.String()),
	}

	if respErr != nil {
		return out, respErr
	}

	respJSON, err := (protojson.MarshalOptions{
		Indent: "  ",
	}).Marshal(resp)
	if err != nil {
		return out, err
	}

	out.Body = string(respJSON)
	return out, nil
}

func (s *Service) prepareAuth(req *domain.GRPCRequestSpec) *metadata.MD {
	if req.Auth.Type == domain.AuthTypeNone {
		return nil
	}

	md := metadata.New(nil)
	if req.Auth.Type == domain.AuthTypeToken {
		md.Append("Authorization", fmt.Sprintf("Bearer %s", req.Auth.TokenAuth.Token))
		return &md
	}

	if req.Auth.Type == domain.AuthTypeBasic && req.Auth.BasicAuth != nil {
		md.Append("Authorization", fmt.Sprintf("Basic %s:%s", req.Auth.BasicAuth.Username, req.Auth.BasicAuth.Password))
		return &md
	}

	if req.Auth.Type == domain.AuthTypeAPIKey {
		md.Append(req.Auth.APIKeyAuth.Key, req.Auth.APIKeyAuth.Value)
		return &md
	}

	return nil
}

func (s *Service) getMethodDesc(id, envID, fullname string) (protoreflect.MethodDescriptor, error) {
	registryFiles, exist := s.protoFiles.Get(id)
	if !exist {
		// reload the proto files we don't have them in registry
		if _, err := s.GetServices(id, envID); err != nil {
			return nil, err
		}

		// get the proto files from the registry
		registryFiles, _ = s.protoFiles.Get(id)
	}

	name := strings.Replace(fullname[1:], "/", ".", 1)
	desc, err := registryFiles.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, fmt.Errorf("app: failed to find descriptor: %v", err)
	}

	methodDesc, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("app: descriptor was not a method: %T", desc)
	}

	return methodDesc, nil
}

func (s *Service) GetServices(id, activeEnvironmentID string) ([]domain.GRPCService, error) {
	req := s.requests.GetRequest(id)
	if req == nil {
		return nil, ErrRequestNotFound
	}

	req = req.Clone()

	var activeEnvironment = s.getActiveEnvironment(activeEnvironmentID)
	vars := variables.GetVariables()
	variables.ApplyToGRPCRequest(vars, req.Spec.GRPC)

	if activeEnvironment != nil {
		variables.ApplyToEnv(vars, &activeEnvironment.Spec)
		activeEnvironment.ApplyToGRPCRequest(req.Spec.GRPC)
	}

	conn, err := s.Dial(req.Spec.GRPC.ServerInfo.Address)
	if err != nil {
		return nil, err
	}

	if req.Spec.GRPC.ServerInfo.ServerReflection {
		protoRegistryFiles, err := ProtoFilesFromReflectionAPI(context.Background(), conn)
		if err != nil {
			return nil, err
		}

		s.protoFiles.Set(id, protoRegistryFiles)

		return s.parseRegistryFiles(protoRegistryFiles)
	} else if len(req.Spec.GRPC.ServerInfo.ProtoFiles) > 0 {
		protoRegistryFiles, err := ProtoFilesFromDisk(getImportPaths(req.Spec.GRPC.ServerInfo.ProtoFiles))
		if err != nil {
			return nil, err
		}

		s.protoFiles.Set(id, protoRegistryFiles)
		return s.parseRegistryFiles(protoRegistryFiles)
	}

	return nil, fmt.Errorf("no server reflection or proto files found")
}

func (s *Service) getActiveEnvironment(id string) *domain.Environment {
	if id == "" {
		return nil
	}

	activeEnvironment := s.environments.GetEnvironment(id)
	if activeEnvironment == nil {
		return nil
	}

	return activeEnvironment
}

func getImportPaths(files []string) ([]string, []string) {
	importPaths := make([]string, 0, len(files))
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		// extract the directory path from the file path
		importPaths = append(importPaths, filepath.Dir(file))
		fileNames = append(fileNames, filepath.Base(file))
	}
	return importPaths, fileNames
}

func (s *Service) parseRegistryFiles(in *protoregistry.Files) ([]domain.GRPCService, error) {
	services := make([]domain.GRPCService, 0)
	in.RangeFiles(func(ds protoreflect.FileDescriptor) bool {
		for i := 0; i < ds.Services().Len(); i++ {
			svc := ds.Services().Get(i)
			srv := domain.GRPCService{
				Name:    string(svc.Name()),
				Methods: make([]domain.GRPCMethod, 0, svc.Methods().Len()),
			}

			for j := 0; j < svc.Methods().Len(); j++ {
				mth := svc.Methods().Get(j)
				fname := fmt.Sprintf("/%s/%s", svc.FullName(), mth.Name())
				srv.Methods = append(srv.Methods, domain.GRPCMethod{
					FullName:          fname,
					Name:              string(mth.Name()),
					IsStreamingClient: mth.IsStreamingClient(),
					IsStreamingServer: mth.IsStreamingServer(),
				})
			}

			sort.SliceStable(srv.Methods, func(i, j int) bool {
				return srv.Methods[i].Name < srv.Methods[j].Name
			})

			services = append(services, srv)
		}
		return true
	})

	sort.SliceStable(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}
