package syntax

import (
	"iter"

	"gioui.org/text"
	"github.com/oligo/gvcode/internal/layout"
	"github.com/oligo/gvcode/internal/painter"
	"golang.org/x/image/math/fixed"
)

// lineSplitter split line into RenderRun on behalf of the TextPainter.
type lineSplitter struct {
	current painter.RenderRun
	// the rune offset while iterating through the line.
	runeOff int
	// the advance offset while iterating through the line.
	advance   fixed.Int26_6
	nextGlyph func() (text.Glyph, bool)
	stopFunc  func()
}

func (rb *lineSplitter) setup(line *layout.Line) {
	lineIter := line.All()
	rb.nextGlyph, rb.stopFunc = iter.Pull(lineIter)
	rb.current = painter.RenderRun{}
	rb.advance = fixed.I(0)
}

func (rb *lineSplitter) commitLast(runs *[]painter.RenderRun) {
	if rb.current.Size() > 0 {
		*runs = append(*runs, rb.current)
		rb.current = painter.RenderRun{
			Offset: rb.advance,
		}
	}
}

func (rb *lineSplitter) Split(line *layout.Line, textTokens *TextTokens, runs *[]painter.RenderRun) {
	*runs = (*runs)[:0]
	rb.runeOff = line.RuneOff

	tokens := textTokens.QueryRange(line.RuneOff, line.RuneOff+line.Runes)
	if len(tokens) == 0 {
		run := painter.RenderRun{
			Glyphs: line.GetGlyphs(0, len(line.Glyphs)),
			Offset: 0,
		}

		*runs = append(*runs, run)
		return
	}

	rb.setup(line)
	defer rb.stopFunc()

	for _, token := range tokens {
		// check if there is any glyphs not covered by the token and put them in
		// one run.
		rb.readUntil(token.Start)
		if rb.current.Size() > 0 {
			// no style
			rb.commitLast(runs)
		}

		// next read the entire token range to the current run.
		rb.readUntil(token.End)
		if rb.current.Size() > 0 {
			if token.Style == 0 {
				continue
			}

			fg := textTokens.GetColor(token.Style.Foreground())
			bg := textTokens.GetColor(token.Style.Background())
			if fg.IsSet() {
				rb.current.Fg = fg.Op(nil)
			}
			if bg.IsSet() {
				rb.current.Bg = bg.Op(nil)
			}

			textStyle := token.Style.TextStyle()
			if textStyle.HasStyle(Underline) {
				rb.current.Underline = &painter.UnderlineStyle{}
			}
			if textStyle.HasStyle(Squiggle) {
				rb.current.Squiggle = &painter.SquiggleStyle{}
			}
			if textStyle.HasStyle(Strikethrough) {
				rb.current.Strikethrough = &painter.StrikethroughStyle{}
			}
			if textStyle.HasStyle(Border) {
				rb.current.Border = &painter.BorderStyle{}
			}

			rb.commitLast(runs)
		}
	}

	// check if there is any glyphs left over and put them in one run.
	rb.readUntil(line.RuneOff + line.Runes)
	if rb.current.Size() > 0 {
		// no style
		rb.commitLast(runs)
	}

}

func (rb *lineSplitter) readUntil(runeOff int) {
	for rb.runeOff < runeOff {
		g, ok := rb.nextGlyph()
		if !ok {
			break
		}
		rb.advance += g.Advance
		rb.current.Glyphs = append(rb.current.Glyphs, g)
		rb.runeOff += int(g.Runes)
	}
}
