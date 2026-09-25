package egress

import (
	"net/http"
	"time"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
)

// Timeline step phase constants.
const (
	TimelinePhaseApp     = "app"
	TimelinePhaseNetwork = "network"
)

// TimelineStep is one timed stage of a request (app pipeline or network).
type TimelineStep struct {
	Name     string
	Phase    string // TimelinePhaseApp | TimelinePhaseNetwork
	Duration time.Duration
	Detail   string
	Err      string
}

type Response struct {
	// http and graphql
	StatusCode      int
	ResponseHeaders map[string]string
	RequestHeaders  map[string]string
	Cookies         []*http.Cookie

	// CookieEvents is what the cookie jar did with each Set-Cookie header,
	// including those on redirect hops. SentCookies are the jar cookies that
	// were attached to the request. Both are empty when no jar was used.
	CookieEvents []cookies.Event
	SentCookies  []*http.Cookie

	// grpc
	RequestMetadata  []domain.KeyValue
	ResponseMetadata []domain.KeyValue
	Trailers         []domain.KeyValue
	Size             int
	Error            error

	StatueCode int
	Status     string

	Body       []byte
	TimePassed time.Duration
	IsJSON     bool
	JSON       string

	// Pretty is a formatted body for display when available; Body stays raw.
	Pretty   string
	BodyKind string // util.BodyKindJSON | XML | HTML | Text

	Timeline []TimelineStep
}
