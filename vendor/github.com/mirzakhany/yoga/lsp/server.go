package lsp

import (
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ServerConfig describes how to launch a language server for one language.
type ServerConfig struct {
	LanguageID string   // LSP language id sent in didOpen (e.g. "go")
	Command    string   // executable name, resolved on PATH
	Args       []string // arguments passed to the executable
}

// registry maps a file extension to its language server. gopls (.go) is built
// in; add more with Register. Servers whose Command is not on PATH degrade
// gracefully to a no-op (see Manager.Open).
var (
	registryMu sync.RWMutex
	registry   = map[string]ServerConfig{
		".go": {LanguageID: "go", Command: "gopls", Args: []string{"serve"}},
	}
)

// Register maps a file extension (with the leading dot, case-insensitive) to a
// language server. It is the supported way to add languages without editing this
// package: call it once at startup, before opening files of that type. A later
// Register for the same extension replaces the earlier entry.
//
//	lsp.Register(".json", lsp.ServerConfig{
//	    LanguageID: "json",
//	    Command:    "vscode-json-language-server",
//	    Args:       []string{"--stdio"},
//	})
func Register(ext string, cfg ServerConfig) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[strings.ToLower(ext)] = cfg
}

// ForPath returns the server configured for path's extension, if any.
func ForPath(path string) (ServerConfig, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	cfg, ok := registry[strings.ToLower(filepath.Ext(path))]
	return cfg, ok
}

// Doc is the per-document handle the editor depends on. It is the LSP analog of
// highlight.Highlighter: every method is non-blocking on the UI thread, and
// results are retrieved via Poll. A no-op implementation is returned whenever no
// server is available, so callers never need to nil-check.
type Doc interface {
	// DidChange notifies the server that the full document content changed.
	DidChange(content []byte)
	// Completion requests suggestions at pos; the result arrives via Poll as an
	// EventCompletion tagged with the returned request id.
	Completion(pos Position) (reqID int)
	// Hover requests hover info at pos; the result arrives via Poll as an
	// EventHover tagged with the returned request id.
	Hover(pos Position) (reqID int)
	// Poll drains pending async events for this document (non-blocking).
	Poll() (events []Event, ok bool)
	// Diagnostics returns the latest published diagnostics for this document.
	Diagnostics() []Diagnostic
	// Encoding reports the negotiated position encoding ("utf-8" or "utf-16").
	Encoding() string
	// Close notifies the server the document closed and releases the session.
	Close()
}

// Manager owns language-server processes and shares one per (root, language) so
// every open file in a workspace talks to a single server, the way LSP expects.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
}

// NewManager returns an empty manager. Processes start lazily on Open.
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*session)}
}

// Open registers a document with its language server and returns a handle. If no
// server is configured for the path, the binary is missing, or path is empty, a
// no-op Doc is returned — so the editor works normally without a server.
func (m *Manager) Open(path string, content []byte) Doc {
	if path == "" {
		return noopDoc{}
	}
	cfg, ok := ForPath(path)
	if !ok {
		return noopDoc{}
	}
	if _, err := exec.LookPath(cfg.Command); err != nil {
		return noopDoc{}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	root := findRoot(abs)
	sess := m.session(cfg, root)
	uri := pathToURI(abs)
	d := &doc{sess: sess, mgr: m, uri: uri, languageID: cfg.LanguageID, version: 1}
	text := string(content)
	sess.do(func(c *client) { c.didOpen(uri, cfg.LanguageID, 1, text) })
	return d
}

// session is one running server shared by Manager.Open callers. The client is
// created asynchronously (the initialize round-trip is slow) so Open never
// blocks the UI; operations issued before the handshake completes are queued.
type session struct {
	cfg  ServerConfig
	root string

	mu     sync.Mutex
	client *client
	queue  []func(*client)
	refs   int
	dead   bool
}

func (m *Manager) session(cfg ServerConfig, root string) *session {
	key := cfg.LanguageID + "\x00" + root
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[key]; ok {
		s.mu.Lock()
		s.refs++
		s.mu.Unlock()
		return s
	}
	s := &session{cfg: cfg, root: root, refs: 1}
	m.sessions[key] = s
	go s.start()
	return s
}

func (s *session) start() {
	rwc, err := spawn(s.cfg)
	if err != nil {
		s.markDead()
		return
	}
	c, err := newClient(rwc, pathToURI(s.root))
	if err != nil {
		rwc.Close()
		s.markDead()
		return
	}
	s.mu.Lock()
	s.client = c
	q := s.queue
	s.queue = nil
	s.mu.Unlock()
	for _, fn := range q {
		fn(c)
	}
}

func (s *session) markDead() {
	s.mu.Lock()
	s.dead = true
	s.queue = nil
	s.mu.Unlock()
}

// do runs fn against the client, queueing it if the handshake is still pending.
func (s *session) do(fn func(*client)) {
	s.mu.Lock()
	if s.client != nil {
		c := s.client
		s.mu.Unlock()
		fn(c)
		return
	}
	if s.dead {
		s.mu.Unlock()
		return
	}
	s.queue = append(s.queue, fn)
	s.mu.Unlock()
}

func (s *session) getClient() *client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client
}

