package painter

import (
	"image"
	"reflect"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	lt "github.com/oligo/gvcode/internal/layout"

	"golang.org/x/image/math/fixed"
)

// TextPainter computes the bounding box of and paints text.
type TextPainter struct {
	// viewport is the rectangle of document coordinates that the painter is
	// trying to fill with text.
	viewport  image.Rectangle
	scrollOff image.Point
	// padding is the space needed outside of the bounds of the text to ensure no
	// part of a glyph is clipped.
	padding image.Rectangle
	// runBuffer buffers runs of a line. This is passed down to the splitter to
	// decrease allocations. It works as we are rendering line by line.
	runBuffer []RenderRun
}

func (tp *TextPainter) SetViewport(viewport image.Rectangle, scrollOff image.Point) {
	tp.viewport = viewport
	tp.scrollOff = scrollOff
}

// Paint paints text and various styles originated from syntax hignlighting or decorations.
func (tp *TextPainter) Paint(gtx layout.Context, shaper *text.Shaper, lines []*lt.Line, defaultColor op.CallOp,
	syntaxTokens LineSplitter, decorations LineSplitter) {
	m := op.Record(gtx.Ops)
	viewport := tp.viewport

	for _, line := range lines {
		if line.Descent.Ceil()+line.YOff < tp.viewport.Min.Y {
			continue
		}
		if line.YOff-line.Ascent.Floor() > tp.viewport.Max.Y {
			break
		}

		if len(line.Glyphs) <= 0 {
			continue
		}

		lineOff := f32.Point{
			X: fixedToFloat(line.XOff),
			Y: float32(line.YOff),
		}.Sub(layout.FPt(tp.viewport.Min))

		// draw text with syntax token styles first.
		tp.paintText(gtx, shaper, lineOff, line, defaultColor, syntaxTokens)
		// And then draw decorations.
		tp.paintDecorations(gtx, shaper, lineOff, line, defaultColor, decorations)
	}

	call := m.Stop()
	viewport.Min = viewport.Min.Add(tp.padding.Min)
	viewport.Max = viewport.Max.Add(tp.padding.Max)
	// clip to make it fit the viewport.
	defer clip.Rect(viewport.Sub(tp.scrollOff)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

func (tp *TextPainter) paintText(gtx layout.Context, shaper *text.Shaper, lineOff f32.Point, line *lt.Line,
	defaultMaterial op.CallOp, syntaxTokens LineSplitter) {
	// split the line into runs.
	if !isNil(syntaxTokens) {
		syntaxTokens.Split(line, &tp.runBuffer)
	} else {
		tp.runBuffer = tp.runBuffer[:0]
		tp.runBuffer = append(tp.runBuffer, RenderRun{
			Glyphs: line.GetGlyphs(0, len(line.Glyphs)),
			Offset: 0,
		})
	}

	tp.paintLine(gtx, shaper, lineOff, tp.runBuffer, defaultMaterial, false)
}

func (tp *TextPainter) paintDecorations(gtx layout.Context, shaper *text.Shaper, lineOff f32.Point, line *lt.Line,
	defaultMaterial op.CallOp, decorations LineSplitter) {
	if isNil(decorations) {
		return
	}
	decorations.Split(line, &tp.runBuffer)
	tp.paintLine(gtx, shaper, lineOff, tp.runBuffer, defaultMaterial, true)
}

func (tp *TextPainter) paintLine(gtx layout.Context, shaper *text.Shaper, lineOffset f32.Point, runs []RenderRun,
	defaultMaterial op.CallOp, noText bool) {
	// Let drawing begin at the offset of the entire line.
	defer op.Affine(f32.Affine2D{}.Offset(lineOffset)).Push(gtx.Ops).Pop()

	// Iterate through the runs to paint the text.
	for _, run := range runs {
		// paint at the run offset.
		spanOffset := op.Affine(f32.Affine2D{}.Offset(f32.Point{X: float32(run.Offset.Round())})).Push(gtx.Ops)

		// draw background
		if run.Bg != (op.CallOp{}) {
			rect := run.Bounds()
			bgClip := clip.Rect(rect).Push(gtx.Ops)
			run.Bg.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			bgClip.Pop()
		}

		if !noText {
			// draw glyph
			tp.drawText(gtx, shaper, &run, defaultMaterial)
		}

		// draw underline and other styles.
		if run.Underline != nil {
			tp.drawUnderline(gtx, &run, defaultMaterial)
		}
		if run.Strikethrough != nil {
			tp.drawStrikethrough(gtx, &run, defaultMaterial)
		}
		if run.Border != nil {
			tp.drawBorder(gtx, &run, defaultMaterial)
		}
		if run.Squiggle != nil {
			tp.drawSquiggle(gtx, &run, defaultMaterial)
		}

		spanOffset.Pop()
	}

}

func (tp *TextPainter) drawText(gtx layout.Context, shaper *text.Shaper, run *RenderRun, defaultMaterial op.CallOp) {
	// draw glyph
	path := shaper.Shape(run.Glyphs)
	outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)
	if run.Fg == (op.CallOp{}) {
		run.Fg = defaultMaterial
	}
	run.Fg.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	outline.Pop()
	if call := shaper.Bitmaps(run.Glyphs); call != (op.CallOp{}) {
		call.Add(gtx.Ops)
	}
}

func (tp *TextPainter) drawStroke(gtx layout.Context, path clip.PathSpec, material op.CallOp) {
	if material == (op.CallOp{}) {
		return
	}

	shape := clip.Stroke{
		Path:  path,
		Width: float32(gtx.Dp(unit.Dp(1))),
	}.Op()

	defer shape.Push(gtx.Ops).Pop()
	material.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func (tp *TextPainter) drawUnderline(gtx layout.Context, run *RenderRun, material op.CallOp) {
	descent := run.Glyphs[0].Descent
	path := clip.Path{}
	path.Begin(gtx.Ops)
	// No need to move in x axis as the outer code already set the x offset.
	path.Move(f32.Pt(0, fixedToFloat(descent)))

	width := fixedToFloat(run.Advance())
	path.Line(f32.Point{X: width})
	path.Close()

	if run.Underline.Color != (op.CallOp{}) {
		material = run.Underline.Color
	}

	tp.drawStroke(gtx, path.End(), material)
}

func (tp *TextPainter) drawStrikethrough(gtx layout.Context, run *RenderRun, material op.CallOp) {
	path := clip.Path{}
	path.Begin(gtx.Ops)

	ascent := run.Glyphs[0].Ascent
	descent := run.Glyphs[0].Descent

	deltaY := (ascent+descent)/2 - ascent
	path.Move(f32.Point{X: 0, Y: fixedToFloat(deltaY)})

	width := fixedToFloat(run.Advance())
	path.Line(f32.Point{X: width})
	path.Close()

	if run.Strikethrough.Color != (op.CallOp{}) {
		material = run.Strikethrough.Color
	}

	tp.drawStroke(gtx, path.End(), material)
}

func (tp *TextPainter) drawBorder(gtx layout.Context, run *RenderRun, material op.CallOp) {
	rect := clip.Rect(run.Bounds())
	if run.Border.Color != (op.CallOp{}) {
		material = run.Border.Color
	}

	tp.drawStroke(gtx, rect.Path(), material)
}

// drawSquiggleQuad draw a wavy line using quadratic Bézier curve.
//
// Some parameters:
//
// startPoint: (x, y) - Where the squiggle begins.
// endPoint: (x, y) - Where the squiggle ends.
// amplitude: How "tall" each wave is (distance from the center line).
// numWaves(or frequency): How many complete waves (up-down-up) you want.
// segmentsPerWave: How many curve segments make up one complete wave. Usually 2 (one up, one down)
// for quadratic.
//
// Calculate Points:
//
// Determine the total length of the squiggle (if horizontal, endPoint.x - startPoint.x).
// Divide the length by the number of segments to find the segmentLength.
// Iterate along the path, calculating control points and end points for each curve segment.
func (tp *TextPainter) drawSquiggle(gtx layout.Context, run *RenderRun, material op.CallOp) {
	path := clip.Path{}
	path.Begin(gtx.Ops)

	descent := run.Glyphs[0].Descent
	// calculate a amplitude based on the descent size.
	var amplitude fixed.Int26_6 = descent / 2
	// set the pen relative to the dot of the start glyph.
	startX := fixed.I(0)
	startY := descent
	numWaves := run.Advance() / amplitude.Mul(fixed.I(2))
	if numWaves <= 0 {
		path.End() // pop the macro from the stack.
		return
	}

	// Each wave has 2 segments (one up, one down)
	numSegments := numWaves * 2
	segmentWidth := run.Advance() / numSegments

	path.MoveTo(f32.Pt(fixedToFloat(startX), fixedToFloat(startY)))

	currentX := startX
	currentAmplitude := amplitude // Start with positive amplitude

	for range numSegments {
		nextX := currentX + segmentWidth
		controlX := currentX + segmentWidth/2.0
		controlY := startY + currentAmplitude // Control point is at the peak or trough

		path.QuadTo(f32.Pt(fixedToFloat(controlX), fixedToFloat(controlY)),
			f32.Pt(fixedToFloat(nextX), fixedToFloat(startY))) // start and end are equal.
		currentX = nextX
		// Alternate amplitude for next segment (up/down)
		currentAmplitude *= -1
	}

	//path.Close()
	if run.Squiggle.Color != (op.CallOp{}) {
		material = run.Squiggle.Color
	}

	tp.drawStroke(gtx, path.End(), material)
}

// processGlyph checks whether the glyph is visible within the configured
// viewport and (if so) updates the text dimensions to include the glyph.
func (tp *TextPainter) processGlyph(g text.Glyph) (visible bool) {
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

func fixedToFloat(i fixed.Int26_6) float32 {
	return float32(i) / 64.0
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}

	value := reflect.ValueOf(i)
	kind := value.Kind()

	if kind >= reflect.Chan && kind <= reflect.Struct && value.IsNil() {
		return true
	}

	return false
}
