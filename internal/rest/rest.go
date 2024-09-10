package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/internal/variables"
)

type Response struct {
	StatusCode int
	Headers    map[string]string
	Cookies    []*http.Cookie
	Body       []byte

	TimePassed time.Duration

	IsJSON bool
	JSON   string
}

type Service struct {
	requests     *state.Requests
	environments *state.Environments
}

func New(requests *state.Requests, environments *state.Environments) *Service {
	return &Service{
		requests:     requests,
		environments: environments,
	}
}

func (s *Service) SendRequest(requestID, activeEnvironmentID string) (*Response, error) {
	req := s.requests.GetRequest(requestID)
	if req == nil {
		return nil, fmt.Errorf("request with id %s not found", requestID)
	}

	// clone the request to make sure we do not modify the original request
	r := req.Clone()

	var activeEnvironment *domain.Environment
	// Get environment if provided
	if activeEnvironmentID != "" {
		activeEnvironment = s.environments.GetEnvironment(activeEnvironmentID)
		if activeEnvironment == nil {
			return nil, fmt.Errorf("environment with id %s not found", activeEnvironmentID)
		}
	}

	response, err := s.sendRequest(r.Spec.HTTP, activeEnvironment)
	if err != nil {
		return nil, err
	}

	// handle post request
	if err := s.handlePostRequest(r.Spec.HTTP.Request.PostRequest, response, activeEnvironment); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *Service) handlePostRequest(r domain.PostRequest, response *Response, env *domain.Environment) error {
	if r == (domain.PostRequest{}) {
		return nil
	}

	if response == nil {
		return nil
	}

	if r.Type == domain.PrePostTypeSetEnv {
		// only handle post request if the status code is the same as the one provided
		if response.StatusCode != r.PostRequestSet.StatusCode {
			return nil
		}

		switch r.PostRequestSet.From {
		case domain.PostRequestSetFromResponseBody:
			if err := s.handlePostRequestFromBody(r, response, env); err != nil {
				return err
			}
		case domain.PostRequestSetFromResponseHeader:
			if err := s.handlePostRequestFromHeader(r, response, env); err != nil {
				return err
			}
		case domain.PostRequestSetFromResponseCookie:
			if err := s.handlePostRequestFromCookie(r, response, env); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) handlePostRequestFromBody(r domain.PostRequest, response *Response, env *domain.Environment) error {
	// handle post request
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseBody {
		return nil
	}

	if response.JSON != "" && response.IsJSON {
		data, err := jsonpath.Get(response.JSON, r.PostRequestSet.FromKey)
		if err != nil {
			return err
		}

		if data == nil {
			return nil
		}

		if result, ok := data.(string); ok {
			if env != nil {
				env.SetKey(r.PostRequestSet.Target, result)

				if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Service) handlePostRequestFromHeader(r domain.PostRequest, response *Response, env *domain.Environment) error {
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseHeader {
		return nil
	}

	if result, ok := response.Headers[r.PostRequestSet.FromKey]; ok {
		if env != nil {
			env.SetKey(r.PostRequestSet.Target, result)

			if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) handlePostRequestFromCookie(r domain.PostRequest, response *Response, env *domain.Environment) error {
	if r.PostRequestSet.From != domain.PostRequestSetFromResponseCookie {
		return nil
	}

	for _, c := range response.Cookies {
		if c.Name == r.PostRequestSet.FromKey {
			if env != nil {
				env.SetKey(r.PostRequestSet.Target, c.Value)

				if err := s.environments.UpdateEnvironment(env, state.SourceRestService, false); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Service) sendRequest(req *domain.HTTPRequestSpec, e *domain.Environment) (*Response, error) {
	// prepare request
	// - apply environment
	// - apply variables
	// - apply authentication (if any) is not already applied to the headers

	if e == nil {
		applyVariables(req, nil)
	} else {
		env := e.Clone()
		applyVariables(req, &env.Spec)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, nil)
	if err != nil {
		return nil, err
	}

	// apply headers
	for _, h := range req.Request.Headers {
		if !h.Enable {
			continue
		}

		httpReq.Header.Add(h.Key, h.Value)
	}

	// apply path params as single brace
	for _, p := range req.Request.PathParams {
		if !p.Enable {
			continue
		}

		httpReq.URL.Path = strings.ReplaceAll(httpReq.URL.Path, "{"+p.Key+"}", p.Value)
	}

	// TODO queries are already assembled to the url, should we do it here instead?
	// apply query params
	// query := httpReq.URL.Query()
	// for _, q := range req.Request.QueryParams {
	//	query.Add(q.Key, q.Value)
	// }

	// httpReq.URL.RawQuery = query.Encode()

	if err := s.applyBody(req, httpReq); err != nil {
		return nil, err
	}

	// apply authentication
	if req.Request.Auth != (domain.Auth{}) {
		if req.Request.Auth.Type == domain.AuthTypeToken {
			if req.Request.Auth.TokenAuth != nil && req.Request.Auth.TokenAuth.Token != "" {
				httpReq.Header.Add("Authorization", "Bearer "+req.Request.Auth.TokenAuth.Token)
			}
		}

		if req.Request.Auth.Type == domain.AuthTypeBasic {
			if req.Request.Auth.BasicAuth != nil && req.Request.Auth.BasicAuth.Username != "" && req.Request.Auth.BasicAuth.Password != "" {
				httpReq.SetBasicAuth(req.Request.Auth.BasicAuth.Username, req.Request.Auth.BasicAuth.Password)
			}
		}

		if req.Request.Auth.Type == domain.AuthTypeAPIKey {
			if req.Request.Auth.APIKeyAuth != nil && req.Request.Auth.APIKeyAuth.Key != "" && req.Request.Auth.APIKeyAuth.Value != "" {
				httpReq.Header.Add(req.Request.Auth.APIKeyAuth.Key, req.Request.Auth.APIKeyAuth.Value)
			}
		}
	}

	// send request
	// - measure time
	// - handle response
	// - handle error
	// - handle cookies
	// - handle redirects
	// - handle status code

	// send request
	start := time.Now()
	res, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// read body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// measure time
	elapsed := time.Since(start)

	// handle response
	response := &Response{
		StatusCode: res.StatusCode,
		Headers:    map[string]string{},
		Cookies:    res.Cookies(),
		Body:       body,
		TimePassed: elapsed,
		IsJSON:     false,
	}

	if IsJSON(string(body)) {
		response.IsJSON = true
		if js, err := PrettyJSON(body); err != nil {
			return nil, err
		} else {
			response.JSON = js
		}
	}

	// handle headers
	for k, v := range res.Header {
		response.Headers[k] = strings.Join(v, ", ")
	}

	return response, nil
}

func (s *Service) applyBody(req *domain.HTTPRequestSpec, httpReq *http.Request) error {
	// apply body
	switch req.Request.Body.Type {
	case domain.BodyTypeJSON, domain.BodyTypeXML, domain.BodyTypeText:
		if req.Request.Body.Data != "" {
			httpReq.Body = io.NopCloser(strings.NewReader(req.Request.Body.Data))
		}

	case domain.BodyTypeBinary:
		if req.Request.Body.BinaryFilePath != "" {
			// read file
			file, err := os.ReadFile(req.Request.Body.BinaryFilePath)
			if err != nil {
				return err
			}

			httpReq.Body = io.NopCloser(bytes.NewReader(file))
			httpReq.ContentLength = int64(len(file))
			httpReq.Header.Add("Content-Type", "application/octet-stream")
			httpReq.Header.Add("Content-Disposition", "attachment; filename="+req.Request.Body.BinaryFilePath)
			httpReq.Header.Add("Content-Transfer-Encoding", "binary")
			httpReq.Header.Add("Connection", "Keep-Alive")
			httpReq.Header.Add("Content-Length", strconv.Itoa(len(file)))
		}

	case domain.BodyTypeFormData:
		// apply form body
		if len(req.Request.Body.FormData.Fields) > 0 {
			var b bytes.Buffer
			w := multipart.NewWriter(&b)
			for _, field := range req.Request.Body.FormData.Fields {
				if !field.Enable {
					continue
				}

				if field.Type == domain.FormFieldTypeText {
					fw, err := w.CreateFormField(field.Key)
					if err != nil {
						return err
					}
					if _, err := io.Copy(fw, strings.NewReader(field.Value)); err != nil {
						return err
					}
				}

				if field.Type == domain.FormFieldTypeFile {
					for _, ff := range field.Files {
						file, err := os.Open(ff)
						if err != nil {
							return err
						}
						defer file.Close()

						fw, err := w.CreateFormFile(field.Key, file.Name())
						if err != nil {
							return err
						}
						if _, err = io.Copy(fw, file); err != nil {
							return err
						}
					}
				}
			}

			if err := w.Close(); err != nil {
				return err
			}

			httpReq.Body = io.NopCloser(&b)
			httpReq.Header.Set("Content-Type", w.FormDataContentType())
			httpReq.Header.Add("Connection", "Keep-Alive")
			httpReq.Header.Add("Content-Length", strconv.Itoa(b.Len()))
			httpReq.ContentLength = int64(b.Len())
		}

	case domain.BodyTypeUrlencoded:
		// apply url encoded
		if len(req.Request.Body.URLEncoded) > 0 {
			form := url.Values{}
			for _, f := range req.Request.Body.URLEncoded {
				if !f.Enable {
					continue
				}

				form.Add(f.Key, f.Value)
			}
			httpReq.PostForm = form
		}
	}

	return nil
}

// TODO refactor
// nolint:gocyclo
func applyVariables(req *domain.HTTPRequestSpec, env *domain.EnvSpec) {
	// apply internal variables to environment
	// apply environment to request
	vars := variables.GetVariables()

	// apply environment variables if any
	if env != nil {
		variables.ApplyToEnv(vars, env)
	}

	// apply variables to request
	for k, v := range vars {
		for i, kv := range req.Request.Headers {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+k+"}}") {
				req.Request.Headers[i].Value = strings.ReplaceAll(kv.Value, "{{"+k+"}}", v)
			}
		}

		for i, kv := range req.Request.PathParams {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+k+"}}") {
				req.Request.PathParams[i].Value = strings.ReplaceAll(kv.Value, "{{"+k+"}}", v)
			}
		}

		for i, kv := range req.Request.QueryParams {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+k+"}}") {
				req.Request.QueryParams[i].Value = strings.ReplaceAll(kv.Value, "{{"+k+"}}", v)
			}
		}

		if strings.Contains(req.URL, "{{"+k+"}}") {
			req.URL = strings.ReplaceAll(req.URL, "{{"+k+"}}", v)
		}

		if strings.Contains(req.Request.Body.Data, "{{"+k+"}}") {
			req.Request.Body.Data = strings.ReplaceAll(req.Request.Body.Data, "{{"+k+"}}", v)
		}

		for i, field := range req.Request.Body.FormData.Fields {
			if field.Type == domain.FormFieldTypeFile {
				continue
			}
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(field.Value, "{{"+k+"}}") {
				req.Request.Body.FormData.Fields[i].Value = strings.ReplaceAll(field.Value, "{{"+k+"}}", v)
			}
		}

		for i, kv := range req.Request.Body.URLEncoded {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+k+"}}") {
				req.Request.Body.URLEncoded[i].Value = strings.ReplaceAll(kv.Value, "{{"+k+"}}", v)
			}
		}

		if req.Request.Auth != (domain.Auth{}) && req.Request.Auth.TokenAuth != nil {
			if strings.Contains(req.Request.Auth.TokenAuth.Token, "{{"+k+"}}") {
				req.Request.Auth.TokenAuth.Token = strings.ReplaceAll(req.Request.Auth.TokenAuth.Token, "{{"+k+"}}", v)
			}
		}

		if req.Request.Auth != (domain.Auth{}) && req.Request.Auth.BasicAuth != nil {
			if strings.Contains(req.Request.Auth.BasicAuth.Username, "{{"+k+"}}") {
				req.Request.Auth.BasicAuth.Username = strings.ReplaceAll(req.Request.Auth.BasicAuth.Username, "{{"+k+"}}", v)
			}

			if strings.Contains(req.Request.Auth.BasicAuth.Password, "{{"+k+"}}") {
				req.Request.Auth.BasicAuth.Password = strings.ReplaceAll(req.Request.Auth.BasicAuth.Password, "{{"+k+"}}", v)
			}
		}

		if req.Request.Auth != (domain.Auth{}) && req.Request.Auth.APIKeyAuth != nil {
			if strings.Contains(req.Request.Auth.APIKeyAuth.Key, "{{"+k+"}}") {
				req.Request.Auth.APIKeyAuth.Key = strings.ReplaceAll(req.Request.Auth.APIKeyAuth.Key, "{{"+k+"}}", v)
			}

			if strings.Contains(req.Request.Auth.APIKeyAuth.Value, "{{"+k+"}}") {
				req.Request.Auth.APIKeyAuth.Value = strings.ReplaceAll(req.Request.Auth.APIKeyAuth.Value, "{{"+k+"}}", v)
			}
		}

	}
}

func IsJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func PrettyJSON(data []byte) (string, error) {
	out := bytes.Buffer{}
	if err := json.Indent(&out, data, "", "    "); err != nil {
		return "", err
	}
	return out.String(), nil
}

func ParseJSON(text string) (map[string]any, error) {
	var js map[string]any
	if err := json.Unmarshal([]byte(text), &js); err != nil {
		return nil, err
	}
	return js, nil
}

func EncodeJSON(data any) ([]byte, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return b, nil
}
