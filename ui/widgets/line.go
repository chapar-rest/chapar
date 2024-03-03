package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

func DrawLineFlex(background color.NRGBA, height, width unit.Dp) layout.FlexChild {
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
