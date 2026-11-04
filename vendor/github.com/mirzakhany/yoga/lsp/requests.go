package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ---- protocol types for navigation, editing and symbols ----

// Location is a range in a document.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Path returns the location's file path.
func (l Location) Path() string { return URIToPath(l.URI) }

type locationLink struct {
	TargetURI            string `json:"targetUri"`
	TargetRange          Range  `json:"targetRange"`
	TargetSelectionRange Range  `json:"targetSelectionRange"`
}

// TextEdit replaces Range with NewText.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// WorkspaceEdit is a set of edits across documents, keyed by URI.
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
	// DocumentChanges is folded into Changes by Manager methods; file
	// create/rename/delete operations are not supported and are dropped.
	DocumentChanges []json.RawMessage `json:"documentChanges,omitempty"`
}

// Edits returns the edits by file path, DocumentChanges included.
func (w WorkspaceEdit) Edits() map[string][]TextEdit {
	out := map[string][]TextEdit{}
	for uri, es := range w.Changes {
		out[URIToPath(uri)] = append(out[URIToPath(uri)], es...)
	}
	for _, raw := range w.DocumentChanges {
		var dc struct {
			TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
			Edits        []TextEdit                      `json:"edits"`
		}
		if json.Unmarshal(raw, &dc) == nil && dc.TextDocument.URI != "" {
			p := URIToPath(dc.TextDocument.URI)
			out[p] = append(out[p], dc.Edits...)
		}
	}
	return out
}

// FormattingOptions are the editor settings sent with formatting requests.
type FormattingOptions struct {
	TabSize      int  `json:"tabSize"`
	InsertSpaces bool `json:"insertSpaces"`
}

// SymbolKind is the LSP symbol kind (1 File … 26 TypeParameter).
type SymbolKind int

// DocumentSymbol is one symbol in a document outline.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolInformation is a workspace symbol search result.
type SymbolInformation struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Location      Location   `json:"location"`
	ContainerName string     `json:"containerName,omitempty"`
}

// SignatureHelp describes the call being typed.
type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature"`
	ActiveParameter int                    `json:"activeParameter"`
}

// SignatureInformation is one overload.
type SignatureInformation struct {
	Label         string                 `json:"label"`
	Documentation json.RawMessage        `json:"documentation,omitempty"`
	Parameters    []ParameterInformation `json:"parameters,omitempty"`
}

// ParameterInformation labels one parameter: a substring of the signature
// label, or [start, end] offsets into it.
type ParameterInformation struct {
	Label json.RawMessage `json:"label"`
}

// CodeAction is a fix or refactoring the server offers.
type CodeAction struct {
	Title       string          `json:"title"`
	Kind        string          `json:"kind,omitempty"`
	IsPreferred bool            `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit  `json:"edit,omitempty"`
	Command     *Command        `json:"command,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
}

// Command is a server command, run with Manager.ExecuteCommand.
type Command struct {
	Title     string            `json:"title"`
	Command   string            `json:"command"`
	Arguments []json.RawMessage `json:"arguments,omitempty"`
}

// ---- errors ----

var (
	// ErrNoDocument: the path is not open in any editor.
	ErrNoDocument = errors.New("lsp: document is not open")
	// ErrNoServer: no language server is attached to the document.
	ErrNoServer = errors.New("lsp: no language server for this file")
	// ErrUnsupported: the server does not offer the feature.
	ErrUnsupported = errors.New("lsp: the language server does not support this")
)

// ---- Manager request API ----
//
// These methods block until the server answers or ctx ends, so call them from
// a goroutine, never the UI thread. Positions use the server's encoding;
// Encoding reports it, and Editor converts offsets for you.

// OnApplyEdit sets the handler for a server's workspace/applyEdit request
// (sent while running some commands and code actions). It runs on a
// background goroutine and reports whether the edit was applied; without a
// handler such requests are refused. encoding is the requesting server's
// position encoding.
func (m *Manager) OnApplyEdit(fn func(edit WorkspaceEdit, encoding string) bool) {
	m.mu.Lock()
	m.applyEdit = fn
	m.mu.Unlock()
}

