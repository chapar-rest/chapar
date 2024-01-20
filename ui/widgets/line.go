package widgets

import (
	"image"
	"image/color"

	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"gioui.org/unit"

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

func DrawLineFlex(gtx layout.Context, background color.NRGBA, height, width unit.Dp) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return DrawLine(gtx, background, height, width)
	})
}

func DrawLine(gtx layout.Context, background color.NRGBA, height, width unit.Dp) layout.Dimensions {
	w, h := gtx.Dp(width), gtx.Dp(height)
	tabRect := image.Rect(0, 0, w, h)
	paint.FillShape(gtx.Ops, background, clip.Rect(tabRect).Op())
	return layout.Dimensions{Size: image.Pt(w, h)}
}
