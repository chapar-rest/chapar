package lsp

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ServerConfig describes how to launch a language server for one language.
type ServerConfig struct {
	LanguageID string   // LSP language id sent in didOpen (e.g. "go")
	Command    string   // executable name resolved on PATH, or an absolute path
	Args       []string // arguments passed to the executable
	// RootMarkers are file names that mark a workspace root (e.g. "go.mod").
	// The root is the nearest ancestor directory holding one of them; with no
	// markers, or none found, it is the document's own directory.
	RootMarkers []string
}

// registry maps a file extension to its language server. gopls (.go) is built
// in; add more with Register. Servers whose Command is not on PATH degrade
// gracefully to a no-op (see Manager.Open).
var (
	registryMu sync.RWMutex
	registry   = map[string]ServerConfig{
		".go": {LanguageID: "go", Command: "gopls", Args: []string{"serve"}, RootMarkers: []string{"go.mod"}},
	}
)

// Register maps a file extension (with the leading dot, case-insensitive) to a
// language server. It is the supported way to add languages without editing this
// package. A later Register for the same extension replaces the earlier entry;
// documents already open pick the change up on Manager.Restart.
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

// Unregister removes the server for an extension, so documents of that type
// get no language server. Open documents detach on Manager.Restart.
func Unregister(ext string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(registry, strings.ToLower(ext))
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

// ErrorKind classifies a ServerError.
type ErrorKind int

const (
	// ErrNotFound: the server command is not on PATH. Reported once per
	// command until the next Restart.
	ErrNotFound ErrorKind = iota
	// ErrStart: the process failed to start or to complete the handshake.
	ErrStart
	// ErrExited: the process exited while documents were using it. It stays
	// down until Restart.
	ErrExited
	// ErrServer: the server showed an error message (window/showMessage).
	ErrServer
	// ErrRequest: a completion or hover request failed. Cancellations and
	// stale-content replies are routine and not reported.
	ErrRequest
)

func (k ErrorKind) String() string {
	switch k {
	case ErrNotFound:
		return "not found"
	case ErrStart:
		return "failed to start"
	case ErrExited:
		return "exited"
	case ErrServer:
		return "server error"
	case ErrRequest:
		return "request failed"
	}
	return "error"
}

// ServerError is a language-server problem reported to Manager.OnError.
type ServerError struct {
	Kind       ErrorKind
	LanguageID string
	Command    string
	Err        error
	// Stderr is the tail of the server's standard error, when it has any. It
	// usually explains a failed start or a crash.
	Stderr string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("%s language server (%s) %s: %v", e.LanguageID, e.Command, e.Kind, e.Err)
}

func (e *ServerError) Unwrap() error { return e.Err }

// Manager owns language-server processes and shares one per (root, language) so
// every open file in a workspace talks to a single server, the way LSP expects.
//
// Open, Restart, and the Doc methods belong to the UI thread; OnError and
// OnActivity callbacks run on background goroutines.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
	docs     map[*doc]struct{}
	missing  map[string]bool // commands already reported as not found
	onError  func(*ServerError)
	activity func()
}

// NewManager returns an empty manager. Processes start lazily on Open.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*session),
		docs:     make(map[*doc]struct{}),
		missing:  make(map[string]bool),
	}
}

// OnError sets the callback that receives language-server problems. It is
// called from background goroutines and must not block; hand the error to the
// UI thread before touching UI state.
func (m *Manager) OnError(fn func(*ServerError)) {
	m.mu.Lock()
	m.onError = fn
	m.mu.Unlock()
}

// OnActivity sets a callback fired from background goroutines whenever a
// server delivers something an editor should poll for (diagnostics, a
// completion or hover reply). Use it to request a repaint, so results show up
// in editors that are not otherwise animating.
func (m *Manager) OnActivity(fn func()) {
	m.mu.Lock()
	m.activity = fn
	m.mu.Unlock()
}

func (m *Manager) report(e *ServerError) {
	m.mu.Lock()
	fn := m.onError
	m.mu.Unlock()
	if fn != nil {
		fn(e)
	}
}

