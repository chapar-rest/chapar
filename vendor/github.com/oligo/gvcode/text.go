package gvcode

import (
	"image"
	"math"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/oligo/gvcode/internal/buffer"
	lt "github.com/oligo/gvcode/internal/layout"
	"golang.org/x/exp/slices"
	"golang.org/x/image/math/fixed"
)

// TextRange contains the range of text of interest in the document. It can used for
// search, styling text, or any other purposes.
type TextRange struct {
	// offset of the start rune in the document.
	Start int
	// offset of the end rune in the document.
	End int
}

// TextStyle defines style for a range of text in the document.
type TextStyle struct {
	TextRange
	// Color of the text..
	Color op.CallOp
	// Background color of the painted text in the range.
	Background op.CallOp
}

// Region describes the position and baseline of an area of interest within
// shaped text.
type Region = lt.Region

type caretPos struct {
	// xoff is the offset to the current position when moving between lines.
	xoff fixed.Int26_6
	// start is the current caret position in runes, and also the start position of
	// selected text. end is the end position of selected text. If start
	// == end, then there's no selection. Note that it's possible (and
	// common) that the caret (start) is after the end, e.g. after
	// Shift-DownArrow.
	start int
	end   int
}

// textView provides efficient shaping and indexing of interactive text. When provided
// with a TextSource, textView will shape and cache the runes within that source.
// It provides methods for configuring a viewport onto the shaped text which can
// be scrolled, and for configuring and drawing text selection boxes.
type textView struct {
	// Font set the font used to draw the text.
	Font font.Font
	// TextSize set the size of both the main text and line number.
	TextSize unit.Sp
	// Alignment controls the alignment of text within the editor.
	Alignment text.Alignment
	// LineHeight controls the distance between the baselines of lines of text.
	// If zero, the font size will be used.
	LineHeight unit.Sp
	// LineHeightScale applies a scaling factor to the LineHeight. If zero, a default
	// value 1.2 will be used.
	LineHeightScale float32

	// CaretWidth set the visual width of a caret.
	CaretWidth unit.Dp

	// SoftTab controls the behaviour when user try to insert a Tab character.
	// If set to true, the editor will insert the amount of space characters specified by
	// TabWidth, else the editor insert a \t character.
	SoftTab bool

	// TabWidth set how many spaces to represent a tab character. In the case of
	// soft tab, this determines the number of space characters to insert into the editor.
	// While for hard tab, this controls the maximum width of the 'tab' glyph to expand to.
	TabWidth int

	// WrapLine configures whether the displayed text will be broken into lines or not.
	WrapLine bool

	// WordSeperators configures a set of characters that will be used as word separators
	// when doing word related operations, like navigating or deleting by word.
	WordSeperators string

	src    buffer.TextSource
	params text.Parameters
	shaper *text.Shaper
	// dimensions of the layouted document.
	dims layout.Dimensions
	// viewport size
	viewSize image.Point
	// line height used by shaper.
	lineHeight fixed.Int26_6
	// scrolled offset relative to the start of dims.
	scrollOff image.Point
	layouter  lt.TextLayout

	// The layout is valid or not. Invalid layout requires a re-layout.
	valid bool
	// caret position in the view.
	caret   caretPos
	regions []Region
}

// SetSource initializes the underlying data source for the Text. This
// must be done before invoking any other methods on Text.
func (e *textView) SetSource(source buffer.TextSource) {
	e.src = source
	e.layouter = lt.NewTextLayout(e.src)
	e.invalidate()
}

func (e *textView) Changed() bool {
	return e.src.Changed()
}

// Dimensions returns the dimensions of the visible text.
func (e *textView) Dimensions() layout.Dimensions {
	basePos := e.dims.Size.Y - e.dims.Baseline
	return layout.Dimensions{Size: e.viewSize, Baseline: e.viewSize.Y - basePos}
}

