package lsp

import (
	"encoding/json"
	"strings"
)

// Method names we use. The LSP defines many more; this is the working subset.
const (
	methodInitialize         = "initialize"
	methodInitialized        = "initialized"
	methodShutdown           = "shutdown"
	methodExit               = "exit"
	methodDidOpen            = "textDocument/didOpen"
	methodDidChange          = "textDocument/didChange"
	methodDidClose           = "textDocument/didClose"
	methodCompletion         = "textDocument/completion"
	methodHover              = "textDocument/hover"
	methodPublishDiagnostics = "textDocument/publishDiagnostics"
)

// Position is a zero-based line/character offset. The unit of Character depends
// on the negotiated position encoding (utf-8 → bytes, utf-16 → code units).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is a half-open [Start, End) span.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// TextDocumentIdentifier names an open document by URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier names a document plus its monotonic version.
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentItem is the payload of didOpen.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type didOpenParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// textChange is one whole-document replacement (full text sync).
type textChange struct {
	Text string `json:"text"`
}

type didChangeParams struct {
	TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []textChange                    `json:"contentChanges"`
}

type didCloseParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// textDocumentPositionParams is the common request shape for completion/hover.
type textDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// DiagnosticSeverity values per the LSP spec.
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// Diagnostic is a single problem reported for a document.
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Code     json.RawMessage    `json:"code,omitempty"`
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// PublishDiagnosticsParams is the payload of the publishDiagnostics notification.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// CompletionItem is one suggestion.
type CompletionItem struct {
	Label      string          `json:"label"`
	Detail     string          `json:"detail,omitempty"`
	Kind       int             `json:"kind,omitempty"`
	SortText   string          `json:"sortText,omitempty"`
	FilterText string          `json:"filterText,omitempty"`
	InsertText string          `json:"insertText,omitempty"`
	TextEdit   json.RawMessage `json:"textEdit,omitempty"`
}

// Insert returns the text to insert when this item is accepted, preferring the
// explicit insert text and falling back to the label.
func (c CompletionItem) Insert() string {
	if c.InsertText != "" {
		return c.InsertText
	}
	return c.Label
}

// completionList is the object form of a completion response. The result may
// also be a bare array of items; parseCompletion handles both.
type completionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// parseCompletion decodes a completion result that may be either a
// CompletionItem[] or a CompletionList.
func parseCompletion(raw json.RawMessage) []CompletionItem {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var list completionList
	if err := json.Unmarshal(raw, &list); err == nil && list.Items != nil {
		return list.Items
	}
	var items []CompletionItem
	if err := json.Unmarshal(raw, &items); err == nil {
		return items
	}
	return nil
}

// MarkupContent is markdown/plaintext documentation.
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Hover is the result of a hover request. Contents is decoded lazily because
// the spec permits a string, a MarkupContent, or an array of marked strings.
type Hover struct {
	Contents json.RawMessage `json:"contents"`
	Range    *Range          `json:"range,omitempty"`
}

// Text flattens Hover.Contents into a plain string.
func (h *Hover) Text() string {
	if h == nil || len(h.Contents) == 0 {
		return ""
	}
	// MarkupContent { kind, value }
	var mc MarkupContent
	if err := json.Unmarshal(h.Contents, &mc); err == nil && mc.Value != "" {
		return mc.Value
	}
	// Plain string.
	var s string
	if err := json.Unmarshal(h.Contents, &s); err == nil {
		return s
	}
	// Array of strings or { language, value } objects.
	var arr []json.RawMessage
	if err := json.Unmarshal(h.Contents, &arr); err == nil {
		var b strings.Builder
		for _, e := range arr {
			if json.Unmarshal(e, &s) == nil {
				b.WriteString(s)
				b.WriteByte('\n')
				continue
			}
			var mv struct {
				Value string `json:"value"`
			}
			if json.Unmarshal(e, &mv) == nil {
				b.WriteString(mv.Value)
				b.WriteByte('\n')
			}
		}
		return b.String()
	}
	return ""
}

// ---- Initialize handshake ----

type clientInfo struct {
	Name string `json:"name"`
}

type initializeParams struct {
	ProcessID    int                `json:"processId"`
	ClientInfo   clientInfo         `json:"clientInfo"`
	RootURI      string             `json:"rootUri"`
	Capabilities clientCapabilities `json:"capabilities"`
}

type clientCapabilities struct {
	General      generalCapabilities      `json:"general"`
	TextDocument textDocumentCapabilities `json:"textDocument"`
}

type generalCapabilities struct {
	PositionEncodings []string `json:"positionEncodings"`
}

type textDocumentCapabilities struct {
	Synchronization    syncCapabilities        `json:"synchronization"`
	Completion         completionCapabilities  `json:"completion"`
	Hover              hoverCapabilities       `json:"hover"`
	PublishDiagnostics publishDiagCapabilities `json:"publishDiagnostics"`
}

type syncCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type completionCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type hoverCapabilities struct {
	ContentFormat []string `json:"contentFormat"`
}

type publishDiagCapabilities struct {
	RelatedInformation bool `json:"relatedInformation"`
}

type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
}

type serverCapabilities struct {
	PositionEncoding string `json:"positionEncoding"`
}
