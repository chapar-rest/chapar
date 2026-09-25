package langsrv

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/lsp"
)

// TestPyrightKnowsScriptGlobals runs the real pyright against the virtual
// workspace: the script API (chapar, request, response) must type-check, and
// genuine mistakes must still be reported. Skipped without pyright on PATH.
func TestPyrightKnowsScriptGlobals(t *testing.T) {
	bin, err := exec.LookPath("pyright-langserver")
	if err != nil {
		t.Skip("pyright-langserver not on PATH")
	}
	dir := t.TempDir()
	if err := writeWorkspace(dir); err != nil {
		t.Fatal(err)
	}
	lsp.Register(".py", lsp.ServerConfig{LanguageID: "python", Command: bin, Args: []string{"--stdio"}})
	t.Cleanup(func() { lsp.Unregister(".py") })

	m := lsp.NewManager()
	var mu sync.Mutex
	var serverErrs []string
	m.OnError(func(e *lsp.ServerError) {
		mu.Lock()
		serverErrs = append(serverErrs, e.Error())
		mu.Unlock()
	})

	script := strings.Join([]string{
		`import chapar as api`,
		`token = response.json()["token"]`,
		`chapar.set_env("token", token)`,
		`api.set_env("status", response.status_code)`,
		`print(request.url, chapar.get_env("base"))`,
		`undefined_thing()`,
	}, "\n") + "\n"
	d := m.Open(filepath.Join(dir, "pre-1.py"), func() []byte { return []byte(script) })
	defer d.Close()

	var diags []lsp.Diagnostic
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		d.Poll()
		if diags = d.Diagnostics(); len(diags) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(serverErrs) > 0 {
		t.Fatalf("server errors: %v", serverErrs)
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "undefined_thing") || diags[0].Range.Start.Line != 5 {
		t.Fatalf("diagnostics = %+v, want only undefined_thing on line 6", diags)
	}
}
