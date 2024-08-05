// SPDX-License-Identifier: Unlicense OR MIT

package editor

import (
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"

	"golang.org/x/image/math/fixed"
)

type TextStyle struct {
	Line int
	// offset of start in the text
	Start int
	// offset of end in the text
	End        int
	Color      op.CallOp
	Background op.CallOp
}

type glyphStyle struct {
	g  text.Glyph
	fg op.CallOp
	bg op.CallOp
}

// A glyphSpan is a group of adjacent glyphs sharing the same fg and bg.
type glyphSpan struct {
	glyphs []text.Glyph
	fg     op.CallOp
	bg     op.CallOp
	offset float32
}

// textIterator computes the bounding box of and paints text.
type textIterator struct {
	// viewport is the rectangle of document coordinates that the iterator is
	// trying to fill with text.
	viewport image.Rectangle
	// maxLines is the maximum number of text lines that should be displayed.
	maxLines int

	// truncated tracks the count of truncated runes in the text.
	truncated int
	// linesSeen tracks the quantity of line endings this iterator has seen.
	linesSeen int
	// lineOff tracks the origin for the glyphs in the current line.
	lineOff f32.Point
	// padding is the space needed outside of the bounds of the text to ensure no
	// part of a glyph is clipped.
	padding image.Rectangle
	// bounds is the logical bounding box of the text.
	bounds image.Rectangle
	// visible tracks whether the most recently iterated glyph is visible within
	// the viewport.
	visible bool
	// first tracks whether the iterator has processed a glyph yet.
	first bool
	// baseline tracks the location of the first line of text's baseline.
	baseline int
}

// processGlyph checks whether the glyph is visible within the iterator's configured
// viewport and (if so) updates the iterator's text dimensions to include the glyph.
func (it *textIterator) processGlyph(g text.Glyph, ok bool) (visibleOrBefore bool) {
	if it.maxLines > 0 {
		if g.Flags&text.FlagTruncator != 0 && g.Flags&text.FlagClusterBreak != 0 {
			// A glyph carrying both of these flags provides the count of truncated runes.
			it.truncated = int(g.Runes)
		}
		if g.Flags&text.FlagLineBreak != 0 {
			it.linesSeen++
		}
		if it.linesSeen == it.maxLines && g.Flags&text.FlagParagraphBreak != 0 {
			return false
		}
	}
	// Compute the maximum extent to which glyphs overhang on the horizontal
	// axis.
	if d := g.Bounds.Min.X.Floor(); d < it.padding.Min.X {
		// If the distance between the dot and the left edge of this glyph is
		// less than the current padding, increase the left padding.
		it.padding.Min.X = d
	}
	if d := (g.Bounds.Max.X - g.Advance).Ceil(); d > it.padding.Max.X {
		// If the distance between the dot and the right edge of this glyph
		// minus the logical advance of this glyph is greater than the current
		// padding, increase the right padding.
		it.padding.Max.X = d
	}
	if d := (g.Bounds.Min.Y + g.Ascent).Floor(); d < it.padding.Min.Y {
		// If the distance between the dot and the top of this glyph is greater
		// than the ascent of the glyph, increase the top padding.
		it.padding.Min.Y = d
	}
	if d := (g.Bounds.Max.Y - g.Descent).Ceil(); d > it.padding.Max.Y {
		// If the distance between the dot and the bottom of this glyph is greater
		// than the descent of the glyph, increase the bottom padding.
		it.padding.Max.Y = d
	}
	logicalBounds := image.Rectangle{
		Min: image.Pt(g.X.Floor(), int(g.Y)-g.Ascent.Ceil()),
		Max: image.Pt((g.X + g.Advance).Ceil(), int(g.Y)+g.Descent.Ceil()),
	}
	if !it.first {
		it.first = true
		it.baseline = int(g.Y)
		it.bounds = logicalBounds
	}

	above := logicalBounds.Max.Y < it.viewport.Min.Y
	below := logicalBounds.Min.Y > it.viewport.Max.Y
	left := logicalBounds.Max.X < it.viewport.Min.X
	right := logicalBounds.Min.X > it.viewport.Max.X
	it.visible = !above && !below && !left && !right
	if it.visible {
		it.bounds.Min.X = min(it.bounds.Min.X, logicalBounds.Min.X)
		it.bounds.Min.Y = min(it.bounds.Min.Y, logicalBounds.Min.Y)
		it.bounds.Max.X = max(it.bounds.Max.X, logicalBounds.Max.X)
		it.bounds.Max.Y = max(it.bounds.Max.Y, logicalBounds.Max.Y)
	}
	return ok && !below
}

