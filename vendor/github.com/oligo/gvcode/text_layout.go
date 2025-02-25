package gvcode

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
	"github.com/oligo/gvcode/buffer"
	"golang.org/x/image/math/fixed"
)

type textLayout struct {
	src        buffer.TextSource
	reader     *bufio.Reader
	params     text.Parameters
	spaceGlyph text.Glyph
	wrapper    lineWrapper

	// positions contain all possible caret positions, sorted by rune index.
	positions []combinedPos
	// screenLines contains metadata about the size and position of each line of
	// text on the screen.
	lines []*line
	// lineRanges contain all line pixel coordinates in the document coordinates.
	lineRanges []lineRange
	// graphemes tracks the indices of grapheme cluster boundaries within text source.
	graphemes []int
	seg       segmenter.Segmenter

	// bounds is the logical bounding box of the text.
	bounds image.Rectangle
	// baseline tracks the location of the first line's baseline.
	baseline int
}

func newTextLayout(src buffer.TextSource) textLayout {
	return textLayout{
		src:    src,
		reader: bufio.NewReader(src),
	}
}

// Calculate line height. Maybe there's a better way?
func (tl *textLayout) calcLineHeight(params *text.Parameters) fixed.Int26_6 {
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
func (tl *textLayout) reset() {
	tl.src.Seek(0, io.SeekStart)
	tl.reader.Reset(tl.src)
	tl.positions = tl.positions[:0]
	tl.lines = tl.lines[:0]
	tl.lineRanges = tl.lineRanges[:0]
	tl.graphemes = tl.graphemes[:0]
	tl.bounds = image.Rectangle{}
}

func (tl *textLayout) Layout(shaper *text.Shaper, params *text.Parameters, tabWidth int, wrapLine bool) layout.Dimensions {
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
				text, readErr := tl.reader.ReadBytes('\n')
				// the last line returned by ReadBytes returns EOF and may have remaining bytes to process.
				if len(text) > 0 {
					tl.layoutNextParagraph(shaper, string(text), paragraphCount-1 == currentIdx, tabWidth, wrapLine)

					paragraphRunes := []rune(string(text))
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

		tl.calculateXOffsets(tl.lines)
		tl.calculateYOffsets(tl.lines)

		// build position index
		for idx, line := range tl.lines {
			tl.indexGlyphs(idx, line)
			tl.updateBounds(line)
			// log.Printf("line[%d]: %s", idx, line)

		}

		tl.trackLines(tl.lines)
	}

	dims := layout.Dimensions{Size: tl.bounds.Size()}
	dims.Baseline = dims.Size.Y - tl.baseline
	return dims
}

func (tl *textLayout) layoutNextParagraph(shaper *text.Shaper, paragraph string, isLastParagrah bool, tabWidth int, wrapLine bool) {
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

	tl.lines = append(tl.lines, lines...)
}

func (tl *textLayout) wrapParagraph(glyphs glyphIter, paragraph []rune, maxWidth int, tabWidth int, spaceGlyph *text.Glyph) []*line {
	return tl.wrapper.WrapParagraph(glyphs.All(), paragraph, maxWidth, tabWidth, spaceGlyph)
}

func (tl *textLayout) fakeLayout() {
	// Make a fake glyph for every rune in the reader.
	b := bufio.NewReader(tl.src)
	for _, _, err := b.ReadRune(); err != io.EOF; _, _, err = b.ReadRune() {
		g := text.Glyph{Runes: 1, Flags: text.FlagClusterBreak}
		line := line{}
		line.append(g)
		tl.indexGlyphs(0, &line)
	}

	var graphemeReader graphemeReader
	graphemeReader.SetSource(tl.src)

	for g := graphemeReader.Graphemes(); len(g) > 0; g = graphemeReader.Graphemes() {
		if len(tl.graphemes) > 0 && g[0] == tl.graphemes[len(tl.graphemes)-1] {
			g = g[1:]
		}
		tl.graphemes = append(tl.graphemes, g...)
	}
}

func (tl *textLayout) calculateYOffsets(lines []*line) {
	if len(lines) <= 0 {
		return
	}

	lineHeight := tl.calcLineHeight(&tl.params)
	// Ceil the first value to ensure that we don't baseline it too close to the top of the
	// viewport and cut off the top pixel.
	currentY := lines[0].ascent.Ceil()
	for i := range lines {
		if i > 0 {
			currentY += lineHeight.Round()
		}
		lines[i].adjustYOff(currentY)
	}
}

func (tl *textLayout) calculateXOffsets(lines []*line) {
	runeOff := 0
	for _, line := range lines {
		alignOff := tl.params.Alignment.Align(tl.params.Locale.Direction, line.width, tl.params.MaxWidth)
		line.recompute(alignOff, runeOff)
		runeOff += line.runes
	}
}

func (tl *textLayout) shapeRune(shaper *text.Shaper, params text.Parameters, r rune) (text.Glyph, error) {
	shaper.LayoutString(params, string(r))
	glyph, ok := shaper.NextGlyph()
	if !ok {
		return text.Glyph{}, fmt.Errorf("shaper.LayoutString failed for rune: %s", string(r))
	}

	return glyph, nil
}

func (tl *textLayout) indexGraphemeClusters(paragraph []rune, runeOffset int) {
	tl.seg.Init(paragraph)
	iter := tl.seg.GraphemeIterator()
	if len(tl.graphemes) == 0 {
		if iter.Next() {
			grapheme := iter.Grapheme()
			tl.graphemes = append(tl.graphemes,
				runeOffset+grapheme.Offset,
				runeOffset+grapheme.Offset+len(grapheme.Text),
			)
		}
	}

	for iter.Next() {
		grapheme := iter.Grapheme()
		tl.graphemes = append(tl.graphemes, runeOffset+grapheme.Offset+len(grapheme.Text))
	}

}

func (tl *textLayout) updateBounds(line *line) {
	logicalBounds := line.bounds()
	if tl.bounds == (image.Rectangle{}) {
		tl.baseline = int(line.yOff)
		tl.bounds = logicalBounds
	} else {
		tl.bounds.Min.X = min(tl.bounds.Min.X, logicalBounds.Min.X)
		tl.bounds.Min.Y = min(tl.bounds.Min.Y, logicalBounds.Min.Y)
		tl.bounds.Max.X = max(tl.bounds.Max.X, logicalBounds.Max.X)
		tl.bounds.Max.Y = max(tl.bounds.Max.Y, logicalBounds.Max.Y)
	}
}

func (tl *textLayout) trackLines(lines []*line) {
	if len(lines) <= 0 {
		tl.lineRanges = append(tl.lineRanges, lineRange{})
		return
	}

	rng := lineRange{}
	var lastGlyph *text.Glyph
	for _, l := range lines {
		if rng == (lineRange{}) {
			rng.start(l.glyphs[0])
			lastGlyph = l.glyphs[len(l.glyphs)-1]
			rng.end(lastGlyph)
		} else {
			lastGlyph = l.glyphs[len(l.glyphs)-1]
			rng.end(lastGlyph)
		}

		if lastGlyph.Flags&text.FlagParagraphBreak != 0 {
			tl.lineRanges = append(tl.lineRanges, rng)
			rng = lineRange{}
		}
	}

	if rng != (lineRange{}) {
		tl.lineRanges = append(tl.lineRanges, rng)
	}
}

func (tl *textLayout) insertPosition(pos combinedPos) {
	lastIdx := len(tl.positions) - 1
	if lastIdx >= 0 {
		lastPos := tl.positions[lastIdx]
		if lastPos.runes == pos.runes && (lastPos.y != pos.y || (lastPos.x == pos.x)) {
			// If we insert a consecutive position with the same logical position,
			// overwrite the previous position with the new one.
			tl.positions[lastIdx] = pos
			return
		}
	}
	tl.positions = append(tl.positions, pos)
}

// Glyph indexes the provided glyph, generating text cursor positions for it.
func (tl *textLayout) indexGlyphs(idx int, line *line) {
	pos := combinedPos{}
	pos.runes = line.runeOff
	pos.lineCol.line = idx

	midCluster := false
	clusterAdvance := fixed.I(0)
	var direction text.Flags

	for _, gl := range line.glyphs {
		// needsNewLine := gl.Flags&text.FlagLineBreak != 0
		needsNewRun := gl.Flags&text.FlagRunBreak != 0
		breaksParagraph := gl.Flags&text.FlagParagraphBreak != 0
		breaksCluster := gl.Flags&text.FlagClusterBreak != 0
		// We should insert new positions if the glyph we're processing terminates
		// a glyph cluster, has nonzero runes, and is not a hard newline.
		insertPositionsWithin := breaksCluster && !breaksParagraph && gl.Runes > 0

		// Get the text direction.
		direction = gl.Flags & text.FlagTowardOrigin
		pos.towardOrigin = direction == text.FlagTowardOrigin
		if !midCluster {
			// Create the text position prior to the glyph.
			pos.x = gl.X
			pos.y = int(gl.Y)
			pos.ascent = gl.Ascent
			pos.descent = gl.Descent
			if pos.towardOrigin {
				pos.x += gl.Advance
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
			pos.runes += int(gl.Runes)
		}

		// Always track the cumulative advance added by the glyph, even if it
		// doesn't terminate a cluster itself.
		clusterAdvance += gl.Advance
		if insertPositionsWithin {
			// Construct the text positions _within_ gl.
			pos.y = int(gl.Y)
			pos.ascent = gl.Ascent
			pos.descent = gl.Descent
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
			if pos.towardOrigin {
				// If RTL, subtract increments from the width of the cluster
				// instead of adding.
				adjust = width
				perRune = -perRune
			}
			for i := 1; i <= positionCount; i++ {
				pos.x = gl.X + adjust + perRune*fixed.Int26_6(i)
				pos.runes += runesPerPosition
				pos.lineCol.col += runesPerPosition
				tl.insertPosition(pos)
			}
			clusterAdvance = 0
		}

		if needsNewRun {
			pos.runIndex++
		}
	}

}

// This method and some of the following methods are adapted from the Gio's package gioui.org/widget.
// Original copyright (c) 2018-2025 Elias Naur and Gio contributors.
//
// incrementPosition returns the next position after pos (if any). Pos _must_ be
// an unmodified position acquired from one of the closest* methods. If eof is
// true, there was no next position.
func (tl *textLayout) incrementPosition(pos combinedPos) (next combinedPos, eof bool) {
	candidate, index := tl.closestToRune(pos.runes)
	for candidate != pos && index+1 < len(tl.positions) {
		index++
		candidate = tl.positions[index]
	}
	if index+1 < len(tl.positions) {
		return tl.positions[index+1], false
	}
	return candidate, true
}

func (tl *textLayout) closestToRune(runeIdx int) (combinedPos, int) {
	if len(tl.positions) == 0 {
		return combinedPos{}, 0
	}
	i := sort.Search(len(tl.positions), func(i int) bool {
		pos := tl.positions[i]
		return pos.runes >= runeIdx
	})
	if i > 0 {
		i--
	}
	closest := tl.positions[i]
	closestI := i
	for ; i < len(tl.positions); i++ {
		if tl.positions[i].runes == runeIdx {
			return tl.positions[i], i
		}
	}
	return closest, closestI
}

func (tl *textLayout) closestToLineCol(lineCol screenPos) combinedPos {
	if len(tl.positions) == 0 {
		return combinedPos{}
	}
	i := sort.Search(len(tl.positions), func(i int) bool {
		pos := tl.positions[i]
		return pos.lineCol.line > lineCol.line || (pos.lineCol.line == lineCol.line && pos.lineCol.col >= lineCol.col)
	})
	if i > 0 {
		i--
	}
	prior := tl.positions[i]
	if i+1 >= len(tl.positions) {
		return prior
	}
	next := tl.positions[i+1]
	if next.lineCol != lineCol {
		return prior
	}
	return next
}

func (tl *textLayout) closestToXY(x fixed.Int26_6, y int) combinedPos {
	if len(tl.positions) == 0 {
		return combinedPos{}
	}
	i := sort.Search(len(tl.positions), func(i int) bool {
		pos := tl.positions[i]
		return pos.y+pos.descent.Round() >= y
	})
	// If no position was greater than the provided Y, the text is too
	// short. Return either the last position or (if there are no
	// positions) the zero position.
	if i == len(tl.positions) {
		return tl.positions[i-1]
	}
	first := tl.positions[i]
	// Find the best X coordinate.
	closest := i
	closestDist := dist(first.x, x)
	line := first.lineCol.line
	// NOTE(whereswaldon): there isn't a simple way to accelerate this. Bidi text means that the x coordinates
	// for positions have no fixed relationship. In the future, we can consider sorting the positions
	// on a line by their x coordinate and caching that. It'll be a one-time O(nlogn) per line, but
	// subsequent uses of this function for that line become O(logn). Right now it's always O(n).
	for i := i + 1; i < len(tl.positions) && tl.positions[i].lineCol.line == line; i++ {
		candidate := tl.positions[i]
		distance := dist(candidate.x, x)
		// If we are *really* close to the current position candidate, just choose it.
		if distance.Round() == 0 {
			return tl.positions[i]
		}
		if distance < closestDist {
			closestDist = distance
			closest = i
		}
	}
	return tl.positions[closest]
}

// locate returns highlight regions covering the glyphs that represent the runes in
// [startRune,endRune). If the rects parameter is non-nil, locate will use it to
// return results instead of allocating, provided that there is enough capacity.
// The returned regions have their Bounds specified relative to the provided
// viewport.
func (tl *textLayout) locate(viewport image.Rectangle, startRune, endRune int, rects []Region) []Region {
	if startRune > endRune {
		startRune, endRune = endRune, startRune
	}
	rects = rects[:0]
	caretStart, _ := tl.closestToRune(startRune)
	caretEnd, _ := tl.closestToRune(endRune)

	for lineIdx := caretStart.lineCol.line; lineIdx < len(tl.lines); lineIdx++ {
		if lineIdx > caretEnd.lineCol.line {
			break
		}
		pos := tl.closestToLineCol(screenPos{line: lineIdx})
		if int(pos.y)+pos.descent.Ceil() < viewport.Min.Y {
			continue
		}
		if int(pos.y)-pos.ascent.Ceil() > viewport.Max.Y {
			break
		}
		line := tl.lines[lineIdx]
		if lineIdx > caretStart.lineCol.line && lineIdx < caretEnd.lineCol.line {
			startX := line.xOff
			endX := startX + line.width
			// The entire line is selected.
			rects = append(rects, makeRegion(line, pos.y, startX, endX))
			continue
		}
		selectionStart := caretStart
		selectionEnd := caretEnd
		if lineIdx != caretStart.lineCol.line {
			// This line does not contain the beginning of the selection.
			selectionStart = tl.closestToLineCol(screenPos{line: lineIdx})
		}
		if lineIdx != caretEnd.lineCol.line {
			// This line does not contain the end of the selection.
			selectionEnd = tl.closestToLineCol(screenPos{line: lineIdx, col: math.MaxInt})
		}

		var (
			startX, endX fixed.Int26_6
			eof          bool
		)
	lineLoop:
		for !eof {
			startX = selectionStart.x
			if selectionStart.runIndex == selectionEnd.runIndex {
				// Commit selection.
				endX = selectionEnd.x
				rects = append(rects, makeRegion(line, pos.y, startX, endX))
				break
			} else {
				currentDirection := selectionStart.towardOrigin
				previous := selectionStart
			runLoop:
				for !eof {
					// Increment the start position until the next logical run.
					for startRun := selectionStart.runIndex; selectionStart.runIndex == startRun; {
						previous = selectionStart
						selectionStart, eof = tl.incrementPosition(selectionStart)
						if eof {
							endX = selectionStart.x
							rects = append(rects, makeRegion(line, pos.y, startX, endX))
							break runLoop
						}
					}
					if selectionStart.towardOrigin != currentDirection {
						endX = previous.x
						rects = append(rects, makeRegion(line, pos.y, startX, endX))
						break
					}
					if selectionStart.runIndex == selectionEnd.runIndex {
						// Commit selection.
						endX = selectionEnd.x
						rects = append(rects, makeRegion(line, pos.y, startX, endX))
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
