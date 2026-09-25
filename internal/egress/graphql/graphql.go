package graphql

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/cookies"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/util"
	"github.com/chapar-rest/chapar/internal/variables"
	"github.com/chapar-rest/chapar/version"
)

type Service struct {
	cookies *cookies.Store
}

// SetCookieStore makes requests send and store cookies in the jar of the
// environment they are sent with. A nil store disables cookie handling.
func (s *Service) SetCookieStore(store *cookies.Store) {
	s.cookies = store
}

// nolint: gocyclo
func (s *Service) sendRequest(req *domain.GraphQLRequestSpec, e *domain.Environment) (*egress.Response, error) {
	// prepare request
	// - apply environment
	// - apply variables
	// - apply authentication (if any) is not already applied to the headers

	vars := variables.GetVariables()
	variables.ApplyToGraphQLRequest(vars, req)

	if e != nil {
		variables.ApplyToEnv(vars, &e.Spec)
		e.ApplyToGraphQLRequest(req)
	}

	// Prepare GraphQL request body
	requestBody := map[string]interface{}{
		"query": req.Query,
	}

	// Parse variables JSON string to map
	if req.Variables != "" && req.Variables != "{}" {
		var variablesMap map[string]interface{}
		if err := json.Unmarshal([]byte(req.Variables), &variablesMap); err != nil {
			return nil, fmt.Errorf("invalid variables JSON: %w", err)
		}
		requestBody["variables"] = variablesMap
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequest("POST", req.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	// Set Content-Type header for GraphQL
	httpReq.Header.Set("Content-Type", "application/json")

	// apply headers
	for _, h := range req.Headers {
		if !h.Enable {
			continue
		}
		httpReq.Header.Add(h.Key, h.Value)
	}

	// apply authentication
	if req.Auth != (domain.Auth{}) {
		if req.Auth.Type == domain.AuthTypeToken {
			if req.Auth.TokenAuth != nil && req.Auth.TokenAuth.Token != "" {
				httpReq.Header.Add("Authorization", "Bearer "+req.Auth.TokenAuth.Token)
			}
		}

		if req.Auth.Type == domain.AuthTypeBasic {
			if req.Auth.BasicAuth != nil && req.Auth.BasicAuth.Username != "" && req.Auth.BasicAuth.Password != "" {
				httpReq.SetBasicAuth(req.Auth.BasicAuth.Username, req.Auth.BasicAuth.Password)
			}
		}

		if req.Auth.Type == domain.AuthTypeAPIKey {
			if req.Auth.APIKeyAuth != nil && req.Auth.APIKeyAuth.Key != "" && req.Auth.APIKeyAuth.Value != "" {
				httpReq.Header.Add(req.Auth.APIKeyAuth.Key, req.Auth.APIKeyAuth.Value)
			}
		}
	}

	// send request
	globalConfig := prefs.GetGlobalConfig()

	start := time.Now()

	client := &http.Client{
		Timeout: time.Duration(globalConfig.Spec.General.RequestTimeoutSec) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:           10,
			MaxResponseHeaderBytes: int64(globalConfig.Spec.General.ResponseSizeMb * 1024 * 1024),
		},
	}
	if !globalConfig.Spec.General.FollowRedirects {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	if !globalConfig.Spec.General.VaidateTLSCertificates {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	if globalConfig.Spec.General.HTTPVersion == "http/2" {
		client.Transport = egress.NewHTTP2Transport(globalConfig.Spec.General.ResponseSizeMb*1024*1024, nil)
	}

	if globalConfig.Spec.General.SendNoCacheHeader {
		httpReq.Header.Add("Cache-Control", "no-cache")
	}

	if globalConfig.Spec.General.SendChaparAgentHeader {
		httpReq.Header.Add("User-Agent", version.GetAgentName())
	}

	envID := ""
	if e != nil {
		envID = e.ID()
	}
	jar, err := s.cookies.Attach(envID, client, httpReq)
	if err != nil {
		return nil, fmt.Errorf("cookie jar: %w", err)
	}

	traceCol := egress.NewHTTPTraceCollector()
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), traceCol.ClientTrace()))

	res, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	// read body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	downloadEnd := time.Now()

	// measure time
	elapsed := time.Since(start)

	// handle response
	response := &egress.Response{
		StatusCode:      res.StatusCode,
		ResponseHeaders: map[string]string{},
		RequestHeaders:  map[string]string{},
		Cookies:         res.Cookies(),
		Body:            body,
		TimePassed:      elapsed,
		Timeline:        traceCol.Steps(downloadEnd),
	}

	// handle headers
	for k, v := range res.Header {
		response.ResponseHeaders[k] = strings.Join(v, ", ")
	}

	for k, v := range httpReq.Header {
		response.RequestHeaders[k] = strings.Join(v, ", ")
	}

	if jar != nil {
		response.CookieEvents = jar.Events()
		response.SentCookies = jar.Sent()
	}

	ct := response.ResponseHeaders["Content-Type"]
	if ct == "" {
		ct = response.ResponseHeaders["content-type"]
	}
	kind, pretty, jsonStr, isJSON := util.ApplyBodyFormat(ct, body)
	response.BodyKind = kind
	response.Pretty = pretty
	response.IsJSON = isJSON
	response.JSON = jsonStr

	return response, nil
}
