package giosvg

import (
	"bytes"
	"image"
	"io"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"github.com/inkeliz/giosvg/internal/svgdraw"
	"github.com/inkeliz/giosvg/internal/svgparser"
)

// Vector hold the information from the XML/SVG file, in order to avoid
// decoding of the XML.
type Vector func(ops *op.Ops, constraints Constraints) layout.Dimensions

// Layout implements layout.Widget, that renders the current vector without any cache.
// Consider using NewIcon instead.
//
// You should avoid it, that functions only exists to simplify integration
// to custom cache implementations.
func (v Vector) Layout(gtx layout.Context) layout.Dimensions {
	return v(gtx.Ops, newConstraintsFromGio(gtx.Constraints))
}

// NewVector creates an IconOp from the given data. The data is
// expected to be an SVG/XML
func NewVector(data []byte) (Vector, error) { return NewVectorReader(bytes.NewReader(data)) }

// NewVectorReader creates an IconOp from the given io.Reader. The data is
// expected to be an SVG/XML
func NewVectorReader(reader io.Reader) (Vector, error) {
	render, err := svgparser.ReadIcon(reader)
	if err != nil {
		return nil, err
	}

	return func(ops *op.Ops, constraints Constraints) layout.Dimensions {
		var w, h float32
		if constraints.Max != constraints.Min {
			if render.ViewBox.W >= render.ViewBox.H {
				d := float32(render.ViewBox.W) / float32(render.ViewBox.H)
				if constraints.Max.Y*d > constraints.Max.X {
					w, h = constraints.Max.X, constraints.Max.X/d
				} else {
					w, h = constraints.Max.Y*d, constraints.Max.Y
				}
			} else {
				d := float32(render.ViewBox.H) / float32(render.ViewBox.W)
				if constraints.Max.X*d > constraints.Max.Y {
					w, h = constraints.Max.Y/d, constraints.Max.Y
				} else {
					w, h = constraints.Max.X, constraints.Max.X*d
				}
			}
		}

		if constraints.Min.X > w {
			w = constraints.Min.X
		}
		if constraints.Min.Y > h {
			h = constraints.Min.Y
		}

		render.SetTarget(0-render.ViewBox.X, 0-render.ViewBox.Y, float64(w), float64(h))
		scale := float32(float32(float64(w)/render.ViewBox.W)+float32(float64(h)/render.ViewBox.H)) / 2
		render.Draw(&svgdraw.Driver{Ops: ops, Scale: scale}, 1.0)

		return layout.Dimensions{Size: image.Point{X: int(w), Y: int(h)}}
	}, nil
}

// Constraints is the layout.Constraints with f32.Pt instead of image.Point.
// This is used to keep aspect ratio, and to keep the size of the icon
// within the constraints.
type Constraints struct {
	Max f32.Point
	Min f32.Point
}

func newConstraintsFromGio(constraints layout.Constraints) Constraints {
	return Constraints{
		Max: f32.Pt(float32(constraints.Max.X), float32(constraints.Max.Y)),
		Min: f32.Pt(float32(constraints.Min.X), float32(constraints.Min.Y)),
	}
}

// Icon keeps a cache from the last frame and re-uses it if
// the size didn't change.
type Icon struct {
	vector Vector

	lastDimensions layout.Dimensions
	lastSize       layout.Constraints
	macro          op.CallOp
	op             *op.Ops
}

// NewIcon creates the layout.Widget from the iconOp.
// Similar to widget.List, the Icon keeps the state from the last draw,
// and the drawing is used if the size remains unchanged. You should
// reuse the same Icon across multiples frames.
//
// Make sure to not reuse the Icon with different sizes in the same frame,
// if the same Icon is used twice  in the same frame you MUST create
// two Icon, for each one.
func NewIcon(vector Vector) *Icon {
	return &Icon{
		vector: vector,
		op:     new(op.Ops),
	}
}

// Layout implements widget.Layout.
// It will render the icon based on the given layout.Constraints.Max.
// If the SVG uses `currentColor` you can set the color using
// paint.ColorOp.
func (icon *Icon) Layout(gtx layout.Context) layout.Dimensions {
	if icon.lastSize != gtx.Constraints {
		// If the size changes, we can't re-use the macro.
		icon.lastSize = gtx.Constraints

		icon.op.Reset()
		macro := op.Record(icon.op)
		icon.lastDimensions = icon.vector(icon.op, newConstraintsFromGio(gtx.Constraints))
		icon.macro = macro.Stop()
	}

	icon.macro.Add(gtx.Ops)
	return icon.lastDimensions
}