// FullDimensions returns the dimensions of all shaped text, including
// text that isn't visible within the current viewport.
func (e *textView) FullDimensions() layout.Dimensions {
	return e.dims
}

func (e *textView) makeValid() {
	if e.valid {
		return
	}
	e.layoutText(e.shaper)
	e.valid = true
}

func (e *textView) closestToRune(runeIdx int) lt.CombinedPos {
	e.makeValid()
	pos, _ := e.layouter.ClosestToRune(runeIdx)
	return pos
}

func (e *textView) closestToLineCol(line, col int) lt.CombinedPos {
	e.makeValid()
	return e.layouter.ClosestToLineCol(lt.ScreenPos{Line: line, Col: col})
}

func (e *textView) closestToXY(x fixed.Int26_6, y int) lt.CombinedPos {
	e.makeValid()
	return e.layouter.ClosestToXY(x, y)
}

func (e *textView) closestToXYGraphemes(x fixed.Int26_6, y int) lt.CombinedPos {
	// Find the closest existing rune position to the provided coordinates.
	pos := e.closestToXY(x, y)
	// Resolve cluster boundaries on either side of the rune position.
	firstOption := e.moveByGraphemes(pos.Runes, 0)
	distance := 1
	if firstOption > pos.Runes {
		distance = -1
	}
	secondOption := e.moveByGraphemes(firstOption, distance)
	// Choose the closest grapheme cluster boundary to the desired point.
	first := e.closestToRune(firstOption)
	firstDist := absFixed(first.X - x)
	second := e.closestToRune(secondOption)
	secondDist := absFixed(second.X - x)
	if firstDist > secondDist {
		return second
	} else {
		return first
	}
}

// MaxLines moves the cursor the specified number of lines vertically, ensuring
// that the resulting position is aligned to a grapheme cluster.
func (e *textView) MoveLines(distance int, selAct selectionAction) {
	caretStart := e.closestToRune(e.caret.start)
	x := caretStart.X + e.caret.xoff
	// Seek to line.
	pos := e.closestToLineCol(caretStart.LineCol.Line+distance, 0)
	pos = e.closestToXYGraphemes(x, pos.Y)
	e.caret.start = pos.Runes
	e.caret.xoff = x - pos.X
	e.updateSelection(selAct)
}

// Layout the text, reshaping it as necessary.
func (e *textView) Layout(gtx layout.Context, lt *text.Shaper) {
	e.params.DisableSpaceTrim = true

	if e.params.Locale != gtx.Locale {
		e.params.Locale = gtx.Locale
		e.invalidate()
	}
	textSize := fixed.I(gtx.Sp(e.TextSize))
	if e.params.Font != e.Font || e.params.PxPerEm != textSize {
		e.invalidate()
		e.params.Font = e.Font
		e.params.PxPerEm = textSize
	}

	maxWidth := gtx.Constraints.Max.X
	minWidth := gtx.Constraints.Min.X
	if maxWidth != e.params.MaxWidth {
		e.params.MaxWidth = maxWidth
		if e.WrapLine {
			e.invalidate()
		}
	}
	if minWidth != e.params.MinWidth {
		e.params.MinWidth = minWidth
		if e.WrapLine {
			e.invalidate()
		}
	}

	if lt != e.shaper {
		e.shaper = lt
		e.invalidate()
	}
	if e.Alignment != e.params.Alignment {
		e.params.Alignment = e.Alignment
		e.invalidate()
	}

	if lh := fixed.I(gtx.Sp(e.LineHeight)); lh != e.params.LineHeight {
		e.params.LineHeight = lh
		e.invalidate()
	}
	if e.LineHeightScale != e.params.LineHeightScale {
		e.params.LineHeightScale = e.LineHeightScale
		e.invalidate()
	}

	// calculate the final line height used by Shaper
	e.lineHeight = e.calcLineHeight()
	e.makeValid()

	if viewSize := e.calculateViewSize(gtx); viewSize != e.viewSize {
		e.viewSize = viewSize
		if e.WrapLine {
			e.invalidate()
		}
	}

	e.makeValid()
}

