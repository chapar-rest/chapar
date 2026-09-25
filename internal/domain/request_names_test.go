package domain

import "testing"

func TestRequestAutoName_HTTP(t *testing.T) {
	req := NewHTTPRequest(DefaultRequestName)
	req.Spec.HTTP.URL = "https://example.com/foo/bar?x=1"
	if got := RequestAutoName(req); got != "example.com/foo/bar" {
		t.Fatalf("got %q want example.com/foo/bar", got)
	}
}

func TestRequestAutoName_HTTPVariableURL(t *testing.T) {
	req := NewHTTPRequest(DefaultRequestName)
	req.Spec.HTTP.URL = "{{baseUrl}}/users"
	if got := RequestAutoName(req); got != "{{baseUrl}}/users" {
		t.Fatalf("got %q want {{baseUrl}}/users", got)
	}
}

func TestRequestAutoName_GraphQL(t *testing.T) {
	req := NewGraphQLRequest(DefaultRequestName)
	req.Spec.GraphQL.URL = "https://api.example.com/graphql"
	if got := RequestAutoName(req); got != "api.example.com/graphql" {
		t.Fatalf("got %q want api.example.com/graphql", got)
	}
}

func TestRequestAutoName_GRPC(t *testing.T) {
	req := NewGRPCRequest(DefaultRequestName)
	req.Spec.GRPC.LasSelectedMethod = "helloworld.Greeter/SayHello"
	if got := RequestAutoName(req); got != "helloworld.Greeter/SayHello" {
		t.Fatalf("got %q want helloworld.Greeter/SayHello", got)
	}
}

func TestRequestDisplayName_CustomName(t *testing.T) {
	req := NewHTTPRequest("My API")
	req.Spec.HTTP.URL = "https://example.com/foo"
	if got := RequestDisplayName(req); got != "My API" {
		t.Fatalf("got %q want My API", got)
	}
}

func TestRequestDisplayName_AutoWhenDefault(t *testing.T) {
	req := NewHTTPRequest(DefaultRequestName)
	req.Spec.HTTP.URL = "https://example.com/foo"
	if got := RequestDisplayName(req); got != "example.com/foo" {
		t.Fatalf("got %q want example.com/foo", got)
	}
}

func TestRequestInfoNameValue(t *testing.T) {
	req := NewHTTPRequest(DefaultRequestName)
	if got := RequestInfoNameValue(req); got != "" {
		t.Fatalf("got %q want empty", got)
	}
	req.MetaData.Name = "Custom"
	if got := RequestInfoNameValue(req); got != "Custom" {
		t.Fatalf("got %q want Custom", got)
	}
}

func TestSetRequestInfoName(t *testing.T) {
	req := NewHTTPRequest("Custom")
	SetRequestInfoName(req, "")
	if req.MetaData.Name != DefaultRequestName {
		t.Fatalf("got %q want %q", req.MetaData.Name, DefaultRequestName)
	}
	SetRequestInfoName(req, "  Renamed  ")
	if req.MetaData.Name != "Renamed" {
		t.Fatalf("got %q want Renamed", req.MetaData.Name)
	}
}
