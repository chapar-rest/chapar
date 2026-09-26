package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chapar-rest/chapar/internal/codegen"
	"github.com/chapar-rest/chapar/internal/domain"
)

func headerMap(kv []domain.KeyValue) map[string]string {
	out := make(map[string]string, len(kv))
	for _, h := range kv {
		out[h.Key] = h.Value
	}
	return out
}

func TestParseCurl_SimpleGet(t *testing.T) {
	req, err := ParseCurl(`curl https://api.example.com/users?page=2&limit=10`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, domain.RequestTypeHTTP, req.MetaData.Type)
	assert.Equal(t, domain.DefaultRequestName, req.MetaData.Name)
	assert.Equal(t, "GET", spec.Method)
	assert.Equal(t, "https://api.example.com/users?page=2&limit=10", spec.URL)
	require.Len(t, spec.Request.QueryParams, 2)
	assert.Equal(t, "page", spec.Request.QueryParams[0].Key)
	assert.Equal(t, "2", spec.Request.QueryParams[0].Value)
	assert.True(t, spec.Request.QueryParams[0].Enable)
	assert.Empty(t, spec.Request.Headers)
	assert.Equal(t, domain.RequestBodyTypeNone, spec.Request.Body.Type)
}

func TestParseCurl_PostJSONWithHeaders(t *testing.T) {
	cmd := `curl -X POST 'https://api.example.com/users' \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer abc123" \
  -d '{"name": "Ada", "tags": ["x"]}'`
	req, err := ParseCurl(cmd)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "POST", spec.Method)
	assert.Equal(t, "https://api.example.com/users", spec.URL)
	assert.Equal(t, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer abc123",
	}, headerMap(spec.Request.Headers))
	assert.Equal(t, domain.RequestBodyTypeJSON, spec.Request.Body.Type)
	assert.Equal(t, `{"name": "Ada", "tags": ["x"]}`, spec.Request.Body.Data)
}

func TestParseCurl_DataImpliesPost(t *testing.T) {
	req, err := ParseCurl(`curl https://example.com/login -d 'user=ada&pass=p%40ss'`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "POST", spec.Method)
	assert.Equal(t, domain.RequestBodyTypeUrlencoded, spec.Request.Body.Type)
	require.Len(t, spec.Request.Body.URLEncoded, 2)
	assert.Equal(t, "user", spec.Request.Body.URLEncoded[0].Key)
	assert.Equal(t, "ada", spec.Request.Body.URLEncoded[0].Value)
	assert.Equal(t, "p@ss", spec.Request.Body.URLEncoded[1].Value)
}

func TestParseCurl_JSONBodyWithoutContentType(t *testing.T) {
	req, err := ParseCurl(`curl https://example.com -d '{"a":1}'`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, domain.RequestBodyTypeJSON, spec.Request.Body.Type)
	assert.Equal(t, "application/json", headerMap(spec.Request.Headers)["Content-Type"])
}

func TestParseCurl_JSONFlag(t *testing.T) {
	req, err := ParseCurl(`curl --json '{"a":1}' https://example.com`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "POST", spec.Method)
	assert.Equal(t, domain.RequestBodyTypeJSON, spec.Request.Body.Type)
	assert.Equal(t, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}, headerMap(spec.Request.Headers))
}

func TestParseCurl_ChromeCopyAsCurl(t *testing.T) {
	// Chrome's "Copy as cURL (bash)" output: $'...' body, --data-raw, flag noise.
	cmd := `curl 'https://example.com/api/items' \
  -H 'accept: */*' \
  -H 'content-type: application/json' \
  -b 'session=xyz; theme=dark' \
  -H 'user-agent: Mozilla/5.0' \
  --data-raw $'{"note":"it\'s\\n\\u00e9"}' \
  --compressed`
	req, err := ParseCurl(cmd)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "POST", spec.Method)
	assert.Equal(t, "https://example.com/api/items", spec.URL)
	h := headerMap(spec.Request.Headers)
	assert.Equal(t, "session=xyz; theme=dark", h["Cookie"])
	assert.Equal(t, "*/*", h["accept"])
	assert.Equal(t, domain.RequestBodyTypeJSON, spec.Request.Body.Type)
	assert.Equal(t, "{\"note\":\"it's\\n\\u00e9\"}", spec.Request.Body.Data)
}

