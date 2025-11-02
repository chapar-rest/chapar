package gvcode

import (
	"image"
	"io"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/internal/buffer"
	gestureExt "github.com/oligo/gvcode/internal/gesture"
	"github.com/oligo/gvcode/textview"
)

// Editor implements an editable and scrollable text area.
type Editor struct {
	// mode sets the mode of the editor to work in.
	mode EditorMode

	// text manages the text buffer and provides shaping and cursor positioning
	// services.
	text       *textview.TextView
	buffer     buffer.TextSource
	snippetCtx *snippetContext
	// colorPalette configures the color scheme used for syntax highlighting.
	colorPalette *color.ColorPalette
	// LineNumberGutterGap specifies the right inset between the line number and the
	// editor text area.
	lineNumberGutterGap unit.Dp
	showLineNumber      bool
	// hooks
	onPaste   BeforePasteHook
	completor Completion
	// last input when the editor received an EditEvent.
	lastInput *key.EditEvent

	// scratch is a byte buffer that is reused to efficiently read portions of text
	// from the textView.
	scratch    []byte
	blinkStart time.Time

	// ime tracks the state relevant to input methods.
	ime struct {
		imeState
		scratch []byte
	}

	dragging    bool
	dragger     gesture.Drag
	scroller    gestureExt.Scroll
	hover       gestureExt.Hover
	scrollCaret bool
	showCaret   bool
	clicker     gesture.Click
	pending     []EditorEvent
	// commands is a registry of key commands.
	commands map[key.Name][]keyCommand
	// autoInsertions tracks recently inserted closing brackets or quotes.
	autoInsertions map[int]rune
	// gutterWidth can be used to guide to set the horizontal offset when
	// laying out a horizontal scrollbar.
	gutterWidth int
}

type imeState struct {
	selection struct {
		rng   key.Range
		caret key.Caret
	}
	snippet    key.Snippet
	start, end int
}

type EditorEvent interface {
	isEditorEvent()
}

// A ChangeEvent is generated for every user change to the text.
type ChangeEvent struct{}

// A SelectEvent is generated when the user selects some text, or changes the
// selection (e.g. with a shift-click), including if they remove the
// selection. The selected text is not part of the event, on the theory that
// it could be a relatively expensive operation (for a large editor), most
// applications won't actually care about it, and those that do can call
// Editor.SelectedText() (which can be empty).
type SelectEvent struct{}

// A HoverEvent is generated if the pointer hovers and keep still or maybe some
// small movement for some time.
type HoverEvent struct {
	PixelOff image.Point
	Pos      Position
	IsCancel bool
}

const (
	blinksPerSecond  = 1
	maxBlinkDuration = 10 * time.Second
)

// initBuffer should be invoked first in every exported function that accesses
// text state. It ensures that the underlying text widget is both ready to use
// and has its fields synced with the editor.
func (e *Editor) initBuffer() {
	if e.buffer == nil {
		e.text = textview.NewTextView()
		e.buffer = e.text.Source()
	}

	e.text.CaretWidth = unit.Dp(1)
}

// Update the state of the editor in response to input events. Update consumes editor
// input events until there are no remaining events or an editor event is generated.
// To fully update the state of the editor, callers should call Update until it returns
// false.
func (e *Editor) Update(gtx layout.Context) (EditorEvent, bool) {
	e.initBuffer()
	event, ok := e.processEvents(gtx)
	// Notify IME of selection if it changed.
	newSel := e.ime.selection
	start, end := e.text.Selection()
	newSel.rng = key.Range{
		Start: start,
		End:   end,
	}
	caretPos, carAsc, carDesc := e.text.CaretInfo()
	newSel.caret = key.Caret{
		Pos:     layout.FPt(caretPos),
		Ascent:  float32(carAsc),
		Descent: float32(carDesc),
	}
	if newSel != e.ime.selection {
		e.ime.selection = newSel
		gtx.Execute(key.SelectionCmd{Tag: e, Range: newSel.rng, Caret: newSel.caret})
	}

	e.updateSnippet(gtx, e.ime.start, e.ime.end)
	return event, ok
}

