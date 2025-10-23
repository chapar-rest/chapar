package gvcode

// Mode defines a mode for the editor. The editor can be switched
// back and forth bewteen different modes, depending on the context.
type EditorMode uint8

const (
	// ModeNormal is the default mode for the editor. Users can
	// insert or select text or anything else.
	ModeNormal EditorMode = iota

	// ModeReadOnly controls whether the contents of the editor can be
	// altered by user interaction. If set, the editor will allow selecting
	// text and copying it interactively, but not modifying it. Users can
	// enter or quit this mode via user commands.
	ModeReadOnly

	// ModeSnippet put the editor into the snippet mode required by LSP protocol.
	// The users can navigate bewteen the snippet locations/placeholders using
	// the tab/shift-tab keys. And clicking or pressing the ESC key switched the
	// editor to normal mode.
	//
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#snippet_syntax.
	ModeSnippet
)

func (e *Editor) setMode(mode EditorMode) {
	switch mode {
	case ModeNormal, ModeReadOnly:
		if e.snippetCtx != nil {
			e.snippetCtx.Cancel()
			e.snippetCtx = nil
		}
	}

	e.mode = mode
}
