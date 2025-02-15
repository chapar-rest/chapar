package importer

import (
	"encoding/json"
	"errors"
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

func ImportPostmanCollection(data []byte, repo repository.Repository) error {
	data = replaceVariables(data)

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		// it might be v2 of the collection. inform the user to convert it to v2.1
		if strings.Contains(string(data), "https://schema.getpostman.com/json/collection/v2.0.0/collection.json") {
			return errors.New("it seems like you are using v2 of the collection. Please export it as v2.1 and try again")
		}

		return fmt.Errorf("error parsing JSON: %w", err)
	}

	col := domain.NewCollection(collection.Info.Name)
	if err := repo.Create(col); err != nil {
		return fmt.Errorf("error saving collection: %w", err)
	}

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
		if apiKey != nil {
			req.Spec.HTTP.Request.Auth = domain.Auth{
				Type:       "API Key",
				APIKeyAuth: apiKey,
			}
		}

		req.SetDefaultValues()

		if err := repo.CreateRequestInCollection(col, req); err != nil {
			return fmt.Errorf("error saving request: %w", err)
		}
	}

	return nil
}

func ImportPostmanCollectionFromFile(filePath string, repo repository.Repository) error {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return ImportPostmanCollection(fileContent, repo)
}

func findApiKey(coll PostmanCollection) *domain.APIKeyAuth {
	if coll.Auth == nil {
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

func ImportPostmanEnvironment(data []byte, repo repository.Repository) error {
	data = replaceVariables(data)

	var env PostmanEnvironment
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("error parsing JSON: %w", err)
	}

	environment := domain.NewEnvironment(env.Name)
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

	if err := repo.Create(environment); err != nil {
		return fmt.Errorf("error saving environment: %w", err)
	}

	return nil
}

func ImportPostmanEnvironmentFromFile(filePath string, repo repository.Repository) error {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %v\n", err)
	}

	return ImportPostmanEnvironment(fileContent, repo)
}

func replaceVariables(input []byte) []byte {
	for k, v := range variablesMap {
		input = []byte(strings.ReplaceAll(string(input), k, v))
	}

	return input
}