func (m *Manager) wake() {
	m.mu.Lock()
	fn := m.activity
	m.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// Open registers a document with its language server and returns a handle.
// content returns the document's current text; it is read on the calling
// goroutine now and again whenever the document re-attaches after Restart.
//
// An empty path yields a no-op Doc. Otherwise the document is tracked even when
// no server is configured for it or the binary is missing, so a later Register
// followed by Restart can attach it without reopening the editor.
func (m *Manager) Open(path string, content func() []byte) Doc {
	if path == "" {
		return noopDoc{}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	d := &doc{mgr: m, path: abs, uri: pathToURI(abs), content: content}
	m.mu.Lock()
	m.docs[d] = struct{}{}
	m.mu.Unlock()
	d.attach()
	return d
}

// Restart stops the servers for languageID ("" for every language) and
// re-attaches their documents using the current registry, so it also applies
// Register/Unregister changes. It clears the not-found memory, so a server
// installed since is picked up and a still-missing one is reported again.
func (m *Manager) Restart(languageID string) {
	m.mu.Lock()
	var stop []*session
	for k, s := range m.sessions {
		if languageID == "" || s.cfg.LanguageID == languageID {
			delete(m.sessions, k)
			stop = append(stop, s)
		}
	}
	var docs []*doc
	for d := range m.docs {
		if languageID == "" || d.languageID() == languageID {
			docs = append(docs, d)
			continue
		}
		if cfg, ok := ForPath(d.path); ok && cfg.LanguageID == languageID {
			docs = append(docs, d)
		}
	}
	clear(m.missing)
	m.mu.Unlock()

	stopped := make(map[*session]bool, len(stop))
	for _, s := range stop {
		stopped[s] = true
		s.stop()
	}
	for _, d := range docs {
		d.mu.Lock()
		old := d.sess
		d.sess = nil
		d.reset = true
		d.mu.Unlock()
		if old != nil && !stopped[old] {
			// Its extension now maps to another language; leave the old
			// server properly.
			old.do(func(c *client) { c.didClose(d.uri) })
			m.release(old)
		}
		d.attach()
	}
	m.wake()
}

// Running reports the language ids that currently have a live server.
func (m *Manager) Running() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for _, s := range m.sessions {
		if s.alive() && !seen[s.cfg.LanguageID] {
			seen[s.cfg.LanguageID] = true
			out = append(out, s.cfg.LanguageID)
		}
	}
	return out
}

// session is one running server shared by Manager.Open callers. The client is
// created asynchronously (the initialize round-trip is slow) so Open never
// blocks the UI; operations issued before the handshake completes are queued.
type session struct {
	mgr  *Manager
	key  string
	cfg  ServerConfig
	root string

	mu       sync.Mutex
	client   *client
	proc     *stdio
	queue    []func(*client)
	refs     int
	dead     bool
	stopping bool
}

func (m *Manager) session(cfg ServerConfig, root string) *session {
	key := cfg.LanguageID + "\x00" + cfg.Command + "\x00" + root
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[key]; ok {
		s.mu.Lock()
		s.refs++
		s.mu.Unlock()
		return s
	}
	s := &session{mgr: m, key: key, cfg: cfg, root: root, refs: 1}
	m.sessions[key] = s
	go s.start()
	return s
}

func (s *session) start() {
	proc, err := spawn(s.cfg)
	if err != nil {
		s.fail(ErrStart, err, "")
		return
	}
	s.mu.Lock()
	s.proc = proc
	stopping := s.stopping
	s.mu.Unlock()
	if stopping {
		proc.Close()
		return
	}

	hooks := clientHooks{
		message: func(typ int, text string) {
			if typ == MessageTypeError {
				s.report(ErrServer, errors.New(text), "")
			}
		},
		activity: s.mgr.wake,
	}
	c, err := newClient(proc, pathToURI(s.root), hooks)
	if err != nil {
		// A server that dies during the handshake shows up as a closed pipe;
		// its exit status says more.
		select {
		case <-proc.done:
		case <-time.After(time.Second):
		}
		if xe := proc.exitErr(); xe != nil {
			err = xe
		}
		proc.Close()
		s.fail(ErrStart, err, proc.stderr.String())
		return
	}

	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		go c.shutdown()
		return
	}
	s.client = c
	q := s.queue
	s.queue = nil
	s.mu.Unlock()
	for _, fn := range q {
		fn(c)
	}
	go s.watch(c, proc)
}

