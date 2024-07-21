package widgets

import giovieweditor "github.com/oligo/gioview/editor"

var (
	singleBracket = `(\{[a-zA-Z0-9_]+\})`
	doubleBracket = `(\{\{[a-zA-Z0-9_]+\}\})`
)

// PatternEditor is a widget that allows the user to edit a text like and highlight patterns like {{id}} or {name}
type PatternEditor struct {
	editor giovieweditor.Editor
	Keys   map[string]string
}

// NewPatternEditor creates a new PatternEditor
func NewPatternEditor() *PatternEditor {
	return &PatternEditor{
		editor: giovieweditor.Editor{},
		Keys:   make(map[string]string),
	}
}
