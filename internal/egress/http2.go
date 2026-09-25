package egress

import (
	"crypto/tls"
	"net/http"
)

// maxHTTP2FrameSize is the largest frame size HTTP/2 allows (RFC 9113 §4.2).
const maxHTTP2FrameSize = 1<<24 - 1

// NewHTTP2Transport returns a transport that speaks only HTTP/2: over TLS for
// https URLs and with prior knowledge (h2c) for http ones. maxReadFrameSize is
// capped at the protocol limit; tlsConfig may be nil.
func NewHTTP2Transport(maxReadFrameSize int, tlsConfig *tls.Config) *http.Transport {
	var protocols http.Protocols
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)
	return &http.Transport{
		Protocols:       &protocols,
		TLSClientConfig: tlsConfig,
		HTTP2:           &http.HTTP2Config{MaxReadFrameSize: min(maxReadFrameSize, maxHTTP2FrameSize)},
	}
}