// watch reports a server that goes away on its own.
func (s *session) watch(c *client, proc *stdio) {
	<-c.rpc.closed
	select {
	case <-proc.done:
	case <-time.After(time.Second):
	}
	s.mu.Lock()
	stopping := s.stopping
	s.dead = true
	s.mu.Unlock()
	if stopping {
		return
	}
	err := proc.exitErr()
	if err == nil {
		err = errors.New("connection closed")
	}
	s.report(ErrExited, err, proc.stderr.String())
	s.mgr.wake()
}

func (s *session) fail(kind ErrorKind, err error, stderr string) {
	s.mu.Lock()
	stopping := s.stopping
	s.dead = true
	s.queue = nil
	s.mu.Unlock()
	if !stopping {
		s.report(kind, err, stderr)
	}
}

func (s *session) report(kind ErrorKind, err error, stderr string) {
	s.mgr.report(&ServerError{
		Kind:       kind,
		LanguageID: s.cfg.LanguageID,
		Command:    s.cfg.Command,
		Err:        err,
		Stderr:     stderr,
	})
}

// stop shuts the server down without reporting its exit.
func (s *session) stop() {
	s.mu.Lock()
	s.stopping = true
	s.dead = true
	s.queue = nil
	c, proc := s.client, s.proc
	s.mu.Unlock()
	switch {
	case c != nil:
		go c.shutdown()
	case proc != nil:
		proc.Close() // still in the handshake
	}
}

func (s *session) alive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.dead
}

