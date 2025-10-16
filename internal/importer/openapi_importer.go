package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

// ImportOpenAPISpec imports an OpenAPI 3.0 specification using kin-openapi library
func ImportOpenAPISpec(data []byte, repo repository.RepositoryV2) error {
	// Create a loader
	loader := openapi3.NewLoader()

	// Load the OpenAPI spec
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Validate the spec
	if err := doc.Validate(loader.Context); err != nil {
		return fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	// Create collection from OpenAPI spec
	collectionName := doc.Info.Title
	if collectionName == "" {
		collectionName = "OpenAPI Import"
	}

	col := domain.NewCollection(collectionName)
	if err := repo.CreateCollection(col); err != nil {
		return fmt.Errorf("error saving collection: %w", err)
	}

	// Get base URL from servers
	baseURL := ""
	if len(doc.Servers) > 0 {
		baseURL = doc.Servers[0].URL
	}

	// Convert paths to requests
	var requests []*domain.Request
	for path, pathItem := range doc.Paths.Map() {
		operations := []struct {
			method string
			op     *openapi3.Operation
		}{
			{"GET", pathItem.Get},
			{"POST", pathItem.Post},
			{"PUT", pathItem.Put},
			{"DELETE", pathItem.Delete},
			{"PATCH", pathItem.Patch},
		}

		for _, op := range operations {
			if op.op == nil {
				continue
			}

			requestName := op.op.Summary
			if requestName == "" {
				requestName = fmt.Sprintf("%s %s", op.method, path)
			}

			req := &domain.Request{
				ApiVersion: "v1",
				Kind:       "Request",
				MetaData: domain.RequestMeta{
					ID:   uuid.NewString(),
					Name: requestName,
					Type: domain.RequestTypeHTTP,
				},
				Spec: domain.RequestSpec{
					HTTP: &domain.HTTPRequestSpec{
						Method: op.method,
						URL:    baseURL + path,
						Request: &domain.HTTPRequest{
							Headers:     []domain.KeyValue{},
							PathParams:  []domain.KeyValue{},
							QueryParams: []domain.KeyValue{},
							Body: domain.Body{
								Type: domain.RequestBodyTypeNone,
								Data: "",
							},
						},
					},
				},
			}

			// Add parameters as headers, query params, or path params
			for _, param := range op.op.Parameters {
				if param.Value == nil {
					continue
				}

				paramValue := param.Value
				switch paramValue.In {
				case "header":
					req.Spec.HTTP.Request.Headers = append(req.Spec.HTTP.Request.Headers, domain.KeyValue{
						ID:     uuid.NewString(),
						Key:    paramValue.Name,
						Value:  "",
						Enable: paramValue.Required,
					})
				case "query":
					req.Spec.HTTP.Request.QueryParams = append(req.Spec.HTTP.Request.QueryParams, domain.KeyValue{
						ID:     uuid.NewString(),
						Key:    paramValue.Name,
						Value:  "",
						Enable: paramValue.Required,
					})
				case "path":
					req.Spec.HTTP.Request.PathParams = append(req.Spec.HTTP.Request.PathParams, domain.KeyValue{
						ID:     uuid.NewString(),
						Key:    paramValue.Name,
						Value:  "",
						Enable: paramValue.Required,
					})
				}
			}

			// Add request body if present
			if op.op.RequestBody != nil && op.op.RequestBody.Value != nil {
				requestBody := op.op.RequestBody.Value

				// Find the best content type to use (prefer JSON, then others)
				var selectedContentType string
				var selectedContent *openapi3.MediaType

				// First, try to find JSON content
				for contentType, content := range requestBody.Content {
					if strings.Contains(contentType, "application/json") {
						selectedContentType = contentType
						selectedContent = content
						break
					}
				}

				// If no JSON found, use the first available content type
				if selectedContentType == "" {
					for contentType, content := range requestBody.Content {
						selectedContentType = contentType
						selectedContent = content
						break
					}
				}

				if selectedContentType != "" && selectedContent != nil {
					// Set body type based on content type
					bodyType := determineBodyType(selectedContentType)
					req.Spec.HTTP.Request.Body.Type = bodyType

					// Generate example data based on content type and schema
					if selectedContent.Schema != nil && selectedContent.Schema.Value != nil {
						example := generateExampleFromSchema(selectedContent.Schema.Value)
						switch bodyType {
						case domain.RequestBodyTypeJSON:
							req.Spec.HTTP.Request.Body.Data = example
						case domain.RequestBodyTypeXML:
							req.Spec.HTTP.Request.Body.Data = generateXMLExample(selectedContent.Schema.Value)
						case domain.RequestBodyTypeText:
							req.Spec.HTTP.Request.Body.Data = generateTextExample(selectedContent.Schema.Value)
						default:
							req.Spec.HTTP.Request.Body.Data = example
						}
					} else {
						// No schema, set default based on content type
						req.Spec.HTTP.Request.Body.Data = getDefaultBodyForType(bodyType)
					}

					// Set content type header
					req.Spec.HTTP.Request.Headers = append(req.Spec.HTTP.Request.Headers, domain.KeyValue{
						ID:     uuid.NewString(),
						Key:    "Content-Type",
						Value:  selectedContentType,
						Enable: true,
					})
				}
			}

			req.SetDefaultValues()
			requests = append(requests, req)
		}
	}

	// Save all requests
	for _, req := range requests {
		if err := repo.CreateRequest(req, col); err != nil {
			return fmt.Errorf("error saving request: %w", err)
		}
	}

	return nil
}

// determineBodyType maps OpenAPI content types to Chapar body types
func determineBodyType(contentType string) string {
	switch {
	case strings.Contains(contentType, "application/json"):
		return domain.RequestBodyTypeJSON
	case strings.Contains(contentType, "application/xml") || strings.Contains(contentType, "text/xml"):
		return domain.RequestBodyTypeXML
	case strings.Contains(contentType, "text/plain"):
		return domain.RequestBodyTypeText
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		return domain.RequestBodyTypeUrlencoded
	case strings.Contains(contentType, "multipart/form-data"):
		return domain.RequestBodyTypeFormData
	default:
		// Default to JSON for unknown types
		return domain.RequestBodyTypeJSON
	}
}

// generateExampleFromSchema creates a JSON example from kin-openapi schema
func generateExampleFromSchema(schema *openapi3.Schema) string {
	if schema == nil {
		return ""
	}

	// Use example if available
	if schema.Example != nil {
		if exampleBytes, err := json.MarshalIndent(schema.Example, "", "  "); err == nil {
			return string(exampleBytes)
		}
	}

	// Generate basic example based on type
	if schema.Type != nil {
		switch {
		case schema.Type.Is("object"):
			if schema.Properties != nil {
				example := make(map[string]interface{})
				for key, prop := range schema.Properties {
					if prop.Value != nil {
						example[key] = generateExampleValue(prop.Value)
					}
				}
				if exampleBytes, err := json.MarshalIndent(example, "", "  "); err == nil {
					return string(exampleBytes)
				}
			}
			return "{}"
		case schema.Type.Is("array"):
			if schema.Items != nil && schema.Items.Value != nil {
				// Generate an array with one example item
				itemExample := generateExampleValue(schema.Items.Value)
				arrayExample := []interface{}{itemExample}
				if exampleBytes, err := json.MarshalIndent(arrayExample, "", "  "); err == nil {
					return string(exampleBytes)
				}
			}
			return "[]"
		case schema.Type.Is("string"):
			return `""`
		case schema.Type.Is("number") || schema.Type.Is("integer"):
			return "0"
		case schema.Type.Is("boolean"):
			return "false"
		default:
			return "{}"
		}
	}
	return "{}"
}

// generateExampleValue creates example value from kin-openapi schema
func generateExampleValue(schema *openapi3.Schema) interface{} {
	if schema == nil {
		return nil
	}

	// Use example if available
	if schema.Example != nil {
		return schema.Example
	}

	if schema.Type != nil {
		switch {
		case schema.Type.Is("string"):
			return ""
		case schema.Type.Is("number") || schema.Type.Is("integer"):
			return 0
		case schema.Type.Is("boolean"):
			return false
		case schema.Type.Is("array"):
			if schema.Items != nil && schema.Items.Value != nil {
				return []interface{}{generateExampleValue(schema.Items.Value)}
			}
			return []interface{}{}
		case schema.Type.Is("object"):
			if schema.Properties != nil {
				example := make(map[string]interface{})
				for key, prop := range schema.Properties {
					if prop.Value != nil {
						example[key] = generateExampleValue(prop.Value)
					}
				}
				return example
			}
			return make(map[string]interface{})
		default:
			return nil
		}
	}
	return nil
}

// generateXMLExample creates a simple XML example from schema
func generateXMLExample(schema *openapi3.Schema) string {
	if schema == nil {
		return ""
	}

	if schema.Example != nil {
		if exampleStr, ok := schema.Example.(string); ok {
			return exampleStr
		}
	}

	// Generate basic XML structure
	if schema.Type != nil && schema.Type.Is("object") && schema.Properties != nil {
		var xmlParts []string
		xmlParts = append(xmlParts, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
		xmlParts = append(xmlParts, "<root>")

		for key, prop := range schema.Properties {
			if prop.Value != nil {
				value := generateXMLValue(prop.Value)
				xmlParts = append(xmlParts, fmt.Sprintf("  <%s>%s</%s>", key, value, key))
			}
		}

		xmlParts = append(xmlParts, "</root>")
		return strings.Join(xmlParts, "\n")
	}

	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<root></root>"
}

// generateXMLValue creates XML value from schema
func generateXMLValue(schema *openapi3.Schema) string {
	if schema == nil {
		return ""
	}

	if schema.Example != nil {
		if exampleStr, ok := schema.Example.(string); ok {
			return exampleStr
		}
	}

	if schema.Type != nil {
		switch {
		case schema.Type.Is("string"):
			return ""
		case schema.Type.Is("number") || schema.Type.Is("integer"):
			return "0"
		case schema.Type.Is("boolean"):
			return "false"
		default:
			return ""
		}
	}
	return ""
}

// generateTextExample creates a text example from schema
func generateTextExample(schema *openapi3.Schema) string {
	if schema == nil {
		return ""
	}

	if schema.Example != nil {
		if exampleStr, ok := schema.Example.(string); ok {
			return exampleStr
		}
	}

	if schema.Type != nil {
		switch {
		case schema.Type.Is("string"):
			return ""
		case schema.Type.Is("number") || schema.Type.Is("integer"):
			return "0"
		case schema.Type.Is("boolean"):
			return "false"
		default:
			return ""
		}
	}
	return ""
}

// getDefaultBodyForType returns default body content for different types
func getDefaultBodyForType(bodyType string) string {
	switch bodyType {
	case domain.RequestBodyTypeJSON:
		return "{}"
	case domain.RequestBodyTypeXML:
		return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<root></root>"
	case domain.RequestBodyTypeText:
		return ""
	case domain.RequestBodyTypeFormData, domain.RequestBodyTypeUrlencoded:
		return ""
	default:
		return "{}"
	}
}
