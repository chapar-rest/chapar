package gvcode

import (
	"errors"
	"fmt"

	"gioui.org/io/key"
	"gioui.org/layout"
	"github.com/oligo/gvcode/internal/buffer"
	"github.com/oligo/gvcode/snippet"
	"github.com/oligo/gvcode/textstyle/decoration"
)

const (
	snippetModeDeco = "_snippet_deco"
)

// snippetContext manages a LSP snippet states when the editor enters the
// snippet mode. The user can then use Tab/Shift+Tab to navigate through
// the tabstops defined in the snippet and fill the placeholders.
type snippetContext struct {
	editor *Editor
	state  *snippet.Snippet
	// currentIdx tracks the current tabstop.
	currentIdx int
	// origin marks the starting rune offset of the snippet.
	origin int
	// markers holds a left and right marker pair for each of the tabstop
	// in the current snippet.
	markers [][]*buffer.Marker
}

func newSnippetContext(editor *Editor) *snippetContext {
	sc := &snippetContext{
		editor:     editor,
		currentIdx: -1,
	}
	// register a key command used to quit the snippet mode when pressed.
	editor.RegisterCommand(sc, key.Filter{Name: key.NameEscape},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			editor.setMode(ModeNormal)
			return nil
		})
	return sc
}

func (sc *snippetContext) SetSnippet(snp *snippet.Snippet) (int, error) {
	if snp == nil {
		return 0, errors.New("invalid snippet")
	}

	sc.currentIdx = -1
	sc.state = snp

	tmpl := sc.state.Template()
	runes := sc.editor.Insert(tmpl)

	// move caret to the begining of the text.
	sc.editor.MoveCaret(-runes, -runes)
	// record the initial position of the snippet.
	sc.origin, _ = sc.editor.Selection()

	err := sc.addDecorations()
	if err != nil {
		return 0, fmt.Errorf("add decoration failed: %w", err)
	}

	sc.NextTabStop()
	return runes, nil
}

func (sc *snippetContext) NextTabStop() error {
	if sc.state == nil {
		return errors.New("no snippet is set")
	}

	sc.currentIdx++

	// sc.state.tabstops is sorted, so we can just iterate through it.
	if sc.currentIdx < sc.state.TabStopSize() {
		start, end := sc.getTabStopPosition(sc.currentIdx)
		sc.editor.SetCaret(end, start)
	}

	currentTabStop := sc.state.TabStopAt(sc.currentIdx)

	if sc.currentIdx >= sc.state.TabStopSize() || currentTabStop.IsFinal() {
		// Reached the end of the tabstops
		sc.editor.setMode(ModeNormal)
	}

	return nil
}

func (sc *snippetContext) PrevTabStop() error {
	if sc.state == nil {
		return errors.New("no snippet is set")
	}

	sc.currentIdx--

	if sc.currentIdx >= 0 {
		start, end := sc.getTabStopPosition(sc.currentIdx)
		sc.editor.SetCaret(end, start)
		return nil
	} else {
		// Reached the end of the tabstops
		sc.editor.setMode(ModeNormal)
		return nil
	}
}

func (sc *snippetContext) OnInsertAt(runeStart, runeEnd int) {
	if runeStart > runeEnd {
		runeStart, runeEnd = runeEnd, runeStart
	}

	if sc == nil || sc.state == nil {
		return
	}

	start, end := sc.getTabStopPosition(sc.currentIdx)
	if runeStart < start || runeEnd > end+1 {
		sc.editor.setMode(ModeNormal)
	}

}

func (sc *snippetContext) getTabStopPosition(idx int) (int, int) {
	if len(sc.markers) > 0 && idx < len(sc.markers) {
		markers := sc.markers[idx]
		return markers[0].Offset(), markers[1].Offset()
	}

	start, end := sc.state.TabStopOff(sc.currentIdx)
	return sc.origin + start, sc.origin + end
}

func (sc *snippetContext) addDecorations() error {
	sc.markers = sc.markers[:0]

	// add decorations for the tabstops
	decos := make([]decoration.Decoration, 0)
	for idx := range sc.state.TabStops() {
		start, end := sc.state.TabStopOff(idx)
		decos = append(decos, decoration.Decoration{
			Source: snippetModeDeco,
			Start:  sc.origin + start,
			End:    sc.origin + end,
			Border: &decoration.Border{Color: sc.editor.colorPalette.Foreground},
		})

	}

	if len(decos) > 0 {
		err := sc.editor.AddDecorations(decos...)
		if err != nil {
			return err
		}

		for idx := range decos {
			startMarker, endMarker := decos[idx].Range()
			if startMarker != nil && endMarker != nil {
				sc.markers = append(sc.markers, []*buffer.Marker{startMarker, endMarker})
			} else {
				return fmt.Errorf("invalid marker for decoration: %d", idx)
			}
		}
	}

	return nil
}

func (sc *snippetContext) Cancel() {
	sc.editor.ClearDecorations(snippetModeDeco)
	sc.markers = sc.markers[:0]
	sc.state = nil
	sc.currentIdx = -1
	sc.origin = 0
	sc.editor.RemoveCommands(sc)
}

func (e *Editor) InsertSnippet(body string) (insertedRunes int, err error) {
	snp := snippet.NewSnippet(body)
	err = snp.Parse()
	if err != nil {
		return 0, err
	}

	if e.mode == ModeSnippet {
		// An ongoing snippet session exists. To avoid nested snippet editing,
		// we try to insert the snippet body as plain text.
		runes := e.Insert(snp.Template())
		return runes, nil
	}

	if snp.Template() == body {
		runes := e.Insert(snp.Template())
		return runes, nil
	}

	if e.snippetCtx != nil {
		e.snippetCtx.Cancel()
	}

	e.snippetCtx = newSnippetContext(e)
	e.setMode(ModeSnippet)
	insertedRunes, err = e.snippetCtx.SetSnippet(snp)
	return
}
