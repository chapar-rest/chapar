package domain

import "github.com/google/uuid"

type WebSocketSpec struct {
	URL string `yaml:"url"`

	LastUsedEnvironment LastUsedEnvironment `yaml:"lastUsedEnvironment"`

	Request *WebSocketRequest `yaml:"request"`
}

type WebSocketRequest struct {
	Headers     []KeyValue `yaml:"headers"`
	QueryParams []KeyValue `yaml:"queryParams"`

	BodyType string `yaml:"bodyType"`
	Body     string `yaml:"body"`

	Settings WebSocketSettings `yaml:"settings"`
}

type WebSocketSettings struct {
	HandShakeTimeoutMilliseconds     int `yaml:"handShakeTimeoutMilliseconds"`
	ReconnectionAttempts             int `yaml:"reconnectionAttempts"`
	ReconnectionIntervalMilliseconds int `yaml:"reconnectionIntervalMilliseconds"`
	MessageSizeLimitMB               int `yaml:"messageSizeLimitMB"`
}

func NewWebSocketRequest(name string) *Request {
	return &Request{
		ApiVersion: ApiVersion,
		Kind:       KindRequest,
		MetaData: RequestMeta{
			ID:   uuid.NewString(),
			Name: name,
			Type: RequestTypeHTTP,
		},
		Spec: RequestSpec{
			HTTP: &HTTPRequestSpec{
				Method: RequestMethodGET,
				URL:    "https://example.com",
				Request: &HTTPRequest{
					Headers: []KeyValue{
						{Key: "Content-Type", Value: "application/json"},
					},
				},
			},
		},
	}
}
