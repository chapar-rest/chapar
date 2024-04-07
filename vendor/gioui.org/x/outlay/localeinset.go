package outlay

import (
	"image"

	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Inset adds space around a widget by decreasing its maximum
// constraints. The minimum constraints will be adjusted to ensure
// they do not exceed the maximum. Inset respects the system locale
// provided at layout time, and will swap start/end insets for
// RTL text. This differs from gioui.org/layout.Inset, which never
// swaps the sides contextually.
type Inset struct {
	Top, Bottom unit.Dp
	// Start and End refer to the visual start and end of the widget
	// and surrounding area. In LTR locales, Start is left and End is
	// right. In RTL locales the inverse in true.
	Start, End unit.Dp
}

// Layout a widget.
func (in Inset) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	top := gtx.Dp(in.Top)
	right := gtx.Dp(in.End)
	bottom := gtx.Dp(in.Bottom)
	left := gtx.Dp(in.Start)
	if gtx.Locale.Direction.Progression() == system.TowardOrigin {
		if gtx.Locale.Direction.Axis() == system.Horizontal {
			right, left = left, right
		} else {
			top, bottom = bottom, top
		}
	}
	mcs := gtx.Constraints
	mcs.Max.X -= left + right
	if mcs.Max.X < 0 {
		left = 0
		right = 0
		mcs.Max.X = 0
	}
	if mcs.Min.X > mcs.Max.X {
		mcs.Min.X = mcs.Max.X
	}
	mcs.Max.Y -= top + bottom
	if mcs.Max.Y < 0 {
		bottom = 0
		top = 0
		mcs.Max.Y = 0
	}
	if mcs.Min.Y > mcs.Max.Y {
		mcs.Min.Y = mcs.Max.Y
	}
	gtx.Constraints = mcs
	trans := op.Offset(image.Pt(left, top)).Push(gtx.Ops)
	dims := w(gtx)
	trans.Pop()
	return layout.Dimensions{
		Size:     dims.Size.Add(image.Point{X: right + left, Y: top + bottom}),
		Baseline: dims.Baseline + bottom,
	}
}

// UniformInset returns an Inset with a single inset applied to all
// edges.
func UniformInset(v unit.Dp) Inset {
	return Inset{Top: v, End: v, Bottom: v, Start: v}
}
