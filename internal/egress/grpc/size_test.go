package grpc

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/chapar-rest/chapar/internal/domain"
)

// echoServer answers /test.Echo/Echo with the bytes it was sent, so one
// call moves a message of the same size each way.
func echoServer(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer(grpc.MaxRecvMsgSize(64<<20), grpc.MaxSendMsgSize(64<<20))
	srv.RegisterService(&grpc.ServiceDesc{
		ServiceName: "test.Echo",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "Echo",
			Handler: func(_ any, _ context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
				in := &wrapperspb.BytesValue{}
				if err := dec(in); err != nil {
					return nil, err
				}
				return in, nil
			},
		}},
	}, struct{}{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func TestMessageSizeLimits(t *testing.T) {
	addr := echoServer(t)
	call := func(set domain.GRPCSettings, size int) error {
		set.Insecure = true
		conn, err := (&Service{}).Dial(&domain.GRPCRequestSpec{ServerInfo: domain.ServerInfo{Address: addr}, Settings: set})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		return conn.Invoke(context.Background(), "/test.Echo/Echo", wrapperspb.Bytes(make([]byte, size)), &wrapperspb.BytesValue{})
	}

	if err := call(domain.GRPCSettings{}, 5<<20); status.Code(err) != codes.ResourceExhausted {
		t.Errorf("5 MB response with default limit: got %v, want ResourceExhausted", err)
	}
	if err := call(domain.GRPCSettings{MaxReceiveMessageSizeMB: 8}, 5<<20); err != nil {
		t.Errorf("5 MB response with 8 MB limit: %v", err)
	}
	if err := call(domain.GRPCSettings{MaxReceiveMessageSizeMB: 1}, 2<<20); status.Code(err) != codes.ResourceExhausted {
		t.Errorf("2 MB response with 1 MB limit: got %v, want ResourceExhausted", err)
	}
	if err := call(domain.GRPCSettings{MaxSendMessageSizeMB: 1, MaxReceiveMessageSizeMB: 8}, 2<<20); status.Code(err) != codes.ResourceExhausted {
		t.Errorf("2 MB request with 1 MB send limit: got %v, want ResourceExhausted", err)
	}
	if err := call(domain.GRPCSettings{MaxSendMessageSizeMB: 1}, 512<<10); err != nil {
		t.Errorf("512 KB request with 1 MB send limit: %v", err)
	}
}
