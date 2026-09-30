package langsrv

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/lsp"
)

// TestPyrightKnowsScriptGlobals runs the real pyright against the virtual
// workspace: the script API (chapar, request, response) must type-check,
// response must be flagged in a pre-request script, and genuine mistakes must
// still be reported. Skipped without pyright on PATH.
func TestPyrightKnowsScriptGlobals(t *testing.T) {
	bin, err := exec.LookPath("pyright-langserver")
	if err != nil {
		t.Skip("pyright-langserver not on PATH")
	}
	dir := t.TempDir()
	if err := writeWorkspace(dir); err != nil {
		t.Fatal(err)
	}
	py, _ := ByID("python")
	lsp.Register(".py", lsp.ServerConfig{LanguageID: "python", Command: bin, Args: []string{"--stdio"}, RootMarkers: py.RootMarkers})
	t.Cleanup(func() { lsp.Unregister(".py") })

	m := lsp.NewManager()
	var mu sync.Mutex
	var serverErrs []string
	m.OnError(func(e *lsp.ServerError) {
		mu.Lock()
		serverErrs = append(serverErrs, e.Error())
		mu.Unlock()
	})

	// The same script in both phases: response exists only after the send.
	script := strings.Join([]string{
		`import chapar as api`,
		`token = response.json()["token"]`,
		`chapar.set_env("token", token)`,
		`api.set_env("status", response.status_code)`,
		`print(request.url, chapar.get_env("base"))`,
		`undefined_thing()`,
	}, "\n") + "\n"
	pre := m.Open(filepath.Join(dir, "pre", "x-1.py"), func() []byte { return []byte(script) })
	defer pre.Close()
	post := m.Open(filepath.Join(dir, "post", "x-2.py"), func() []byte { return []byte(script) })
	defer post.Close()

	var preDiags, postDiags []lsp.Diagnostic
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		pre.Poll()
		post.Poll()
		if preDiags, postDiags = pre.Diagnostics(), post.Diagnostics(); len(preDiags) > 0 && len(postDiags) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(serverErrs) > 0 {
		t.Fatalf("server errors: %v", serverErrs)
	}
	if len(postDiags) != 1 || !strings.Contains(postDiags[0].Message, "undefined_thing") || postDiags[0].Range.Start.Line != 5 {
		t.Fatalf("post diagnostics = %+v, want only undefined_thing on line 6", postDiags)
	}
	var lines []int
	for _, d := range preDiags {
		lines = append(lines, d.Range.Start.Line)
	}
	if !slices.Equal(lines, []int{1, 3, 5}) || !strings.Contains(preDiags[1].Message, `"None"`) {
		t.Fatalf("pre diagnostics = %+v, want response on lines 2 and 4 and undefined_thing on line 6", preDiags)
	}
}
