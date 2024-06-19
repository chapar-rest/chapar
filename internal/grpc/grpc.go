package grpc

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/state"

	//lint:ignore SA1019 we have to import this because it appears in exported API
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrReflectionNotSupported = errors.New("server does not support the reflection API")

type Service struct {
	requests     *state.Requests
	environments *state.Environments
}

func NewService(requests *state.Requests, envs *state.Environments) *Service {
	return &Service{
		requests:     requests,
		environments: envs,
	}
}

func (s *Service) Dial(requestID string) (*grpc.ClientConn, error) {
	req := s.requests.GetRequest(requestID)
	if req == nil {
		return nil, fmt.Errorf("request not found: %s", requestID)
	}

	address := req.Spec.GRPC.ServerInfo.Address

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	return grpc.NewClient(address, opts...)
}

func (s *Service) GetServerReflection(id string) ([]domain.GRPCMethod, error) {
	conn, err := s.Dial(id)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ss := serverSource{client: grpcreflect.NewClientAuto(ctx, conn)}
	defer ss.client.Reset()

	svcs, err := ss.ListServices()
	if err != nil {
		return nil, err
	}

	methods := make([]domain.GRPCMethod, 0)

	for _, svc := range svcs {
		mds, err := ss.ListMethods(svc)
		if err != nil {
			return nil, err
		}

		for _, md := range mds {
			methods = append(methods, domain.GRPCMethod{
				Name: md,
			})
		}
	}

	return methods, nil
}

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
