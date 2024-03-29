package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/repository"
)

// PostmanCollection represents the structure of a Postman exported JSON
type PostmanCollection struct {
	Info struct {
		Name string `json:"name"`
	} `json:"info"`
	Item []RequestItem `json:"item"`
}

// KeyValue for headers, pathParams, queryParams, etc.
type KeyValue struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// Request, RequestMeta, RequestSpec, and other struct definitions follow your Golang struct definitions...

type RequestItem struct {
	Name    string `json:"name"`
	Request struct {
		Method string `json:"method"`
		Header []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"header"`
		Body struct {
			Mode string `json:"mode"`
			Raw  string `json:"raw"`
		} `json:"body"`
		URL struct {
			Raw string `json:"raw"`
		} `json:"url"`
	} `json:"request"`
}

// Convert a Postman collection item to our Request structure
func convertItemToRequest(item RequestItem) *domain.Request {
	// Initialize the Request object with basic information
	req := &domain.Request{
		ApiVersion: "v1",
		Kind:       "Request",
		MetaData: domain.RequestMeta{
			ID:   uuid.NewString(), // Generate or assign ID
			Name: item.Name,
			Type: domain.RequestMethodGET, // Modify based on actual method
		},
		Spec: domain.RequestSpec{
			HTTP: &domain.HTTPRequestSpec{
				Method: item.Request.Method,
				URL:    item.Request.URL.Raw,
				Request: &domain.HTTPRequest{
					Headers: []domain.KeyValue{},
					Body: domain.Body{
						Type: "JSON", // Simplification, real mapping needed
						Data: item.Request.Body.Raw,
					},
				},
			},
		},
	}

	// Convert headers
	for _, header := range item.Request.Header {
		req.Spec.HTTP.Request.Headers = append(req.Spec.HTTP.Request.Headers, domain.KeyValue{
			ID:     uuid.NewString(),
			Key:    header.Key,
			Value:  header.Value,
			Enable: true,
		})
	}

	return req
}

func main() {
	// Assume the first argument is the path to the Postman JSON file
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run script.go <path_to_postman_collection.json>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	filesystem := &repository.Filesystem{}

	var collection PostmanCollection
	if err := json.Unmarshal(fileContent, &collection); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	col := domain.NewCollection(collection.Info.Name)
	fp, err := filesystem.GetNewCollectionDir(collection.Info.Name)
	if err != nil {
		fmt.Printf("Error getting new collection directory: %v\n", err)
		os.Exit(1)
	}
	col.FilePath = fp.Path
	col.MetaData.Name = fp.NewName

	if err := filesystem.UpdateCollection(col); err != nil {
		fmt.Printf("Error saving collection: %v\n", err)
		os.Exit(1)
	}

	// Convert each item in the collection to our Request structure
	var requests []*domain.Request
	for _, item := range collection.Item {
		requests = append(requests, convertItemToRequest(item))
	}

	for _, req := range requests {
		fp, err := filesystem.GetCollectionRequestNewFilePath(col, req.MetaData.Name)
		if err != nil {
			fmt.Printf("Error getting new request file path: %v\n", err)
			continue
		}

		req.FilePath = fp.Path
		req.MetaData.Name = fp.NewName

		// Save the request to a file
		if err := filesystem.UpdateRequest(req); err != nil {
			fmt.Printf("Error saving request: %v\n", err)
		}
	}
}
