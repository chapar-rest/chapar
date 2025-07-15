package layout

import (
	"fmt"
	"image"
	"iter"

	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

// Line contains various metrics of a line of text.
type Line struct {
	XOff    fixed.Int26_6
	YOff    int
	Width   fixed.Int26_6
	Ascent  fixed.Int26_6
	Descent fixed.Int26_6
	Glyphs  []*text.Glyph
	// runes is the number of runes represented by this line.
	Runes int
	// runeOff tracks the rune offset of the first rune of the line in the document.
	RuneOff int
}

func (li Line) String() string {
	return fmt.Sprintf("[line] xOff: %d, yOff: %d, width: %d, runes: %d, runeOff: %d, glyphs: %d",
		li.XOff.Round(), li.YOff, li.Width.Ceil(), li.Runes, li.RuneOff, len(li.Glyphs))
}

func (li *Line) append(glyphs ...text.Glyph) {
	for _, gl := range glyphs {
		li.YOff = int(gl.Y)
		if li.XOff > gl.X {
			li.XOff = gl.X
		}

		li.Width += gl.Advance
		// glyph ascent and descent are derived from the line ascent and descent,
		// so it is safe to just set them as the line's ascent and descent.
		li.Ascent = gl.Ascent
		li.Descent = gl.Descent
		li.Runes += int(gl.Runes)
		li.Glyphs = append(li.Glyphs, &gl)
	}

}

// recompute re-compute xOff of the line and each glyph contained in the line,
// and also update the runeOff to the right value.
func (li *Line) recompute(alignOff fixed.Int26_6, runeOff int) {
	xOff := fixed.I(0)
	for idx, gl := range li.Glyphs {
		gl.X = alignOff + fixed.Int26_6(xOff)
		if idx == len(li.Glyphs)-1 {
			gl.Flags |= text.FlagLineBreak
		}

		xOff += gl.Advance
	}

	li.RuneOff = runeOff
}

func (li *Line) adjustYOff(yOff int) {
	li.YOff = yOff
	for _, gl := range li.Glyphs {
		gl.Y = int32(yOff)
	}
}

func (li *Line) bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(li.XOff.Floor(), li.YOff-li.Ascent.Ceil()),
		Max: image.Pt((li.XOff + li.Width).Ceil(), li.YOff+li.Descent.Ceil()),
	}
}

func (li *Line) GetGlyphs(offset, count int) []text.Glyph {
	if count <= 0 {
		return []text.Glyph{}
	}

	out := make([]text.Glyph, count)
	for idx, gl := range li.Glyphs[offset : offset+count] {
		out[idx] = *gl
	}

	return out
}

func (li *Line) All() iter.Seq[text.Glyph] {
	return func(yield func(text.Glyph) bool) {
		for _, gl := range li.Glyphs {
			if !yield(*gl) {
				return
			}
		}
	}
}

// Paragraph contains the pixel coordinates of the start and end position
// of the paragraph.
type Paragraph struct {
	StartX fixed.Int26_6
	StartY int
	EndX   fixed.Int26_6
	EndY   int
	// Runes is the number of runes represented by this paragraph.
	Runes int
	// RuneOff tracks the rune offset of the first rune of the paragraph in the document.
	RuneOff int
}

// Add add a visual line to the paragraph, returning a boolean value indicating
// the end of a paragraph.
func (p *Paragraph) Add(li *Line) bool {
	lastGlyph := li.Glyphs[len(li.Glyphs)-1]

	if p.Runes == 0 {
		start := li.Glyphs[0]
		p.StartX = start.X
		p.StartY = int(start.Y)

		p.EndX = lastGlyph.X
		p.EndY = int(lastGlyph.Y)

		p.RuneOff = li.RuneOff
	} else {
		p.EndX = lastGlyph.X
		p.EndY = int(lastGlyph.Y)
	}

	p.Runes += li.Runes
	return lastGlyph.Flags&text.FlagParagraphBreak != 0
}

type glyphIter struct {
	shaper *text.Shaper
}

func (gi glyphIter) All() iter.Seq[text.Glyph] {
	return func(yield func(text.Glyph) bool) {
		for {
			g, ok := gi.shaper.NextGlyph()
			if !ok {
				return
			}

			if !yield(g) {
				return
			}
		}
	}
}
