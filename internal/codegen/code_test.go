package codegen

import (
	"fmt"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

func TestGeneratePythonRequest(t *testing.T) {
	requestSpec := &domain.HTTPRequestSpec{
		Method: domain.RequestMethodPOST,
		URL:    "https://api.example.com/resource",
		Request: &domain.HTTPRequest{
			Headers: []domain.KeyValue{
				{Key: "Authorization", Value: "Bearer token123"},
				{Key: "Content-Type", Value: "application/json"},
			},
			QueryParams: []domain.KeyValue{
				{Key: "filter", Value: "active"},
			},
			Body: domain.Body{
				Type: "json",
				Data: `{"name": "Test", "age": 30}`,
			},
		},
	}

	svc := &Service{}

	// Generate Python request code
	pythonCode, err := svc.GeneratePythonRequest(requestSpec)
	if err != nil {
		fmt.Println("Error generating Python code:", err)
		return
	}

	// Print the generated Python code
	fmt.Println(pythonCode)
}
