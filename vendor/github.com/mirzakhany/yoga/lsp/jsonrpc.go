// Package lsp is a small Language Server Protocol client. It speaks JSON-RPC 2.0
// over a language server's stdio and surfaces diagnostics, hover, and completion
// to the editor.
//
// The design mirrors the highlight package: all blocking I/O (the RPC read loop
// and request round-trips) runs on goroutines, and the editor only ever does a
// non-blocking Poll. Results cross the goroutine boundary as plain Go structs,
// so the UI thread never touches a live connection.
//
// Transport is hand-rolled on top of an io.ReadWriteCloser (no third-party LSP
// or JSON-RPC dependency, only the standard library). Building on an
// io.ReadWriteCloser rather than directly on os/exec lets the client be unit
// tested over a net.Pipe with a fake in-process server — no real language
// server binary required.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// rpcError is a JSON-RPC 2.0 error object.
type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("lsp rpc error %d: %s", e.Code, e.Message)
}

// rpcMessage is the union of every frame shape we read: responses (id+result or
// id+error), server→client requests (id+method), and notifications (method only).
type rpcMessage struct {
	ID     *json.RawMessage `json:"id,omitempty"`
	Method string           `json:"method,omitempty"`
	Params json.RawMessage  `json:"params,omitempty"`
	Result json.RawMessage  `json:"result,omitempty"`
	Error  *rpcError        `json:"error,omitempty"`
}

// notifyFunc receives server→client notifications (e.g. publishDiagnostics).
type notifyFunc func(method string, params json.RawMessage)

// rpcConn frames JSON-RPC messages over an io.ReadWriteCloser and correlates
// request ids with their replies.
type rpcConn struct {
	rwc io.ReadWriteCloser
	w   *bufio.Writer
	wmu sync.Mutex // serializes frame writes

	mu      sync.Mutex
	nextID  int
	pending map[int]chan rpcMessage

	onNotify notifyFunc

	closeOnce sync.Once
	closed    chan struct{}
}

// newRPC starts the read loop and returns a ready connection. onNotify may be
// nil; it is invoked from the read goroutine, so it must not block.
func newRPC(rwc io.ReadWriteCloser, onNotify notifyFunc) *rpcConn {
	c := &rpcConn{
		rwc:      rwc,
		w:        bufio.NewWriter(rwc),
		pending:  make(map[int]chan rpcMessage),
		onNotify: onNotify,
		closed:   make(chan struct{}),
	}
	go c.readLoop(bufio.NewReader(rwc))
	return c
}

// notify sends a fire-and-forget notification.
func (c *rpcConn) notify(method string, params any) error {
	return c.writeFrame(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

// call sends a request and blocks until the matching response arrives or the
// connection closes. It is meant to be called from worker goroutines, never the
// UI thread.
func (c *rpcConn) call(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan rpcMessage, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	err := c.writeFrame(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	select {
	case msg := <-ch:
		if msg.Error != nil {
			return nil, msg.Error
		}
		return msg.Result, nil
	case <-c.closed:
		return nil, io.ErrClosedPipe
	}
}

// Close shuts down the transport. The read loop unblocks pending calls.
func (c *rpcConn) Close() error {
	err := c.rwc.Close()
	c.markClosed()
	return err
}

func (c *rpcConn) markClosed() {
	c.closeOnce.Do(func() { close(c.closed) })
}

func (c *rpcConn) writeFrame(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if _, err := fmt.Fprintf(c.w, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	if _, err := c.w.Write(data); err != nil {
		return err
	}
	return c.w.Flush()
}

func (c *rpcConn) readLoop(r *bufio.Reader) {
	defer c.markClosed()
	for {
		n, err := readContentLength(r)
		if err != nil {
			return
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(r, body); err != nil {
			return
		}
		var msg rpcMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			continue // skip malformed frames rather than tearing down the link
		}
		c.dispatch(msg)
	}
}

func (c *rpcConn) dispatch(msg rpcMessage) {
	switch {
	case msg.Method != "" && msg.ID != nil:
		// Server→client request. We implement no client-side features, so reply
		// with a null result to keep the server from blocking (e.g. on
		// client/registerCapability or workspace/configuration).
		_ = c.writeFrame(map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(*msg.ID),
			"result":  nil,
		})
	case msg.Method != "":
		// Notification.
		if c.onNotify != nil {
			c.onNotify(msg.Method, msg.Params)
		}
	case msg.ID != nil:
		// Response to one of our requests.
		var id int
		if err := json.Unmarshal(*msg.ID, &id); err != nil {
			return
		}
		c.mu.Lock()
		ch := c.pending[id]
		c.mu.Unlock()
		if ch != nil {
			ch <- msg
		}
	}
}

// readContentLength consumes the header block and returns the body length.
func readContentLength(r *bufio.Reader) (int, error) {
	n := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // end of headers
		}
		if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			n, err = strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return 0, err
			}
		}
	}
	if n < 0 {
		return 0, fmt.Errorf("lsp: frame missing Content-Length header")
	}
	return n, nil
}
