package widgets

import (
	"image"
	"image/color"

	"gioui.org/op/paint"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
)

type SplitView struct {
	// Ratio keeps the current layout.
	// 0 is center, -1 completely to the left, 1 completely to the right.
	Ratio float32

	drag   bool
	dragID pointer.ID
	dragX  float32

	// Bar is the width for resizing the layout
	BarWidth      unit.Dp
	BarColor      color.NRGBA
	BarColorHover color.NRGBA

	MinLeftSize int
	MaxLeftSize int

	MinRightSize int
	MaxRightSize int
}

const defaultBarWidth = unit.Dp(2)

func (s *SplitView) Layout(gtx layout.Context, left, right layout.Widget) layout.Dimensions {
	bar := gtx.Dp(s.BarWidth)
	if bar <= 1 {
		bar = gtx.Dp(defaultBarWidth)
	}

	// 0.18 := (x + 1) / 2
	proportion := (s.Ratio + 1) / 2
	leftSize := int(proportion*float32(gtx.Constraints.Max.X) - float32(bar))
	if leftSize < s.MinLeftSize {
		leftSize = s.MinLeftSize
	}

	if leftSize > s.MaxLeftSize && s.MaxLeftSize > 0 {
		leftSize = s.MaxLeftSize
	}

	rightOffset := leftSize + bar
	rightsize := gtx.Constraints.Max.X - rightOffset

	if rightsize < s.MinRightSize {
		rightsize = s.MinRightSize
	}

	if rightsize > s.MaxRightSize && s.MaxRightSize > 0 {
		rightsize = s.MaxRightSize
	}

	{
		barColor := s.BarColor
		for _, ev := range gtx.Events(s) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Kind {
			case pointer.Press:
				if s.drag {
					break
				}

				barColor = s.BarColorHover
				s.dragID = e.PointerID
				s.dragX = e.Position.X

			case pointer.Drag:
				if s.dragID != e.PointerID {
					break
				}

				barColor = s.BarColorHover
				deltaX := e.Position.X - s.dragX
				s.dragX = e.Position.X

				deltaRatio := deltaX * 2 / float32(gtx.Constraints.Max.X)
				s.Ratio += deltaRatio

			case pointer.Release:
				barColor = s.BarColor
				fallthrough
			case pointer.Cancel:
				s.drag = false
				barColor = s.BarColor
			default:
				continue
			}
		}

		// register for input
		barRect := image.Rect(leftSize, 0, rightOffset, gtx.Constraints.Max.X)
		paint.FillShape(gtx.Ops, barColor, clip.Rect(barRect).Op())
		area := clip.Rect(barRect).Push(gtx.Ops)
		pointer.CursorColResize.Add(gtx.Ops)
		pointer.InputOp{
			Tag:   s,
			Kinds: pointer.Press | pointer.Drag | pointer.Release,
			Grab:  s.drag,
		}.Add(gtx.Ops)
		area.Pop()
	}

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(leftSize, gtx.Constraints.Max.Y))
		left(gtx)
	}

	{
		off := op.Offset(image.Pt(rightOffset, 0)).Push(gtx.Ops)
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(rightsize, gtx.Constraints.Max.Y))
		right(gtx)
		off.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
