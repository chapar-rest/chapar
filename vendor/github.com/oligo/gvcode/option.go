package gvcode

import (
	"gioui.org/font"
	"gioui.org/text"
	"gioui.org/unit"
)

// EditorOption defines a function to configure the editor.
type EditorOption func(*Editor)

// WithOptions applies various options to configure the editor.
func (e *Editor) WithOptions(opts ...EditorOption) {
	for _, opt := range opts {
		opt(e)
	}
}

// WithShaperParams set the basic shaping params for the editor.
func WithShaperParams(font font.Font, textSize unit.Sp, alignment text.Alignment, lineHeight unit.Sp, lineHeightScale float32) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		e.text.Font = font
		e.text.TextSize = textSize
		e.text.Alignment = alignment
		e.text.LineHeight = lineHeight
		e.text.LineHeightScale = lineHeightScale
	}
}

// WithTabWidth set how many spaces to represent a tab character. In the case of
// soft tab, this determines the number of space characters to insert into the editor.
// While for hard tab, this controls the maximum width of the 'tab' glyph to expand to.
func WithTabWidth(tabWidth int) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		e.text.TabWidth = tabWidth
	}
}

// WithSoftTab controls the behaviour when user try to insert a Tab character.
// If set to true, the editor will insert the amount of space characters specified by
// TabWidth, else the editor insert a \t character.
func WithSoftTab(enabled bool) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		e.text.SoftTab = enabled
	}
}

// WithWordSeperators configures a set of characters that will be used as word separators
// when doing word related operations, like navigating or deleting by word.
func WithWordSeperators(seperators string) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		e.text.WordSeperators = seperators
	}
}

// WithQuotePairs configures a set of quote pairs that can be auto-completed when the left
// half is entered.
func WithQuotePairs(quotePairs map[rune]rune) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		e.text.BracketsQuotes.SetQuotes(quotePairs)
	}
}

// WithBracketPairs configures a set of bracket pairs that can be auto-completed when the left
// half is entered.
func WithBracketPairs(bracketPairs map[rune]rune) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		e.text.BracketsQuotes.SetBrackets(bracketPairs)
	}
}

// ReadOnlyMode controls whether the contents of the editor can be altered by
// user interaction. If set to true, the editor will allow selecting text
// and copying it interactively, but not modifying it.
func ReadOnlyMode(enabled bool) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		e.readOnly = enabled
	}
}

// WrapLine configures whether the displayed text will be broken into lines or not.
func WrapLine(enabled bool) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		changed := e.text.WrapLine == enabled
		e.text.WrapLine = enabled
		if changed {
			e.text.invalidate()
		}
	}
}

func WithAutoCompletion(completor Completion) EditorOption {
	return func(e *Editor) {
		e.initBuffer()
		e.completor = completor
	}
}

// BeforePasteHook defines a hook to be called before pasting text to transform the text.
type BeforePasteHook func(text string) string

func AddBeforePasteHook(hook BeforePasteHook) EditorOption {
	return func(ed *Editor) {
		ed.onPaste = hook
	}
}
