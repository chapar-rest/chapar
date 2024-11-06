package domain

import (
	"fmt"
	"testing"
)

func TestGeneratePythonRequest(t *testing.T) {
	requestSpec := HTTPRequestSpec{
		Method: "POST",
		URL:    "https://api.example.com/resource",
		Request: &HTTPRequest{
			Headers: []KeyValue{
				{Key: "Authorization", Value: "Bearer token123"},
				{Key: "Content-Type", Value: "application/json"},
			},
			QueryParams: []KeyValue{
				{Key: "filter", Value: "active"},
			},
			Body: Body{
				Type: "json",
				Data: `{"name": "Test", "age": 30}`,
			},
		},
	}

	// Generate Python request code
	pythonCode, err := GeneratePythonRequest(requestSpec)
	if err != nil {
		fmt.Println("Error generating Python code:", err)
		return
	}

	// Print the generated Python code
	fmt.Println(pythonCode)
}
