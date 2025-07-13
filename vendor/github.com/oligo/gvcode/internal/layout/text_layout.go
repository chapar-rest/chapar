package layout

import (
	"bufio"
	"fmt"
	"image"
	"io"
	"math"
	"sort"
	"strings"

	"gioui.org/layout"
	"gioui.org/text"
	"github.com/go-text/typesetting/segmenter"
	"github.com/oligo/gvcode/internal/buffer"
	"golang.org/x/image/math/fixed"
)

type TextLayout struct {
	src        buffer.TextSource
	reader     *bufio.Reader
	params     text.Parameters
	spaceGlyph text.Glyph
	wrapper    lineWrapper
	seg        segmenter.Segmenter

	// Positions contain all possible caret positions, sorted by rune index.
	Positions []CombinedPos
	// lines contain metadata about the size and position of each line of
	// text on the screen.
	Lines []*Line
	// Paragraphs contain size and position of each paragraph of text on the screen.
	Paragraphs []Paragraph
	// Graphemes tracks the indices of grapheme cluster boundaries within text source.
	Graphemes []int
	// bounds is the logical bounding box of the text.
	bounds image.Rectangle
	// baseline tracks the location of the first line's baseline.
	baseline int
}

func NewTextLayout(src buffer.TextSource) TextLayout {
	return TextLayout{
		src:    src,
		reader: bufio.NewReader(buffer.NewReader(src)),
	}
}

// Calculate line height. Maybe there's a better way?
func (tl *TextLayout) calcLineHeight(params *text.Parameters) fixed.Int26_6 {
	lineHeight := params.LineHeight
	// align with how text.Shaper handles default value of params.LineHeight.
	if lineHeight == 0 {
		lineHeight = params.PxPerEm
	}
	lineHeightScale := params.LineHeightScale
	// align with how text.Shaper handles default value of tv.params.LineHeightScale.
	if lineHeightScale == 0 {
		lineHeightScale = 1.2
	}

	return floatToFixed(fixedToFloat(lineHeight) * lineHeightScale)
}

// reset prepares the index for reuse.
func (tl *TextLayout) reset() {
	tl.reader.Reset(buffer.NewReader(tl.src))
	tl.Positions = tl.Positions[:0]
	tl.Lines = tl.Lines[:0]
	tl.Paragraphs = tl.Paragraphs[:0]
	tl.Graphemes = tl.Graphemes[:0]
	tl.bounds = image.Rectangle{}
	tl.baseline = 0
}

func (tl *TextLayout) Layout(shaper *text.Shaper, params *text.Parameters, tabWidth int, wrapLine bool) layout.Dimensions {
	tl.reset()
	tl.params = *params
	paragraphCount := tl.src.Lines()

	if shaper == nil {
		tl.fakeLayout()
	} else {
		tl.spaceGlyph, _ = tl.shapeRune(shaper, tl.params, '\u0020')
		if paragraphCount > 0 {
			runeOffset := 0
			currentIdx := 0

			for {
				text, readErr := tl.reader.ReadString('\n')
				// the last line returned by ReadBytes returns EOF and may have remaining bytes to process.
				if len(text) > 0 {
					tl.layoutNextParagraph(shaper, text, paragraphCount-1 == currentIdx, tabWidth, wrapLine)

					paragraphRunes := []rune(text)
					tl.indexGraphemeClusters(paragraphRunes, runeOffset)
					runeOffset += len(paragraphRunes)
					currentIdx++
				}

				if readErr != nil {
					break
				}
			}
		} else {
			tl.layoutNextParagraph(shaper, "", true, tabWidth, wrapLine)
		}

		tl.calculateXOffsets(tl.Lines)
		tl.calculateYOffsets(tl.Lines)

		// build position index
		for idx, line := range tl.Lines {
			tl.indexGlyphs(idx, line)
			tl.updateBounds(line)
			// log.Printf("line[%d]: %s", idx, line)

		}

		tl.trackLines(tl.Lines)
	}

	dims := layout.Dimensions{Size: tl.bounds.Size()}
	dims.Baseline = dims.Size.Y - tl.baseline
	return dims
}

