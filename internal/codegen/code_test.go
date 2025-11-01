package codegen

import (
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

func TestGenerateCurlCommand(t *testing.T) {
	tests := []struct {
		name     string
		spec     *domain.HTTPRequestSpec
		contains []string
	}{
		{
			name: "GET request",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodGET,
				URL:    "https://api.example.com/users",
				Request: &domain.HTTPRequest{
					Headers: []domain.KeyValue{
						{Key: "Authorization", Value: "Bearer token123", Enable: true},
					},
				},
			},
			contains: []string{"curl", "-X GET", "https://api.example.com/users", "-H \"Authorization: Bearer token123\""},
		},
		{
			name: "HEAD request",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodHEAD,
				URL:    "https://api.example.com/users",
				Request: &domain.HTTPRequest{},
			},
			contains: []string{"curl --head", "https://api.example.com/users"},
		},
		{
			name: "POST with JSON",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodPOST,
				URL:    "https://api.example.com/users",
				Request: &domain.HTTPRequest{
					Body: domain.Body{
						Type: "json",
						Data: `{"name": "Test"}`,
					},
				},
			},
			contains: []string{"curl", "-X POST", "-d '{\"name\": \"Test\"}'"},
		},
		{
			name: "POST with urlEncoded",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodPOST,
				URL:    "https://api.example.com/users",
				Request: &domain.HTTPRequest{
					Body: domain.Body{
						Type: "urlEncoded",
						URLEncoded: []domain.KeyValue{
							{Key: "name", Value: "test", Enable: true},
						},
					},
				},
			},
			contains: []string{"curl", "-X POST", "-d \"name=test\""},
		},
	}

	svc := &Service{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := svc.GenerateCurlCommand(tt.spec)
			if err != nil {
				t.Fatalf("GenerateCurlCommand() error = %v", err)
			}
			for _, substr := range tt.contains {
				if !strings.Contains(code, substr) {
					t.Errorf("GenerateCurlCommand() result should contain %q, got:\n%s", substr, code)
				}
			}
		})
	}
}

func TestGeneratePythonRequest(t *testing.T) {
	tests := []struct {
		name     string
		spec     *domain.HTTPRequestSpec
		contains []string
	}{
		{
			name: "POST with JSON",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodPOST,
				URL:    "https://api.example.com/resource",
				Request: &domain.HTTPRequest{
					Headers: []domain.KeyValue{
						{Key: "Authorization", Value: "Bearer token123", Enable: true},
					},
					Body: domain.Body{
						Type: "json",
						Data: `{"name": "Test"}`,
					},
				},
			},
			contains: []string{"import requests", "import json", "json_data = json.dumps", "requests.post"},
		},
		{
			name: "GET with text body",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodGET,
				URL:    "https://api.example.com/resource",
				Request: &domain.HTTPRequest{
					Body: domain.Body{
						Type: "text",
						Data: "plain text",
					},
				},
			},
			contains: []string{"import requests", "data = '''plain text'''", "requests.get"},
		},
		{
			name: "POST with urlEncoded",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodPOST,
				URL:    "https://api.example.com/resource",
				Request: &domain.HTTPRequest{
					Body: domain.Body{
						Type: "urlEncoded",
						URLEncoded: []domain.KeyValue{
							{Key: "username", Value: "test", Enable: true},
						},
					},
				},
			},
			contains: []string{"import requests", "data = {", "\"username\": \"test\""},
		},
	}

	svc := &Service{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := svc.GeneratePythonRequest(tt.spec)
			if err != nil {
				t.Fatalf("GeneratePythonRequest() error = %v", err)
			}
			for _, substr := range tt.contains {
				if !strings.Contains(code, substr) {
					t.Errorf("GeneratePythonRequest() result should contain %q, got:\n%s", substr, code)
				}
			}
		})
	}
}

func TestGenerateGoRequest(t *testing.T) {
	tests := []struct {
		name     string
		spec     *domain.HTTPRequestSpec
		contains []string
	}{
		{
			name: "POST with JSON",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodPOST,
				URL:    "https://api.example.com/resource",
				Request: &domain.HTTPRequest{
					Body: domain.Body{
						Type: "json",
						Data: `{"name": "Test"}`,
					},
				},
			},
			contains: []string{"package main", "import (", "\"strings\"", "strings.NewReader", "http.NewRequest"},
		},
		{
			name: "POST with urlEncoded",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodPOST,
				URL:    "https://api.example.com/resource",
				Request: &domain.HTTPRequest{
					Body: domain.Body{
						Type: "urlEncoded",
						URLEncoded: []domain.KeyValue{
							{Key: "username", Value: "test", Enable: true},
						},
					},
				},
			},
			contains: []string{"package main", "url.Values{}", "data.Set", "application/x-www-form-urlencoded"},
		},
	}

	svc := &Service{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := svc.GenerateGoRequest(tt.spec)
			if err != nil {
				t.Fatalf("GenerateGoRequest() error = %v", err)
			}
			for _, substr := range tt.contains {
				if !strings.Contains(code, substr) {
					t.Errorf("GenerateGoRequest() result should contain %q, got:\n%s", substr, code)
				}
			}
		})
	}
}

