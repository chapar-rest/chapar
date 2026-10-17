package mitm

import (
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

// State is where a captured flow is in its lifetime.
type State int

const (
	StatePending State = iota
	StateDone
	StateFailed
)

// Kind groups flows by content for the page's filter chips.
type Kind string

const (
	KindJSON   Kind = "json"
	KindXML    Kind = "xml"
	KindHTML   Kind = "html"
	KindCSS    Kind = "css"
	KindJS     Kind = "js"
	KindImage  Kind = "image"
	KindMedia  Kind = "media"
	KindFont   Kind = "font"
	KindForm   Kind = "form"
	KindText   Kind = "text"
	KindWS     Kind = "ws"
	KindTunnel Kind = "tunnel"
	KindOther  Kind = "other"
)

// Flow is one captured request and its response. A published Flow is
// never modified: the store swaps in a new value instead, so the UI can
// read a snapshot without locking.
type Flow struct {
	ID    uint64
	Start time.Time
	End   time.Time
	State State
	Error string

	Method string
	URL    string
	Scheme string
	Host   string // without port
	Port   string
	Path   string // path and query
	Proto  string

	ReqHeader http.Header
	ReqBody   []byte
	// ReqStreamed is set when the body was larger than the capture
	// limit and passed through without being kept.
	ReqStreamed bool

	Status     int
	RespHeader http.Header
	// RespBody is decoded (gzip, br, deflate, zstd removed).
	RespBody     []byte
	RespSize     int64 // bytes on the wire, before decoding
	RespStreamed bool

	Kind Kind
	// Tunnel marks a CONNECT that was passed through without decrypting.
	Tunnel bool

	ClientAddr string
	App        string
}

// Duration is how long the flow took, or has taken so far.
func (f *Flow) Duration() time.Duration {
	if f.End.IsZero() {
		return time.Since(f.Start)
	}
	return f.End.Sub(f.Start)
}

// HTTPS reports whether the flow was sent over TLS.
func (f *Flow) HTTPS() bool { return f.Scheme == "https" || f.Scheme == "wss" || f.Tunnel }

// Classify picks a Kind from the response content type, falling back
// to the request path's extension.
func Classify(ct, urlPath string) Kind {
	if ct != "" {
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil {
			mt = strings.ToLower(strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]))
		}
		switch {
		case mt == "application/json" || strings.HasSuffix(mt, "+json") || mt == "application/x-ndjson":
			return KindJSON
		case mt == "text/html" || mt == "application/xhtml+xml":
			return KindHTML
		case mt == "text/css":
			return KindCSS
		case strings.Contains(mt, "javascript") || mt == "application/ecmascript":
			return KindJS
		case mt == "application/xml" || mt == "text/xml" || strings.HasSuffix(mt, "+xml"):
			return KindXML
		case strings.HasPrefix(mt, "image/"):
			return KindImage
		case strings.HasPrefix(mt, "audio/") || strings.HasPrefix(mt, "video/"):
			return KindMedia
		case strings.HasPrefix(mt, "font/") || strings.Contains(mt, "font"):
			return KindFont
		case mt == "application/x-www-form-urlencoded" || mt == "multipart/form-data":
			return KindForm
		case strings.HasPrefix(mt, "text/"):
			return KindText
		}
	}
	switch strings.ToLower(path.Ext(urlPath)) {
	case ".json":
		return KindJSON
	case ".html", ".htm":
		return KindHTML
	case ".css":
		return KindCSS
	case ".js", ".mjs":
		return KindJS
	case ".xml":
		return KindXML
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".avif", ".bmp":
		return KindImage
	case ".mp4", ".webm", ".mp3", ".m4a", ".ogg", ".wav", ".m3u8", ".ts":
		return KindMedia
	case ".woff", ".woff2", ".ttf", ".otf", ".eot":
		return KindFont
	}
	return KindOther
}