// Calculate line height. Maybe there's a better way?
func (tv *textView) calcLineHeight() fixed.Int26_6 {
	lineHeight := tv.params.LineHeight
	// align with how text.Shaper handles default value of tv.params.LineHeight.
	if lineHeight == 0 {
		lineHeight = tv.params.PxPerEm
	}
	lineHeightScale := tv.params.LineHeightScale
	// align with how text.Shaper handles default value of tv.params.LineHeightScale.
	if lineHeightScale == 0 {
		lineHeightScale = 1.2
	}

	return floatToFixed(fixedToFloat(lineHeight) * lineHeightScale)
}

// ByteOffset returns the start byte of the rune at the given
// rune offset, clamped to the size of the text.
func (e *textView) ByteOffset(runeOffset int) int64 {
	pos := e.closestToRune(runeOffset)
	return int64(e.src.RuneOffset(pos.Runes))
}

// Len is the length of the editor contents, in runes.
func (e *textView) Len() int {
	e.makeValid()
	return e.closestToRune(math.MaxInt).Runes
}

func (e *textView) ScrollBounds() image.Rectangle {
	return image.Rectangle{Max: image.Point{X: e.dims.Size.X - e.viewSize.X, Y: e.dims.Size.Y - e.viewSize.Y}}
}

func (e *textView) ScrollRel(dx, dy int) {
	e.scrollAbs(e.scrollOff.X+dx, e.scrollOff.Y+dy)
}

// ScrollOff returns the scroll offset of the text viewport.
func (e *textView) ScrollOff() image.Point {
	return e.scrollOff
}

func (e *textView) scrollAbs(x, y int) {
	e.scrollOff.X = x
	e.scrollOff.Y = y
	b := e.ScrollBounds()
	if e.scrollOff.X > b.Max.X {
		e.scrollOff.X = b.Max.X
	}
	if e.scrollOff.X < b.Min.X {
		e.scrollOff.X = b.Min.X
	}
	if e.scrollOff.Y > b.Max.Y {
		e.scrollOff.Y = b.Max.Y
	}
	if e.scrollOff.Y < b.Min.Y {
		e.scrollOff.Y = b.Min.Y
	}
}

// MoveCoord moves the caret to the position closest to the provided
// point that is aligned to a grapheme cluster boundary.
func (e *textView) MoveCoord(pos image.Point) {
	x := fixed.I(pos.X + e.scrollOff.X)
	y := pos.Y + e.scrollOff.Y
	e.caret.start = e.closestToXYGraphemes(x, y).Runes
	e.caret.xoff = 0
}

// CaretPos returns the line & column numbers of the caret.
func (e *textView) CaretPos() (line, col int) {
	pos := e.closestToRune(e.caret.start)
	return pos.LineCol.Line, pos.LineCol.Col
}

// CaretCoords returns the coordinates of the caret, relative to the
// editor itself.
func (e *textView) CaretCoords() f32.Point {
	pos := e.closestToRune(e.caret.start)
	return f32.Pt(float32(pos.X)/64-float32(e.scrollOff.X), float32(pos.Y-e.scrollOff.Y))
}

// invalidate mark the layout as invalid.
func (e *textView) invalidate() {
	e.valid = false
}

// Set the text of the buffer. It returns the number of runes inserted.
func (e *textView) SetText(s string) int {
	e.src.SetText([]byte(s))
	sc := e.src.Len()

	// e.SetCaret(0, 0)
	e.invalidate()
	return sc
}

