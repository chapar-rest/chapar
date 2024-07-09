package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

var variablesMap = map[string]string{
	"{{$guid}}":         "{{randomUUID4}}",
	"{{$timestamp}}":    "{{unixTimestamp}}",
	"{{$isoTimestamp}}": "{{timeNow}}",
}

// PostmanCollection represents the structure of a Postman exported JSON
type PostmanCollection struct {
	Info struct {
		Name string `json:"name"`
	} `json:"info"`
	Item []RequestItem `json:"item"`
	Auth *struct {
		Type   string   `json:"type"`
		ApiKey []ApiKey `json:"apikey,omitempty"`
	} `json:"auth"`
}

type ApiKey struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type RequestItem struct {
	Name string `json:"name"`
	// if request is a folder, it will have an item array
	Item    []RequestItem `json:"item,omitempty"`
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

type PostmanEnvironment struct {
	ID     string                       `json:"id"`
	Name   string                       `json:"name"`
	Values []PostmanEnvironmentVariable `json:"values"`
}

type PostmanEnvironmentVariable struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
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
			Type: domain.RequestTypeHTTP, // Modify based on actual method
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

func ImportPostmanCollection(data []byte) error {
	filesystem, err := repository.NewFilesystem()
	if err != nil {
		fmt.Printf("Error creating filesystem: %v\n", err)
		return err
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return err
	}

	col := domain.NewCollection(collection.Info.Name)
	fp, err := filesystem.GetNewCollectionDir(collection.Info.Name)
	if err != nil {
		fmt.Printf("Error getting new collection directory: %v\n", err)
		return err
	}
	col.FilePath = fp.Path
	col.MetaData.Name = fp.NewName

	if err := filesystem.UpdateCollection(col); err != nil {
		fmt.Printf("Error saving collection: %v\n", err)
		return err
	}

	fmt.Println("collection auth", collection.Auth)

	// Convert each item in the collection to our Request structure
	var requests = make([]*domain.Request, 0, len(collection.Item))
	for _, item := range collection.Item {
		if item.Item != nil {
			// Convert the folder to a collection
			// This is a simplification, in reality we need to handle nested folders
			// and convert them to nested collections
			for _, subItem := range item.Item {
				requests = append(requests, convertItemToRequest(subItem))
			}
			continue
		}

		requests = append(requests, convertItemToRequest(item))
	}

	apiKey := findApiKey(collection)

	for _, req := range requests {
		fp, err := filesystem.GetCollectionRequestNewFilePath(col, req.MetaData.Name)
		if err != nil {
			fmt.Printf("Error getting new request file path: %v\n", err)
			continue
		}

		if apiKey != nil {
			req.Spec.HTTP.Request.Auth = domain.Auth{
				Type:       "API Key",
				APIKeyAuth: apiKey,
			}
		}

		req.FilePath = fp.Path
		req.MetaData.Name = fp.NewName

		req.SetDefaultValues()

		// Save the request to a file
		if err := filesystem.UpdateRequest(req); err != nil {
			return err
		}

		// Replace variables in the request file
		if err := findAndReplaceVariables(req.FilePath); err != nil {
			return err
		}
	}

	return nil
}

func ImportPostmanCollectionFromFile(filePath string) error {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return err
	}

	return ImportPostmanCollection(fileContent)
}

func findAndReplaceVariables(filename string) error {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	for k, v := range variablesMap {
		fileContent = []byte(strings.ReplaceAll(string(fileContent), k, v))
	}

	return os.WriteFile(filename, fileContent, 0644)
}

func findApiKey(coll PostmanCollection) *domain.APIKeyAuth {
	if coll.Auth == nil {
		fmt.Printf("auth is nil")
		return nil
	}

	var out domain.APIKeyAuth

	if coll.Auth.Type == "apikey" && len(coll.Auth.ApiKey) > 0 {
		if i, ok := findInApiKey(coll.Auth.ApiKey, func(val ApiKey) bool { return val.Key == "value" }); ok {
			out.Value = coll.Auth.ApiKey[i].Value
		}

		if i, ok := findInApiKey(coll.Auth.ApiKey, func(val ApiKey) bool { return val.Key == "key" }); ok {
			out.Key = coll.Auth.ApiKey[i].Value
		}
	}

	if out.Key == "" || out.Value == "" {
		fmt.Println("fail to find key or value", out)
		return nil
	}

	return &out
}

func findInApiKey(arr []ApiKey, filter func(apiKey ApiKey) bool) (int, bool) {
	for i, v := range arr {
		if filter(v) {
			return i, true
		}
	}
	return 0, false
}

func ImportPostmanEnvironment(data []byte) error {
	filesystem, err := repository.NewFilesystem()
	if err != nil {
		fmt.Printf("Error creating filesystem: %v\n", err)
		return err
	}

	var env PostmanEnvironment
	if err := json.Unmarshal(data, &env); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return err
	}

	// Convert Postman environment to our Environment structure
	environment := domain.NewEnvironment(env.Name)
	fp, err := filesystem.GetNewEnvironmentFilePath(env.Name)
	if err != nil {
		fmt.Printf("Error getting new environment file path: %v\n", err)
		return err
	}

	environment.FilePath = fp.Path
	environment.MetaData.Name = fp.NewName

	// Convert each variable in the Postman environment to our KeyValue structure
	var variables = make([]domain.KeyValue, 0, len(env.Values))
	for _, variable := range env.Values {
		variables = append(variables, domain.KeyValue{
			ID:     uuid.NewString(),
			Key:    variable.Key,
			Value:  variable.Value,
			Enable: variable.Enabled,
		})
	}

	environment.Spec.Values = variables

	if err := filesystem.UpdateEnvironment(environment); err != nil {
		fmt.Printf("Error saving environment: %v\n", err)
		return err
	}

	// Replace variables in the request file
	if err := findAndReplaceVariables(environment.FilePath); err != nil {
		return err
	}

	return nil
}

func ImportPostmanEnvironmentFromFile(filePath string) error {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return err
	}

	return ImportPostmanEnvironment(fileContent)
}