func TestGenerateAxiosCommand(t *testing.T) {
	tests := []struct {
		name     string
		spec     *domain.HTTPRequestSpec
		contains []string
	}{
		{
			name: "POST with JSON",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodPOST,
				URL:    "https://api.example.com/resource",
				Request: &domain.HTTPRequest{
					Body: domain.Body{
						Type: "json",
						Data: `{"name": "Test"}`,
					},
				},
			},
			contains: []string{"const axios = require('axios')", "method: 'POST'", "data: {\"name\": \"Test\"}"},
		},
		{
			name: "POST with urlEncoded",
			spec: &domain.HTTPRequestSpec{
				Method: domain.RequestMethodPOST,
				URL:    "https://api.example.com/resource",
				Request: &domain.HTTPRequest{
					Body: domain.Body{
						Type: "urlEncoded",
						URLEncoded: []domain.KeyValue{
							{Key: "username", Value: "test", Enable: true},
						},
					},
				},
			},
			contains: []string{"const axios = require('axios')", "URLSearchParams"},
		},
	}

	svc := &Service{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := svc.GenerateAxiosCommand(tt.spec)
			if err != nil {
				t.Fatalf("GenerateAxiosCommand() error = %v", err)
			}
			for _, substr := range tt.contains {
				if !strings.Contains(code, substr) {
					t.Errorf("GenerateAxiosCommand() result should contain %q, got:\n%s", substr, code)
				}
			}
		})
	}
}

func TestGenerateKotlinOkHttpCommand(t *testing.T) {
	svc := &Service{}
	spec := &domain.HTTPRequestSpec{
		Method: domain.RequestMethodPOST,
		URL:    "https://api.example.com/resource",
		Request: &domain.HTTPRequest{
			Body: domain.Body{
				Type: "json",
				Data: `{"name": "Test"}`,
			},
		},
	}

	code, err := svc.GenerateKotlinOkHttpCommand(spec)
	if err != nil {
		t.Fatalf("GenerateKotlinOkHttpCommand() error = %v", err)
	}

	contains := []string{"import okhttp3.*", "OkHttpClient()", "Request.Builder()"}
	for _, substr := range contains {
		if !strings.Contains(code, substr) {
			t.Errorf("GenerateKotlinOkHttpCommand() result should contain %q", substr)
		}
	}
}

func TestGenerateJavaOkHttpCommand(t *testing.T) {
	svc := &Service{}
	spec := &domain.HTTPRequestSpec{
		Method: domain.RequestMethodPOST,
		URL:    "https://api.example.com/resource",
		Request: &domain.HTTPRequest{
			Body: domain.Body{
				Type: "json",
				Data: `{"name": "Test"}`,
			},
		},
	}

	code, err := svc.GenerateJavaOkHttpCommand(spec)
	if err != nil {
		t.Fatalf("GenerateJavaOkHttpCommand() error = %v", err)
	}

	contains := []string{"import okhttp3.*", "OkHttpClient", "public class ApiRequest"}
	for _, substr := range contains {
		if !strings.Contains(code, substr) {
			t.Errorf("GenerateJavaOkHttpCommand() result should contain %q", substr)
		}
	}
}

func TestGenerateRubyNetHttpCommand(t *testing.T) {
	svc := &Service{}
	spec := &domain.HTTPRequestSpec{
		Method: domain.RequestMethodPOST,
		URL:    "https://api.example.com/resource",
		Request: &domain.HTTPRequest{
			Body: domain.Body{
				Type: "json",
				Data: `{"name": "Test"}`,
			},
		},
	}

	code, err := svc.GenerateRubyNetHttpCommand(spec)
	if err != nil {
		t.Fatalf("GenerateRubyNetHttpCommand() error = %v", err)
	}

	contains := []string{"require 'net/http'", "URI.parse", "Net::HTTP::Post"}
	for _, substr := range contains {
		if !strings.Contains(code, substr) {
			t.Errorf("GenerateRubyNetHttpCommand() result should contain %q", substr)
		}
	}
}

func TestGenerateDotNetHttpClientCommand(t *testing.T) {
	svc := &Service{}
	spec := &domain.HTTPRequestSpec{
		Method: domain.RequestMethodPOST,
		URL:    "https://api.example.com/resource",
		Request: &domain.HTTPRequest{
			Body: domain.Body{
				Type: "json",
				Data: `{"name": "Test"}`,
			},
		},
	}

	code, err := svc.GenerateDotNetHttpClientCommand(spec)
	if err != nil {
		t.Fatalf("GenerateDotNetHttpClientCommand() error = %v", err)
	}

	contains := []string{"using System.Net.Http", "HttpClient", "HttpRequestMessage"}
	for _, substr := range contains {
		if !strings.Contains(code, substr) {
			t.Errorf("GenerateDotNetHttpClientCommand() result should contain %q", substr)
		}
	}
}
