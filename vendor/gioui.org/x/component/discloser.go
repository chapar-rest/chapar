package component

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// DiscloserState holds state for a widget that can hide and reveal
// content.
type DiscloserState struct {
	VisibilityAnimation
	widget.Clickable
}

// Layout updates the state of the Discloser.
func (d *DiscloserState) Layout(gtx C) D {
	if d.Duration == time.Duration(0) {
		d.Duration = time.Millisecond * 100
		d.State = Invisible
	}
	if d.Clicked(gtx) {
		d.ToggleVisibility(gtx.Now)
	}
	return D{}
}

// Side represents a preference for left or right.
type Side bool

const (
	Left  Side = false
	Right Side = true
)

// DiscloserStyle defines the presentation of a discloser widget.
type DiscloserStyle struct {
	*DiscloserState
	// ControlSide defines whether the control widget is drawn to the
	// left or right of the summary widget.
	ControlSide Side
	// Alignment dictates how the control and summary are aligned relative
	// to one another.
	Alignment layout.Alignment
}

// Discloser configures a discloser from the provided theme and state.
func Discloser(th *material.Theme, state *DiscloserState) DiscloserStyle {
	return DiscloserStyle{
		DiscloserState: state,
		Alignment:      layout.Middle,
	}
}

// Layout the discloser with the provided toggle control, summary widget, and
// detail widget. The toggle widget will be wrapped in a clickable area
// automatically.
//
// The structure of the resulting discloser is:
//
//	control | summary
//	-----------------
//	detail
//
// If d.ControlSide is set to Right, the control will appear after the summary
// instead of before it.
func (d DiscloserStyle) Layout(gtx C, control, summary, detail layout.Widget) D {
	d.DiscloserState.Layout(gtx)
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			controlWidget := func(gtx C) D {
				return d.Clickable.Layout(gtx, control)
			}
			return layout.Flex{
				Alignment: d.Alignment,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					if d.ControlSide == Left {
						return controlWidget(gtx)
					}
					return summary(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					if d.ControlSide == Left {
						return summary(gtx)
					}
					return controlWidget(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			if !d.Visible() {
				return D{}
			}
			if !d.Animating() {
				return detail(gtx)
			}
			progress := d.Revealed(gtx)
			macro := op.Record(gtx.Ops)
			dims := detail(gtx)
			call := macro.Stop()
			height := int(math.Round(float64(float32(dims.Size.Y) * progress)))
			dims.Size.Y = height
			defer clip.Rect(image.Rectangle{
				Max: dims.Size,
			}).Push(gtx.Ops).Pop()
			call.Add(gtx.Ops)
			return dims
		}),
	)
}

// DiscloserArrowStyle defines the presentation of a simple triangular
// Discloser control that rotates downward as the content is revealed.
type DiscloserArrowStyle struct {
	Color color.NRGBA
	Size  unit.Dp
	State *DiscloserState
	Side
	Margin layout.Inset
}

// DiscloserArrow creates and configures a DiscloserArrow for use with
// the provided DiscloserStyle.
func DiscloserArrow(th *material.Theme, style DiscloserStyle) DiscloserArrowStyle {
	return DiscloserArrowStyle{
		Color:  color.NRGBA{A: 200},
		State:  style.DiscloserState,
		Size:   unit.Dp(10),
		Side:   style.ControlSide,
		Margin: layout.UniformInset(unit.Dp(4)),
	}
}

// Width returns the width of the arrow and surrounding whitespace.
func (d DiscloserArrowStyle) Width() unit.Dp {
	return d.Size + d.Margin.Right + d.Margin.Left
}

// DetailInset returns a layout.Inset that can be used to align a
// Discloser's details with its summary.
func (d DiscloserArrowStyle) DetailInset() layout.Inset {
	if d.Side == Left {
		return layout.Inset{
			Left: d.Width(),
		}
	}
	return layout.Inset{
		Right: d.Width(),
	}
}

const halfPi float32 = math.Pi * .5

// Layout the arrow.
func (d DiscloserArrowStyle) Layout(gtx C) D {
	return d.Margin.Layout(gtx, func(gtx C) D {
		// Draw a triangle.
		path := clip.Path{}
		path.Begin(gtx.Ops)
		size := float32(gtx.Dp(d.Size))
		halfSize := size * .5
		path.LineTo(f32.Pt(0, size))
		path.LineTo(f32.Pt(size, halfSize))
		path.Close()
		outline := clip.Outline{
			Path: path.End(),
		}
		affine := f32.Affine2D{}
		if d.State.Visible() {
			// Rotate the triangle.
			origin := f32.Pt(halfSize, halfSize)
			rotation := halfPi
			if d.State.Animating() {
				rotation *= d.State.Revealed(gtx)
			}
			affine = affine.Rotate(origin, rotation)
		}
		if d.Side == Right {
			// Mirror the triangle.
			affine = affine.Scale(f32.Point{}, f32.Point{
				X: -1,
				Y: 1,
			}).Offset(f32.Pt(size, 0))
		}
		defer op.Affine(affine).Push(gtx.Ops).Pop()
		paint.FillShape(gtx.Ops, d.Color, outline.Op())
		return D{
			Size: image.Pt(int(size), int(size)),
		}
	})
}

// SimpleDiscloserStyle configures a default discloser that uses a simple
// rotating triangle control and indents its details.
type SimpleDiscloserStyle struct {
	DiscloserStyle
	DiscloserArrowStyle
}

// SimpleDiscloser creates a SimpleDiscloserStyle for the given theme and
// DiscloserState.
func SimpleDiscloser(th *material.Theme, state *DiscloserState) SimpleDiscloserStyle {
	sd := SimpleDiscloserStyle{
		DiscloserStyle: Discloser(th, state),
	}
	sd.DiscloserArrowStyle = DiscloserArrow(th, sd.DiscloserStyle)
	return sd
}

// Layout the discloser with the provided summary and detail widget content.
func (sd SimpleDiscloserStyle) Layout(gtx C, summary, details layout.Widget) D {
	return sd.DiscloserStyle.Layout(gtx, sd.DiscloserArrowStyle.Layout, summary, func(gtx C) D {
		return sd.DiscloserArrowStyle.DetailInset().Layout(gtx, details)
	})
}
