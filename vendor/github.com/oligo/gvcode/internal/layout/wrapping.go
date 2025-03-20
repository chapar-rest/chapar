package layout

import (
	"iter"

	"gioui.org/text"
	"github.com/go-text/typesetting/segmenter"
	"golang.org/x/image/math/fixed"
)

// breakOption represents a rune index in rune slice at which it is
// safe to break a line.
type breakOption int

type breaker struct {
	wordSegmenter     *segmenter.LineIterator
	graphemeSegmenter *segmenter.GraphemeIterator
	runes             int
	// committed break marks the point before which all break options
	// are handled.
	committed breakOption
	// wordBreak marks the cursor position of the wordSegmenter.
	wordBreak breakOption
	// graphemeBreak marks the cursor position of the graphemeSegmenter.
	graphemeBreak breakOption
	// prevWordUnread marks the runes between committed and wordBreak as
	// unread. They should be re-evaluated in the next round.
	prevWordUnread bool
	// prevGraphemeUnread marks the runes between committed and graphemeBreak as
	// unread. They should be re-evaluated in the next round.
	prevGraphemeUnread bool
}

// newBreaker returns a breaker initialized to break the text.
func newBreaker(seg *segmenter.Segmenter, text []rune) *breaker {
	seg.Init(text)
	br := &breaker{
		wordSegmenter:     seg.LineIterator(),
		graphemeSegmenter: seg.GraphemeIterator(),
		runes:             len(text),
	}
	return br
}

func (b *breaker) nextWordBreak() (breakOption, bool) {
	if b.prevWordUnread && b.wordBreak > b.committed {
		b.prevWordUnread = false
		return b.wordBreak, true
	}

	var opt breakOption
	for b.wordSegmenter.Next() {
		line := b.wordSegmenter.Line()
		opt = breakOption(line.Offset + len(line.Text))
		if opt > b.wordBreak {
			b.wordBreak = opt
			return opt, true
		}
	}

	return 0, false
}

func (b *breaker) nextGraphemeBreak() (breakOption, bool) {
	if b.prevGraphemeUnread && b.graphemeBreak > b.committed {
		b.prevGraphemeUnread = false
		return b.graphemeBreak, true
	}

	var opt breakOption
	for b.graphemeSegmenter.Next() {
		grapheme := b.graphemeSegmenter.Grapheme()
		opt = breakOption(grapheme.Offset + len(grapheme.Text))
		if opt > b.graphemeBreak {
			b.graphemeBreak = opt
			return opt, true
		}

	}

	return 0, false
}

func (b *breaker) markPrevWordUnread() {
	b.prevWordUnread = true
}

func (b *breaker) markPrevGraphemeUnread() {
	b.prevGraphemeUnread = true
}

func (b *breaker) markCommitted() {
	if !b.prevWordUnread && b.committed < b.wordBreak {
		b.committed = b.wordBreak
		if b.graphemeBreak < b.committed {
			b.graphemeBreak = b.committed
		}
	}

	if !b.prevGraphemeUnread && b.committed < b.graphemeBreak {
		b.committed = b.graphemeBreak
		if b.wordBreak < b.committed {
			b.wordBreak = b.committed
		}
	}

}

// glyphReader is a buffered glyph reader to read from the shaped glyphs.
type glyphReader struct {
	nextGlyph func() (text.Glyph, bool)
	// total glyph read from the shaper
	buf []text.Glyph
	// runes marks how many runes of glyphs we have alrealy read from the nextGlyph(from the shaper).
	runes int
	// offset is the rune offset that marks the first unread glyphs.
	offset int
	// glyphOff is the glyph offset that marks the first unread glyphs. It should align the offset.
	glyphOff int
}

// read one glyph until offset is equal to runeOff.
func (b *glyphReader) next(runeOff int) text.Glyph {
	// if runeOff<0, bypass this check.
	if runeOff >= 0 && runeOff <= b.offset {
		return text.Glyph{}
	}

	if b.offset < b.runes {
		g := b.buf[b.glyphOff]
		b.offset += int(g.Runes) // Runes is only set for glyph cluster break.
		b.glyphOff++
		return g
	}

	// no more unread glyph. read one more from the shaper.
	gl, ok := b.nextGlyph()
	if !ok {
		return text.Glyph{}
	}

	b.buf = append(b.buf, gl)
	b.runes += int(gl.Runes)
	b.offset += int(gl.Runes)
	b.glyphOff++
	return gl
}

func (b *glyphReader) seekTo(runeOff int) {
	if b.offset == runeOff {
		return
	}

	if runeOff-b.offset > 0 {
		for runeOff > b.offset {
			if b.glyphOff > len(b.buf) {
				break
			}
			if gl := b.buf[b.glyphOff-1]; gl.Runes > 0 {
				b.offset++
			}
			b.glyphOff++
		}
	} else {
		for b.offset > runeOff {
			if b.glyphOff <= 0 {
				break
			}
			if gl := b.buf[b.glyphOff-1]; gl.Runes > 0 {
				b.offset--
			}
			b.glyphOff--
		}
	}
}

func (b *glyphReader) reset() {
	b.buf = b.buf[:0]
	b.runes = 0
	b.offset = 0
	b.glyphOff = 0
}

// advance calculates the advance of all glyphs.
func advanceOfGlyphs(glyphs []text.Glyph) fixed.Int26_6 {
	width := fixed.I(0)

	for _, gl := range glyphs {
		width += gl.Advance
	}

	return width
}