// do runs fn against the client, queueing it if the handshake is still pending.
func (s *session) do(fn func(*client)) {
	s.mu.Lock()
	if s.dead {
		s.mu.Unlock()
		return
	}
	if s.client != nil {
		c := s.client
		s.mu.Unlock()
		fn(c)
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
	last := s.refs <= 0
	s.mu.Unlock()
	if !last {
		return
	}
	m.mu.Lock()
	if m.sessions[s.key] == s {
		delete(m.sessions, s.key)
	}
	m.mu.Unlock()
	s.stop()
}

// doc is the live Doc implementation. It is attached to a session while a
// server is configured and available, and detached (inert) otherwise.
type doc struct {
	mgr     *Manager
	path    string
	uri     string
	content func() []byte

	mu      sync.Mutex
	sess    *session
	version int
	nextReq int
	reset   bool // deliver an EventDiagnostics on the next Poll
}

// attach connects the document to its configured server, if any.
func (d *doc) attach() {
	cfg, ok := ForPath(d.path)
	if !ok {
		return
	}
	if _, err := exec.LookPath(cfg.Command); err != nil {
		d.mgr.reportMissing(cfg, err)
		return
	}
	sess := d.mgr.session(cfg, findRoot(d.path, cfg.RootMarkers))
	d.mu.Lock()
	d.sess = sess
	d.version++
	v := d.version
	d.mu.Unlock()
	var text string
	if d.content != nil {
		text = string(d.content())
	}
	uri := d.uri
	sess.do(func(c *client) { c.didOpen(uri, cfg.LanguageID, v, text) })
}

func (m *Manager) reportMissing(cfg ServerConfig, err error) {
	m.mu.Lock()
	seen := m.missing[cfg.Command]
	m.missing[cfg.Command] = true
	m.mu.Unlock()
	if !seen {
		m.report(&ServerError{Kind: ErrNotFound, LanguageID: cfg.LanguageID, Command: cfg.Command, Err: err})
	}
}

func (d *doc) session() *session {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sess
}

func (d *doc) languageID() string {
	if s := d.session(); s != nil {
		return s.cfg.LanguageID
	}
	return ""
}

func (d *doc) DidChange(content []byte) {
	d.mu.Lock()
	d.version++
	v := d.version
	s := d.sess
	d.mu.Unlock()
	if s == nil {
		return // no server: the text is read from content on attach
	}
	text := string(content)
	s.do(func(c *client) { c.didChange(d.uri, v, text) })
}

func (d *doc) Completion(pos Position) int {
	id := d.reqID()
	s := d.session()
	if s == nil {
		return id
	}
	s.do(func(c *client) {
		go func() {
			items, err := c.completion(d.uri, pos)
			if err != nil {
				s.requestFailed("completion", err)
				return
			}
			c.enqueue(Event{Kind: EventCompletion, URI: d.uri, ReqID: id, Completions: items})
		}()
	})
	return id
}

func (d *doc) Hover(pos Position) int {
	id := d.reqID()
	s := d.session()
	if s == nil {
		return id
	}
	s.do(func(c *client) {
		go func() {
			h, err := c.hover(d.uri, pos)
			if err != nil {
				s.requestFailed("hover", err)
				return
			}
			c.enqueue(Event{Kind: EventHover, URI: d.uri, ReqID: id, Hover: h})
		}()
	})
	return id
}

// requestFailed reports a failed request unless the failure is routine: a
// server that went away is reported once by watch, and cancelled or stale
// requests are part of normal typing.
func (s *session) requestFailed(method string, err error) {
	if errors.Is(err, io.ErrClosedPipe) {
		return
	}
	var re *rpcError
	if errors.As(err, &re) {
		switch re.Code {
		case -32800, -32801, -32802: // RequestCancelled, ContentModified, ServerCancelled
			return
		}
	}
	s.report(ErrRequest, fmt.Errorf("%s: %w", method, err), "")
}

func (d *doc) Poll() ([]Event, bool) {
	d.mu.Lock()
	s := d.sess
	reset := d.reset
	d.reset = false
	d.mu.Unlock()
	var evs []Event
	if reset {
		// The server changed: have the editor re-read (and so clear) its
		// diagnostics.
		evs = append(evs, Event{Kind: EventDiagnostics, URI: d.uri})
	}
	if s != nil {
		if c := s.getClient(); c != nil {
			evs = append(evs, c.drain(d.uri)...)
		}
	}
	return evs, len(evs) > 0
}

func (d *doc) Diagnostics() []Diagnostic {
	s := d.session()
	if s == nil {
		return nil
	}
	c := s.getClient()
	if c == nil {
		return nil
	}
	return c.diagnostics(d.uri)
}

func (d *doc) Encoding() string {
	if s := d.session(); s != nil {
		if c := s.getClient(); c != nil {
			return c.encoding
		}
	}
	return "utf-8"
}

func (d *doc) Close() {
	d.mgr.mu.Lock()
	delete(d.mgr.docs, d)
	d.mgr.mu.Unlock()
	d.mu.Lock()
	s := d.sess
	d.sess = nil
	d.mu.Unlock()
	if s == nil {
		return
	}
	uri := d.uri
	s.do(func(c *client) { c.didClose(uri) })
	d.mgr.release(s)
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

// stdio adapts a subprocess's stdin/stdout into one io.ReadWriteCloser and
// keeps the tail of its stderr for error reports.
type stdio struct {
	in     io.WriteCloser
	out    io.ReadCloser
	cmd    *exec.Cmd
	stderr *tailBuffer

	done    chan struct{} // closed when the process has exited
	waitErr error
}

func (s *stdio) Read(p []byte) (int, error)  { return s.out.Read(p) }
func (s *stdio) Write(p []byte) (int, error) { return s.in.Write(p) }

func (s *stdio) Close() error {
	_ = s.in.Close()
	_ = s.out.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	return nil
}

// exitErr returns the process's exit error, or nil if it has not exited or
// exited cleanly.
func (s *stdio) exitErr() error {
	select {
	case <-s.done:
		if s.waitErr != nil {
			return s.waitErr
		}
		return errors.New("stopped unexpectedly (exit status 0)")
	default:
		return nil
	}
}

// spawn launches the configured server and wires its stdio.
func spawn(cfg ServerConfig) (*stdio, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	s := &stdio{in: in, out: out, cmd: cmd, stderr: &tailBuffer{max: 8 << 10}, done: make(chan struct{})}
	cmd.Stderr = s.stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		s.waitErr = cmd.Wait()
		close(s.done)
	}()
	return s, nil
}

// tailBuffer is an io.Writer that keeps the last max bytes written.
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.max; over > 0 {
		t.buf = append(t.buf[:0], t.buf[over:]...)
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.TrimSpace(string(t.buf))
}

// findRoot walks up from a file looking for one of markers, falling back to the
// file's directory. This is the workspace root reported to the server.
func findRoot(absPath string, markers []string) string {
	dir := filepath.Dir(absPath)
	if len(markers) == 0 {
		return dir
	}
	for d := dir; ; {
		for _, m := range markers {
			if fileExists(filepath.Join(d, m)) {
				return d
			}
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
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // Windows drive paths: file:///C:/...
	}
	u := url.URL{Scheme: "file", Path: p}
	return u.String()
}