// release drops one reference; the process is shut down when the last document
// using it closes.
func (m *Manager) release(s *session) {
	s.mu.Lock()
	s.refs--
	dead := s.refs <= 0
	c := s.client
	s.mu.Unlock()
	if !dead {
		return
	}
	m.mu.Lock()
	for k, v := range m.sessions {
		if v == s {
			delete(m.sessions, k)
			break
		}
	}
	m.mu.Unlock()
	if c != nil {
		go c.shutdown()
	}
}

// doc is the live Doc implementation backed by a shared session.
type doc struct {
	sess       *session
	mgr        *Manager
	uri        string
	languageID string

	mu      sync.Mutex
	version int
	nextReq int
}

func (d *doc) DidChange(content []byte) {
	d.mu.Lock()
	d.version++
	v := d.version
	d.mu.Unlock()
	text := string(content)
	d.sess.do(func(c *client) { c.didChange(d.uri, v, text) })
}

func (d *doc) Completion(pos Position) int {
	id := d.reqID()
	d.sess.do(func(c *client) {
		go func() {
			items, err := c.completion(d.uri, pos)
			if err != nil {
				return
			}
			c.enqueue(Event{Kind: EventCompletion, URI: d.uri, ReqID: id, Completions: items})
		}()
	})
	return id
}

func (d *doc) Hover(pos Position) int {
	id := d.reqID()
	d.sess.do(func(c *client) {
		go func() {
			h, err := c.hover(d.uri, pos)
			if err != nil {
				return
			}
			c.enqueue(Event{Kind: EventHover, URI: d.uri, ReqID: id, Hover: h})
		}()
	})
	return id
}

func (d *doc) Poll() ([]Event, bool) {
	c := d.sess.getClient()
	if c == nil {
		return nil, false
	}
	evs := c.drain(d.uri)
	return evs, len(evs) > 0
}

func (d *doc) Diagnostics() []Diagnostic {
	c := d.sess.getClient()
	if c == nil {
		return nil
	}
	return c.diagnostics(d.uri)
}

func (d *doc) Encoding() string {
	if c := d.sess.getClient(); c != nil {
		return c.encoding
	}
	return "utf-8"
}

func (d *doc) Close() {
	uri := d.uri
	d.sess.do(func(c *client) { c.didClose(uri) })
	d.mgr.release(d.sess)
}

func (d *doc) reqID() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nextReq++
	return d.nextReq
}

// noopDoc is the graceful fallback when no language server is available.
type noopDoc struct{}

func (noopDoc) DidChange([]byte)          {}
func (noopDoc) Completion(Position) int   { return 0 }
func (noopDoc) Hover(Position) int        { return 0 }
func (noopDoc) Poll() ([]Event, bool)     { return nil, false }
func (noopDoc) Diagnostics() []Diagnostic { return nil }
func (noopDoc) Encoding() string          { return "utf-8" }
func (noopDoc) Close()                    {}

// ---- process + URI helpers ----

// stdio adapts a subprocess's stdin/stdout into one io.ReadWriteCloser.
type stdio struct {
	in  io.WriteCloser
	out io.ReadCloser
	cmd *exec.Cmd
}

func (s *stdio) Read(p []byte) (int, error)  { return s.out.Read(p) }
func (s *stdio) Write(p []byte) (int, error) { return s.in.Write(p) }

func (s *stdio) Close() error {
	_ = s.in.Close()
	_ = s.out.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	go s.cmd.Wait() // reap the process without blocking the caller
	return nil
}

// spawn launches the configured server and wires its stdio.
func spawn(cfg ServerConfig) (io.ReadWriteCloser, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = nil // discard server logging
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &stdio{in: in, out: out, cmd: cmd}, nil
}

// findRoot walks up from a file looking for a go.mod, falling back to the file's
// directory. This is the workspace root reported to the server.
func findRoot(absPath string) string {
	dir := filepath.Dir(absPath)
	for d := dir; ; {
		if fileExists(filepath.Join(d, "go.mod")) {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return dir
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// pathToURI converts an absolute filesystem path to a file:// URI.
func pathToURI(path string) string {
	u := url.URL{Scheme: "file", Path: path}
	return u.String()
}