func (tl *TextLayout) layoutNextParagraph(shaper *text.Shaper, paragraph string, isLastParagrah bool, tabWidth int, wrapLine bool) {
	params := tl.params
	maxWidth := params.MaxWidth
	params.MaxWidth = 1e6
	if !wrapLine {
		maxWidth = params.MaxWidth
	}
	shaper.LayoutString(params, paragraph)

	lines := tl.wrapParagraph(glyphIter{shaper: shaper}, []rune(paragraph), maxWidth, tabWidth, &tl.spaceGlyph)
	if strings.HasSuffix(paragraph, "\n") && len(lines) > 0 && !isLastParagrah {
		lines = lines[:len(lines)-1]
	}

	tl.Lines = append(tl.Lines, lines...)
}

func (tl *TextLayout) wrapParagraph(glyphs glyphIter, paragraph []rune, maxWidth int, tabWidth int, spaceGlyph *text.Glyph) []*Line {
	return tl.wrapper.WrapParagraph(glyphs.All(), paragraph, maxWidth, tabWidth, spaceGlyph)
}

func (tl *TextLayout) fakeLayout() {
	srcReader := buffer.NewReader(tl.src)
	// Make a fake glyph for every rune in the reader.
	b := bufio.NewReader(srcReader)
	for _, _, err := b.ReadRune(); err != io.EOF; _, _, err = b.ReadRune() {
		g := text.Glyph{Runes: 1, Flags: text.FlagClusterBreak}
		line := Line{}
		line.append(g)
		tl.indexGlyphs(0, &line)
	}

	var graphemeReader graphemeReader
	graphemeReader.SetSource(tl.src)

	for g := graphemeReader.Graphemes(); len(g) > 0; g = graphemeReader.Graphemes() {
		if len(tl.Graphemes) > 0 && g[0] == tl.Graphemes[len(tl.Graphemes)-1] {
			g = g[1:]
		}
		tl.Graphemes = append(tl.Graphemes, g...)
	}
}

func (tl *TextLayout) calculateYOffsets(lines []*Line) {
	if len(lines) <= 0 {
		return
	}

	lineHeight := tl.calcLineHeight(&tl.params)
	// Ceil the first value to ensure that we don't baseline it too close to the top of the
	// viewport and cut off the top pixel.
	currentY := lines[0].Ascent.Ceil()
	for i := range lines {
		if i > 0 {
			currentY += lineHeight.Round()
		}
		lines[i].adjustYOff(currentY)
	}
}

func (tl *TextLayout) calculateXOffsets(lines []*Line) {
	runeOff := 0
	for _, line := range lines {
		alignOff := tl.params.Alignment.Align(tl.params.Locale.Direction, line.Width, tl.params.MaxWidth)
		line.recompute(alignOff, runeOff)
		runeOff += line.Runes
	}
}

func (tl *TextLayout) shapeRune(shaper *text.Shaper, params text.Parameters, r rune) (text.Glyph, error) {
	shaper.LayoutString(params, string(r))
	glyph, ok := shaper.NextGlyph()
	if !ok {
		return text.Glyph{}, fmt.Errorf("shaper.LayoutString failed for rune: %s", string(r))
	}

	return glyph, nil
}

func (tl *TextLayout) indexGraphemeClusters(paragraph []rune, runeOffset int) {
	tl.seg.Init(paragraph)
	iter := tl.seg.GraphemeIterator()
	if len(tl.Graphemes) == 0 {
		if iter.Next() {
			grapheme := iter.Grapheme()
			tl.Graphemes = append(tl.Graphemes,
				runeOffset+grapheme.Offset,
				runeOffset+grapheme.Offset+len(grapheme.Text),
			)
		}
	}

	for iter.Next() {
		grapheme := iter.Grapheme()
		tl.Graphemes = append(tl.Graphemes, runeOffset+grapheme.Offset+len(grapheme.Text))
	}

}