// Encoding reports the position encoding of path's server ("utf-8" when no
// server is ready).
func (m *Manager) Encoding(path string) string {
	if d := m.docFor(path); d != nil {
		return d.Encoding()
	}
	return "utf-8"
}

// Supports reports whether path's server advertises a capability, such as
// "definitionProvider" or "documentFormattingProvider".
func (m *Manager) Supports(path, capability string) bool {
	c, _, err := m.ready(context.Background(), path, false)
	if err != nil {
		return false
	}
	raw, ok := c.caps[capability]
	if !ok {
		return false
	}
	s := strings.TrimSpace(string(raw))
	return s != "false" && s != "null"
}

// Definition returns where the symbol at pos is defined.
func (m *Manager) Definition(ctx context.Context, path string, pos Position) ([]Location, error) {
	return m.locations(ctx, path, "textDocument/definition", "definitionProvider", pos, nil)
}

// TypeDefinition returns where the type of the symbol at pos is defined.
func (m *Manager) TypeDefinition(ctx context.Context, path string, pos Position) ([]Location, error) {
	return m.locations(ctx, path, "textDocument/typeDefinition", "typeDefinitionProvider", pos, nil)
}

// Implementation returns the implementations of the interface or method at pos.
func (m *Manager) Implementation(ctx context.Context, path string, pos Position) ([]Location, error) {
	return m.locations(ctx, path, "textDocument/implementation", "implementationProvider", pos, nil)
}

// References returns the uses of the symbol at pos.
func (m *Manager) References(ctx context.Context, path string, pos Position, includeDeclaration bool) ([]Location, error) {
	return m.locations(ctx, path, "textDocument/references", "referencesProvider", pos,
		map[string]any{"includeDeclaration": includeDeclaration})
}

func (m *Manager) locations(ctx context.Context, path, method, capability string, pos Position, refCtx map[string]any) ([]Location, error) {
	params := map[string]any{"textDocument": TextDocumentIdentifier{URI: pathToURI(abs(path))}, "position": pos}
	if refCtx != nil {
		params["context"] = refCtx
	}
	raw, err := m.call(ctx, path, method, capability, params)
	if err != nil {
		return nil, err
	}
	return parseLocations(raw), nil
}

// parseLocations accepts Location, []Location and []LocationLink replies.
func parseLocations(raw json.RawMessage) []Location {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return nil
	}
	if s[0] == '{' {
		var l Location
		if json.Unmarshal(raw, &l) == nil && l.URI != "" {
			return []Location{l}
		}
		return nil
	}
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	out := make([]Location, 0, len(items))
	for _, it := range items {
		var link locationLink
		if json.Unmarshal(it, &link) == nil && link.TargetURI != "" {
			out = append(out, Location{URI: link.TargetURI, Range: link.TargetSelectionRange})
			continue
		}
		var l Location
		if json.Unmarshal(it, &l) == nil && l.URI != "" {
			out = append(out, l)
		}
	}
	return out
}

// HoverText returns the hover documentation for pos as plain text.
func (m *Manager) HoverText(ctx context.Context, path string, pos Position) (string, error) {
	raw, err := m.call(ctx, path, "textDocument/hover", "hoverProvider", map[string]any{
		"textDocument": TextDocumentIdentifier{URI: pathToURI(abs(path))},
		"position":     pos,
	})
	if err != nil {
		return "", err
	}
	if s := strings.TrimSpace(string(raw)); s == "" || s == "null" {
		return "", nil
	}
	var h Hover
	if err := json.Unmarshal(raw, &h); err != nil {
		return "", err
	}
	return h.Text(), nil
}

