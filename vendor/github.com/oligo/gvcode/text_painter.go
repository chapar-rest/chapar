package gvcode

import (
	"image"
	"sort"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
	lt "github.com/oligo/gvcode/internal/layout"

)

// A glyphSpan is a group of adjacent glyphs sharing the same fg and bg.
type glyphSpan struct {
	// glyphs is a range mapped into the line's glyphs.
	glyphs text.Range
	fg     op.CallOp
	bg     op.CallOp
	// offset is an visual offset relative to the start of the line.
	offset fixed.Int26_6
}

// textPainter computes the bounding box of and paints text.
type textPainter struct {
	// viewport is the rectangle of document coordinates that the painter is
	// trying to fill with text.
	viewport image.Rectangle

	// padding is the space needed outside of the bounds of the text to ensure no
	// part of a glyph is clipped.
	padding image.Rectangle
	// styles used to paint the text.
	styles []*TextStyle

	// the styling spans of the current line.
	spans []glyphSpan
}

// processGlyph checks whether the glyph is visible within the configured
// viewport and (if so) updates the text dimensions to include the glyph.
func (tp *textPainter) processGlyph(g text.Glyph) (visible bool) {
	// Compute the maximum extent to which glyphs overhang on the horizontal
	// axis.
	if d := g.Bounds.Min.X.Floor(); d < tp.padding.Min.X {
		// If the distance between the dot and the left edge of this glyph is
		// less than the current padding, increase the left padding.
		tp.padding.Min.X = d
	}
	if d := (g.Bounds.Max.X - g.Advance).Ceil(); d > tp.padding.Max.X {
		// If the distance between the dot and the right edge of this glyph
		// minus the logical advance of this glyph is greater than the current
		// padding, increase the right padding.
		tp.padding.Max.X = d
	}
	if d := (g.Bounds.Min.Y + g.Ascent).Floor(); d < tp.padding.Min.Y {
		// If the distance between the dot and the top of this glyph is greater
		// than the ascent of the glyph, increase the top padding.
		tp.padding.Min.Y = d
	}
	if d := (g.Bounds.Max.Y - g.Descent).Ceil(); d > tp.padding.Max.Y {
		// If the distance between the dot and the bottom of this glyph is greater
		// than the descent of the glyph, increase the bottom padding.
		tp.padding.Max.Y = d
	}
	logicalBounds := image.Rectangle{
		Min: image.Pt(g.X.Floor(), int(g.Y)-g.Ascent.Ceil()),
		Max: image.Pt((g.X + g.Advance).Ceil(), int(g.Y)+g.Descent.Ceil()),
	}

	above := logicalBounds.Max.Y < tp.viewport.Min.Y
	below := logicalBounds.Min.Y > tp.viewport.Max.Y
	left := logicalBounds.Max.X < tp.viewport.Min.X
	right := logicalBounds.Min.X > tp.viewport.Max.X

	return !above && !below && !left && !right
}

func (tp *textPainter) paintLine(gtx layout.Context, shaper *text.Shaper, line *lt.Line, defaultMaterial op.CallOp) {
	if len(line.Glyphs) <= 0 {
		return
	}

	lineOff := f32.Point{X: fixedToFloat(line.XOff), Y: float32(line.YOff)}.Sub(layout.FPt(tp.viewport.Min))
	t := op.Affine(f32.Affine2D{}.Offset(lineOff)).Push(gtx.Ops)

	tp.stylingLine(line, defaultMaterial)

	for _, span := range tp.spans {
		spanOffset := op.Affine(f32.Affine2D{}.Offset(f32.Point{X: float32(span.offset.Round())})).Push(gtx.Ops)

		glyphs := line.GetGlyphs(span.glyphs.Offset, span.glyphs.Count)
		// draw background
		if span.bg != (op.CallOp{}) {
			rect := span.bounds(line)
			bgClip := clip.Rect(rect).Push(gtx.Ops)
			span.bg.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			bgClip.Pop()
		}

		// draw glyph
		path := shaper.Shape(glyphs)
		outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)
		span.fg.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		outline.Pop()
		if call := shaper.Bitmaps(glyphs); call != (op.CallOp{}) {
			call.Add(gtx.Ops)
		}
		spanOffset.Pop()
	}

	t.Pop()
}

// stylingLine determines the style for each of the glyphs of the line using TextStyles.
// Text styles should be sorted by rune range in ascending order.
func (tp *textPainter) stylingLine(line *lt.Line, defaultMaterial op.CallOp) {
	tp.spans = tp.spans[:0]

	idx := sort.Search(len(tp.styles), func(i int) bool {
		s := tp.styles[i]
		return s.End > line.RuneOff
	})

	if idx == len(tp.styles) {
		span := glyphSpan{
			glyphs: text.Range{Count: len(line.Glyphs), Offset: 0},
			fg:     defaultMaterial,
			offset: 0,
		}

		tp.spans = append(tp.spans, span)
		return
	}

	style := tp.styles[idx]
	span := glyphSpan{}

	var fg, bg op.CallOp
	runeOff := line.RuneOff
	advance := fixed.I(0)

	for glyphIdx, g := range line.Glyphs {
		if style != nil && style.Start <= runeOff && style.End > runeOff {
			fg = style.Color
			bg = style.Background
		} else {
			fg = defaultMaterial
			bg = op.CallOp{}
		}

		if span.fg != fg {
			if span.glyphs.Count > 0 {
				if span.fg == (op.CallOp{}) {
					span.fg = defaultMaterial
				}
				tp.spans = append(tp.spans, span)
			}
			span = glyphSpan{}
			span.offset = advance
			span.glyphs.Offset = glyphIdx
			span.fg = fg
			span.bg = bg
		}

		span.glyphs.Count += 1
		runeOff += int(g.Runes)
		advance += g.Advance

		if style != nil && runeOff >= style.End {
			idx++
			if idx < len(tp.styles) {
				style = tp.styles[idx]
			} else {
				style = nil
			}
		}

	}

	if span.glyphs.Count > 0 {
		if span.fg == (op.CallOp{}) {
			span.fg = defaultMaterial
		}
		tp.spans = append(tp.spans, span)
	}
}

// bounds returns the bounding box relative to the dot of the first
// glyph of the span.
func (s *glyphSpan) bounds(line *lt.Line) image.Rectangle {
	rect := image.Rectangle{}

	if s.glyphs.Count <= 0 {
		return rect
	}

	for _, g := range line.Glyphs[s.glyphs.Offset : s.glyphs.Offset+s.glyphs.Count] {
		rect.Min.Y = min(rect.Min.Y, -g.Ascent.Round())
		rect.Max.Y = max(rect.Max.Y, g.Descent.Round())
		rect.Max.X += g.Advance.Round()
	}

	return rect
}
