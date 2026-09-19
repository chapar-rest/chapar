package langsrv

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/lsp"
	"github.com/mirzakhany/yoga/ui"
)

// toastInterval limits how often one kind of problem from one server pops a
// toast; every occurrence is still logged.
const toastInterval = 30 * time.Second

// Notice is a server problem ready to show: Title for a toast, Message for
// the notification list, Detail (message plus server output) for the console.
type Notice struct {
	// Missing is set when the server is not installed; the app offers to
	// install it instead of showing a plain toast.
	Missing *Language
	Title   string
	Message string
	Detail  string
	Warning bool // a setup problem rather than a failure
	Toast   bool // false when a toast for the same problem was shown recently
}

// Service applies the language-server settings and turns server errors into
// notices. Apply, Restart, and Drain belong to the UI thread.
type Service struct {
	mgr *lsp.Manager
	dir string // virtual workspace holding the editors' documents

	mu        sync.Mutex
	pending   []*lsp.ServerError
	pathReady bool
	retry     bool // PATH changed: re-resolve servers reported missing
	wake      func()

	installing map[string]bool // guarded by mu
	installs   []InstallEvent  // guarded by mu

	applied   map[string]domain.LanguageServerConfig
	lastToast map[string]time.Time
	seq       atomic.Int64
}

// New creates the service and hooks it to yoga's language-server manager.
// Server-not-found reports are held until PathReady, since the PATH may still
// be growing (see FixPath).
func New(wake func()) *Service {
	s := &Service{
		mgr:       ui.LanguageServers(),
		dir:       workspaceDir(),
		wake:      wake,
		applied:   map[string]domain.LanguageServerConfig{},
		lastToast: map[string]time.Time{},
	}
	s.mgr.OnError(s.onError)
	s.mgr.OnActivity(s.poke)
	if err := writeWorkspace(s.dir); err != nil {
		s.pending = append(s.pending, &lsp.ServerError{
			Kind: lsp.ErrStart, LanguageID: "python", Command: "pyright-langserver",
			Err: fmt.Errorf("preparing workspace %s: %w", s.dir, err),
		})
	}
	return s
}

func (s *Service) onError(e *lsp.ServerError) {
	s.mu.Lock()
	s.pending = append(s.pending, e)
	s.mu.Unlock()
	s.poke()
}

func (s *Service) poke() {
	s.mu.Lock()
	wake := s.wake
	s.mu.Unlock()
	if wake != nil {
		wake()
	}
}

// SetWake sets the function that requests a repaint. Safe from any goroutine.
func (s *Service) SetWake(fn func()) {
	s.mu.Lock()
	s.wake = fn
	s.mu.Unlock()
}

// PathReady reports that PATH is final; servers reported missing before it
// are looked up again on the next Drain. Safe from any goroutine.
func (s *Service) PathReady() {
	s.mu.Lock()
	s.pathReady = true
	s.retry = true
	s.mu.Unlock()
	s.poke()
}

// Apply registers the configured servers with yoga and restarts every
// language whose configuration changed since the last Apply.
func (s *Service) Apply(cfg domain.LanguageServersConfig) {
	for i, c := range Effective(cfg) {
		l := Languages[i]
		if c.Enabled && c.Command != "" {
			lsp.Register(l.Ext, lsp.ServerConfig{
				LanguageID: l.LSPID,
				Command:    expandHome(c.Command),
				Args:       c.Args,
			})
		} else {
			lsp.Unregister(l.Ext)
		}
		if prev, ok := s.applied[l.ID]; ok && prev.Changed(c) {
			s.mgr.Restart(l.LSPID)
		}
		s.applied[l.ID] = c
	}
}

// Restart restarts the server for the language with config key id, or every
// server when id is empty.
func (s *Service) Restart(id string) {
	if id == "" {
		s.mgr.Restart("")
		return
	}
	if l, ok := ByID(id); ok {
		s.mgr.Restart(l.LSPID)
	}
}

// Running reports whether the language with config key id has a live server.
func (s *Service) Running(id string) bool {
	l, ok := ByID(id)
	return ok && slices.Contains(s.mgr.Running(), l.LSPID)
}

// Drain returns the notices collected since the last call.
func (s *Service) Drain() []Notice {
	s.mu.Lock()
	ready, retry := s.pathReady, s.retry
	s.retry = false
	var errs, keep []*lsp.ServerError
	for _, e := range s.pending {
		if e.Kind == lsp.ErrNotFound && !ready {
			keep = append(keep, e)
			continue
		}
		errs = append(errs, e)
	}
	s.pending = keep
	s.mu.Unlock()

	if retry {
		// Servers reported missing may have been found on the final PATH.
		var still []*lsp.ServerError
		for _, e := range errs {
			if e.Kind == lsp.ErrNotFound {
				if _, err := exec.LookPath(e.Command); err == nil {
					s.mgr.Restart(e.LanguageID)
					continue
				}
			}
			still = append(still, e)
		}
		errs = still
	}

	out := make([]Notice, 0, len(errs))
	now := time.Now()
	for _, e := range errs {
		n := describe(e)
		key := e.LanguageID + "\x00" + e.Kind.String()
		if now.Sub(s.lastToast[key]) >= toastInterval {
			s.lastToast[key] = now
			n.Toast = true
		}
		out = append(out, n)
	}
	return out
}