func fixedToFloat(i fixed.Int26_6) float32 {
	return float32(i) / 64.0
}

// paintGlyph buffers up and paints text glyphs. It should be invoked iteratively upon each glyph
// until it returns false. The line parameter should be a slice with
// a backing array of sufficient size to buffer multiple glyphs.
// A modified slice will be returned with each invocation, and is
// expected to be passed back in on the following invocation.
// This design is awkward, but prevents the line slice from escaping
// to the heap.
func (it *textIterator) paintGlyph(gtx layout.Context, shaper *text.Shaper, glyph glyphStyle, line []glyphStyle) ([]glyphStyle, bool) {
	visibleOrBefore := it.processGlyph(glyph.g, true)
	if it.visible {
		if len(line) == 0 {
			it.lineOff = f32.Point{X: fixedToFloat(glyph.g.X), Y: float32(glyph.g.Y)}.Sub(layout.FPt(it.viewport.Min))
		}
		line = append(line, glyph)
	}
	if glyph.g.Flags&text.FlagLineBreak != 0 || cap(line)-len(line) == 0 || !visibleOrBefore {
		t := op.Affine(f32.Affine2D{}.Offset(it.lineOff)).Push(gtx.Ops)
		//var glyphLine []text.Glyph
		spans := it.groupGlyphs(line)

		for _, span := range spans {
			glyphOffset := op.Affine(f32.Affine2D{}.Offset(f32.Point{X: span.offset})).Push(gtx.Ops)

			// draw background
			if span.bg != (op.CallOp{}) {
				rect := span.calculateBgRect()
				bgClip := rect.Push(gtx.Ops)
				span.bg.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				bgClip.Pop()
			}

			// draw glyph
			path := shaper.Shape(span.glyphs)
			outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)
			span.fg.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			outline.Pop()
			if call := shaper.Bitmaps(span.glyphs); call != (op.CallOp{}) {
				call.Add(gtx.Ops)
			}
			glyphOffset.Pop()
		}

		t.Pop()
		line = line[:0]
	}
	return line, visibleOrBefore
}

func (it *textIterator) groupGlyphs(line []glyphStyle) []*glyphSpan {
	spans := make([]*glyphSpan, 0)
	idx := 0
	if len(line) > 0 {
		spans = append(spans, &glyphSpan{})
	}

	for _, s := range line {
		span := spans[idx]
		if len(span.glyphs) <= 0 {
			span.addFirstGlyph(s, float32(it.viewport.Min.X)+it.lineOff.X)
			continue
		}

		if span.fg == s.fg && span.bg == s.bg {
			span.glyphs = append(span.glyphs, s.g)
			continue
		} else {
			idx++
			spans = append(spans, &glyphSpan{})
			span := spans[idx]
			span.addFirstGlyph(s, float32(it.viewport.Min.X)+it.lineOff.X)
		}
	}

	//log.Printf("line is splitted into %d spans", len(spans))
	return spans
}

func (span *glyphSpan) addFirstGlyph(s glyphStyle, lineOff float32) {
	span.glyphs = append(span.glyphs, s.g)
	span.fg = s.fg
	span.bg = s.bg
	// offset is where the first glyph character starts
	// thanks to setting an offset, the rectangle and the glyph can be drawn from X: 0
	span.offset = fixedToFloat(s.g.X) - lineOff
}

func (span *glyphSpan) calculateBgRect() clip.Rect {
	if len(span.glyphs) <= 0 {
		return clip.Rect{}
	}

	var minY, maxX, maxY int
	maxX = 0
	for _, g := range span.glyphs {
		y := 0 - g.Ascent.Ceil()
		if minY > y {
			minY = y
		}

		my := 0 + g.Descent.Ceil()
		if maxY < my {
			maxY = my
		}

		maxX += g.Advance.Ceil()
	}

	return clip.Rect{
		Min: image.Point{X: 0, Y: minY},
		Max: image.Point{X: maxX, Y: maxY},
	}
}

func toGlyphStyle(g text.Glyph, start int, detaultMaterial op.CallOp, styles []*TextStyle) glyphStyle {
	var style *TextStyle
	for _, s := range styles {
		if start >= s.Start && start < s.End {
			style = s
			break
		}
	}

	gs := glyphStyle{g: g}

	if style == nil {
		gs.fg = detaultMaterial
		return gs
	}

	gs.fg = style.Color
	gs.bg = style.Background

	if style.Color == (op.CallOp{}) {
		gs.fg = detaultMaterial
	}

	return gs
}
