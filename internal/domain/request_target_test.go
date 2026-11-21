package domain

import "testing"

func TestRequestTarget(t *testing.T) {
	tests := []struct {
		name string
		req  *Request
		want string
	}{
		{"nil", nil, ""},
		{"http", &Request{MetaData: RequestMeta{Type: RequestTypeHTTP}, Spec: RequestSpec{HTTP: &HTTPRequestSpec{Method: "POST", URL: " https://api.example.com/users "}}}, "POST https://api.example.com/users"},
		{"http without url", &Request{MetaData: RequestMeta{Type: RequestTypeHTTP}, Spec: RequestSpec{HTTP: &HTTPRequestSpec{Method: "GET"}}}, ""},
		{"graphql", &Request{MetaData: RequestMeta{Type: RequestTypeGraphQL}, Spec: RequestSpec{GraphQL: &GraphQLRequestSpec{URL: "https://api.example.com/graphql"}}}, "https://api.example.com/graphql"},
		{"websocket", &Request{MetaData: RequestMeta{Type: RequestTypeWebSocket}, Spec: RequestSpec{WebSocket: &WebSocketRequestSpec{URL: "wss://echo.example.com"}}}, "wss://echo.example.com"},
		{"grpc", &Request{MetaData: RequestMeta{Type: RequestTypeGRPC}, Spec: RequestSpec{GRPC: &GRPCRequestSpec{ServerInfo: ServerInfo{Address: "localhost:50051"}, LasSelectedMethod: "/greeter.Greeter/SayHello"}}}, "localhost:50051 /greeter.Greeter/SayHello"},
	}
	for _, tt := range tests {
		if got := RequestTarget(tt.req); got != tt.want {
			t.Errorf("%s: RequestTarget = %q, want %q", tt.name, got, tt.want)
		}
	}
}