func TestParseCurl_ANSIQuoting(t *testing.T) {
	req, err := ParseCurl(`curl https://example.com -H $'X-A: tab\there' --data-raw $'line1\nline2 \xc3\xa9 é'`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "tab\there", headerMap(spec.Request.Headers)["X-A"])
	assert.Equal(t, "line1\nline2 é é", spec.Request.Body.Data)
}

func TestParseCurl_ShortFlagClusters(t *testing.T) {
	req, err := ParseCurl(`curl -sSL -XPUT -m 30 -o out.json -HAccept:text/plain https://example.com/x`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "PUT", spec.Method)
	assert.Equal(t, "https://example.com/x", spec.URL)
	assert.Equal(t, map[string]string{"Accept": "text/plain"}, headerMap(spec.Request.Headers))
}

func TestParseCurl_BasicAuthAndBearer(t *testing.T) {
	req, err := ParseCurl(`curl -u ada:s3cr:et https://example.com`)
	require.NoError(t, err)
	auth := req.Spec.HTTP.Request.Auth
	assert.Equal(t, domain.AuthTypeBasic, auth.Type)
	require.NotNil(t, auth.BasicAuth)
	assert.Equal(t, "ada", auth.BasicAuth.Username)
	assert.Equal(t, "s3cr:et", auth.BasicAuth.Password)

	req, err = ParseCurl(`curl --oauth2-bearer tok https://example.com`)
	require.NoError(t, err)
	auth = req.Spec.HTTP.Request.Auth
	assert.Equal(t, domain.AuthTypeToken, auth.Type)
	require.NotNil(t, auth.TokenAuth)
	assert.Equal(t, "tok", auth.TokenAuth.Token)
}

