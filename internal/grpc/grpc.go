package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	appVersion string

	requests     *state.Requests
	environments *state.Environments
	protoFiles   *state.ProtoFiles

	protoFilesRegistry *safemap.Map[*protoregistry.Files]
}

type Response struct {
	Body             string
	RequestMetadata  []domain.KeyValue
	ResponseMetadata []domain.KeyValue
	Trailers         []domain.KeyValue
	TimePassed       time.Duration
	Size             int
	Error            error

	StatueCode int
	Status     string
}

var (
	appName = "Chapar"
)

func NewService(appVersion string, requests *state.Requests, envs *state.Environments, protoFiles *state.ProtoFiles) *Service {
	return &Service{
		appVersion:         appVersion,
		requests:           requests,
		environments:       envs,
		protoFiles:         protoFiles,
		protoFilesRegistry: safemap.New[*protoregistry.Files](),
	}
}

func (s *Service) Dial(req *domain.GRPCRequestSpec) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithUserAgent(fmt.Sprintf("%s/%s", appName, s.appVersion)),
	}

	if !req.Settings.Insecure {
		var tlsCfg tls.Config
		tlsCfg.InsecureSkipVerify = req.Settings.Insecure

		if req.Settings.ClientCertFile != "" {
			certFile, err := os.ReadFile(req.Settings.ClientCertFile)
			if err != nil {
				return nil, err
			}

			keyFile, err := os.ReadFile(req.Settings.ClientCertFile)
			if err != nil {
				return nil, err
			}

			cert, err := tls.X509KeyPair(certFile, keyFile)
			if err != nil {
				return nil, err
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}

		var err error
		tlsCfg.RootCAs, err = x509.SystemCertPool()
		if err != nil {
			tlsCfg.RootCAs = x509.NewCertPool()
		}
		if req.Settings.RootCertFile != "" {
			rootFile, err := os.ReadFile(req.Settings.RootCertFile)
			if err != nil {
				return nil, err
			}

			tlsCfg.RootCAs.AppendCertsFromPEM(rootFile)
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return grpc.NewClient(req.ServerInfo.Address, opts...)
}

func (s *Service) GetRequestStruct(id, environmentID string) (string, error) {
	req := s.requests.GetRequest(id)
	if req == nil {
		return "", ErrRequestNotFound
	}

	method := req.Spec.GRPC.LasSelectedMethod
	if method == "" {
		return "", errors.New("no method selected")
	}

	// get the method descriptor
	md, err := s.getMethodDesc(id, environmentID, method)
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.MarshalIndent(generateExampleJSON(md.Input()), "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

func generateExampleJSON(messageDescriptor protoreflect.MessageDescriptor) map[string]interface{} {
	out := make(map[string]interface{})

	fields := messageDescriptor.Fields()

	castField := func(field protoreflect.FieldDescriptor) any {
		var out any
		switch field.Kind() {
		case protoreflect.StringKind:
			out = "string"
		case protoreflect.DoubleKind, protoreflect.FloatKind:
			out = 123.456
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind,
			protoreflect.Fixed64Kind, protoreflect.Int32Kind, protoreflect.Sint32Kind,
			protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind,
			protoreflect.Sfixed64Kind:
			out = 123
		case protoreflect.BytesKind:
			out = "bytes"
		case protoreflect.EnumKind:
			enum := field.Enum()
			out = string(enum.Values().Get(0).Name())
		case protoreflect.BoolKind:
			out = true
		case protoreflect.MessageKind:
			nestedMessageDescriptor := field.Message()
			out = generateExampleJSON(nestedMessageDescriptor)
		default:
			out = "string"
		}

		return out
	}

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)

		// Check if the field is repeated
		switch {
		case field.IsMap():
			// Handle map fields
			keyField := field.MapKey()
			valueField := field.MapValue()

			// Generate a key as a string
			var mapKey string
			switch keyField.Kind() {
			case protoreflect.StringKind:
				mapKey = "key_string"
			case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind,
				protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind:
				mapKey = "123"
			case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind,
				protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind:
				mapKey = "123456789"
			default:
				mapKey = "key"
			}

			mapValue := castField(valueField)
			out[string(field.Name())] = map[string]interface{}{
				mapKey: mapValue,
			}
		case field.Cardinality() == protoreflect.Repeated:
			// Handle repeated fields
			var repeatedValues []interface{}
			repeatedValues = append(repeatedValues, castField(field))
			out[string(field.Name())] = repeatedValues
		default:
			// Handle singular fields
			out[string(field.Name())] = castField(field)
		}
	}

	return out
}

func (s *Service) Invoke(id, activeEnvironmentID string) (*Response, error) {
	req := s.requests.GetRequest(id)
	if req == nil {
		return nil, ErrRequestNotFound
	}

	spec := req.Clone().Spec.GRPC
	if spec == nil {
		return nil, nil
	}

	var activeEnvironment = s.getActiveEnvironment(activeEnvironmentID)

	vars := variables.GetVariables()
	variables.ApplyToGRPCRequest(vars, spec)

	if activeEnvironment != nil {
		variables.ApplyToEnv(vars, &activeEnvironment.Spec)
		activeEnvironment.ApplyToGRPCRequest(spec)
	}

	method := spec.LasSelectedMethod
	if method == "" {
		return nil, errors.New("no method selected")
	}

	rawJSON := []byte(spec.Body)

	conn, err := s.Dial(spec)
	if err != nil {
		return nil, err
	}

	// get the method descriptor
	md, err := s.getMethodDesc(id, activeEnvironmentID, method)
	if err != nil {
		return nil, err
	}

	// create the message
	request := dynamicpb.NewMessage(md.Input())
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, request); err != nil {
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

	timeOut := 5000 * time.Millisecond
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

	out := &Response{
		TimePassed:       elapsed,
		ResponseMetadata: domain.MetadataToKeyValue(respHeaders),
		RequestMetadata:  domain.MetadataToKeyValue(outgoingMetadata),
		Trailers:         domain.MetadataToKeyValue(respTrailers),
		Error:            respErr,
		StatueCode:       int(status.Code(respErr)),
		Status:           status.Code(respErr).String(),
		Size:             len(respStr),
		Body:             respStr,
	}

	if respErr != nil {
		return out, respErr
	}

	return out, nil
}

func (s *Service) invokeServerStream(ctx context.Context, conn *grpc.ClientConn, method string, req proto.Message, md protoreflect.MethodDescriptor, opts ...grpc.CallOption) (string, error) {
	if conn == nil {
		return "", errors.New("no connection")
	}

	sd := &grpc.StreamDesc{
		StreamName:    method,
		ClientStreams: false,
		ServerStreams: true,
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := conn.NewStream(ctx, sd, method, opts...)
	if err != nil {
		return "", err
	}

	if err := stream.SendMsg(req); err != nil {
		return "", err
	}

	if err := stream.CloseSend(); err != nil {
		return "", err
	}

	var out string
	counter := 0
	for {
		resp := dynamicpb.NewMessage(md.Output())
		err := stream.RecvMsg(resp)
		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}

		respJSON, err := (protojson.MarshalOptions{
			Indent: "  ",
		}).Marshal(resp)
		if err != nil {
			return "", err
		}

		// concat responses with a new line and message counter
		out += fmt.Sprintf("// Message %d:\n%s\n\n", counter, string(respJSON))
		counter++
	}

	return out, nil
}

func (s *Service) invokeUnary(ctx context.Context, conn *grpc.ClientConn, method string, req proto.Message, md protoreflect.MethodDescriptor, opts ...grpc.CallOption) (string, error) {
	if conn == nil {
		return "", errors.New("no connection")
	}

	resp := dynamicpb.NewMessage(md.Output())
	if err := conn.Invoke(ctx, method, req, resp, opts...); err != nil {
		return "", err
	}

	respJSON, err := (protojson.MarshalOptions{
		Indent: "  ",
	}).Marshal(resp)
	if err != nil {
		return "", err
	}

	return string(respJSON), nil
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

func (s *Service) getMethodDesc(id, envID, fullName string) (protoreflect.MethodDescriptor, error) {
	registryFiles, exist := s.protoFilesRegistry.Get(id)
	if !exist {
		// reload the proto files we don't have them in registry
		if _, err := s.GetServices(id, envID); err != nil {
			return nil, err
		}

		// get the proto files from the registry
		registryFiles, _ = s.protoFilesRegistry.Get(id)
	}

	name := strings.Replace(fullName[1:], "/", ".", 1)
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

	conn, err := s.Dial(req.Spec.GRPC)
	if err != nil {
		return nil, err
	}

	if req.Spec.GRPC.ServerInfo.ServerReflection {
		protoRegistryFiles, err := ProtoFilesFromReflectionAPI(context.Background(), conn)
		if err != nil {
			return nil, err
		}

		s.protoFilesRegistry.Set(id, protoRegistryFiles)

		return s.parseRegistryFiles(protoRegistryFiles)
	} else if len(req.Spec.GRPC.ServerInfo.ProtoFiles) > 0 {
		protoFiles, err := s.protoFiles.LoadProtoFiles()
		if err != nil {
			return nil, err
		}

		protoRegistryFiles, err := ProtoFilesFromDisk(GetImportPaths(protoFiles, req.Spec.GRPC.ServerInfo.ProtoFiles))
		if err != nil {
			return nil, err
		}

		s.protoFilesRegistry.Set(id, protoRegistryFiles)
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

func GetImportPaths(protoFiles []*domain.ProtoFile, files []string) ([]string, []string) {
	importPaths := make([]string, 0, len(protoFiles)+len(files))
	fileNames := make([]string, 0, len(protoFiles)+len(files))
	for _, file := range files {
		// extract the directory path from the file path
		importPaths = append(importPaths, filepath.Dir(file))
		fileNames = append(fileNames, filepath.Base(file))
	}

	for _, protoFile := range protoFiles {
		if protoFile.Spec.IsImportPath {
			importPaths = append(importPaths, protoFile.Spec.Path)
		} else {
			importPaths = append(importPaths, filepath.Dir(protoFile.Spec.Path))
			fileNames = append(fileNames, filepath.Base(protoFile.Spec.Path))
		}
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