// Rename renames the symbol at pos everywhere and returns the edits to make.
func (m *Manager) Rename(ctx context.Context, path string, pos Position, newName string) (WorkspaceEdit, error) {
	var we WorkspaceEdit
	raw, err := m.call(ctx, path, "textDocument/rename", "renameProvider", map[string]any{
		"textDocument": TextDocumentIdentifier{URI: pathToURI(abs(path))},
		"position":     pos,
		"newName":      newName,
	})
	if err != nil {
		return we, err
	}
	err = unmarshalResult(raw, &we)
	return we, err
}

// Format returns edits that format the whole document.
func (m *Manager) Format(ctx context.Context, path string, opts FormattingOptions) ([]TextEdit, error) {
	var edits []TextEdit
	raw, err := m.call(ctx, path, "textDocument/formatting", "documentFormattingProvider", map[string]any{
		"textDocument": TextDocumentIdentifier{URI: pathToURI(abs(path))},
		"options":      opts,
	})
	if err != nil {
		return nil, err
	}
	err = unmarshalResult(raw, &edits)
	return edits, err
}

// FormatRange returns edits that format part of the document.
func (m *Manager) FormatRange(ctx context.Context, path string, r Range, opts FormattingOptions) ([]TextEdit, error) {
	var edits []TextEdit
	raw, err := m.call(ctx, path, "textDocument/rangeFormatting", "documentRangeFormattingProvider", map[string]any{
		"textDocument": TextDocumentIdentifier{URI: pathToURI(abs(path))},
		"range":        r,
		"options":      opts,
	})
	if err != nil {
		return nil, err
	}
	err = unmarshalResult(raw, &edits)
	return edits, err
}

// DocumentSymbols returns the document's outline. Servers that answer with a
// flat SymbolInformation list get it converted, without nesting.
func (m *Manager) DocumentSymbols(ctx context.Context, path string) ([]DocumentSymbol, error) {
	raw, err := m.call(ctx, path, "textDocument/documentSymbol", "documentSymbolProvider", map[string]any{
		"textDocument": TextDocumentIdentifier{URI: pathToURI(abs(path))},
	})
	if err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if err := unmarshalResult(raw, &items); err != nil {
		return nil, err
	}
	out := make([]DocumentSymbol, 0, len(items))
	for _, it := range items {
		var probe struct {
			Location *Location `json:"location"`
		}
		_ = json.Unmarshal(it, &probe)
		if probe.Location != nil {
			var si SymbolInformation
			if json.Unmarshal(it, &si) == nil {
				out = append(out, DocumentSymbol{Name: si.Name, Detail: si.ContainerName, Kind: si.Kind,
					Range: si.Location.Range, SelectionRange: si.Location.Range})
			}
			continue
		}
		var ds DocumentSymbol
		if json.Unmarshal(it, &ds) == nil {
			out = append(out, ds)
		}
	}
	return out, nil
}

// WorkspaceSymbols searches symbols across the workspace of path's server.
func (m *Manager) WorkspaceSymbols(ctx context.Context, path, query string) ([]SymbolInformation, error) {
	var out []SymbolInformation
	raw, err := m.call(ctx, path, "workspace/symbol", "workspaceSymbolProvider", map[string]any{"query": query})
	if err != nil {
		return nil, err
	}
	err = unmarshalResult(raw, &out)
	return out, err
}

// SignatureHelp describes the call around pos.
func (m *Manager) SignatureHelp(ctx context.Context, path string, pos Position) (*SignatureHelp, error) {
	raw, err := m.call(ctx, path, "textDocument/signatureHelp", "signatureHelpProvider", map[string]any{
		"textDocument": TextDocumentIdentifier{URI: pathToURI(abs(path))},
		"position":     pos,
	})
	if err != nil {
		return nil, err
	}
	if s := strings.TrimSpace(string(raw)); s == "" || s == "null" {
		return nil, nil
	}
	var sh SignatureHelp
	if err := json.Unmarshal(raw, &sh); err != nil {
		return nil, err
	}
	return &sh, nil
}

