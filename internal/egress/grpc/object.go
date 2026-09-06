package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/internal/variables"
)

// NewDirect builds a gRPC service that does not read from internal/state.
func NewDirect() *Service {
	return &Service{
		protoFilesRegistry: safemap.New[*protoregistry.Files](),
	}
}

// GetServicesFrom lists methods using the given request, environment, and proto files.
func (s *Service) GetServicesFrom(req *domain.Request, env *domain.Environment, protoFiles []*domain.ProtoFile) ([]domain.GRPCService, error) {
	if req == nil || req.Spec.GRPC == nil {
		return nil, ErrRequestNotFound
	}

	cloned := req.Clone()
	cloned.MetaData.ID = req.MetaData.ID
	spec := cloned.Spec.GRPC

	vars := variables.GetVariables()
	variables.ApplyToGRPCRequest(vars, spec)
	if env != nil {
		variables.ApplyToEnv(vars, &env.Spec)
		env.ApplyToGRPCRequest(spec)
	}

	conn, err := s.Dial(spec)
	if err != nil {
		return nil, err
	}

	id := req.MetaData.ID
	if spec.ServerInfo.ServerReflection {
		protoRegistryFiles, err := ProtoFilesFromReflectionAPI(context.Background(), conn)
		if err != nil {
			return nil, err
		}
		s.protoFilesRegistry.Set(id, protoRegistryFiles)
		return s.parseRegistryFiles(protoRegistryFiles)
	}
	if len(spec.ServerInfo.ProtoFiles) > 0 {
		protoRegistryFiles, err := ProtoFilesFromDisk(GetImportPaths(protoFiles, spec.ServerInfo.ProtoFiles))
		if err != nil {
			return nil, err
		}
		s.protoFilesRegistry.Set(id, protoRegistryFiles)
		return s.parseRegistryFiles(protoRegistryFiles)
	}

	return nil, fmt.Errorf("no server reflection or proto files found")
}

// GetRequestStructFrom returns an example JSON body for the selected method.
func (s *Service) GetRequestStructFrom(req *domain.Request, env *domain.Environment, protoFiles []*domain.ProtoFile) (string, error) {
	if req == nil || req.Spec.GRPC == nil {
		return "", ErrRequestNotFound
	}
	method := req.Spec.GRPC.LasSelectedMethod
	if method == "" {
		return "", errors.New("no method selected")
	}
	md, err := s.methodDescFrom(req, env, protoFiles, method)
	if err != nil {
		return "", err
	}
	jsonBytes, err := json.MarshalIndent(GenerateExampleJSON(md.Input()), "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// SendObject invokes a gRPC method from domain objects instead of looking them up in state.
func (s *Service) SendObject(req *domain.Request, env *domain.Environment, collection *domain.Collection, protoFiles []*domain.ProtoFile) (*egress.Response, error) {
	if req == nil || req.Spec.GRPC == nil {
		return nil, ErrRequestNotFound
	}

	cloned := req.Clone()
	cloned.MetaData.ID = req.MetaData.ID
	cloned.CollectionID = req.CollectionID
	spec := cloned.Spec.GRPC

	if collection != nil {
		spec.Metadata = s.mergeMetadata(collection.Spec.Headers, spec.Metadata)
		if spec.Auth.Type == domain.AuthTypeInherit {
			spec.Auth = collection.Spec.Auth
		}
	}

	vars := variables.GetVariables()
	variables.ApplyToGRPCRequest(vars, spec)
	if env != nil {
		variables.ApplyToEnv(vars, &env.Spec)
		env.ApplyToGRPCRequest(spec)
	}

	method := spec.LasSelectedMethod
	if method == "" {
		return nil, errors.New("no method selected")
	}

	conn, err := s.Dial(spec)
	if err != nil {
		return nil, err
	}

	md, err := s.methodDescFrom(cloned, env, protoFiles, method)
	if err != nil {
		return nil, err
	}

	request := dynamicpb.NewMessage(md.Input())
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(spec.Body), request); err != nil {
		return nil, err
	}

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.New(nil))
	for _, item := range spec.Metadata {
		if !item.Enable {
			continue
		}
		ctx = metadata.AppendToOutgoingContext(ctx, item.Key, item.Value)
	}
	if authHeaders := s.prepareAuth(spec); authHeaders != nil {
		ctx = metadata.NewOutgoingContext(ctx, *authHeaders)
	}

	var respHeaders, respTrailers metadata.MD
	timeOut := 2 * time.Hour
	if spec.Settings.TimeoutMilliseconds > 0 {
		timeOut = time.Duration(spec.Settings.TimeoutMilliseconds) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	outgoingMetadata, _ := metadata.FromOutgoingContext(ctx)
	callOpts := []grpc.CallOption{
		grpc.Header(&respHeaders),
		grpc.Trailer(&respTrailers),
	}

	var (
		respErr error
		respStr string
	)
	start := time.Now()
	if md.IsStreamingServer() {
		respStr, respErr = s.invokeServerStream(ctx, conn, method, request, md, callOpts...)
	} else {
		respStr, respErr = s.invokeUnary(ctx, conn, method, request, md, callOpts...)
	}
	elapsed := time.Since(start)

	return &egress.Response{
		TimePassed:       elapsed,
		ResponseMetadata: domain.MetadataToKeyValue(respHeaders),
		RequestMetadata:  domain.MetadataToKeyValue(outgoingMetadata),
		Trailers:         domain.MetadataToKeyValue(respTrailers),
		Error:            respErr,
		StatueCode:       int(status.Code(respErr)),
		Status:           status.Code(respErr).String(),
		Size:             len(respStr),
		Body:             []byte(respStr),
		JSON:             respStr,
		IsJSON:           json.Valid([]byte(respStr)),
		StatusCode:       int(status.Code(respErr)),
	}, nil
}

func (s *Service) methodDescFrom(req *domain.Request, env *domain.Environment, protoFiles []*domain.ProtoFile, fullName string) (protoreflect.MethodDescriptor, error) {
	id := req.MetaData.ID
	registryFiles, exist := s.protoFilesRegistry.Get(id)
	if !exist {
		if _, err := s.GetServicesFrom(req, env, protoFiles); err != nil {
			return nil, err
		}
		registryFiles, _ = s.protoFilesRegistry.Get(id)
	}
	if registryFiles == nil {
		return nil, errors.New("proto registry is empty")
	}

	name := stringsReplaceMethod(fullName)
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

func stringsReplaceMethod(fullName string) string {
	if fullName == "" {
		return fullName
	}
	if fullName[0] == '/' {
		fullName = fullName[1:]
	}
	for i := 0; i < len(fullName); i++ {
		if fullName[i] == '/' {
			return fullName[:i] + "." + fullName[i+1:]
		}
	}
	return fullName
}