func describe(e *lsp.ServerError) Notice {
	name, install := e.LanguageID, ""
	if l, ok := ByLSPID(e.LanguageID); ok {
		name, install = l.Name, l.Install
	}
	var title, msg string
	warning := false
	switch e.Kind {
	case lsp.ErrNotFound:
		warning = true
		title = name + " language server not found"
		msg = fmt.Sprintf("%s language server not found: %s.", name, e.Command)
		if install != "" {
			msg += " Install it with: " + install + "."
		}
		msg += " Or set its path in Settings → Language servers."
	case lsp.ErrStart:
		title = name + " language server failed to start"
		msg = fmt.Sprintf("%s language server failed to start: %v", name, e.Err)
	case lsp.ErrExited:
		title = name + " language server stopped"
		msg = fmt.Sprintf("%s language server stopped: %v. Restart it from the command palette.", name, e.Err)
	case lsp.ErrServer:
		title = name + " language server error"
		msg = fmt.Sprintf("%s language server: %v", name, e.Err)
	default:
		title = name + " language server error"
		msg = fmt.Sprintf("%s language server %s: %v", name, e.Kind, e.Err)
	}
	detail := msg
	if e.Stderr != "" {
		detail += "\n" + e.Stderr
	}
	n := Notice{Title: title, Message: msg, Detail: detail, Warning: warning}
	if e.Kind == lsp.ErrNotFound {
		if l, ok := ByLSPID(e.LanguageID); ok {
			n.Missing = &l
		}
	}
	return n
}

// Status describes the server for c in one line, for settings.
func (s *Service) Status(c domain.LanguageServerConfig) string {
	if !c.Enabled {
		return "Disabled"
	}
	path, err := exec.LookPath(expandHome(c.Command))
	if err != nil {
		msg := "Not found: " + c.Command
		if l, ok := ByID(c.Language); ok && l.Install != "" {
			msg += ". Install: " + l.Install
		}
		return msg
	}
	if s.Running(c.Language) {
		return "Running · " + path
	}
	return "Ready · " + path + " (starts when an editor needs it)"
}

// Missing reports whether c's command cannot be found.
func (s *Service) Missing(c domain.LanguageServerConfig) bool {
	_, err := exec.LookPath(expandHome(c.Command))
	return err != nil
}

// ---- editors ----

// docPath returns a unique virtual document path in the workspace. The files
// are never written: the server gets the text from the editor.
func (s *Service) docPath(name, ext string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s-%d%s", sanitize(name), s.seq.Add(1), ext))
}

// NewScriptEditor returns an editor for a Python pre/post-request script.
func (s *Service) NewScriptEditor(name, script string) *ui.Editor {
	return ui.NewEditorFor(s.docPath(name, ".py"), []byte(script))
}

// NewBodyEditor returns an editor for a request body of the given
// domain.RequestBodyType*. JSON and XML bodies get highlighting and, when
// enabled, their language server; other types are plain text.
func (s *Service) NewBodyEditor(name, bodyType string, body []byte, opts ...ui.EditorOption) *ui.Editor {
	switch bodyType {
	case domain.RequestBodyTypeJSON:
		return ui.NewEditorFor(s.docPath(name, ".json"), body, opts...)
	case domain.RequestBodyTypeXML:
		return ui.NewEditorFor(s.docPath(name, ".xml"), body, opts...)
	default:
		return ui.NewEditor(body, highlight.Noop{}, opts...)
	}
}

// codegenLanguages maps the code dialog's generators to languages.
var codegenLanguages = map[string]string{
	"curl":        "bash",
	"python":      "python",
	"golang":      "go",
	"axios":       "javascript",
	"node-fetch":  "javascript",
	"java-okhttp": "java",
	"ruby-net":    "ruby",
	"dot-net":     "csharp",
}

// NewCodeViewer returns an editor for code generated by the code dialog's
// generator gen (e.g. "golang").
func (s *Service) NewCodeViewer(gen, code string) *ui.Editor {
	l, ok := ByID(codegenLanguages[gen])
	if !ok {
		return ui.NewEditor([]byte(code), highlight.Noop{})
	}
	return ui.NewEditorFor(s.docPath("codegen", l.Ext), []byte(code))
}

// ---- helpers ----

func workspaceDir() string {
	base, err := domain.LegacyConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "lsp-workspace")
}

// writeWorkspace creates the workspace directory and its stub files.
func writeWorkspace(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var errs []error
	for name, content := range workspaceFiles {
		p := filepath.Join(dir, name)
		if old, err := os.ReadFile(p); err == nil && string(old) == content {
			continue
		}
		errs = append(errs, os.WriteFile(p, []byte(content), 0o644))
	}
	return errors.Join(errs...)
}

func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		}
		return '_'
	}, name)
}

func expandHome(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return p
}
