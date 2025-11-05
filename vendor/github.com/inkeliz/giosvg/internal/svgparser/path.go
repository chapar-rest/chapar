package svgparser

import (
	"fmt"
	"gioui.org/f32"
	"strings"
)

// This file defines the basic path structure

// Operation groups the different SVG commands
type Operation interface {
	// SVG text representation of the command
	fmt.Stringer

	// add itself on the driver `d`, after aplying the transform `M`
	drawTo(d Drawer, M Matrix2D)
}

// OpMoveTo moves the current point.
type OpMoveTo f32.Point

// OpLineTo draws a line from the current point,
// and updates it.
type OpLineTo f32.Point

// OpQuadTo draws a quadratic Bezier curve from the current point,
// and updates it.
type OpQuadTo [2]f32.Point

// OpCubicTo draws a cubic Bezier curve from the current point,
// and updates it.
type OpCubicTo [3]f32.Point

// OpClose close the current path.
type OpClose struct{}

// starts a new path at the given point.
func (op OpMoveTo) drawTo(d Drawer, M Matrix2D) {
	d.Stop(false) // implicit close if currently in path.
	d.Start(M.TFixed(f32.Point(op)))
}

// draw a line
func (op OpLineTo) drawTo(d Drawer, M Matrix2D) {
	d.Line(M.TFixed(f32.Point(op)))
}

// draw a quadratic bezier curve
func (op OpQuadTo) drawTo(d Drawer, M Matrix2D) {
	d.QuadBezier(M.TFixed(op[0]), M.TFixed(op[1]))
}

// draw a cubic bezier curve
func (op OpCubicTo) drawTo(d Drawer, M Matrix2D) {
	d.CubeBezier(M.TFixed(op[0]), M.TFixed(op[1]), M.TFixed(op[2]))
}

func (op OpClose) drawTo(d Drawer, _ Matrix2D) {
	d.Stop(true)
}

func (op OpMoveTo) String() string {
	return fmt.Sprintf("M%4.3f,%4.3f", float32(op.X)/64, float32(op.Y)/64)
}

func (op OpLineTo) String() string {
	return fmt.Sprintf("L%4.3f,%4.3f", float32(op.X)/64, float32(op.Y)/64)
}

func (op OpQuadTo) String() string {
	return fmt.Sprintf("Q%4.3f,%4.3f,%4.3f,%4.3f", float32(op[0].X)/64, float32(op[0].Y)/64,
		float32(op[1].X)/64, float32(op[1].Y)/64)
}

func (op OpCubicTo) String() string {
	return "C" + fmt.Sprintf("C%4.3f,%4.3f,%4.3f,%4.3f,%4.3f,%4.3f", float32(op[0].X)/64, float32(op[0].Y)/64,
		float32(op[1].X)/64, float32(op[1].Y)/64, float32(op[2].X)/64, float32(op[2].Y)/64)
}

func (op OpClose) String() string {
	return "Z"
}

// Path describes a sequence of basic SVG operations, which should not be nil
// Higher-level shapes may be reduced to a path.
type Path []Operation

// ToSVGPath returns a string representation of the path
func (p Path) ToSVGPath() string {
	chunks := make([]string, len(p))
	for i, op := range p {
		chunks[i] = op.String()
	}
	return strings.Join(chunks, " ")
}

// String returns a readable representation of a Path.
func (p Path) String() string {
	return p.ToSVGPath()
}

// Clear zeros the path slice
func (p *Path) Clear() {
	*p = (*p)[:0]
}

// Start starts a new curve at the given point.
func (p *Path) Start(a f32.Point) {
	*p = append(*p, OpMoveTo(a))
}

// Line adds a linear segment to the current curve.
func (p *Path) Line(b f32.Point) {
	*p = append(*p, OpLineTo(b))
}

// QuadBezier adds a quadratic segment to the current curve.
func (p *Path) QuadBezier(b, c f32.Point) {
	*p = append(*p, OpQuadTo{b, c})
}

// CubeBezier adds a cubic segment to the current curve.
func (p *Path) CubeBezier(b, c, d f32.Point) {
	*p = append(*p, OpCubicTo{b, c, d})
}

// Stop joins the ends of the path
func (p *Path) Stop(closeLoop bool) {
	if closeLoop {
		*p = append(*p, OpClose{})
	}
}
