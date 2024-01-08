package widgets

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// Rect creates a rectangle of the provided background color with
// Dimensions specified by size and a corner radius (on all corners)
// specified by radii.
type Rect struct {
	Color color.NRGBA
	Size  f32.Point
	Radii float32
}

// Layout renders the Rect into the provided context
func (r Rect) Layout(gtx layout.Context) layout.Dimensions {
	return DrawRect(gtx, r.Color, r.Size, r.Radii)
}

// DrawRect creates a rectangle of the provided background color with
// Dimensions specified by size and a corner radius (on all corners)
// specified by radii.
func DrawRect(gtx layout.Context, background color.NRGBA, size f32.Point, radii float32) layout.Dimensions {
	bounds := image.Rectangle{
		Max: image.Point{
			X: int(size.X),
			Y: int(size.Y),
		},
	}
	paint.FillShape(gtx.Ops, background, clip.UniformRRect(bounds, int(radii)).Op(gtx.Ops))
	return layout.Dimensions{Size: image.Pt(int(size.X), int(size.Y))}
}
