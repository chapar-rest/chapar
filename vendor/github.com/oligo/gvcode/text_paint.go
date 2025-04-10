package gvcode

import (
	"image"
	"math"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	lt "github.com/oligo/gvcode/internal/layout"
)

// calculateViewSize determines the size of the current visible content,
// ensuring that even if there is no text content, some space is reserved
// for the caret.
func (e *textView) calculateViewSize(gtx layout.Context) image.Point {
	base := e.dims.Size
	if caretWidth := gtx.Dp(e.CaretWidth); base.X < caretWidth {
		base.X = caretWidth
	}
	return gtx.Constraints.Constrain(base)
}

func (e *textView) layoutText(shaper *text.Shaper) {
	//e.layoutByParagraph(shaper, &it)
	e.dims = e.layouter.Layout(shaper, &e.params, e.TabWidth, e.WrapLine)
}

// PaintText clips and paints the visible text glyph outlines using the provided
// material to fill the glyphs.
func (e *textView) PaintText(gtx layout.Context, material op.CallOp, textStyles []*TextStyle) {
	m := op.Record(gtx.Ops)
	viewport := image.Rectangle{
		Min: e.scrollOff,
		Max: e.viewSize.Add(e.scrollOff),
	}

	tp := textPainter{
		viewport: viewport,
		styles:   textStyles,
	}

	for _, line := range e.layouter.Lines {
		if line.Descent.Ceil()+line.YOff < viewport.Min.Y {
			continue
		}
		if line.YOff-line.Ascent.Floor() > viewport.Max.Y {
			break
		}

		tp.paintLine(gtx, e.shaper, line, material)
	}

	call := m.Stop()
	viewport.Min = viewport.Min.Add(tp.padding.Min)
	viewport.Max = viewport.Max.Add(tp.padding.Max)
	defer clip.Rect(viewport.Sub(e.scrollOff)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

// PaintSelection clips and paints the visible text selection rectangles using
// the provided material to fill the rectangles.
func (e *textView) PaintSelection(gtx layout.Context, material op.CallOp) {
	localViewport := image.Rectangle{Max: e.viewSize}
	docViewport := image.Rectangle{Max: e.viewSize}.Add(e.scrollOff)
	defer clip.Rect(localViewport).Push(gtx.Ops).Pop()
	e.regions = e.layouter.Locate(docViewport, e.caret.start, e.caret.end, e.regions)
	//log.Println("regions count: ", len(e.regions), e.regions)
	expandEmptyRegion := len(e.regions) > 1
	for _, region := range e.regions {
		bounds := e.adjustPadding(region.Bounds)
		if expandEmptyRegion && bounds.Dx() <= 0 {
			bounds.Max.X += gtx.Dp(unit.Dp(2))
		}
		area := clip.Rect(bounds).Push(gtx.Ops)
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		area.Pop()
	}
}

// paintRegions clips and paints the visible text rectangles using
// the provided material to fill the rectangles. Regions passed in should be constrained
// in the viewport.
func (e *textView) PaintRegions(gtx layout.Context, regions []Region, material op.CallOp) {
	localViewport := image.Rectangle{Max: e.viewSize}
	defer clip.Rect(localViewport).Push(gtx.Ops).Pop()
	for _, region := range regions {
		area := clip.Rect(e.adjustPadding(region.Bounds)).Push(gtx.Ops)
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		area.Pop()
	}
}

func (e *textView) PaintOverlay(gtx layout.Context, offset image.Point, overlay layout.Widget) {
	viewport := image.Rectangle{
		Min: e.scrollOff,
		Max: e.viewSize.Add(e.scrollOff),
	}

	macro := op.Record(gtx.Ops)
	dims := overlay(gtx)
	call := macro.Stop()

	if offset.X+dims.Size.X-e.scrollOff.X > gtx.Constraints.Max.X {
		offset.X = max(offset.X-dims.Size.X, 0)
	}

	padding := e.adjustDescentPadding()
	if offset.Y+dims.Size.Y+padding-e.scrollOff.Y > gtx.Constraints.Max.Y {
		offset.Y = max(offset.Y-dims.Size.Y-int(e.lineHeight.Ceil())+padding, 0)
	} else {
		offset.Y += padding
	}

	defer op.Offset(offset.Sub(e.scrollOff)).Push(gtx.Ops).Pop()
	defer clip.Rect(viewport.Sub(e.scrollOff)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

func (e *textView) highlightMatchingBrackets(gtx layout.Context, material op.CallOp) {
	left, right := e.bracketHandler.NearestMatchingBrackets()
	if left < 0 || right < 0 {
		// no matching found
		return
	}
	localViewport := image.Rectangle{Max: e.viewSize}
	docViewport := image.Rectangle{Max: e.viewSize}.Add(e.scrollOff)
	leftRegion := e.layouter.Locate(docViewport, left, left+1, nil)
	rightRegion := e.layouter.Locate(docViewport, right, right+1, nil)

	e.regions = e.regions[:0]
	e.regions = append(e.regions, leftRegion...)
	e.regions = append(e.regions, rightRegion...)

	defer clip.Rect(localViewport).Push(gtx.Ops).Pop()
	for _, region := range e.regions {
		area := clip.Rect(e.adjustPadding(region.Bounds))
		stack := area.Push(gtx.Ops)
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		stack.Pop()

		stroke := clip.Stroke{
			Path:  area.Path(),
			Width: float32(gtx.Dp(unit.Dp(1))),
		}.Op().Push(gtx.Ops)
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		stroke.Pop()
	}
}

// caretCurrentLine returns the current paragraph that the carent is in.
// Only the start position is checked.
func (e *textView) caretCurrentLine() (start lt.CombinedPos, end lt.CombinedPos) {
	caretStart := e.closestToRune(e.caret.start)
	lines := e.selectedParagraphs()
	if len(lines) == 0 {
		return caretStart, caretStart
	}

	line := lines[0]
	start = e.closestToXY(line.StartX, line.StartY)
	end = e.closestToXY(line.EndX, line.EndY)

	return
}

// paintLineHighlight clips and paints the visible line that the caret is in when there is no
// text selected.
func (e *textView) paintLineHighlight(gtx layout.Context, material op.CallOp) {
	if e.caret.start != e.caret.end {
		return
	}

	start, end := e.caretCurrentLine()
	if start == (lt.CombinedPos{}) || end == (lt.CombinedPos{}) {
		return
	}

	bounds := image.Rectangle{Min: image.Point{X: 0, Y: start.Y - start.Ascent.Ceil()},
		Max: image.Point{X: gtx.Constraints.Max.X, Y: end.Y + end.Descent.Ceil()}}.Sub(e.scrollOff)

	area := clip.Rect(e.adjustPadding(bounds)).Push(gtx.Ops)
	material.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	area.Pop()
}

func (e *textView) PaintLineNumber(gtx layout.Context, lt *text.Shaper, material op.CallOp) layout.Dimensions {
	m := op.Record(gtx.Ops)
	viewport := image.Rectangle{
		Min: e.scrollOff,
		Max: e.viewSize.Add(e.scrollOff),
	}

	dims := paintLineNumber(gtx, lt, e.params, viewport, e.layouter.Paragraphs, material)
	call := m.Stop()

	rect := viewport.Sub(e.scrollOff)
	rect.Max.X = dims.Size.X
	defer clip.Rect(rect).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)

	return dims
}

// PaintCaret clips and paints the caret rectangle, adding material immediately
// before painting to set the appropriate paint material.
func (e *textView) PaintCaret(gtx layout.Context, material op.CallOp) {
	carWidth2 := gtx.Dp(e.CaretWidth)
	caretPos, carAsc, carDesc := e.CaretInfo()

	carRect := image.Rectangle{
		Min: caretPos.Sub(image.Pt(carWidth2, carAsc)),
		Max: caretPos.Add(image.Pt(carWidth2, carDesc)),
	}
	cl := image.Rectangle{Max: e.viewSize}
	carRect = cl.Intersect(carRect)
	if !carRect.Empty() {
		defer clip.Rect(e.adjustPadding(carRect)).Push(gtx.Ops).Pop()
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}
}

func (e *textView) CaretInfo() (pos image.Point, ascent, descent int) {
	caretStart := e.closestToRune(e.caret.start)

	ascent = caretStart.Ascent.Ceil()
	descent = caretStart.Descent.Ceil()

	pos = image.Point{
		X: caretStart.X.Round(),
		Y: caretStart.Y,
	}
	pos = pos.Sub(e.scrollOff)
	return
}

// adjustPadding adjusts the vertical padding of a bounding box around the texts.
// This improves the visual effects of selected texts, or any other texts to be highlighted.
func (e *textView) adjustPadding(bounds image.Rectangle) image.Rectangle {
	if e.lineHeight <= 0 {
		e.lineHeight = e.calcLineHeight()
	}

	if e.lineHeight.Ceil() <= bounds.Dy() {
		return bounds
	}

	leading := e.lineHeight.Ceil() - bounds.Dy()
	adjust := int(math.Round(float64(float32(leading) / 2.0)))

	bounds.Min.Y -= adjust
	bounds.Max.Y += leading - adjust
	return bounds
}

func (e *textView) adjustDescentPadding() int {
	caretStart := e.closestToRune(e.caret.start)
	height := caretStart.Ascent + caretStart.Descent

	if e.lineHeight <= 0 {
		e.lineHeight = e.calcLineHeight()
	}

	if e.lineHeight.Ceil() <= height.Ceil() {
		return 0
	}

	leading := (e.lineHeight - height).Round()
	return int(math.Round(float64(float32(leading) / 2.0)))
}