// lineWrapper wraps a paragraph of text to lines using the greedy line breaking
// algorithm. Unlike the normal line breaking routine, it expands tab characters
// to the next tabstop before wrapping.
type lineWrapper struct {
	seg             segmenter.Segmenter
	breaker         *breaker
	maxWidth        int
	spaceGlyph      *text.Glyph
	tabStopInterval fixed.Int26_6
	currentLine     Line
	glyphBuf        glyphReader
	glyphs          []text.Glyph
}

func (w *lineWrapper) setup(nextGlyph func() (text.Glyph, bool), paragraph []rune, maxWidth int, tabWidth int, spaceGlyph *text.Glyph) {
	w.breaker = newBreaker(&w.seg, paragraph)
	w.maxWidth = maxWidth
	w.tabStopInterval = spaceGlyph.Advance.Mul(fixed.I(tabWidth))
	w.spaceGlyph = spaceGlyph
	w.currentLine = Line{}
	w.glyphBuf.nextGlyph = nextGlyph
	w.glyphBuf.reset()
	w.glyphs = w.glyphs[:0]
}

// WrapParagraph wraps a paragraph of text using a policy similar to the WhenNecessary LineBreakPolicy from gotext/typesetting.
// It is also the default policy used by Gio.
func (w *lineWrapper) WrapParagraph(glyphsIter iter.Seq[text.Glyph], paragraph []rune, maxWidth int, tabWidth int, spaceGlyph *text.Glyph) []*Line {
	nextGlyph, stop := iter.Pull(glyphsIter)
	defer stop()
	w.setup(nextGlyph, paragraph, maxWidth, tabWidth, spaceGlyph)

	lines := make([]*Line, 0)

	for {
		l := w.wrapNextLine(paragraph)
		if len(l.Glyphs) == 0 {
			break
		}

		lines = append(lines, &l)
		w.currentLine = Line{}
	}

	return lines
}

// wrapNextLine breaking lines by looking at the break opportunities defined in https://unicode.org/reports/tr14 first.
// If no break opportunities can be found, it'll try to break at the grapheme cluster bounderies.
func (w *lineWrapper) wrapNextLine(paragraph []rune) Line {
	for {
		// try to break at each word boundaries.
		nextBreak, ok := w.breaker.nextWordBreak()
		if !ok {
			break
		}

		lastOff := w.glyphBuf.offset
		glyphs := w.readToNextBreak(nextBreak, paragraph)
		// check if the line will exceeds the maxWidth if we put the glyph in the current line.
		if w.currentLine.Width+advanceOfGlyphs(glyphs) > fixed.I(w.maxWidth) {
			w.breaker.markPrevWordUnread()
			w.glyphBuf.seekTo(lastOff)
			break
		}

		w.currentLine.append(glyphs...)
		w.breaker.markCommitted()
	}

	if len(w.currentLine.Glyphs) > 0 {
		return w.currentLine
	}

	for {
		// try to break at grapheme cluster boundaries.
		nextBreak, ok := w.breaker.nextGraphemeBreak()
		if !ok {
			break
		}

		lastOff := w.glyphBuf.offset
		glyphs := w.readToNextBreak(nextBreak, paragraph)
		// check if the line will exceeds the maxWidth if we put the glyph in the current line.
		if w.currentLine.Width+advanceOfGlyphs(glyphs) > fixed.I(w.maxWidth) {
			w.breaker.markPrevGraphemeUnread()
			w.glyphBuf.seekTo(lastOff)
			break
		}

		w.currentLine.append(glyphs...)
		w.breaker.markCommitted()
	}

	if len(w.currentLine.Glyphs) > 0 {
		return w.currentLine
	}

	// left over glyphs that cannot be treated as break opportunities, usually the fake glyph starts a new paragraph.
	w.glyphs = w.glyphs[:0]

	for {
		gl := w.glyphBuf.next(-1)
		if gl == (text.Glyph{}) {
			break
		}
		w.glyphs = append(w.glyphs, gl)
	}

	if len(w.glyphs) > 0 {
		w.currentLine.append(w.glyphs...)
	}

	return w.currentLine
}

// readToNextBreak read glyphs from the iterator until it reached to break option.
// It returns a boolean value indicating whether it has terminated early.
func (w *lineWrapper) readToNextBreak(breakAtIdx breakOption, paragraph []rune) []text.Glyph {
	w.glyphs = w.glyphs[:0]

	for {
		gl := w.glyphBuf.next(int(breakAtIdx))
		if gl == (text.Glyph{}) {
			break
		}

		advance := advanceOfGlyphs(w.glyphs)

		if gl.Flags&text.FlagClusterBreak != 0 {
			//log.Println("rune: ", string(paragraph[w.glyphBuf.offset-1]), gl.Flags&text.FlagParagraphStart != 0)
			isTab := paragraph[w.glyphBuf.offset-1] == '\t'
			if isTab {
				// the rune is a tab, expand it before line wrapping.
				w.expandTabGlyph(w.currentLine.Width+advance, &gl)
			}
		}

		w.glyphs = append(w.glyphs, gl)
	}

	return w.glyphs
}

// expandTabGlyph expand the tab to the next tab stop.
func (w *lineWrapper) expandTabGlyph(lineWidth fixed.Int26_6, gl *text.Glyph) {
	tabStopInterval := w.tabStopInterval
	if tabStopInterval <= 0 {
		tabStopInterval = gl.Advance
	}
	nextTabStop := (lineWidth/tabStopInterval + 1) * tabStopInterval
	gl.Advance = nextTabStop - lineWidth
	gl.Offset = fixed.Point26_6{}
	gl.ID = w.spaceGlyph.ID
	gl.Ascent = w.spaceGlyph.Ascent
	gl.Descent = w.spaceGlyph.Descent
}