// Replace the text between start and end with s. Indices are in runes.
// It returns the number of runes inserted.
func (e *textView) Replace(start, end int, s string) int {
	if start > end {
		start, end = end, start
	}
	startPos := e.closestToRune(start)
	endPos := e.closestToRune(end)
	startOff := startPos.Runes
	sc := utf8.RuneCountInString(s)
	newEnd := startPos.Runes + sc

	e.src.Replace(startOff, endPos.Runes, s)
	adjust := func(pos int) int {
		switch {
		case newEnd < pos && pos <= endPos.Runes:
			pos = newEnd
		case endPos.Runes < pos:
			diff := newEnd - endPos.Runes
			pos = pos + diff
		}
		return pos
	}
	e.caret.start = adjust(e.caret.start)
	e.caret.end = adjust(e.caret.end)
	e.invalidate()
	return sc
}

// MovePages moves the caret position by vertical pages of text, ensuring that
// the final position is aligned to a grapheme cluster boundary.
func (e *textView) MovePages(pages int, selAct selectionAction) {
	caret := e.closestToRune(e.caret.start)
	x := caret.X + e.caret.xoff
	y := caret.Y + pages*e.viewSize.Y
	pos := e.closestToXYGraphemes(x, y)
	e.caret.start = pos.Runes
	e.caret.xoff = x - pos.X
	e.updateSelection(selAct)
}

// moveByGraphemes returns the rune index resulting from moving the
// specified number of grapheme clusters from startRuneidx.
func (e *textView) moveByGraphemes(startRuneidx, graphemes int) int {
	if len(e.layouter.Graphemes) == 0 {
		return startRuneidx
	}
	startGraphemeIdx, _ := slices.BinarySearch(e.layouter.Graphemes, startRuneidx)
	startGraphemeIdx = max(startGraphemeIdx+graphemes, 0)
	startGraphemeIdx = min(startGraphemeIdx, len(e.layouter.Graphemes)-1)
	startRuneIdx := e.layouter.Graphemes[startGraphemeIdx]
	return e.closestToRune(startRuneIdx).Runes
}

// clampCursorToGraphemes ensures that the final start/end positions of
// the cursor are on grapheme cluster boundaries.
func (e *textView) clampCursorToGraphemes() {
	e.caret.start = e.moveByGraphemes(e.caret.start, 0)
	e.caret.end = e.moveByGraphemes(e.caret.end, 0)
}

// MoveCaret moves the caret (aka selection start) and the selection end
// relative to their current positions. Positive distances moves forward,
// negative distances moves backward. Distances are in grapheme clusters which
// better match the expectations of users than runes.
func (e *textView) MoveCaret(startDelta, endDelta int) {
	e.caret.xoff = 0
	e.caret.start = e.moveByGraphemes(e.caret.start, startDelta)
	e.caret.end = e.moveByGraphemes(e.caret.end, endDelta)
}

