package widgets

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
)

func HorizontalFullLine() layout.FlexChild {
	return HorizontalLine(0)
}

func HorizontalLine(width float32) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		if width == 0 {
			width = float32(gtx.Constraints.Max.X)
		}
		return Rect{
			// gray 300
			Color: color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			Size:  f32.Point{X: width, Y: 2},
			Radii: 1,
		}.Layout(gtx)
	})
}

func VerticalFullLine() layout.FlexChild {
	return VerticalLine(0)
}

func VerticalLine(height float32) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		if height == 0 {
			height = float32(gtx.Constraints.Max.Y)
		}

		return Rect{
			// gray 300
			Color: color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			Size:  f32.Point{X: 2, Y: height},
			Radii: 1,
		}.Layout(gtx)
	})
}
