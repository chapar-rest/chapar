package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/repository"
)

// ImportProtoFile imports a proto file and creates a collection with gRPC requests
func ImportProtoFile(data []byte, repo repository.RepositoryV2, filePath ...string) error {
	// Write the data to a temporary file for parsing
	tempFile, err := os.CreateTemp("", "proto_import_*.proto")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write the proto content to the temporary file
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("error writing to temporary file: %w", err)
	}
	tempFile.Close()

	// Parse the proto file using protoreflect
	parser := protoparse.Parser{
		ImportPaths: []string{filepath.Dir(tempFile.Name())},
	}

	descriptors, err := parser.ParseFiles(filepath.Base(tempFile.Name()))
	if err != nil {
		return fmt.Errorf("error parsing proto file: %w", err)
	}

	if len(descriptors) == 0 {
		return fmt.Errorf("no descriptors found in proto file")
	}

	// Convert to protoregistry.Files format
	registryFiles, err := convertToRegistryFiles(descriptors)
	if err != nil {
		return fmt.Errorf("error converting to registry files: %w", err)
	}

	// Parse services from registry files
	services := parseServicesFromRegistry(registryFiles)

	// Determine collection name based on number of services
	var collectionName string
	if len(services) == 1 {
		// Single service: use service name as collection name
		collectionName = services[0].Name
	} else {
		collectionName = "ProtoCollection"
	}

	// Create collection
	col := domain.NewCollection(collectionName)
	if err := repo.CreateCollection(col); err != nil {
		return fmt.Errorf("error saving collection: %w", err)
	}

	// Convert services to gRPC requests
	var requests []*domain.Request
	for _, service := range services {
		for _, method := range service.Methods {
			// Determine request name based on number of services
			var requestName string
			if len(services) == 1 {
				// Single service: don't prefix with service name
				requestName = method.Name
			} else {
				// Multiple services: prefix with service name
				requestName = fmt.Sprintf("%s.%s", service.Name, method.Name)
			}

			// Generate example body from the method's input type
			exampleBody := generateExampleBodyFromMethodDescriptor(registryFiles, method.FullName)

			req := &domain.Request{
				ApiVersion: "v1",
				Kind:       "Request",
				MetaData: domain.RequestMeta{
					ID:   uuid.NewString(),
					Name: requestName,
					Type: domain.RequestTypeGRPC,
				},
				Spec: domain.RequestSpec{
					GRPC: &domain.GRPCRequestSpec{
						LasSelectedMethod: method.FullName,
						Metadata:          []domain.KeyValue{},
						Auth:              domain.Auth{Type: domain.AuthTypeNone},
						ServerInfo: domain.ServerInfo{
							Address:          "localhost:50051", // Default address
							ServerReflection: false,
							ProtoFiles:       getProtoFiles(filePath),
						},
						Settings: domain.GRPCSettings{
							Insecure:            true, // Default to insecure for development
							TimeoutMilliseconds: 30000,
						},
						Body:      exampleBody,
						Services:  []domain.GRPCService{service},
						Variables: []domain.Variable{},
						PreRequest: domain.PreRequest{
							Type: domain.PrePostTypeNone,
						},
						PostRequest: domain.PostRequest{
							Type: domain.PrePostTypeNone,
						},
					},
				},
			}

			req.SetDefaultValues()
			requests = append(requests, req)
		}
	}

	// Save all requests
	for _, req := range requests {
		if err := repo.CreateRequest(req, col); err != nil {
			return fmt.Errorf("error saving request: %w", err)
		}
	}

	return nil
}

// ImportProtoFileFromFile imports a proto file from a file path and creates a collection with gRPC requests
func ImportProtoFileFromFile(filePath string, repo repository.RepositoryV2) error {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading proto file: %w", err)
	}

	return ImportProtoFile(fileContent, repo)
}

// convertToRegistryFiles converts protoreflect descriptors to protoregistry.Files
func convertToRegistryFiles(descriptors []*desc.FileDescriptor) (*protoregistry.Files, error) {
	// Convert the descriptors directly to a FileDescriptorSet
	fdset := &descriptorpb.FileDescriptorSet{}
	seen := make(map[string]struct{})

	for _, fd := range descriptors {
		fdset.File = append(fdset.File, walkFileDescriptors(seen, fd)...)
	}

	return protodesc.NewFiles(fdset)
}

// parseServicesFromRegistry parses services from protoregistry.Files
func parseServicesFromRegistry(registryFiles *protoregistry.Files) []domain.GRPCService {
	services := make([]domain.GRPCService, 0)

	registryFiles.RangeFiles(func(ds protoreflect.FileDescriptor) bool {
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

			services = append(services, srv)
		}
		return true
	})

	return services
}

// generateExampleBodyFromMethodDescriptor generates a JSON example body from a method descriptor
func generateExampleBodyFromMethodDescriptor(registryFiles *protoregistry.Files, methodFullName string) string {
	// Find the method descriptor
	methodDesc, err := findMethodDescriptor(registryFiles, methodFullName)
	if err != nil {
		// Fallback to empty object if we can't find the method
		return "{}"
	}

	// Get the input message descriptor
	inputType := methodDesc.Input()
	if inputType == nil {
		return "{}"
	}

	// Generate example JSON from the message descriptor
	example := grpc.GenerateExampleJSON(inputType)

	// Convert to JSON string
	jsonBytes, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(jsonBytes)
}

// findMethodDescriptor finds a method descriptor by its full name
func findMethodDescriptor(registryFiles *protoregistry.Files, methodFullName string) (protoreflect.MethodDescriptor, error) {
	// Convert method full name to descriptor name
	// Method full name format: /package.Service/Method
	// Descriptor name format: package.Service.Method
	descriptorName := methodFullName[1:]                          // Remove leading slash
	descriptorName = strings.Replace(descriptorName, "/", ".", 1) // Replace first slash with dot

	desc, err := registryFiles.FindDescriptorByName(protoreflect.FullName(descriptorName))
	if err != nil {
		return nil, err
	}

	methodDesc, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor was not a method: %T", desc)
	}

	return methodDesc, nil
}

// getProtoFiles returns the appropriate proto file path
func getProtoFiles(filePath []string) []string {
	// If original file path is provided, use it; otherwise, don't set any proto files
	if len(filePath) > 0 && filePath[0] != "" {
		return []string{filePath[0]}
	}
	// Don't set proto files if we don't have the original path
	// This way the user won't see a confusing temporary file path
	return []string{}
}

// walkFileDescriptors walks through file descriptors and their dependencies
func walkFileDescriptors(seen map[string]struct{}, fd *desc.FileDescriptor) []*descriptorpb.FileDescriptorProto {
	fds := []*descriptorpb.FileDescriptorProto{}

	if _, ok := seen[fd.GetName()]; ok {
		return fds
	}
	seen[fd.GetName()] = struct{}{}
	fds = append(fds, fd.AsFileDescriptorProto())

	for _, dep := range fd.GetDependencies() {
		deps := walkFileDescriptors(seen, dep)
		fds = append(fds, deps...)
	}

	return fds
}
