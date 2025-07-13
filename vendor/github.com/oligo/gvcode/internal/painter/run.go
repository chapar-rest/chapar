package painter

import (
	"image"

	"gioui.org/op"
	"gioui.org/text"
	lt "github.com/oligo/gvcode/internal/layout"
	"golang.org/x/image/math/fixed"
)

type UnderlineStyle struct {
	Color op.CallOp
}

type SquiggleStyle struct {
	Color op.CallOp
}

type StrikethroughStyle struct {
	Color op.CallOp
}

type BorderStyle struct {
	Color op.CallOp
}

// A RenderRun is a run of adjacent glyphs sharing the same style. This is
// used as the painting unit of the TextPainter.
type RenderRun struct {
	Glyphs []text.Glyph
	// Offset is the pixel offset relative to the start of the line.
	Offset fixed.Int26_6

	// Fg is the text color encoded to Gio ops. This should be left empty
	// for runs originated from decorations.
	Fg op.CallOp
	// Bg is the background color encoded to Gio ops.
	Bg op.CallOp
	// Underline style for the run.
	Underline     *UnderlineStyle
	Squiggle      *SquiggleStyle
	Strikethrough *StrikethroughStyle
	Border        *BorderStyle
}

// Bounds returns the bounding box relative to the dot of the first glyph of the run.
func (s *RenderRun) Bounds() image.Rectangle {
	if len(s.Glyphs) == 0 {
		return image.Rectangle{}
	}

	rect := fixed.Rectangle26_6{}
	for _, g := range s.Glyphs {
		rect.Min.Y = min(rect.Min.Y, -g.Ascent)
		rect.Max.Y = max(rect.Max.Y, g.Descent)
		rect.Max.X += g.Advance
	}

	return image.Rectangle{
		Min: image.Point{Y: rect.Min.Y.Floor()},
		Max: image.Point{
			X: rect.Max.X.Ceil(),
			Y: rect.Max.Y.Ceil(),
		},
	}
}

// Advance returns the width of the run in pixels.
func (s *RenderRun) Advance() fixed.Int26_6 {
	w := fixed.I(0)
	if len(s.Glyphs) == 0 {
		return w
	}

	for _, g := range s.Glyphs {
		w += g.Advance
	}

	return w
}

// size returns the number of glyphs this run covers.
func (s *RenderRun) Size() int {
	return len(s.Glyphs)
}

// LineSplitter defines the interface external styling source must implement
// to work with the painter. It used the Split api to split a line into a few
// RenderRuns.
type LineSplitter interface {
	// Split the line into runs and put the result in the runs array.
	Split(line *lt.Line, runs *[]RenderRun)
}