// CodeActions returns fixes and refactorings for r. diags are the
// diagnostics overlapping r, which most quick fixes key on.
func (m *Manager) CodeActions(ctx context.Context, path string, r Range, diags []Diagnostic) ([]CodeAction, error) {
	if diags == nil {
		diags = []Diagnostic{}
	}
	raw, err := m.call(ctx, path, "textDocument/codeAction", "codeActionProvider", map[string]any{
		"textDocument": TextDocumentIdentifier{URI: pathToURI(abs(path))},
		"range":        r,
		"context":      map[string]any{"diagnostics": diags},
	})
	if err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if err := unmarshalResult(raw, &items); err != nil {
		return nil, err
	}
	var out []CodeAction
	for _, it := range items {
		var ca CodeAction
		if json.Unmarshal(it, &ca) != nil {
			continue
		}
		if ca.Command == nil && ca.Edit == nil {
			// A bare Command, not a CodeAction literal.
			var cmd Command
			if json.Unmarshal(it, &cmd) == nil && cmd.Command != "" {
				ca = CodeAction{Title: cmd.Title, Command: &cmd}
			}
		}
		out = append(out, ca)
	}
	return out, nil
}

// ResolveCodeAction fills in a lazily computed action's edit.
func (m *Manager) ResolveCodeAction(ctx context.Context, path string, ca CodeAction) (CodeAction, error) {
	raw, err := m.call(ctx, path, "codeAction/resolve", "", ca)
	if err != nil {
		return ca, err
	}
	var out CodeAction
	if err := json.Unmarshal(raw, &out); err != nil {
		return ca, err
	}
	return out, nil
}

// ExecuteCommand runs a server command. Its edits arrive as applyEdit
// requests (see OnApplyEdit).
func (m *Manager) ExecuteCommand(ctx context.Context, path string, cmd Command) error {
	_, err := m.call(ctx, path, "workspace/executeCommand", "executeCommandProvider", map[string]any{
		"command":   cmd.Command,
		"arguments": cmd.Arguments,
	})
	return err
}

// call sends method to path's server once it is ready. capability, when set,
// must be advertised.
func (m *Manager) call(ctx context.Context, path, method, capability string, params any) (json.RawMessage, error) {
	c, _, err := m.ready(ctx, path, true)
	if err != nil {
		return nil, err
	}
	if capability != "" {
		raw, ok := c.caps[capability]
		if s := strings.TrimSpace(string(raw)); !ok || s == "false" || s == "null" {
			return nil, ErrUnsupported
		}
	}
	return c.rpc.callCtx(ctx, method, params)
}

// ready returns the client serving path, waiting for its handshake when wait
// is set.
func (m *Manager) ready(ctx context.Context, path string, wait bool) (*client, *doc, error) {
	d := m.docFor(path)
	if d == nil {
		return nil, nil, ErrNoDocument
	}
	s := d.session()
	if s == nil {
		return nil, d, ErrNoServer
	}
	if wait {
		select {
		case <-s.ready:
		case <-ctx.Done():
			return nil, d, ctx.Err()
		}
	}
	c := s.getClient()
	if c == nil {
		return nil, d, ErrNoServer
	}
	return c, d, nil
}

