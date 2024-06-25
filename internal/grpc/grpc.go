package grpc

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/internal/state"

	//lint:ignore SA1019 we have to import this because it appears in exported API
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrReflectionNotSupported = errors.New("server does not support the reflection API")
	ErrRequestNotFound        = errors.New("request not found")
)

type Service struct {
	requests     *state.Requests
	environments *state.Environments

	protoFiles *safemap.Map[*protoregistry.Files]
}

type Response struct {
	Body string
}

func NewService(requests *state.Requests, envs *state.Environments) *Service {
	return &Service{
		requests:     requests,
		environments: envs,
		protoFiles:   safemap.New[*protoregistry.Files](),
	}
}

func (s *Service) Dial(requestID string) (*grpc.ClientConn, error) {
	req := s.requests.GetRequest(requestID)
	if req == nil {
		return nil, ErrRequestNotFound
	}

	address := req.Spec.GRPC.ServerInfo.Address

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	return grpc.NewClient(address, opts...)
}

func (s *Service) Invoke(id string, envID string) (*Response, error) {
	req := s.requests.GetRequest(id)
	if req == nil {
		return nil, ErrRequestNotFound
	}

	method := req.Spec.GRPC.LasSelectedMethod
	rawJSON := []byte(req.Spec.GRPC.Body)

	conn, err := s.Dial(id)
	if err != nil {
		return nil, err
	}

	// get the method descriptor
	md, err := s.getMethodDesc(id, method)
	if err != nil {
		return nil, err
	}

	// create the message
	request := dynamicpb.NewMessage(md.Input())
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, request); err != nil {
		return nil, err
	}

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.New(nil))
	for _, item := range req.Spec.GRPC.Metadata {
		ctx = metadata.AppendToOutgoingContext(ctx, item.Key, item.Value)
	}

	resp := dynamicpb.NewMessage(md.Output())
	if err := conn.Invoke(ctx, method, request, resp); err != nil {
		return nil, err
	}

	respJSON, err := (protojson.MarshalOptions{}).Marshal(resp)
	if err != nil {
		return nil, err
	}

	return &Response{
		Body: string(respJSON),
	}, nil
}

func (s *Service) getMethodDesc(id, fullname string) (protoreflect.MethodDescriptor, error) {
	registryFiles, exist := s.protoFiles.Get(id)
	if !exist {
		// reload the proto files we don't have them in registry
		if _, err := s.GetServices(id); err != nil {
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

func (s *Service) GetServices(id string) ([]domain.GRPCService, error) {
	req := s.requests.GetRequest(id)
	if req == nil {
		return nil, ErrRequestNotFound
	}

	conn, err := s.Dial(id)
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

//func (s *Service) InvokeMethod(id string, method string) error {
//	conn, err := s.Dial(id)
//	if err != nil {
//		return err
//	}
//
//	svc, mth := parseSymbol(method)
//	if svc == "" || mth == "" {
//		return fmt.Errorf("given method name %q is not in expected format: 'service/method' or 'service.method'", method)
//	}
//
//	ctx := context.Background()
//	ss := serverSource{client: grpcreflect.NewClientAuto(ctx, conn)}
//	defer ss.client.Reset()
//
//	dsc, err := ss.FindSymbol(method)
//	if err != nil {
//		return err
//	}
//
//	sd, ok := dsc.(*desc.ServiceDescriptor)
//	if !ok {
//		return fmt.Errorf("target server does not expose service %q", svc)
//	}
//	mtd := sd.FindMethodByName(mth)
//	if mtd == nil {
//		return fmt.Errorf("service %q does not include a method named %q", svc, mth)
//	}
//
//	msgFactory := dynamic.NewMessageFactoryWithDefaults()
//	req := msgFactory.NewMessage(mtd.GetInputType())
//
//	return nil
//
//}

type serverSource struct {
	client *grpcreflect.Client
}

func (ss serverSource) ListServices() ([]string, error) {
	svcs, err := ss.client.ListServices()
	return svcs, reflectionSupport(err)
}

func (ss serverSource) FindSymbol(fullyQualifiedName string) (desc.Descriptor, error) {
	file, err := ss.client.FileContainingSymbol(fullyQualifiedName)
	if err != nil {
		return nil, reflectionSupport(err)
	}
	d := file.FindSymbol(fullyQualifiedName)
	if d == nil {
		return nil, fmt.Errorf("%s not found: %s", "Symbol", fullyQualifiedName)
	}
	return d, nil
}

// ListMethods uses the given descriptor source to return a sorted list of method names
// for the specified fully-qualified service name.
func (ss serverSource) ListMethods(serviceName string) ([]string, error) {
	dsc, err := ss.FindSymbol(serviceName)
	if err != nil {
		return nil, err
	}
	if sd, ok := dsc.(*desc.ServiceDescriptor); !ok {
		return nil, fmt.Errorf("%s not found: %s", "Service", serviceName)
	} else {
		methods := make([]string, 0, len(sd.GetMethods()))
		for _, method := range sd.GetMethods() {
			methods = append(methods, method.GetFullyQualifiedName())
		}
		sort.Strings(methods)
		return methods, nil
	}
}

func (ss serverSource) AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error) {
	var exts []*desc.FieldDescriptor
	nums, err := ss.client.AllExtensionNumbersForType(typeName)
	if err != nil {
		return nil, reflectionSupport(err)
	}
	for _, fieldNum := range nums {
		ext, err := ss.client.ResolveExtension(typeName, fieldNum)
		if err != nil {
			return nil, reflectionSupport(err)
		}
		exts = append(exts, ext)
	}
	return exts, nil
}

func reflectionSupport(err error) error {
	if err == nil {
		return nil
	}
	if stat, ok := status.FromError(err); ok && stat.Code() == codes.Unimplemented {
		return ErrReflectionNotSupported
	}
	return err
}

func parseSymbol(svcAndMethod string) (string, string) {
	pos := strings.LastIndex(svcAndMethod, "/")
	if pos < 0 {
		pos = strings.LastIndex(svcAndMethod, ".")
		if pos < 0 {
			return "", ""
		}
	}
	return svcAndMethod[:pos], svcAndMethod[pos+1:]
}