func TestParseCurl_NoAuthDefaults(t *testing.T) {
	req, err := ParseCurl(`curl https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, "None", req.Spec.HTTP.Request.Auth.Type)
}

func TestParseCurl_Form(t *testing.T) {
	req, err := ParseCurl(`curl -F 'name=Ada' -F 'avatar=@/tmp/me.png;type=image/png' --form-string 'raw=@not-a-file' https://example.com/upload`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "POST", spec.Method)
	assert.Equal(t, domain.RequestBodyTypeFormData, spec.Request.Body.Type)
	fields := spec.Request.Body.FormData.Fields
	require.Len(t, fields, 3)
	assert.Equal(t, domain.FormField{ID: fields[0].ID, Type: domain.FormFieldTypeText, Key: "name", Value: "Ada", Enable: true}, fields[0])
	assert.Equal(t, domain.FormFieldTypeFile, fields[1].Type)
	assert.Equal(t, []string{"/tmp/me.png"}, fields[1].Files)
	assert.Equal(t, domain.FormFieldTypeText, fields[2].Type)
	assert.Equal(t, "@not-a-file", fields[2].Value)
}

func TestParseCurl_BinaryBody(t *testing.T) {
	req, err := ParseCurl(`curl --data-binary @payload.bin https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, "POST", req.Spec.HTTP.Method)
	assert.Equal(t, domain.RequestBodyTypeBinary, req.Spec.HTTP.Request.Body.Type)
	assert.Equal(t, "payload.bin", req.Spec.HTTP.Request.Body.BinaryFilePath)

	req, err = ParseCurl(`curl -T file.txt https://example.com/put`)
	require.NoError(t, err)
	assert.Equal(t, "PUT", req.Spec.HTTP.Method)
	assert.Equal(t, "file.txt", req.Spec.HTTP.Request.Body.BinaryFilePath)
}

func TestParseCurl_GetMovesDataToQuery(t *testing.T) {
	req, err := ParseCurl(`curl -G https://example.com/search?x=1 -d q=hello --data-urlencode 'tag=a b'`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "GET", spec.Method)
	assert.Equal(t, "https://example.com/search?x=1&q=hello&tag=a+b", spec.URL)
	assert.Len(t, spec.Request.QueryParams, 3)
	assert.Equal(t, domain.RequestBodyTypeNone, spec.Request.Body.Type)
}

func TestParseCurl_HeadAndMisc(t *testing.T) {
	req, err := ParseCurl(`curl -I example.com/health -A 'bot/1.0' -e https://ref.example`)
	require.NoError(t, err)

	spec := req.Spec.HTTP
	assert.Equal(t, "HEAD", spec.Method)
	assert.Equal(t, "http://example.com/health", spec.URL)
	assert.Equal(t, map[string]string{"User-Agent": "bot/1.0", "Referer": "https://ref.example"}, headerMap(spec.Request.Headers))
}

func TestParseCurl_HeaderEdgeCases(t *testing.T) {
	req, err := ParseCurl(`curl -H 'X-Empty;' -H 'Accept:' -H 'X-Url: http://a:b' --url https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"X-Empty": "", "X-Url": "http://a:b"}, headerMap(req.Spec.HTTP.Request.Headers))
	assert.Equal(t, "https://example.com", req.Spec.HTTP.URL)
}

func TestParseCurl_KeepsVariables(t *testing.T) {
	req, err := ParseCurl(`curl '{{baseUrl}}/users/{id}' -H 'Authorization: Bearer {{token}}'`)
	require.NoError(t, err)
	assert.Equal(t, "{{baseUrl}}/users/{id}", req.Spec.HTTP.URL)
	assert.Equal(t, "Bearer {{token}}", headerMap(req.Spec.HTTP.Request.Headers)["Authorization"])
}

func TestParseCurl_WindowsContinuation(t *testing.T) {
	req, err := ParseCurl("curl \"https://example.com\" ^\r\n  -H \"Accept: text/html\"")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", req.Spec.HTTP.URL)
	assert.Equal(t, "text/html", headerMap(req.Spec.HTTP.Request.Headers)["Accept"])
}

func TestParseCurl_Errors(t *testing.T) {
	for name, cmd := range map[string]string{
		"not curl":      `wget https://example.com`,
		"no url":        `curl -X POST`,
		"missing value": `curl https://example.com -H`,
		"open quote":    `curl 'https://example.com`,
		"open dquote":   `curl "https://example.com`,
		"open ansi":     `curl $'https://example.com`,
		"empty":         ``,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseCurl(cmd)
			assert.Error(t, err)
		})
	}
}

func TestIsCurlCommand(t *testing.T) {
	assert.True(t, IsCurlCommand("curl https://x"))
	assert.True(t, IsCurlCommand("  curl\\\n https://x"))
	assert.False(t, IsCurlCommand("https://curl.se"))
	assert.False(t, IsCurlCommand("curly"))
}

// A request survives being turned into curl by the code generator and back.
func TestParseCurl_RoundTripsGeneratedCurl(t *testing.T) {
	spec := &domain.HTTPRequestSpec{
		Method: domain.RequestMethodPOST,
		URL:    "https://api.example.com/users?active=true",
		Request: &domain.HTTPRequest{
			Headers: []domain.KeyValue{
				{Key: "Content-Type", Value: "application/json", Enable: true},
				{Key: "X-Trace", Value: "1", Enable: true},
			},
			Body: domain.Body{Type: domain.RequestBodyTypeJSON, Data: `{"name":"Ada"}`},
		},
	}
	cmd, err := codegen.New().GenerateCurlCommand(spec, nil, nil)
	require.NoError(t, err)

	req, err := ParseCurl(cmd)
	require.NoError(t, err, cmd)
	got := req.Spec.HTTP
	assert.Equal(t, spec.Method, got.Method)
	assert.Equal(t, spec.URL, got.URL)
	assert.Equal(t, headerMap(spec.Request.Headers), headerMap(got.Request.Headers))
	assert.Equal(t, spec.Request.Body.Type, got.Request.Body.Type)
	assert.Equal(t, spec.Request.Body.Data, got.Request.Body.Data)
}
