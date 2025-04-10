package gvcode

import (
	"image"
	"slices"

	"gioui.org/io/key"
	"gioui.org/layout"
)

// Completion is the main auto-completion interface for the editor. A Completion object
// schedules flow between the editor, the visual popup widget and completion algorithms(the Completor).
type Completion interface {
	// Set in what conditions the completion should be activated.
	SetTriggers(triggers ...Trigger)
	// SetCompletors adds Completors to Completion. Completors should run independently and return
	// candicates to Completion. All candicates are then re-ranked and presented to the user.
	SetCompletors(completors ...Completor)

	// SetPopup set the popup widget to be displayed when completion is activated.
	SetPopup(popup CompletionPopup)

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

type CompletionPopup func(gtx layout.Context, items []CompletionCandidate) layout.Dimensions

type CompletionContext struct {
	// start new session if there is no active session.
	New bool
	// Input is the text to complete.
	Input    string
	Position struct {
		// Line number of the caret where the typing is happening.
		Line int
		// Column is the rune offset from the start of the line.
		Column int
		// Coordinates of the caret. Scroll off will change after we update the position,
		// so we use doc view position instead of viewport position.
		Coords image.Point
		// Start is the start offset in the editor text of the input, measured in runes.
		Start int
		// End is the end offset in the editor text of the input, measured in runes.
		End int
	}
}

type CompletionCandidate struct {
	Label       string
	InsertText  string
	Description string
	Kind        string
}

type Completor interface {
	Suggest(ctx CompletionContext) []CompletionCandidate
}

type Trigger interface {
	ImplementsTrigger()
}

// Completion works as you type.
// This is mutually exclusive with PrefixTrigger.
type AutoTrigger struct {
	// The minimum length in runes of the input to trigger completion.
	MinSize int
}

// Special prefix characters triggers the completion.
// This is mutually exclusive with AutoTrigger.
type PrefixTrigger struct {
	// Prefix that must be present to trigger the completion.
	// If it is empty, any character will trigger the completion.
	Prefix string
}

// Special key binding triggers the completion.
type KeyTrigger struct {
	Name      key.Name
	Modifiers key.Modifiers
}

func GetCompletionTrigger[T Trigger](triggers []Trigger) (T, bool) {
	idx := slices.IndexFunc(triggers, func(item Trigger) bool {
		_, ok := item.(T)
		return ok
	})

	if idx < 0 {
		return *new(T), false
	}

	return triggers[idx].(T), true
}

func (a AutoTrigger) ImplementsTrigger()   {}
func (p PrefixTrigger) ImplementsTrigger() {}
func (k KeyTrigger) ImplementsTrigger()    {}
