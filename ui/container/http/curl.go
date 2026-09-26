package httpc

import (
	"strings"

	"github.com/mirzakhany/yoga/input"

	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/ui/container"
)

// pastedCurl returns the curl command pasted into the URL field, if the edit
// was one. The field keeps only the first line of a paste, so a multi-line
// command is read back whole from the clipboard; a command typed by hand
// never matches and so is not parsed half-written.
func pastedCurl(clip input.Clipboard, value string) (string, bool) {
	if clip == nil || !strings.Contains(value, "curl") {
		return "", false
	}
	text := strings.TrimSpace(clip.Get())
	if !importer.IsCurlCommand(text) {
		return "", false
	}
	first, _, _ := strings.Cut(text, "\n")
	first = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(first), "\\"))
	if !strings.Contains(value, first) {
		return "", false
	}
	return text, true
}

// applyCurl replaces the request's method, URL, params, headers, body and
// auth with those of a curl command, keeping its name, collection, scripts
// and variables. It reports whether the command parsed.
func (c *Container) applyCurl(cmd string) bool {
	if cmd == c.rejectedCurl {
		return false
	}
	parsed, err := importer.ParseCurl(cmd)
	if err != nil {
		// Report it once; later edits of the same text are plain URL edits.
		c.rejectedCurl = cmd
		c.deps.ShowError(err)
		return false
	}
	src := parsed.Spec.HTTP
	http := c.req.Spec.HTTP
	http.Method = src.Method
	http.URL = src.URL
	r := http.Request
	r.Headers = src.Request.Headers
	r.QueryParams = src.Request.QueryParams
	r.PathParams = src.Request.PathParams
	r.Body = src.Request.Body
	r.Auth = src.Request.Auth

	container.LoadKV(c.headers, r.Headers)
	container.LoadKV(c.queryParams, r.QueryParams)
	container.LoadKV(c.pathParams, r.PathParams)
	container.LoadKV(c.urlEncoded, r.Body.URLEncoded)

	c.bodyEd.Close()
	c.bodyEd = container.NewBodyEditor(c.deps, "body-"+c.req.MetaData.ID, r.Body.Type, []byte(r.Body.Data))
	container.AssistEditor(c.bodyEd, c.varSrc)

	c.authState = container.LoadAuthState(r.Auth)
	c.authState.Vars = c.varSrc
	c.authState.AllowInherit = true
	c.authState.CollectionID = c.req.CollectionID

	c.markDirty()
	c.deps.Toast("Request filled from curl command")
	return true
}
