package lsp

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

// EventKind tags the variant of an async Event.
type EventKind int

const (
	// EventDiagnostics signals that Diagnostics() has fresh data for the URI.
	EventDiagnostics EventKind = iota
	// EventCompletion carries the result of a Completion request (ReqID set).
	EventCompletion
	// EventHover carries the result of a Hover request (ReqID set).
	EventHover
)

// Event is an async result delivered to the editor via Doc.Poll. It is a tagged
// union: only the fields relevant to Kind are populated.
type Event struct {
	Kind        EventKind
	URI         string
	ReqID       int
	Completions []CompletionItem
	Hover       *Hover
}

// client wraps one language-server connection. A single client is shared by all
// open documents that belong to the same server (see Manager); per-document
// fan-out is done by URI on the editor side.
type client struct {
	rpc      *rpcConn
	encoding string // negotiated position encoding ("utf-8" or "utf-16")

	mu     sync.Mutex
	diags  map[string][]Diagnostic
	events []Event
}

// newClient performs the initialize/initialized handshake over rwc and returns a
// ready client. rootURI is the workspace root (a file:// URI).
func newClient(rwc io.ReadWriteCloser, rootURI string) (*client, error) {
	c := &client{
		encoding: "utf-16", // LSP default until the server says otherwise
		diags:    make(map[string][]Diagnostic),
	}
	c.rpc = newRPC(rwc, c.onNotify)

	params := initializeParams{
		ProcessID:  os.Getpid(),
		ClientInfo: clientInfo{Name: "yoga"},
		RootURI:    rootURI,
		Capabilities: clientCapabilities{
			General: generalCapabilities{
				// Prefer utf-8 so an LSP character offset equals the byte column
				// the editor already computes; fall back to utf-16 if refused.
				PositionEncodings: []string{"utf-8", "utf-16"},
			},
			TextDocument: textDocumentCapabilities{
				Hover:              hoverCapabilities{ContentFormat: []string{"plaintext", "markdown"}},
				PublishDiagnostics: publishDiagCapabilities{RelatedInformation: false},
			},
		},
	}
	raw, err := c.rpc.call(methodInitialize, params)
	if err != nil {
		c.rpc.Close()
		return nil, err
	}
	var res initializeResult
	if json.Unmarshal(raw, &res) == nil && res.Capabilities.PositionEncoding != "" {
		c.encoding = res.Capabilities.PositionEncoding
	}
	if err := c.rpc.notify(methodInitialized, struct{}{}); err != nil {
		c.rpc.Close()
		return nil, err
	}
	return c, nil
}

func (c *client) onNotify(method string, params json.RawMessage) {
	if method != methodPublishDiagnostics {
		return
	}
	var p PublishDiagnosticsParams
	if json.Unmarshal(params, &p) != nil {
		return
	}
	c.mu.Lock()
	c.diags[p.URI] = p.Diagnostics
	c.events = append(c.events, Event{Kind: EventDiagnostics, URI: p.URI})
	c.mu.Unlock()
}

// ---- text synchronization (notifications) ----

func (c *client) didOpen(uri, languageID string, version int, text string) {
	_ = c.rpc.notify(methodDidOpen, didOpenParams{
		TextDocument: TextDocumentItem{URI: uri, LanguageID: languageID, Version: version, Text: text},
	})
}

func (c *client) didChange(uri string, version int, text string) {
	_ = c.rpc.notify(methodDidChange, didChangeParams{
		TextDocument:   VersionedTextDocumentIdentifier{URI: uri, Version: version},
		ContentChanges: []textChange{{Text: text}},
	})
}

func (c *client) didClose(uri string) {
	_ = c.rpc.notify(methodDidClose, didCloseParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})
}

// ---- requests (blocking; call from worker goroutines) ----

func (c *client) completion(uri string, pos Position) ([]CompletionItem, error) {
	raw, err := c.rpc.call(methodCompletion, textDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
	})
	if err != nil {
		return nil, err
	}
	return parseCompletion(raw), nil
}

func (c *client) hover(uri string, pos Position) (*Hover, error) {
	raw, err := c.rpc.call(methodHover, textDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
	})
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var h Hover
	if err := json.Unmarshal(raw, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// ---- event queue (drained by the editor, filtered per document) ----

func (c *client) enqueue(ev Event) {
	c.mu.Lock()
	c.events = append(c.events, ev)
	c.mu.Unlock()
}

// drain removes and returns events for the given URI, leaving others queued.
func (c *client) drain(uri string) []Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.events) == 0 {
		return nil
	}
	var out, keep []Event
	for _, e := range c.events {
		if e.URI == uri {
			out = append(out, e)
		} else {
			keep = append(keep, e)
		}
	}
	c.events = keep
	return out
}

func (c *client) diagnostics(uri string) []Diagnostic {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.diags[uri]
}

func (c *client) shutdown() {
	// Best-effort graceful stop; ignore errors since the process may be dying.
	_, _ = c.rpc.call(methodShutdown, nil)
	_ = c.rpc.notify(methodExit, nil)
	c.rpc.Close()
}
