package layout

import (
	"bufio"
	"fmt"
	"image"
	"io"

	"github.com/go-text/typesetting/segmenter"
	"golang.org/x/image/math/fixed"
)

// ScreenPos represents a character position in text line and column numbers,
// not pixels.
type ScreenPos struct {
	// col is the column, measured in runes.
	Col  int
	Line int
}

// CombinedPos is a point in the editor.
type CombinedPos struct {
	// runes is the offset in runes.
	Runes int
	LineCol ScreenPos
	// Pixel coordinates
	X fixed.Int26_6
	Y int
	Ascent, Descent fixed.Int26_6

	// RunIndex tracks which run this position is within, counted each time
	// we processes an end of run marker.
	RunIndex int
	// towardOrigin tracks whether this glyph's run is progressing toward the
	// origin or away from it.
	TowardOrigin bool
}

func (c CombinedPos) String() string {
	return fmt.Sprintf("[combinedPos] runes: %d, lineCol: [line: %d, col: %d], x: %d, y: %d, ascent: %d, descent: %d, runIndex: %d",
		c.Runes, c.LineCol.Line, c.LineCol.Col, c.X.Ceil(), c.Y, c.Ascent.Ceil(), c.Descent.Ceil(), c.RunIndex)
}

// makeRegion creates a text-aligned rectangle from start to end. The vertical
// dimensions of the rectangle are derived from the provided line's ascent and
// descent, and the y offset of the line's baseline is provided as y.
func makeRegion(line *Line, y int, start, end fixed.Int26_6) Region {
	if start > end {
		start, end = end, start
	}
	dotStart := image.Pt(start.Round(), y)
	dotEnd := image.Pt(end.Round(), y)
	return Region{
		Bounds: image.Rectangle{
			Min: dotStart.Sub(image.Point{Y: line.Ascent.Ceil()}),
			Max: dotEnd.Add(image.Point{Y: line.Descent.Floor()}),
		},
		Baseline: line.Descent.Floor(),
	}
}

// Region describes the position and baseline of an area of interest within
// shaped text.
type Region struct {
	// Bounds is the coordinates of the bounding box relative to the containing
	// widget.
	Bounds image.Rectangle
	// Baseline is the quantity of vertical pixels between the baseline and
	// the bottom of bounds.
	Baseline int
}

// graphemeReader segments paragraphs of text into grapheme clusters.
type graphemeReader struct {
	segmenter.Segmenter
	graphemes  []int
	paragraph  []rune
	source     io.ReaderAt
	cursor     int64
	reader     *bufio.Reader
	runeOffset int
}

// SetSource configures the reader to pull from source.
func (p *graphemeReader) SetSource(source io.ReaderAt) {
	p.source = source
	p.cursor = 0
	p.reader = bufio.NewReader(p)
	p.runeOffset = 0
}

// Read exists to satisfy io.Reader. It should not be directly invoked.
func (p *graphemeReader) Read(b []byte) (int, error) {
	n, err := p.source.ReadAt(b, p.cursor)
	p.cursor += int64(n)
	return n, err
}

// next decodes one paragraph of rune data.
func (p *graphemeReader) next() ([]rune, bool) {
	p.paragraph = p.paragraph[:0]
	var err error
	var r rune
	for err == nil {
		r, _, err = p.reader.ReadRune()
		if err != nil {
			break
		}
		p.paragraph = append(p.paragraph, r)
		if r == '\n' {
			break
		}
	}
	return p.paragraph, err == nil
}

// Graphemes will return the next paragraph's grapheme cluster boundaries,
// if any. If it returns an empty slice, there is no more data (all paragraphs
// have been segmented).
func (p *graphemeReader) Graphemes() []int {
	var more bool
	p.graphemes = p.graphemes[:0]
	p.paragraph, more = p.next()
	if len(p.paragraph) == 0 && !more {
		return nil
	}
	p.Segmenter.Init(p.paragraph)
	iter := p.Segmenter.GraphemeIterator()
	if iter.Next() {
		graph := iter.Grapheme()
		p.graphemes = append(p.graphemes,
			p.runeOffset+graph.Offset,
			p.runeOffset+graph.Offset+len(graph.Text),
		)
	}
	for iter.Next() {
		graph := iter.Grapheme()
		p.graphemes = append(p.graphemes, p.runeOffset+graph.Offset+len(graph.Text))
	}
	p.runeOffset += len(p.paragraph)
	return p.graphemes
}
