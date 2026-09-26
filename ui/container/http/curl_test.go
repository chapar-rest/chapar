package httpc

import (
	"testing"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/container"
)

func setupText(t *testing.T) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	yoga.SetResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
}

func TestPastedCurl(t *testing.T) {
	multi := "curl -X POST 'https://api.example.com/users' \\\n  -H 'Accept: application/json' \\\n  -d '{\"a\":1}'"
	clip := &input.MemClipboard{}
	clip.Set(multi)

	// The field only got the first line of the paste; the clipboard has it all.
	cmd, ok := pastedCurl(clip, `curl -X POST 'https://api.example.com/users' \`)
	assert.True(t, ok)
	assert.Equal(t, multi, cmd)

	// Pasted after text already in the field, without selecting it first.
	_, ok = pastedCurl(clip, `https://example.comcurl -X POST 'https://api.example.com/users' \`)
	assert.True(t, ok)

	// Typing a command by hand does not match what is on the clipboard.
	_, ok = pastedCurl(clip, `curl -X PO`)
	assert.False(t, ok)

	// A plain URL edit never reads as a paste.
	_, ok = pastedCurl(clip, `https://api.example.com/users`)
	assert.False(t, ok)

	clip.Set("https://example.com")
	_, ok = pastedCurl(clip, "curl https://example.com")
	assert.False(t, ok)

	_, ok = pastedCurl(nil, "curl https://example.com")
	assert.False(t, ok)
}

func TestApplyCurl(t *testing.T) {
	setupText(t)
	var errs []error
	deps := container.Deps{Report: container.Reporter{
		Error: func(err error) { errs = append(errs, err) },
		Toast: func(string) {},
	}}
	req := domain.NewHTTPRequest("Create user")
	req.CollectionID = "col-1"
	req.Spec.HTTP.Request.PreRequest = domain.PreRequest{Type: domain.PrePostTypePython, Script: "print(1)"}
	c := Open(req, deps)

	ok := c.applyCurl(`curl -X POST 'https://api.example.com/users?x=1' -H 'Content-Type: application/json' -u ada:pw -d '{"name":"Ada"}'`)
	require.True(t, ok)
	assert.True(t, c.Dirty())
	c.flush()

	got := c.req
	assert.Equal(t, req.MetaData.ID, got.MetaData.ID)
	assert.Equal(t, "Create user", got.MetaData.Name)
	assert.Equal(t, "col-1", got.CollectionID)
	assert.Equal(t, "print(1)", got.Spec.HTTP.Request.PreRequest.Script)

	http := got.Spec.HTTP
	assert.Equal(t, "POST", http.Method)
	assert.Equal(t, "https://api.example.com/users?x=1", http.URL)
	require.Len(t, http.Request.QueryParams, 1)
	assert.Equal(t, "x", http.Request.QueryParams[0].Key)
	require.Len(t, http.Request.Headers, 1)
	assert.Equal(t, "Content-Type", http.Request.Headers[0].Key)
	assert.Equal(t, domain.RequestBodyTypeJSON, http.Request.Body.Type)
	assert.Equal(t, `{"name":"Ada"}`, http.Request.Body.Data)
	assert.Equal(t, domain.AuthTypeBasic, http.Request.Auth.Type)
	require.NotNil(t, http.Request.Auth.BasicAuth)
	assert.Equal(t, "ada", http.Request.Auth.BasicAuth.Username)
	assert.Empty(t, errs)
}

func TestApplyCurlReportsBadCommandOnce(t *testing.T) {
	setupText(t)
	var errs []error
	deps := container.Deps{Report: container.Reporter{
		Error: func(err error) { errs = append(errs, err) },
		Toast: func(string) {},
	}}
	c := Open(domain.NewHTTPRequest("r"), deps)

	assert.False(t, c.applyCurl(`curl 'https://example.com`))
	assert.False(t, c.applyCurl(`curl 'https://example.com`))
	require.Len(t, errs, 1)
	assert.Equal(t, "https://example.com", c.req.Spec.HTTP.URL)
}
