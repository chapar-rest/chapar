package widgets

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
)

func HorizontalLine() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return Rect{
			// gray 300
			Color: color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			Size:  f32.Point{X: float32(gtx.Constraints.Max.X), Y: 2},
			Radii: 1,
		}.Layout(gtx)
	})
}

func VerticalLine() layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return Rect{
			// gray 300
			Color: color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			Size:  f32.Point{X: 2, Y: float32(gtx.Constraints.Max.Y)},
			Radii: 1,
		}.Layout(gtx)
	})
}
