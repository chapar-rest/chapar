package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/version"
)

var (
	ErrRequestNotFound = errors.New("request not found")
)

type Service struct {
	protoFilesRegistry *safemap.Map[*protoregistry.Files]
}

func (s *Service) Dial(req *domain.GRPCRequestSpec, extraOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithUserAgent(version.GetAgentName()),
	}

	if !req.Settings.Insecure {
		// NameOverride is the name the server's certificate is verified
		// against; empty means the host of the address.
		tlsCfg := tls.Config{ServerName: req.Settings.NameOverride}

		certPath, keyPath := req.Settings.ClientCertFile, req.Settings.ClientKeyFile
		if (certPath == "") != (keyPath == "") {
			return nil, errors.New("mutual TLS needs both a client certificate and a client key")
		}
		if certPath != "" {
			certFile, err := os.ReadFile(certPath)
			if err != nil {
				return nil, err
			}

			keyFile, err := os.ReadFile(keyPath)
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

			if !tlsCfg.RootCAs.AppendCertsFromPEM(rootFile) {
				return nil, fmt.Errorf("trusted root certificate %s holds no PEM certificate", req.Settings.RootCertFile)
			}
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, extraOpts...)
	return grpc.NewClient(req.ServerInfo.Address, opts...)
}

func GenerateExampleJSON(messageDescriptor protoreflect.MessageDescriptor) map[string]interface{} {
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
			out = GenerateExampleJSON(nestedMessageDescriptor)
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

// mergeMetadata merges collection headers (as metadata) with request metadata
// Collection headers are the base, request metadata override collection headers with the same key
// Only enabled items are included
func (s *Service) mergeMetadata(collectionHeaders, requestMetadata []domain.KeyValue) []domain.KeyValue {
	// Create a map of request metadata by key (case-insensitive) for quick lookup
	requestMetadataMap := make(map[string]domain.KeyValue)
	for _, m := range requestMetadata {
		if m.Enable {
			requestMetadataMap[strings.ToLower(m.Key)] = m
		}
	}

	// Start with collection headers (as metadata)
	merged := make([]domain.KeyValue, 0)

	// Add collection headers that don't have request overrides
	for _, ch := range collectionHeaders {
		if !ch.Enable {
			continue
		}
		keyLower := strings.ToLower(ch.Key)
		if _, hasOverride := requestMetadataMap[keyLower]; !hasOverride {
			merged = append(merged, ch)
		}
	}

	// Add all request metadata (they override collection headers)
	for _, rm := range requestMetadata {
		if rm.Enable {
			merged = append(merged, rm)
		}
	}

	return merged
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

// isReflectionService returns true for the built-in gRPC reflection service(s),
// which should be hidden from the user's method list and collections.
func isReflectionService(svc protoreflect.ServiceDescriptor) bool {
	fullName := string(svc.FullName())
	return strings.HasPrefix(fullName, "grpc.reflection.")
}

func (s *Service) parseRegistryFiles(in *protoregistry.Files) ([]domain.GRPCService, error) {
	services := make([]domain.GRPCService, 0)
	in.RangeFiles(func(ds protoreflect.FileDescriptor) bool {
		for i := 0; i < ds.Services().Len(); i++ {
			svc := ds.Services().Get(i)
			if isReflectionService(svc) {
				continue
			}
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

// messageJSON is body as a JSON message. A blank body is the empty message,
// which protojson would otherwise reject as a syntax error.
func messageJSON(body string) []byte {
	if strings.TrimSpace(body) == "" {
		return []byte("{}")
	}
	return []byte(body)
}