func (tl *TextLayout) updateBounds(line *Line) {
	logicalBounds := line.bounds()
	if tl.bounds == (image.Rectangle{}) {
		tl.baseline = int(line.YOff)
		tl.bounds = logicalBounds
	} else {
		tl.bounds.Min.X = min(tl.bounds.Min.X, logicalBounds.Min.X)
		tl.bounds.Min.Y = min(tl.bounds.Min.Y, logicalBounds.Min.Y)
		tl.bounds.Max.X = max(tl.bounds.Max.X, logicalBounds.Max.X)
		tl.bounds.Max.Y = max(tl.bounds.Max.Y, logicalBounds.Max.Y)
	}
}

func (tl *TextLayout) trackLines(lines []*Line) {
	if len(lines) <= 0 {
		tl.Paragraphs = append(tl.Paragraphs, Paragraph{})
		return
	}

	rng := Paragraph{}
	for _, l := range lines {
		hasBreak := rng.Add(l)

		if hasBreak {
			tl.Paragraphs = append(tl.Paragraphs, rng)
			rng = Paragraph{}
		}
	}

	if rng != (Paragraph{}) {
		tl.Paragraphs = append(tl.Paragraphs, rng)
	}
}

func (tl *TextLayout) insertPosition(pos CombinedPos) {
	lastIdx := len(tl.Positions) - 1
	if lastIdx >= 0 {
		lastPos := tl.Positions[lastIdx]
		if lastPos.Runes == pos.Runes && (lastPos.Y != pos.Y || (lastPos.X == pos.X)) {
			// If we insert a consecutive position with the same logical position,
			// overwrite the previous position with the new one.
			tl.Positions[lastIdx] = pos
			return
		}
	}
	tl.Positions = append(tl.Positions, pos)
}

// Glyph indexes the provided glyph, generating text cursor positions for it.
func (tl *TextLayout) indexGlyphs(idx int, line *Line) {
	pos := CombinedPos{}
	pos.Runes = line.RuneOff
	pos.LineCol.Line = idx

	midCluster := false
	clusterAdvance := fixed.I(0)
	var direction text.Flags

	for _, gl := range line.Glyphs {
		// needsNewLine := gl.Flags&text.FlagLineBreak != 0
		needsNewRun := gl.Flags&text.FlagRunBreak != 0
		breaksParagraph := gl.Flags&text.FlagParagraphBreak != 0
		breaksCluster := gl.Flags&text.FlagClusterBreak != 0
		// We should insert new positions if the glyph we're processing terminates
		// a glyph cluster, has nonzero runes, and is not a hard newline.
		insertPositionsWithin := breaksCluster && !breaksParagraph && gl.Runes > 0

		// Get the text direction.
		direction = gl.Flags & text.FlagTowardOrigin
		pos.TowardOrigin = direction == text.FlagTowardOrigin
		if !midCluster {
			// Create the text position prior to the glyph.
			pos.X = gl.X
			pos.Y = int(gl.Y)
			pos.Ascent = gl.Ascent
			pos.Descent = gl.Descent
			if pos.TowardOrigin {
				pos.X += gl.Advance
			}
			tl.insertPosition(pos)
		}

		midCluster = !breaksCluster
		if breaksParagraph {
			// Paragraph breaking clusters shouldn't have positions generated for both
			// sides of them. They're always zero-width, so doing so would
			// create two visually identical cursor positions. Just reset
			// cluster state, increment by their runes, and move on to the
			// next glyph.
			clusterAdvance = 0
			pos.Runes += int(gl.Runes)
		}

		// Always track the cumulative advance added by the glyph, even if it
		// doesn't terminate a cluster itself.
		clusterAdvance += gl.Advance
		if insertPositionsWithin {
			// Construct the text positions _within_ gl.
			pos.Y = int(gl.Y)
			pos.Ascent = gl.Ascent
			pos.Descent = gl.Descent
			width := clusterAdvance
			positionCount := int(gl.Runes)
			runesPerPosition := 1
			if gl.Flags&text.FlagTruncator != 0 {
				// Treat the truncator as a single unit that is either selected or not.
				positionCount = 1
				runesPerPosition = int(gl.Runes)
			}
			perRune := width / fixed.Int26_6(positionCount)
			adjust := fixed.Int26_6(0)
			if pos.TowardOrigin {
				// If RTL, subtract increments from the width of the cluster
				// instead of adding.
				adjust = width
				perRune = -perRune
			}
			for i := 1; i <= positionCount; i++ {
				pos.X = gl.X + adjust + perRune*fixed.Int26_6(i)
				pos.Runes += runesPerPosition
				pos.LineCol.Col += runesPerPosition
				tl.insertPosition(pos)
			}
			clusterAdvance = 0
		}

		if needsNewRun {
			pos.RunIndex++
		}
	}

}

