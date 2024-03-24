package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mirzakhany/chapar/internal/domain"
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

func SendRequest(r *domain.HTTPRequestSpec, e *domain.Environment) (*Response, error) {
	// prepare request
	// - apply environment
	// - apply variables
	// - apply authentication (if any) is not already applied to the headers

	// clone the request to make sure we don't modify the original request
	req := r.Clone()

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
		httpReq.Header.Add(h.Key, h.Value)
	}

	// apply path params as single brace
	for _, p := range req.Request.PathParams {
		httpReq.URL.Path = strings.ReplaceAll(httpReq.URL.Path, "{"+p.Key+"}", p.Value)
	}

	// apply query params
	query := httpReq.URL.Query()
	for _, q := range req.Request.QueryParams {
		query.Add(q.Key, q.Value)
	}

	httpReq.URL.RawQuery = query.Encode()

	// apply body
	if req.Request.Body.Data != "" {
		httpReq.Body = io.NopCloser(strings.NewReader(req.Request.Body.Data))
	}

	// apply form body
	if len(req.Request.Body.FormBody) > 0 {
		form := url.Values{}
		for _, f := range req.Request.Body.FormBody {
			form.Add(f.Key, f.Value)
		}
		httpReq.PostForm = form
	}

	// apply url encoded
	if len(req.Request.Body.URLEncoded) > 0 {
		form := url.Values{}
		for _, f := range req.Request.Body.URLEncoded {
			form.Add(f.Key, f.Value)
		}
		httpReq.PostForm = form
	}

	// apply authentication
	if req.Request.Auth != (domain.Auth{}) {
		if req.Request.Auth.TokenAuth != nil && req.Request.Auth.TokenAuth.Token != "" {
			httpReq.Header.Add("Authorization", "Bearer "+req.Request.Auth.TokenAuth.Token)
		}

		if req.Request.Auth.BasicAuth != nil && req.Request.Auth.BasicAuth.Username != "" && req.Request.Auth.BasicAuth.Password != "" {
			httpReq.SetBasicAuth(req.Request.Auth.BasicAuth.Username, req.Request.Auth.BasicAuth.Password)
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

func applyVariables(req *domain.HTTPRequestSpec, env *domain.EnvSpec) *domain.HTTPRequestSpec {
	// apply internal variables to environment
	// apply environment to request
	variables := map[string]string{
		"randomUUID4":   uuid.NewString(),
		"timeNow":       time.Now().UTC().Format(time.RFC3339),
		"unixTimestamp": strconv.FormatInt(time.Now().UTC().Unix(), 10),
	}

	// apply environment variables if any
	if env != nil {
		// to through all the variables and replace them in the environment
		for k, v := range variables {
			for i, kv := range env.Values {
				// if value contain the variable in double curly braces then replace it
				if strings.Contains(kv.Value, "{{"+k+"}}") {
					env.Values[i].Value = strings.ReplaceAll(kv.Value, "{{"+k+"}}", v)
				}
			}
		}

		// add env variables to variables
		for _, kv := range env.Values {
			variables[kv.Key] = kv.Value
		}
	}

	// apply variables to request
	for k, v := range variables {
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

		for i, kv := range req.Request.Body.FormBody {
			// if value contain the variable in double curly braces then replace it
			if strings.Contains(kv.Value, "{{"+k+"}}") {
				req.Request.Body.FormBody[i].Value = strings.ReplaceAll(kv.Value, "{{"+k+"}}", v)
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

	}
	return req
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
