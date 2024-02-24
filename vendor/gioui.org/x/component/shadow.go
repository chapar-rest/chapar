// SPDX-License-Identifier: Unlicense OR MIT

package component

/*
This file is derived from work by Egon Elbre in his gio experiments
repository available here:

https://github.com/egonelbre/expgio/tree/master/box-shadows

He generously licensed it under the Unlicense, and thus is is
reproduced here under the same terms.
*/

import (
	"image"
	"image/color"
	"math"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// ShadowStyle defines a shadow cast by a rounded rectangle.
//
// TODO(whereswaldon): make this support RRects that do not have
// uniform corner radii.
type ShadowStyle struct {
	// The radius of the corners of the rectangle casting the surface.
	// Non-rounded rectangles can just provide a zero.
	CornerRadius unit.Dp
	// Elevation is how high the surface casting the shadow is above
	// the background, and therefore determines how diffuse and large
	// the shadow is.
	Elevation unit.Dp
	// The colors of various components of the shadow. The Shadow()
	// constructor populates these with reasonable defaults.
	AmbientColor, PenumbraColor, UmbraColor color.NRGBA
}

// Shadow defines a shadow cast by a rounded rectangle with the given
// corner radius and elevation. It sets reasonable defaults for the
// shadow colors.
func Shadow(radius, elevation unit.Dp) ShadowStyle {
	return ShadowStyle{
		CornerRadius:  radius,
		Elevation:     elevation,
		AmbientColor:  color.NRGBA{A: 0x10},
		PenumbraColor: color.NRGBA{A: 0x20},
		UmbraColor:    color.NRGBA{A: 0x30},
	}
}

// Layout renders the shadow into the gtx. The shadow's size will assume
// that the rectangle casting the shadow is of size gtx.Constraints.Min.
func (s ShadowStyle) Layout(gtx layout.Context) layout.Dimensions {
	sz := gtx.Constraints.Min
	rr := gtx.Dp(s.CornerRadius)

	r := image.Rect(0, 0, sz.X, sz.Y)
	s.layoutShadow(gtx, r, rr)

	return layout.Dimensions{Size: sz}
}

func (s ShadowStyle) layoutShadow(gtx layout.Context, r image.Rectangle, rr int) {
	if s.Elevation <= 0 {
		return
	}

	offset := gtx.Dp(s.Elevation)

	ambient := r
	gradientBox(gtx.Ops, ambient, rr, offset/2, s.AmbientColor)

	penumbra := r.Add(image.Pt(0, offset/2))
	gradientBox(gtx.Ops, penumbra, rr, offset, s.PenumbraColor)

	umbra := outset(penumbra, -offset/2)
	gradientBox(gtx.Ops, umbra, rr/4, offset/2, s.UmbraColor)
}

// TODO(whereswaldon): switch back to commented implementation when radial
// gradients are available in core.
func gradientBox(ops *op.Ops, r image.Rectangle, rr, spread int, col color.NRGBA) {
	/*
		transparent := col
		transparent.A = 0

		// ensure we are aligned to pixel grid
		r = round(r)
		rr = float32(math.Ceil(float64(rr)))
		spread = float32(math.Ceil(float64(spread)))

		// calculate inside and outside boundaries
		inside := imageRect(outset(r, -rr))
		center := imageRect(r)
		outside := imageRect(outset(r, spread))

		radialStop2 := image.Pt(0, int(spread+rr))
		radialOffset1 := rr / (spread + rr)

		corners := []func(image.Rectangle) image.Point{
			topLeft,
			topRight,
			bottomRight,
			bottomLeft,
		}

		for _, corner := range corners {
			func() {
				defer op.Save(ops).Load()
				clipr := image.Rectangle{
					Min: corner(inside),
					Max: corner(outside),
				}.Canon()
				clip.Rect(clipr).Add(ops)
				paint.RadialGradientOp{
					Color1: col, Color2: transparent,
					Stop1:   layout.FPt(corner(inside)),
					Stop2:   layout.FPt(corner(inside).Add(radialStop2)),
					Offset1: radialOffset1,
				}.Add(ops)
				paint.PaintOp{}.Add(ops)
			}()
		}

		// top
		func() {
			defer op.Save(ops).Load()
			clipr := image.Rectangle{
				Min: image.Point{
					X: inside.Min.X,
					Y: outside.Min.Y,
				},
				Max: image.Point{
					X: inside.Max.X,
					Y: center.Min.Y,
				},
			}
			clip.Rect(clipr).Add(ops)
			paint.LinearGradientOp{
				Color1: col, Color2: transparent,
				Stop1: layout.FPt(image.Point{
					X: inside.Min.X,
					Y: center.Min.Y,
				}),
				Stop2: layout.FPt(image.Point{
					X: inside.Min.X,
					Y: outside.Min.Y,
				}),
			}.Add(ops)
			paint.PaintOp{}.Add(ops)
		}()

		// right
		func() {
			defer op.Save(ops).Load()
			clipr := image.Rectangle{
				Min: image.Point{
					X: center.Max.X,
					Y: inside.Min.Y,
				},
				Max: image.Point{
					X: outside.Max.X,
					Y: inside.Max.Y,
				},
			}
			clip.Rect(clipr).Add(ops)
			paint.LinearGradientOp{
				Color1: col, Color2: transparent,
				Stop1: layout.FPt(image.Point{
					X: center.Max.X,
					Y: inside.Min.Y,
				}),
				Stop2: layout.FPt(image.Point{
					X: outside.Max.X,
					Y: inside.Min.Y,
				}),
			}.Add(ops)
			paint.PaintOp{}.Add(ops)
		}()

		// bottom
		func() {
			defer op.Save(ops).Load()
			clipr := image.Rectangle{
				Min: image.Point{
					X: inside.Min.X,
					Y: center.Max.Y,
				},
				Max: image.Point{
					X: inside.Max.X,
					Y: outside.Max.Y,
				},
			}
			clip.Rect(clipr).Add(ops)
			paint.LinearGradientOp{
				Color1: col, Color2: transparent,
				Stop1: layout.FPt(image.Point{
					X: inside.Min.X,
					Y: center.Max.Y,
				}),
				Stop2: layout.FPt(image.Point{
					X: inside.Min.X,
					Y: outside.Max.Y,
				}),
			}.Add(ops)
			paint.PaintOp{}.Add(ops)
		}()

		// left
		func() {
			defer op.Save(ops).Load()
			clipr := image.Rectangle{
				Min: image.Point{
					X: outside.Min.X,
					Y: inside.Min.Y,
				},
				Max: image.Point{
					X: center.Min.X,
					Y: inside.Max.Y,
				},
			}
			clip.Rect(clipr).Add(ops)
			paint.LinearGradientOp{
				Color1: col, Color2: transparent,
				Stop1: layout.FPt(image.Point{
					X: center.Min.X,
					Y: inside.Min.Y,
				}),
				Stop2: layout.FPt(image.Point{
					X: outside.Min.X,
					Y: inside.Min.Y,
				}),
			}.Add(ops)
			paint.PaintOp{}.Add(ops)
		}()

		func() {
			defer op.Save(ops).Load()
			var p clip.Path
			p.Begin(ops)

			inside := layout.FRect(inside)
			center := layout.FRect(center)

			p.MoveTo(inside.Min)
			p.LineTo(f32.Point{X: inside.Min.X, Y: center.Min.Y})
			p.LineTo(f32.Point{X: inside.Max.X, Y: center.Min.Y})
			p.LineTo(f32.Point{X: inside.Max.X, Y: inside.Min.Y})
			p.LineTo(f32.Point{X: center.Max.X, Y: inside.Min.Y})
			p.LineTo(f32.Point{X: center.Max.X, Y: inside.Max.Y})
			p.LineTo(f32.Point{X: inside.Max.X, Y: inside.Max.Y})
			p.LineTo(f32.Point{X: inside.Max.X, Y: center.Max.Y})
			p.LineTo(f32.Point{X: inside.Min.X, Y: center.Max.Y})
			p.LineTo(f32.Point{X: inside.Min.X, Y: inside.Max.Y})
			p.LineTo(f32.Point{X: center.Min.X, Y: inside.Max.Y})
			p.LineTo(f32.Point{X: center.Min.X, Y: inside.Min.Y})
			p.LineTo(inside.Min)

			clip.Outline{Path: p.End()}.Op().Add(ops)
			paint.ColorOp{Color: col}.Add(ops)
			paint.PaintOp{}.Add(ops)
		}()
	*/
	paint.FillShape(ops, col, clip.RRect{
		Rect: outset(r, spread),
		SE:   rr + spread, SW: rr + spread, NW: rr + spread, NE: rr + spread,
	}.Op(ops))
}

func imageRect(r image.Rectangle) image.Rectangle {
	return image.Rectangle{
		Min: image.Point{
			X: int(math.Round(float64(r.Min.X))),
			Y: int(math.Round(float64(r.Min.Y))),
		},
		Max: image.Point{
			X: int(math.Round(float64(r.Max.X))),
			Y: int(math.Round(float64(r.Max.Y))),
		},
	}
}

func round(r image.Rectangle) image.Rectangle {
	return image.Rectangle{
		Min: image.Point{
			X: int(math.Round(float64(r.Min.X))),
			Y: int(math.Round(float64(r.Min.Y))),
		},
		Max: image.Point{
			X: int(math.Round(float64(r.Max.X))),
			Y: int(math.Round(float64(r.Max.Y))),
		},
	}
}

func outset(r image.Rectangle, rr int) image.Rectangle {
	r.Min.X -= rr
	r.Min.Y -= rr
	r.Max.X += rr
	r.Max.Y += rr
	return r
}

func topLeft(r image.Rectangle) image.Point     { return r.Min }
func topRight(r image.Rectangle) image.Point    { return image.Point{X: r.Max.X, Y: r.Min.Y} }
func bottomRight(r image.Rectangle) image.Point { return r.Max }
func bottomLeft(r image.Rectangle) image.Point  { return image.Point{X: r.Min.X, Y: r.Max.Y} }