// This method and some of the following methods are adapted from the Gio's package gioui.org/widget.
// Original copyright (c) 2018-2025 Elias Naur and Gio contributors.
//
// incrementPosition returns the next position after pos (if any). Pos _must_ be
// an unmodified position acquired from one of the closest* methods. If eof is
// true, there was no next position.
func (tl *TextLayout) incrementPosition(pos CombinedPos) (next CombinedPos, eof bool) {
	candidate, index := tl.ClosestToRune(pos.Runes)
	for candidate != pos && index+1 < len(tl.Positions) {
		index++
		candidate = tl.Positions[index]
	}
	if index+1 < len(tl.Positions) {
		return tl.Positions[index+1], false
	}
	return candidate, true
}

func (tl *TextLayout) ClosestToRune(runeIdx int) (CombinedPos, int) {
	if len(tl.Positions) == 0 {
		return CombinedPos{}, 0
	}
	i := sort.Search(len(tl.Positions), func(i int) bool {
		pos := tl.Positions[i]
		return pos.Runes >= runeIdx
	})
	if i > 0 {
		i--
	}
	closest := tl.Positions[i]
	closestI := i
	for ; i < len(tl.Positions); i++ {
		if tl.Positions[i].Runes == runeIdx {
			return tl.Positions[i], i
		}
	}
	return closest, closestI
}

func (tl *TextLayout) ClosestToLineCol(lineCol ScreenPos) CombinedPos {
	if len(tl.Positions) == 0 {
		return CombinedPos{}
	}
	i := sort.Search(len(tl.Positions), func(i int) bool {
		pos := tl.Positions[i]
		return pos.LineCol.Line > lineCol.Line || (pos.LineCol.Line == lineCol.Line && pos.LineCol.Col >= lineCol.Col)
	})
	if i > 0 {
		i--
	}
	prior := tl.Positions[i]
	if i+1 >= len(tl.Positions) {
		return prior
	}
	next := tl.Positions[i+1]
	if next.LineCol != lineCol {
		return prior
	}
	return next
}

func (tl *TextLayout) ClosestToXY(x fixed.Int26_6, y int) CombinedPos {
	if len(tl.Positions) == 0 {
		return CombinedPos{}
	}
	i := sort.Search(len(tl.Positions), func(i int) bool {
		pos := tl.Positions[i]
		return pos.Y+pos.Descent.Round() >= y
	})
	// If no position was greater than the provided Y, the text is too
	// short. Return either the last position or (if there are no
	// positions) the zero position.
	if i == len(tl.Positions) {
		return tl.Positions[i-1]
	}
	first := tl.Positions[i]
	// Find the best X coordinate.
	closest := i
	closestDist := dist(first.X, x)
	line := first.LineCol.Line
	// NOTE(whereswaldon): there isn't a simple way to accelerate this. Bidi text means that the x coordinates
	// for positions have no fixed relationship. In the future, we can consider sorting the positions
	// on a line by their x coordinate and caching that. It'll be a one-time O(nlogn) per line, but
	// subsequent uses of this function for that line become O(logn). Right now it's always O(n).
	for i := i + 1; i < len(tl.Positions) && tl.Positions[i].LineCol.Line == line; i++ {
		candidate := tl.Positions[i]
		distance := dist(candidate.X, x)
		// If we are *really* close to the current position candidate, just choose it.
		if distance.Round() == 0 {
			return tl.Positions[i]
		}
		if distance < closestDist {
			closestDist = distance
			closest = i
		}
	}
	return tl.Positions[closest]
}

