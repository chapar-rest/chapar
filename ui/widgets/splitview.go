package widgets

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/component"
	"github.com/mirzakhany/chapar/ui/chapartheme"
)

type SplitView struct {
	// Ratio keeps the current layout.
	// 0 is center, -1 completely to the left, 1 completely to the right.
	//Bar is the width for resizing the layout
	BarWidth unit.Dp
	component.Resize
}

const defaultBarWidth = unit.Dp(2)

func (s *SplitView) Layout(gtx layout.Context, theme *chapartheme.Theme, left, right layout.Widget) layout.Dimensions {
	bar := gtx.Dp(s.BarWidth)
	if bar <= 1 {
		bar = gtx.Dp(defaultBarWidth)
	}

	return s.Resize.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return left(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			return right(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			rect := image.Rectangle{
				Max: image.Point{
					X: gtx.Dp(unit.Dp(2)),
					Y: gtx.Constraints.Max.Y,
				},
			}
			paint.FillShape(gtx.Ops, theme.SeparatorColor, clip.Rect(rect).Op())
			return layout.Dimensions{Size: rect.Max}
		},
	)
}