func (e *Editor) Layout(gtx layout.Context, lt *text.Shaper) layout.Dimensions {
	for {
		_, ok := e.Update(gtx)
		if !ok {
			break
		}
	}

	// Adjust scrolling for new viewport and layout.
	e.text.ScrollRel(0, 0)

	if e.scrollCaret {
		e.scrollCaret = false
		e.text.ScrollToCaret()
	}

	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	e.scroller.Add(gtx.Ops)
	if e.colorPalette != nil && e.colorPalette.Background.IsSet() {
		e.colorPalette.Background.Op(nil).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}

	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !e.showLineNumber {
				return layout.Dimensions{}
			}

			dims := layout.Inset{Right: max(0, e.lineNumberGutterGap)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					var lineNumberColor color.Color
					if e.colorPalette.LineNumberColor.IsSet() {
						lineNumberColor = e.colorPalette.LineNumberColor
					} else {
						lineNumberColor = color.Color{}.MulAlpha(255)
					}
					return e.text.PaintLineNumber(gtx, lt, lineNumberColor.Op(gtx.Ops))
				})
			e.gutterWidth = dims.Size.X
			return dims
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			e.text.Layout(gtx, lt)
			dims := e.layout(gtx)
			if e.completor != nil {
				e.text.PaintOverlay(gtx, e.completor.Offset(), e.completor.Layout)
			}
			return dims
		}),
	)

}