// locate returns highlight regions covering the glyphs that represent the runes in
// [startRune,endRune). If the rects parameter is non-nil, locate will use it to
// return results instead of allocating, provided that there is enough capacity.
// The returned regions have their Bounds specified relative to the provided
// viewport.
func (tl *TextLayout) Locate(viewport image.Rectangle, startRune, endRune int, rects []Region) []Region {
	if startRune > endRune {
		startRune, endRune = endRune, startRune
	}
	rects = rects[:0]
	caretStart, _ := tl.ClosestToRune(startRune)
	caretEnd, _ := tl.ClosestToRune(endRune)

	for lineIdx := caretStart.LineCol.Line; lineIdx < len(tl.Lines); lineIdx++ {
		if lineIdx > caretEnd.LineCol.Line {
			break
		}
		pos := tl.ClosestToLineCol(ScreenPos{Line: lineIdx})
		if int(pos.Y)+pos.Descent.Ceil() < viewport.Min.Y {
			continue
		}
		if int(pos.Y)-pos.Ascent.Ceil() > viewport.Max.Y {
			break
		}
		line := tl.Lines[lineIdx]
		if lineIdx > caretStart.LineCol.Line && lineIdx < caretEnd.LineCol.Line {
			startX := line.XOff
			endX := startX + line.Width
			// The entire line is selected.
			rects = append(rects, makeRegion(line, pos.Y, startX, endX))
			continue
		}
		selectionStart := caretStart
		selectionEnd := caretEnd
		if lineIdx != caretStart.LineCol.Line {
			// This line does not contain the beginning of the selection.
			selectionStart = tl.ClosestToLineCol(ScreenPos{Line: lineIdx})
		}
		if lineIdx != caretEnd.LineCol.Line {
			// This line does not contain the end of the selection.
			selectionEnd = tl.ClosestToLineCol(ScreenPos{Line: lineIdx, Col: math.MaxInt})
		}

		var (
			startX, endX fixed.Int26_6
			eof          bool
		)
	lineLoop:
		for !eof {
			startX = selectionStart.X
			if selectionStart.RunIndex == selectionEnd.RunIndex {
				// Commit selection.
				endX = selectionEnd.X
				rects = append(rects, makeRegion(line, pos.Y, startX, endX))
				break
			} else {
				currentDirection := selectionStart.TowardOrigin
				previous := selectionStart
			runLoop:
				for !eof {
					// Increment the start position until the next logical run.
					for startRun := selectionStart.RunIndex; selectionStart.RunIndex == startRun; {
						previous = selectionStart
						selectionStart, eof = tl.incrementPosition(selectionStart)
						if eof {
							endX = selectionStart.X
							rects = append(rects, makeRegion(line, pos.Y, startX, endX))
							break runLoop
						}
					}
					if selectionStart.TowardOrigin != currentDirection {
						endX = previous.X
						rects = append(rects, makeRegion(line, pos.Y, startX, endX))
						break
					}
					if selectionStart.RunIndex == selectionEnd.RunIndex {
						// Commit selection.
						endX = selectionEnd.X
						rects = append(rects, makeRegion(line, pos.Y, startX, endX))
						break lineLoop
					}
				}
			}
		}
	}
	for i := range rects {
		rects[i].Bounds = rects[i].Bounds.Sub(viewport.Min)
	}
	return rects
}

func dist(a, b fixed.Int26_6) fixed.Int26_6 {
	if a > b {
		return a - b
	}
	return b - a
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