// MoveTextStart moves the caret to the start of the text.
func (e *textView) MoveTextStart(selAct selectionAction) {
	caret := e.closestToRune(e.caret.end)
	e.caret.start = 0
	e.caret.end = caret.Runes
	e.caret.xoff = -caret.X
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

// MoveTextEnd moves the caret to the end of the text.
func (e *textView) MoveTextEnd(selAct selectionAction) {
	caret := e.closestToRune(math.MaxInt)
	e.caret.start = caret.Runes
	e.caret.xoff = fixed.I(e.params.MaxWidth) - caret.X
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

// MoveLineStart moves the caret to the start of the current line, ensuring that the resulting
// cursor position is on a grapheme cluster boundary.
func (e *textView) MoveLineStart(selAct selectionAction) {
	caret := e.closestToRune(e.caret.start)
	caret = e.closestToLineCol(caret.LineCol.Line, 0)
	e.caret.start = caret.Runes
	e.caret.xoff = -caret.X
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

// MoveLineEnd moves the caret to the end of the current line, ensuring that the resulting
// cursor position is on a grapheme cluster boundary.
func (e *textView) MoveLineEnd(selAct selectionAction) {
	caret := e.closestToRune(e.caret.start)
	caret = e.closestToLineCol(caret.LineCol.Line, math.MaxInt)
	e.caret.start = caret.Runes
	e.caret.xoff = fixed.I(e.params.MaxWidth) - caret.X
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

func (e *textView) ScrollToCaret() {
	caret := e.closestToRune(e.caret.start)

	miny := caret.Y - caret.Ascent.Ceil()
	maxy := caret.Y + caret.Descent.Ceil()
	var dist int
	if d := miny - e.scrollOff.Y; d < 0 {
		dist = d
	} else if d := maxy - (e.scrollOff.Y + e.viewSize.Y); d > 0 {
		dist = d
	}
	e.ScrollRel(0, dist)
}

// SelectionLen returns the length of the selection, in runes; it is
// equivalent to utf8.RuneCountInString(e.SelectedText()).
func (e *textView) SelectionLen() int {
	return abs(e.caret.start - e.caret.end)
}

// Selection returns the start and end of the selection, as rune offsets.
// start can be > end.
func (e *textView) Selection() (start, end int) {
	return e.caret.start, e.caret.end
}

// SetCaret moves the caret to start, and sets the selection end to end. Then
// the two ends are clamped to the nearest grapheme cluster boundary. start
// and end are in runes, and represent offsets into the editor text.
func (e *textView) SetCaret(start, end int) {
	e.caret.start = e.closestToRune(start).Runes
	e.caret.end = e.closestToRune(end).Runes
	e.clampCursorToGraphemes()
}

// SelectedText returns the currently selected text (if any) from the editor,
// filling the provided byte slice if it is large enough or allocating and
// returning a new byte slice if the provided one is insufficient.
// Callers can guarantee that the buf is large enough by providing a buffer
// with capacity e.SelectionLen()*utf8.UTFMax.
func (e *textView) SelectedText(buf []byte) []byte {
	startOff := e.src.RuneOffset(e.caret.start)
	endOff := e.src.RuneOffset(e.caret.end)
	start := min(startOff, endOff)
	end := max(startOff, endOff)
	if cap(buf) < end-start {
		buf = make([]byte, end-start)
	}
	buf = buf[:end-start]
	n, _ := e.src.ReadAt(buf, int64(start))
	// There is no way to reasonably handle a read error here. We rely upon
	// implementations of textSource to provide other ways to signal errors
	// if the user cares about that, and here we use whatever data we were
	// able to read.
	return buf[:n]
}

func (e *textView) updateSelection(selAct selectionAction) {
	if selAct == selectionClear {
		e.ClearSelection()
	}
}

// ClearSelection clears the selection, by setting the selection end equal to
// the selection start.
func (e *textView) ClearSelection() {
	e.caret.end = e.caret.start
}

// Undo revert the last operation(s) and mark the textview invalid.
func (e *textView) Undo() ([]buffer.CursorPos, bool) {
	cursors, ok := e.src.Undo()
	if ok {
		e.invalidate()
	}

	return cursors, ok
}

// Redo revert the last undo operation(s) and mark the textview invalid.
func (e *textView) Redo() ([]buffer.CursorPos, bool) {
	cursors, ok := e.src.Redo()
	if ok {
		e.invalidate()
	}

	return cursors, ok
}

// Regions returns visible regions covering the rune range [start,end).
func (e *textView) Regions(start, end int, regions []Region) []Region {
	viewport := image.Rectangle{
		Min: e.scrollOff,
		Max: e.viewSize.Add(e.scrollOff),
	}
	return e.layouter.Locate(viewport, start, end, regions)
}

func absFixed(i fixed.Int26_6) fixed.Int26_6 {
	if i < 0 {
		return -i
	}
	return i
}

func fixedToFloat(i fixed.Int26_6) float32 {
	return float32(i) / 64.0
}

func floatToFixed(f float32) fixed.Int26_6 {
	return fixed.Int26_6(f * 64)
}
