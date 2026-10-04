package domain

import (
	"github.com/google/uuid"
)

// WebSocket message formats. Binary bodies are written as base64.
const (
	WebSocketFormatText   = "text"
	WebSocketFormatJSON   = "json"
	WebSocketFormatBinary = "binary"
)

// WebSocketFormats lists the formats the message editor offers.
var WebSocketFormats = []string{WebSocketFormatText, WebSocketFormatJSON, WebSocketFormatBinary}

const defaultWebSocketURL = "wss://mocks.chapar.rest/ws/echo"

// WebSocketRequestSpec is a WebSocket connection: the handshake, the message
// being written, and messages saved to send again.
type WebSocketRequestSpec struct {
	URL     string     `yaml:"url"`
	Headers []KeyValue `yaml:"headers"`
	Auth    Auth       `yaml:"auth"`
	// Subprotocols are offered in Sec-WebSocket-Protocol, in order.
	Subprotocols []string `yaml:"subprotocols,omitempty"`

	// Message is the draft in the message editor.
	Message       string             `yaml:"message"`
	MessageFormat string             `yaml:"messageFormat"`
	SavedMessages []WebSocketMessage `yaml:"savedMessages,omitempty"`

	Settings WebSocketSettings `yaml:"settings"`

	LastUsedEnvironment LastUsedEnvironment `yaml:"lastUsedEnvironment"`

	PreRequest PreRequest `yaml:"preRequest"`
}

// WebSocketMessage is a message saved with the request.
type WebSocketMessage struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	Body   string `yaml:"body"`
	Format string `yaml:"format"`
}

// WebSocketSettings tune one connection. Zero values mean the defaults.
type WebSocketSettings struct {
	// ConnectTimeoutSec bounds the handshake; 0 uses the request timeout
	// from settings.
	ConnectTimeoutSec int `yaml:"connectTimeoutSec,omitempty"`
	// PingIntervalSec sends a ping this often while connected; 0 never pings.
	PingIntervalSec int `yaml:"pingIntervalSec,omitempty"`
	// Compression offers permessage-deflate.
	Compression bool `yaml:"compression,omitempty"`
}

func (w *WebSocketRequestSpec) Clone() *WebSocketRequestSpec {
	clone := *w

	if len(w.Headers) > 0 {
		clone.Headers = make([]KeyValue, len(w.Headers))
		copy(clone.Headers, w.Headers)
	}
	if len(w.Subprotocols) > 0 {
		clone.Subprotocols = append([]string(nil), w.Subprotocols...)
	}
	if len(w.SavedMessages) > 0 {
		clone.SavedMessages = append([]WebSocketMessage(nil), w.SavedMessages...)
	}
	if w.Auth != (Auth{}) {
		clone.Auth = w.Auth.Clone()
	}
	if w.PreRequest.TriggerRequest != nil {
		tr := *w.PreRequest.TriggerRequest
		clone.PreRequest.TriggerRequest = &tr
	}
	return &clone
}

func (w *WebSocketRequestSpec) GetPreRequest() PreRequest {
	if w != nil {
		return w.PreRequest
	}
	return PreRequest{}
}

func NewWebSocketRequest(name string) *Request {
	return &Request{
		ApiVersion: ApiVersion,
		Kind:       KindRequest,
		MetaData: RequestMeta{
			ID:   uuid.NewString(),
			Name: name,
			Type: RequestTypeWebSocket,
		},
		Spec: RequestSpec{
			WebSocket: &WebSocketRequestSpec{
				URL:           defaultWebSocketURL,
				MessageFormat: WebSocketFormatText,
				Auth:          Auth{Type: AuthTypeNone},
				PreRequest:    PreRequest{Type: PrePostTypeNone},
			},
		},
	}
}

func (r *Request) SetDefaultValuesForWebSocket() {
	if r.Spec.WebSocket == nil {
		r.Spec.WebSocket = &WebSocketRequestSpec{}
	}
	w := r.Spec.WebSocket
	if w.URL == "" {
		w.URL = defaultWebSocketURL
	}
	if w.MessageFormat == "" {
		w.MessageFormat = WebSocketFormatText
	}
	if w.Auth == (Auth{}) {
		w.Auth = Auth{Type: AuthTypeNone}
	}
	if w.PreRequest == (PreRequest{}) {
		w.PreRequest = PreRequest{Type: PrePostTypeNone}
	}
}
