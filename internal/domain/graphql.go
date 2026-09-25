package domain

import (
	"time"

	"github.com/google/uuid"
)

type GraphQLRequestSpec struct {
	URL       string     `yaml:"url"`
	Query     string     `yaml:"query"`
	Variables string     `yaml:"variables"`
	Headers   []KeyValue `yaml:"headers"`
	Auth      Auth       `yaml:"auth"`

	LastUsedEnvironment LastUsedEnvironment `yaml:"lastUsedEnvironment"`

	VariablesList []Variable `yaml:"variablesList"`

	PreRequest  PreRequest  `yaml:"preRequest"`
	PostRequest PostRequest `yaml:"postRequest"`
}

func (g *GraphQLRequestSpec) Clone() *GraphQLRequestSpec {
	clone := *g

	// Deep clone slices to avoid modifying the original
	if len(g.Headers) > 0 {
		clone.Headers = make([]KeyValue, len(g.Headers))
		copy(clone.Headers, g.Headers)
	}

	if len(g.VariablesList) > 0 {
		clone.VariablesList = make([]Variable, len(g.VariablesList))
		copy(clone.VariablesList, g.VariablesList)
	}

	// Clone Auth
	if g.Auth != (Auth{}) {
		clone.Auth = g.Auth.Clone()
	}

	return &clone
}

type GraphQLResponseDetail struct {
	Response        string
	ResponseHeaders []KeyValue
	RequestHeaders  []KeyValue
	StatusCode      int
	Duration        time.Duration
	Size            int
	Error           error
}

func NewGraphQLRequest(name string) *Request {
	return &Request{
		ApiVersion: ApiVersion,
		Kind:       KindRequest,
		MetaData: RequestMeta{
			ID:   uuid.NewString(),
			Name: name,
			Type: RequestTypeGraphQL,
		},
		Spec: RequestSpec{
			GraphQL: &GraphQLRequestSpec{
				URL:       "https://example.com/graphql",
				Query:     "",
				Variables: "{}",
			},
		},
	}
}

func (r *Request) SetDefaultValuesForGraphQL() {
	if r.Spec.GraphQL.URL == "" {
		r.Spec.GraphQL.URL = "https://example.com/graphql"
	}

	if r.Spec.GraphQL.Variables == "" {
		r.Spec.GraphQL.Variables = "{}"
	}

	if r.Spec.GraphQL.Auth == (Auth{}) {
		r.Spec.GraphQL.Auth = Auth{
			Type: "None",
		}
	}

	if r.Spec.GraphQL.PostRequest == (PostRequest{}) {
		r.Spec.GraphQL.PostRequest = PostRequest{
			Type: "None",
		}
	}

	if r.Spec.GraphQL.PreRequest == (PreRequest{}) {
		r.Spec.GraphQL.PreRequest = PreRequest{
			Type: "None",
		}
	}
}
