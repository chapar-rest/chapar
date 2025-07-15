package decoration

import (
	"cmp"
	"errors"
	"slices"

	"gioui.org/text"
	"github.com/oligo/gvcode/internal/layout"
	"github.com/oligo/gvcode/internal/painter"
	"golang.org/x/image/math/fixed"
)

// lineSplitter split line into RenderRun on behalf of the TextPainter.
type decorationLineSplitter struct {
	current painter.RenderRun
	// the last unread rune offset while iterating through the line.
	runeOff int
	// glyphOff is the last unread glyph offset.
	glyphOff int
	// the last advance offset while iterating through the line.
	advance fixed.Int26_6
}

// func (rb *decorationLineSplitter) setup(line *layout.Line) {
// 	//lineIter := line.All()
// 	//rb.nextGlyph, rb.stopFunc = iter.Pull(lineIter)
// 	rb.current = painter.RenderRun{}
// 	rb.advance = fixed.I(0)
// }

func (rb *decorationLineSplitter) commitLast(runs *[]painter.RenderRun) {
	if rb.current.Size() > 0 {
		*runs = append(*runs, rb.current)
		rb.current = painter.RenderRun{}
	}
}

// Split split line of glyphs into multiple runs for TextPainter to use.
//
// Unlike splitter for syntax tokens, it only cares about the glyphs its range
// covers, thus generating sparse runs. Another difference is that decoration
// generated runs also have glpyhs, but they are not painted, the TextPainter
// is only interested in the styling fields. We may ommit the glyphs in the
// future to save memory if the various metrics the painter needed are stored
// in the RenderRun.
func (rb *decorationLineSplitter) Split(line *layout.Line, decorations *DecorationTree, runs *[]painter.RenderRun) {
	*runs = (*runs)[:0]
	rb.runeOff = line.RuneOff
	rb.current = painter.RenderRun{}
	rb.advance = fixed.I(0)
	rb.glyphOff = 0

	tokens := decorations.QueryRange(line.RuneOff, line.RuneOff+line.Runes)
	if len(tokens) == 0 {
		return
	}

	// sort according to the priority to determine the painting order.
	slices.SortFunc(tokens, func(a, b Decoration) int { return cmp.Compare(a.Priority, b.Priority) })

	for _, token := range tokens {
		// Need to adjust range here as decoration may cross multiple lines.
		start := max(token.Start, line.RuneOff)
		end := min(token.End, line.RuneOff+line.Runes)

		err := rb.readToRun(line, start, end)
		if err != nil {
			panic("read glyph falied: " + err.Error())
		}

		if rb.current.Size() > 0 {
			if token.Background != nil && token.Background.Color.IsSet() {
				rb.current.Bg = token.Background.Color.Op(nil)
			}

			if token.Underline != nil {
				rb.current.Underline = &painter.UnderlineStyle{Color: token.Underline.Color.Op(nil)}
			}

			if token.Squiggle != nil {
				rb.current.Squiggle = &painter.SquiggleStyle{Color: token.Squiggle.Color.Op(nil)}
			}

			if token.Strikethrough != nil {
				rb.current.Strikethrough = &painter.StrikethroughStyle{Color: token.Strikethrough.Color.Op(nil)}
			}

			if token.Border != nil {
				rb.current.Border = &painter.BorderStyle{Color: token.Border.Color.Op(nil)}
			}

			rb.commitLast(runs)
		}
	}

}

func (rb *decorationLineSplitter) readToRun(line *layout.Line, start, end int) error {
	if rb.runeOff > start {
		// start reading from the begining.
		rb.runeOff = line.RuneOff
		rb.advance = 0
		rb.glyphOff = 0
	}

	// or maybe we can reuse exiting offset.
	for rb.runeOff < start {
		_, err := rb.readGlyph(line)
		if err != nil {
			return err
		}
	}

	if rb.runeOff == start {
		// ready to read the data we are interested in.
		rb.current.Offset = rb.advance
		for rb.runeOff < end {
			gl, err := rb.readGlyph(line)
			if err != nil {
				break
			}
			rb.current.Glyphs = append(rb.current.Glyphs, *gl)
		}
	}

	return nil
}

func (rb *decorationLineSplitter) readGlyph(line *layout.Line) (*text.Glyph, error) {
	if rb.glyphOff < len(line.Glyphs) {
		gl := line.Glyphs[rb.glyphOff]
		if gl == nil {
			return nil, errors.New("read glyph failed")
		}

		rb.advance += gl.Advance
		rb.runeOff += int(gl.Runes)
		rb.glyphOff++
		return gl, nil
	}

	return nil, errors.New("invalid glyphOff")
}