func (e *Editor) layout(gtx layout.Context) layout.Dimensions {
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	pointer.CursorText.Add(gtx.Ops)
	event.Op(gtx.Ops, e)

	//e.scroller.Add(gtx.Ops)

	e.clicker.Add(gtx.Ops)
	e.dragger.Add(gtx.Ops)
	e.hover.Add(gtx.Ops)
	e.showCaret = false
	if gtx.Focused(e) {
		now := gtx.Now
		dt := now.Sub(e.blinkStart)
		blinking := dt < maxBlinkDuration
		const timePerBlink = time.Second / blinksPerSecond
		nextBlink := now.Add(timePerBlink/2 - dt%(timePerBlink/2))
		if blinking {
			gtx.Execute(op.InvalidateCmd{At: nextBlink})
		}
		e.showCaret = !blinking || dt%timePerBlink < timePerBlink/2
	}
	semantic.Editor.Add(gtx.Ops)

	// determine the various colors to use.
	if e.colorPalette == nil {
		panic("No color palette is set!")
	}

	textMaterial := color.Color{}
	var selectColor, lineColor color.Color
	if e.colorPalette.Foreground.IsSet() {
		textMaterial = e.colorPalette.Foreground
	}
	if e.colorPalette.SelectColor.IsSet() {
		selectColor = e.colorPalette.SelectColor
	} else {
		selectColor = textMaterial.MulAlpha(0x60)
	}
	if e.colorPalette.LineColor.IsSet() {
		lineColor = e.colorPalette.LineColor
	} else {
		lineColor = textMaterial.MulAlpha(0x30)
	}

	if e.Len() > 0 {
		e.paintSelection(gtx, selectColor)
		e.paintLineHighlight(gtx, lineColor)
		e.text.HighlightMatchingBrackets(gtx, selectColor.Op(gtx.Ops))
		e.paintText(gtx, textMaterial)
	}
	if gtx.Enabled() {
		e.paintCaret(gtx, textMaterial)
	}
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

// PaintOverlay draws a overlay widget over the main editor area.
func (e *Editor) PaintOverlay(gtx layout.Context, position image.Point, w layout.Widget) {
	offset := position.Add(e.text.ScrollOff())
	e.text.PaintOverlay(gtx, offset, w)
}

// paintSelection paints the contrasting background for selected text using the provided
// material to set the painting material for the selection.
func (e *Editor) paintSelection(gtx layout.Context, material color.Color) {
	e.initBuffer()
	e.text.PaintSelection(gtx, material.Op(gtx.Ops))
}

// paintText paints the text glyphs using the provided material to set the fill of the
// glyphs.
func (e *Editor) paintText(gtx layout.Context, material color.Color) {
	e.initBuffer()
	e.text.PaintText(gtx, material.Op(gtx.Ops))
}

// paintCaret paints the text glyphs using the provided material to set the fill material
// of the caret rectangle.
func (e *Editor) paintCaret(gtx layout.Context, material color.Color) {
	e.initBuffer()
	if !e.showCaret || e.mode == ModeReadOnly {
		return
	}
	e.text.PaintCaret(gtx, material.Op(gtx.Ops))
}

func (e *Editor) paintLineHighlight(gtx layout.Context, material color.Color) {
	e.initBuffer()
	e.text.PaintLineHighlight(gtx, material.Op(gtx.Ops))
}

// Len is the length of the editor contents, in runes.
func (e *Editor) Len() int {
	e.initBuffer()
	return e.buffer.Len()
}

// Text returns the contents of the editor. This method is not concurrent safe,
// and you should use the Reader returned from GetReader to read from multiple
// goroutines.
func (e *Editor) Text() string {
	e.initBuffer()

	srcReader := buffer.NewReader(e.text.Source())
	e.scratch = srcReader.ReadAll(e.scratch)
	return string(e.scratch)
}

// GetReader returns a [io.ReadSeeker] to the caller to read the text buffer. This
// is the preferred way to read from the editor, especially when reading from
// multiple goroutines.
func (e *Editor) GetReader() io.ReadSeeker {
	return buffer.NewReader(e.text.Source())
}

func (e *Editor) SetText(s string) {
	e.initBuffer()

	indent, _, size := GuessIndentation(s)
	e.text.SoftTab = indent == Spaces
	e.text.TabWidth = size

	e.text.SetText(s)
	e.ime.start = 0
	e.ime.end = 0
	// Reset xoff and move the caret to the beginning.
	e.SetCaret(0, 0)
}

// CaretPos returns the line & column numbers of the caret.
func (e *Editor) CaretPos() (line, col int) {
	e.initBuffer()
	return e.text.CaretPos()
}

// CaretCoords returns the coordinates of the caret, relative to the
// editor itself.
func (e *Editor) CaretCoords() f32.Point {
	e.initBuffer()
	return e.text.CaretCoords()
}

// ConvertPos convert a line/col position to rune offset.
func (e *Editor) ConvertPos(line, col int) int {
	return e.text.ConvertPos(line, col)
}

// ReadUntil reads in the specified direction from the current caret position until the
// seperator returns false. It returns the read text.
func (e *Editor) ReadUntil(direction int, seperator func(r rune) bool) string {
	return e.text.ReadUntil(direction, seperator)
}

// Delete runes from the caret position. The sign of the argument specifies the
// direction to delete: positive is forward, negative is backward.
//
// If there is a selection, it is deleted and counts as a single grapheme
// cluster.
func (e *Editor) Delete(graphemeClusters int) (deletedRunes int) {
	e.initBuffer()
	if graphemeClusters == 0 {
		return 0
	}

	if graphemeClusters < 0 {
		// update selection based on some rules.
		e.onDeleteBackward()
	}

	start, end := e.text.Selection()
	if start != end {
		graphemeClusters -= sign(graphemeClusters)
	}

	// Move caret by the target quantity of clusters.
	e.text.MoveCaret(0, graphemeClusters)
	// Get the new rune offsets of the selection.
	start, end = e.text.Selection()
	e.replace(start, end, "")
	// Reset xoff.
	e.text.MoveCaret(0, 0)
	e.ClearSelection()
	return end - start
}

// DeleteLine delete the current line, and place the caret at the
// start of the next line.
func (e *Editor) DeleteLine() (deletedRunes int) {
	e.initBuffer()

	start, end := e.text.SelectedLineRange()
	if start == end {
		return 0
	}

	e.replace(start, end, "")
	// Reset xoff.
	e.text.MoveCaret(0, 0)
	e.SetCaret(start, start)

	e.ClearSelection()
	return end - start
}

func (e *Editor) Insert(s string) (insertedRunes int) {
	e.initBuffer()

	if s == "" {
		return
	}

	start, end := e.text.Selection()
	moves := e.replace(start, end, s)
	if end < start {
		start = end
	}
	// Reset xoff.
	e.text.MoveCaret(0, 0)
	e.SetCaret(start+moves, start+moves)
	e.scrollCaret = true
	return moves
}

func isSingleLine(s string) bool {
	return len(s) > 1 && strings.Count(s, "\n") == 1 && s[len(s)-1] == '\n'
}

// InsertLine insert a line of text before the current line, and place the caret at the
// start of the current line.
//
// This single line insertion is mainly for paste operation after copying/cutting the
// current line(paragraph) when there is no selection, but it can also used outside of
// the editor to insert a entire line(paragraph).
func (e *Editor) InsertLine(s string) (insertedRunes int) {
	e.initBuffer()

	if s == "" {
		return
	}

	if isSingleLine(s) && e.text.SelectionLen() == 0 {
		// If s is a paragraph of text, insert s between the current line
		// and the previous line.
		start, end := e.text.SelectedLineRange()
		moves := e.replace(start, start, s)
		// Reset xoff.
		e.text.MoveCaret(0, 0)
		e.SetCaret(end, end)
		e.scrollCaret = true
		return moves
	}

	return
}

// undo revert the last operation(s).
func (e *Editor) undo() (EditorEvent, bool) {
	e.initBuffer()

	positions, ok := e.text.Undo()
	if !ok {
		return nil, false
	}

	var start, end int
	for _, pos := range positions {
		start = pos.Start
		end = pos.End
	}

	e.SetCaret(end, start)
	return ChangeEvent{}, true
}

// redo revert the last undo operation.
func (e *Editor) redo() (EditorEvent, bool) {
	e.initBuffer()

	positions, ok := e.text.Redo()
	if !ok {
		return nil, false
	}

	var start, end int
	for _, pos := range positions {
		start = pos.Start
		end = pos.End
	}

	e.SetCaret(end, start)
	return ChangeEvent{}, true
}

// replace the text between start and end with s. Indices are in runes.
// It returns the number of runes inserted.
func (e *Editor) replace(start, end int, s string) int {
	length := e.text.Len()
	if start > end {
		start, end = end, start
	}
	start = min(start, length)
	end = min(end, length)

	sc := e.text.Replace(start, end, s)
	newEnd := start + sc
	adjust := func(pos int) int {
		switch {
		case newEnd < pos && pos <= end:
			pos = newEnd
		case end < pos:
			diff := newEnd - end
			pos = pos + diff
		}
		return pos
	}
	e.ime.start = adjust(e.ime.start)
	e.ime.end = adjust(e.ime.end)
	return sc
}

// ReplaceAll replaces all texts specifed in TextRange with newStr.
// It returns the number of occurrences replaced.
func (e *Editor) ReplaceAll(texts []TextRange, newStr string) int {
	if len(texts) <= 0 {
		return 0
	}

	// Traverse in reverse order to prevent match offsets from changing after
	// each replace.
	e.buffer.GroupOp()
	finalPos := 0
	for idx := len(texts) - 1; idx >= 0; idx-- {
		start, end := texts[idx].Start, texts[idx].End
		e.replace(start, end, newStr)
		finalPos = start
	}
	e.buffer.UnGroupOp()

	e.SetCaret(finalPos, finalPos)
	return len(texts)
}

// MoveCaret moves the caret (aka selection start) and the selection end
// relative to their current positions. Positive distances moves forward,
// negative distances moves backward. Distances are in grapheme clusters,
// which closely match what users perceive as "characters" even when the
// characters are multiple code points long.
func (e *Editor) MoveCaret(startDelta, endDelta int) {
	e.initBuffer()
	e.text.MoveCaret(startDelta, endDelta)
}

// deleteWord deletes the next word(s) in the specified direction.
// Unlike moveWord, deleteWord treats whitespace as a word itself.
// Positive is forward, negative is backward.
// Absolute values greater than one will delete that many words.
// The selection counts as a single word.
func (e *Editor) deleteWord(distance int) (deletedRunes int) {
	if distance == 0 {
		return
	}

	start, end := e.text.Selection()
	if start != end {
		deletedRunes = e.Delete(1)
		distance -= sign(distance)
	}
	if distance == 0 {
		return deletedRunes
	}

	// split the distance information into constituent parts to be
	// used independently.
	words, direction := distance, 1
	if distance < 0 {
		words, direction = distance*-1, -1
	}
	caret, _ := e.text.Selection()
	// atEnd if offset is at or beyond either side of the buffer.
	atEnd := func(runes int) bool {
		idx := caret + runes*direction
		return idx <= 0 || idx >= e.Len()
	}
	// next returns the appropriate rune given the direction and offset in runes.
	next := func(runes int) rune {
		idx := caret + runes*direction
		if idx < 0 {
			idx = 0
		} else if idx > e.Len() {
			idx = e.Len()
		}
		var r rune
		if direction < 0 {
			r, _ = e.buffer.ReadRuneAt(idx - 1)
		} else {
			r, _ = e.buffer.ReadRuneAt(idx)
		}
		return r
	}
	runes := 1
	for ii := 0; ii < words; ii++ {
		r := next(runes)
		isSeperator := e.text.IsWordSeperator(r)

		for r := next(runes); (isSeperator == e.text.IsWordSeperator(r)) && !atEnd(runes); r = next(runes) {
			runes += 1
		}
	}
	deletedRunes += e.Delete(runes * direction)
	return deletedRunes
}

// SelectionLen returns the length of the selection, in runes; it is
// equivalent to utf8.RuneCountInString(e.SelectedText()).
func (e *Editor) SelectionLen() int {
	e.initBuffer()
	return e.text.SelectionLen()
}

// Selection returns the start and end of the selection, as rune offsets.
// start can be > end.
func (e *Editor) Selection() (start, end int) {
	e.initBuffer()
	return e.text.Selection()
}

// SetCaret moves the caret to start, and sets the selection end to end. start
// and end are in runes, and represent offsets into the editor text.
func (e *Editor) SetCaret(start, end int) {
	e.initBuffer()
	e.text.SetCaret(start, end)
	e.scrollCaret = true
	e.scroller.Stop()
}

// SelectedText returns the currently selected text (if any) from the editor.
func (e *Editor) SelectedText() string {
	e.initBuffer()
	e.scratch = e.text.SelectedText(e.scratch)
	return string(e.scratch)
}

// ClearSelection clears the selection, by setting the selection end equal to
// the selection start.
func (e *Editor) ClearSelection() {
	e.initBuffer()
	e.text.ClearSelection()
}

// ScrollRatio returns the viewport's start and end scrolling offset in ratio
// relating to the reandered document coordinate space.
func (e *Editor) ScrollRatio() (minX, maxX float32, minY, maxY float32) {
	textDims := e.text.FullDimensions()
	visibleDims := e.text.Dimensions()
	scrollOff := e.text.ScrollOff()

	minX = float32(scrollOff.X) / float32(textDims.Size.X)
	maxX = float32(scrollOff.X+visibleDims.Size.X) / float32(textDims.Size.X)
	minY = float32(scrollOff.Y) / float32(textDims.Size.Y)
	maxY = float32(scrollOff.Y+visibleDims.Size.Y) / float32(textDims.Size.Y)
	return
}

// Scroll scrolls the horizontal or vertical scrollbar, using ratio related to
// the rendered document size.
func (e *Editor) Scroll(gtx layout.Context, xRatio, yRatio float32) {
	textDims := e.text.FullDimensions().Size
	xRatio = min(1.0, xRatio)
	xRatio = max(xRatio, -1.0)
	yRatio = min(1.0, yRatio)
	yRatio = max(yRatio, -1.0)

	e.text.ScrollRel(int(float32(textDims.X)*xRatio), int(float32(textDims.Y)*yRatio))
}

// GutterWidth returns the width of the gutter in pixel, which can be used to
// guide to set the horizontal offset when laying out a horizontal scrollbar.
func (e *Editor) GutterWidth() int {
	return e.gutterWidth
}

// Deprecated: use Mode() method please.
func (e *Editor) ReadOnly() bool {
	return e.mode == ModeReadOnly
}

func (e *Editor) Mode() EditorMode {
	return e.mode
}

func (e *Editor) TabStyle() (TabStyle, int) {
	if e.text.SoftTab {
		return Spaces, e.text.TabWidth
	}

	return Tabs, e.text.TabWidth
}

func (e *Editor) ColorPalette() *color.ColorPalette {
	return e.colorPalette
}

// SetDebug enable or disable the debug mode.
// In debug mode, internal buffer state is printed.
func SetDebug(enable bool) {
	buffer.SetDebug(enable)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

func (s ChangeEvent) isEditorEvent() {}
func (s SelectEvent) isEditorEvent() {}
func (s HoverEvent) isEditorEvent()  {}
