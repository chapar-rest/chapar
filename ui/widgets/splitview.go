package widgets

import (
	"image"
	"image/color"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type SplitView struct {
	// Ratio keeps the current layout.
	// 0 is center, -1 completely to the left, 1 completely to the right.
	// Bar is the width for resizing the layout
	BarWidth unit.Dp
	component.Resize

	handleBar *handleBar
}

const defaultBarWidth = unit.Dp(2)

type handleBar struct {
	BarWidth unit.Dp
	click    gesture.Click
	hovering bool

	bkColor, hoveredColor color.NRGBA
}

func (s *handleBar) Layout(gtx layout.Context) layout.Dimensions {
	bar := gtx.Dp(s.BarWidth)
	if bar <= 1 {
		bar = gtx.Dp(defaultBarWidth)
	}

	rect := image.Rectangle{
		Max: image.Point{
			X: bar,
			Y: gtx.Constraints.Max.Y,
		},
	}

	pointer.CursorPointer.Add(gtx.Ops)
	s.click.Add(gtx.Ops)

	for {
		_, ok := s.click.Update(gtx.Source)
		if !ok {
			break
		}
	}

	barColor := s.bkColor
	if isHovered := s.click.Hovered(); isHovered != s.hovering {
		s.hovering = isHovered
		if isHovered {
			barColor = s.hoveredColor
		} else {
			barColor = s.bkColor
		}
	}

	paint.FillShape(gtx.Ops, barColor, clip.Rect(rect).Op())
	return layout.Dimensions{Size: rect.Max}
}

func NewSpitView(theme *chapartheme.Theme, radio float32, barWidth unit.Dp) *SplitView {
	return &SplitView{
		BarWidth: unit.Dp(defaultBarWidth),
		Resize:   component.Resize{},
		handleBar: &handleBar{
			BarWidth:     barWidth,
			click:        gesture.Click{},
			hovering:     false,
			bkColor:      theme.SeparatorColor,
			hoveredColor: theme.SeparatorHoveredColor,
		},
	}
}

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
		s.handleBar.Layout,
	)
}
