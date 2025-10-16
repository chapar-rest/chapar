package gvcode

import (
	"image"

	"gioui.org/io/key"
	"gioui.org/layout"
)

// Completion is the main auto-completion interface for the editor. A Completion object
// schedules flow between the editor, the visual popup widget and completion algorithms(the Completor).
type Completion interface {
	// AddCompletors adds Completors to Completion. Completors should run independently and return
	// candidates to Completion. A popup is also required to present the cadidates to user.
	AddCompletor(completor Completor, popup CompletionPopup) error

	// OnText update the completion context. If there is no ongoing session, it should start one.
	OnText(ctx CompletionContext)
	// OnConfirm set a callback which is called when the user selected the candidates.
	OnConfirm(idx int)
	// Cancel cancels the current completion session.
	Cancel()
	// IsActive reports if the completion popup is visible.
	IsActive() bool

	// Offset returns the offset used to locate the popup when painting.
	Offset() image.Point
	// Layout layouts the completion selection box as popup near the caret.
	Layout(gtx layout.Context) layout.Dimensions
}

type CompletionPopup interface {
	Layout(gtx layout.Context, items []CompletionCandidate) layout.Dimensions
}

// Position is a position in the eidtor. Line/column and Runes may not
// be set at the same time depending on the use cases.
type Position struct {
	// Line number of the caret where the typing is happening.
	Line int
	// Column is the rune offset from the start of the line.
	Column int
	// Runes is the rune offset in the editor text of the input.
	Runes int
}

type EditRange struct {
	Start Position
	End   Position
}

type CompletionContext struct {
	// The last key input.
	Input string
	// // Prefix is the text before the caret.
	// Prefix string
	// // Suffix is the text after the caret.
	// Suffix string
	// Coordinates of the caret. Scroll off will change after we update the position,
	// so we use doc view position instead of viewport position.
	Coords image.Point
	// The position of the caret in line/column and selection range.
	Position Position
}

// CompletionCandidate are results returned from Completor, to be presented
// to the user to select from.
type CompletionCandidate struct {
	// Label is a short text shown to user to indicate
	// what the candicate looks like.
	Label string
	// TextEdit is the real text with range info to be
	// inserted into the editor.
	TextEdit TextEdit
	// A short description of the candicate.
	Description string
	// Kind of the candicate, for example, function,
	// class, keywords etc.
	Kind string
	// TextFormat defines whether the insert text in a completion item
	// should be interpreted as plain text or a snippet. The possible values are
	// PlainText or Snippet.
	TextFormat string
}

// TextEdit is the text with range info to be
// inserted into the editor, used in auto-completion.
type TextEdit struct {
	NewText   string
	EditRange EditRange
}

func NewTextEditWithRuneOffset(text string, start, end int) TextEdit {
	return TextEdit{
		NewText: text,
		EditRange: EditRange{
			Start: Position{Runes: start},
			End:   Position{Runes: end},
		},
	}
}

func NewTextEditWithPos(text string, start Position, end Position) TextEdit {
	return TextEdit{
		NewText: text,
		EditRange: EditRange{
			Start: start,
			End:   end,
		},
	}
}

// Completor defines a interface that each of the delegated completor must implement.
type Completor interface {
	Trigger() Trigger
	Suggest(ctx CompletionContext) []CompletionCandidate

	// FilterAndRank filters the passed in candidates using the pattern, and then
	// rank the result.
	FilterAndRank(pattern string, candidates []CompletionCandidate) []CompletionCandidate
}

// Trigger
type Trigger struct {
	// Characters that must be present before the caret to trigger the completion.
	// If it is empty, any character will trigger the completion.
	Characters []string

	// Trigger completion even the caret is in side of comment.
	Comment bool
	// Trigger completion even the caret is in side of string(quote pair).
	String bool

	// Special key binding triggers the completion.
	KeyBinding struct {
		Name      key.Name
		Modifiers key.Modifiers
	}
}

func (tr Trigger) ActivateOnKey(evt key.Event) bool {
	return tr.KeyBinding.Name == evt.Name &&
		evt.Modifiers.Contain(tr.KeyBinding.Modifiers)
}
