package domain

import "testing"

// An unchecked value stays in the environment but takes no part in requests
// or scripts; a checked one does.
func TestEnvironmentUsesOnlyCheckedValues(t *testing.T) {
	env := &Environment{Spec: EnvSpec{Values: []KeyValue{
		{Key: "host", Value: "api.example", Enable: true},
		{Key: "token", Value: "old", Enable: false},
		{Key: "sealed", Value: "enc:v1:x", Enable: true, Locked: true},
	}}}

	http := &HTTPRequestSpec{
		URL: "https://{{host}}/{{token}}/{{sealed}}",
		Request: &HTTPRequest{
			Headers: []KeyValue{{Key: "Authorization", Value: "Bearer {{token}}", Enable: true}},
			Body:    Body{Data: `{"t": "{{token}}"}`},
		},
	}
	env.ApplyToHTTPRequest(http)
	if http.URL != "https://api.example/{{token}}/{{sealed}}" {
		t.Errorf("http url = %q", http.URL)
	}
	if got := http.Request.Headers[0].Value; got != "Bearer {{token}}" {
		t.Errorf("header = %q", got)
	}
	if http.Request.Body.Data != `{"t": "{{token}}"}` {
		t.Errorf("body = %q", http.Request.Body.Data)
	}

	grpc := &GRPCRequestSpec{ServerInfo: ServerInfo{Address: "{{host}}:443"}, Body: `{"t": "{{token}}"}`}
	env.ApplyToGRPCRequest(grpc)
	if grpc.ServerInfo.Address != "api.example:443" || grpc.Body != `{"t": "{{token}}"}` {
		t.Errorf("grpc = %q, %q", grpc.ServerInfo.Address, grpc.Body)
	}

	gql := &GraphQLRequestSpec{URL: "https://{{host}}/{{token}}"}
	env.ApplyToGraphQLRequest(gql)
	if gql.URL != "https://api.example/{{token}}" {
		t.Errorf("graphql url = %q", gql.URL)
	}

	got := env.GetKeyValues()
	if len(got) != 1 || got["host"] != "api.example" {
		t.Errorf("GetKeyValues = %v, want only host", got)
	}
}

// Setting a value turns it on, so the next request uses it.
func TestEnvironmentSetKeyTurnsValueOn(t *testing.T) {
	env := &Environment{Spec: EnvSpec{Values: []KeyValue{{Key: "token", Value: "old", Enable: false}}}}
	env.SetKey("token", "new")
	if kv := env.Spec.Values[0]; kv.Value != "new" || !kv.Enable {
		t.Fatalf("after SetKey: %+v", kv)
	}
}