func (m *Manager) docFor(path string) *doc {
	p := abs(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	for d := range m.docs {
		if d.path == p {
			return d
		}
	}
	return nil
}

func abs(path string) string {
	if a, err := filepath.Abs(path); err == nil {
		return a
	}
	return path
}

func unmarshalResult(raw json.RawMessage, v any) error {
	if s := strings.TrimSpace(string(raw)); s == "" || s == "null" {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// serverRequest answers requests from the server.
func (s *session) serverRequest(method string, params json.RawMessage) any {
	switch method {
	case "workspace/configuration":
		var p struct {
			Items []struct {
				Section string `json:"section"`
			} `json:"items"`
		}
		_ = json.Unmarshal(params, &p)
		out := make([]any, len(p.Items))
		for i, it := range p.Items {
			if it.Section == "" {
				if s.cfg.Settings != nil {
					out[i] = s.cfg.Settings
				}
				continue
			}
			out[i] = lookupSection(s.cfg.Settings, it.Section)
		}
		return out
	case "workspace/applyEdit":
		var p struct {
			Edit WorkspaceEdit `json:"edit"`
		}
		_ = json.Unmarshal(params, &p)
		s.mgr.mu.Lock()
		fn := s.mgr.applyEdit
		s.mgr.mu.Unlock()
		enc := "utf-16"
		if c := s.getClient(); c != nil {
			enc = c.encoding
		}
		applied := fn != nil && fn(p.Edit, enc)
		return map[string]any{"applied": applied}
	}
	return serverRequestResult(method, params)
}

// lookupSection finds a dotted section ("gopls" or "python.analysis").
func lookupSection(settings map[string]any, section string) any {
	if v, ok := settings[section]; ok {
		return v
	}
	var cur any = settings
	for _, part := range strings.Split(section, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		if cur, ok = m[part]; !ok {
			return nil
		}
	}
	return cur
}

// ---- helpers ----

// URIToPath converts a file:// URI to a filesystem path.
func URIToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return uri
	}
	p := u.Path
	if runtime.GOOS == "windows" {
		p = strings.TrimPrefix(p, "/")
	}
	return filepath.FromSlash(p)
}

// PathToURI converts an absolute path to a file:// URI.
func PathToURI(path string) string { return pathToURI(abs(path)) }

// OffsetAt converts an LSP position to a byte offset in text, for the given
// encoding ("utf-8", "utf-16" or "utf-32"). Positions past a line's end clamp
// to it, and lines past the end clamp to the end of text.
func OffsetAt(text []byte, pos Position, encoding string) int {
	off := 0
	for line := 0; line < pos.Line; line++ {
		i := bytes.IndexByte(text[off:], '\n')
		if i < 0 {
			return len(text)
		}
		off += i + 1
	}
	end := len(text)
	if i := bytes.IndexByte(text[off:], '\n'); i >= 0 {
		end = off + i
	}
	lineText := text[off:end]
	if encoding == "utf-8" {
		return off + min(pos.Character, len(lineText))
	}
	units := 0
	for i, r := range string(lineText) {
		if units >= pos.Character {
			return off + i
		}
		if encoding == "utf-32" {
			units++
		} else {
			units += utf16.RuneLen(r)
		}
	}
	return end
}

// PositionAt converts a byte offset in text to an LSP position.
func PositionAt(text []byte, off int, encoding string) Position {
	off = max(0, min(off, len(text)))
	line, start := 0, 0
	for i := 0; i < off; i++ {
		if text[i] == '\n' {
			line++
			start = i + 1
		}
	}
	seg := text[start:off]
	switch encoding {
	case "utf-8":
		return Position{Line: line, Character: len(seg)}
	case "utf-32":
		return Position{Line: line, Character: utf8.RuneCount(seg)}
	}
	n := 0
	for _, r := range string(seg) {
		n += utf16.RuneLen(r)
	}
	return Position{Line: line, Character: n}
}

// ApplyEdits applies LSP text edits to text. Edits must not overlap; their
// order does not matter.
func ApplyEdits(text []byte, edits []TextEdit, encoding string) ([]byte, error) {
	type span struct {
		lo, hi int
		s      string
	}
	spans := make([]span, 0, len(edits))
	for _, e := range edits {
		lo, hi := OffsetAt(text, e.Range.Start, encoding), OffsetAt(text, e.Range.End, encoding)
		if hi < lo {
			return nil, fmt.Errorf("lsp: edit range ends before it starts")
		}
		spans = append(spans, span{lo, hi, e.NewText})
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].lo < spans[j].lo })
	var b strings.Builder
	b.Grow(len(text))
	pos := 0
	for _, sp := range spans {
		if sp.lo < pos {
			return nil, fmt.Errorf("lsp: overlapping edits")
		}
		b.Write(text[pos:sp.lo])
		b.WriteString(sp.s)
		pos = sp.hi
	}
	b.Write(text[pos:])
	return []byte(b.String()), nil
}
