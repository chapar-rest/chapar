package ui

import (
	"strings"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ---------------------------------------------------------------------------
// Inline suggestions
//
// An inline suggestion is ghost text painted at the caret, such as an AI
// completion. It is not part of the document: Tab inserts it, Escape drops
// it, and moving the caret or editing anywhere else drops it too. Typing the
// characters it starts with keeps the rest.
// ---------------------------------------------------------------------------

// inlineSuggestion is ghost text anchored to a caret offset and revision.
type inlineSuggestion struct {
	text string
	at   int
	rev  uint64
}

// SetInlineSuggestion shows text as ghost text at the caret. It is dropped
// as soon as the caret moves or the document changes in any other way than
// typing its next characters. An empty text clears it.
func (e *Editor) SetInlineSuggestion(text string) {
	if text == "" || e.readOnly {
		e.inline = inlineSuggestion{}
		return
	}
	e.inline = inlineSuggestion{text: text, at: e.caret, rev: e.revision}
}

// InlineSuggestion returns the ghost text showing at the caret, or "".
func (e *Editor) InlineSuggestion() string {
	if !e.inlineLive() {
		return ""
	}
	return e.inline.text
}

// inlineLive reports whether the suggestion still belongs at the caret.
func (e *Editor) inlineLive() bool {
	s := &e.inline
	if s.text == "" {
		return false
	}
	if s.at != e.caret || s.rev != e.revision || e.hasSelection() || e.readOnly {
		e.inline = inlineSuggestion{}
		return false
	}
	return true
}

// acceptInline inserts the suggestion as one undo step.
func (e *Editor) acceptInline() {
	text := e.inline.text
	e.inline = inlineSuggestion{}
	e.closeCompletion()
	e.applyEdit(e.caret, 0, text, mergeNone)
	e.ensureCaretVisible()
	if e.OnInlineAccept != nil {
		e.OnInlineAccept(text)
	}
}

// inlineKey handles Tab and Escape while a suggestion shows and reports
// whether it used the key.
func (e *Editor) inlineKey(ev input.KeyEvent) bool {
	if ev.Mods.Primary() || ev.Mods.Has(input.ModShift) || ev.Mods.Has(input.ModAlt) || !e.inlineLive() {
		return false
	}
	switch ev.Key {
	case input.KeyTab:
		e.acceptInline()
		return true
	case input.KeyEscape:
		e.inline = inlineSuggestion{}
		return true
	}
	return false
}

// inlineAfterType keeps the rest of the suggestion when the typed text is
// what it started with. text is the suggestion before typed was inserted
// at at.
func (e *Editor) inlineAfterType(text string, at int, typed string) {
	if text == "" || typed == "" || !strings.HasPrefix(text, typed) || e.caret != at+len(typed) {
		return
	}
	if rest := text[len(typed):]; rest != "" {
		e.inline = inlineSuggestion{text: rest, at: e.caret, rev: e.revision}
	}
}

// inlinePaintLines splits the live suggestion into lines, or returns nil.
func (e *Editor) inlinePaintLines() []string {
	if !e.focused || !e.inlineLive() {
		return nil
	}
	return strings.Split(e.inline.text, "\n")
}

// paintInline draws the ghost text on the caret's row. The rest of that row,
// suffix, is moved after the ghost text: after its first line, or after its
// last one when the suggestion spans lines, whose further lines cover the
// rows below.
func (e *Editor) paintInline(dl *render.DrawList, engine *shape.Engine, lines []string, cx, cy, x0, textTop float32, suffix []byte, suffixTint func(int) render.Color, area render.Rect) {
	th := theme.Current()
	ghost := th.ForegroundSubtle
	w := engine.DrawStringTopMono(dl, lines[0], cx, textTop, ghost)
	endX := cx + w
	endTop := textTop
	for i, ln := range lines[1:] {
		y := cy + float32(i+1)*e.lineH
		dl.AddRect(render.Rect{X: area.X, Y: y, W: area.W, H: e.lineH}, th.Background)
		endTop = textTop + float32(i+1)*e.lineH
		endX = x0 + engine.DrawStringTopMono(dl, ln, x0, endTop, ghost)
	}
	if len(suffix) > 0 {
		engine.DrawLineGlyphs(dl, e.lineMono(string(suffix)), endX, endTop, suffixTint)
	}
}
