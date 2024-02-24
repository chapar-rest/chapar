package outlay

import (
	"fmt"
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

type Fan struct {
	itemsCache        []cacheItem
	last              fanParams
	animatedLastFrame bool
	Animation

	// The width, in radians, of the full arc that items should occupy.
	// If zero, math.Pi/2 will be used (1/4 of a full circle).
	WidthRadians float32

	// The offset, in radians, above the X axis to apply before rendering the
	// arc. This can be used with a value of Pi/4 to center an arc of width
	// Pi/2. If zero, math.Pi/4 will be used (1/8 of a full circle). To get the
	// equivalent of specifying zero, specify a value of 2*math.Pi.
	OffsetRadians float32

	// The radius of the hollow circle at the center of the fan. Leave nil to
	// use the default heuristic of half the width of the widest item.
	HollowRadius *unit.Dp
}

type fanParams struct {
	arc    float32
	radius float32
	len    int
}

func (f fanParams) String() string {
	return fmt.Sprintf("arc: %v radus: %v len: %v", f.arc, f.radius, f.len)
}

type cacheItem struct {
	elevated bool
	op.CallOp
	layout.Dimensions
}

type FanItem struct {
	W       layout.Widget
	Elevate bool
}

func Item(elevate bool, w layout.Widget) FanItem {
	return FanItem{
		W:       w,
		Elevate: elevate,
	}
}

func (f *Fan) fullWidthRadians() float32 {
	if f.WidthRadians == 0 {
		return math.Pi / 2
	}
	return f.WidthRadians
}

func (f *Fan) offsetRadians() float32 {
	if f.OffsetRadians == 0 {
		return math.Pi / 4
	}
	return f.OffsetRadians
}

func (f *Fan) Layout(gtx layout.Context, items ...FanItem) layout.Dimensions {
	defer op.Offset(image.Point{
		X: gtx.Constraints.Max.X / 2,
		Y: gtx.Constraints.Max.Y / 2,
	}).Push(gtx.Ops).Pop()
	f.itemsCache = f.itemsCache[:0]
	maxWidth := 0
	for i := range items {
		item := items[i]
		macro := op.Record(gtx.Ops)
		dims := item.W(gtx)
		if dims.Size.X > maxWidth {
			maxWidth = dims.Size.X
		}
		f.itemsCache = append(f.itemsCache, cacheItem{
			CallOp:     macro.Stop(),
			Dimensions: dims,
			elevated:   item.Elevate,
		})
	}
	var current fanParams
	current.len = len(items)
	if f.HollowRadius == nil {
		current.radius = float32(maxWidth * 2.0)
	} else {
		current.radius = float32(gtx.Dp(*f.HollowRadius))
	}
	var itemArcFraction float32
	if len(items) > 1 {
		itemArcFraction = float32(1) / float32(len(items)-1)
	} else {
		itemArcFraction = 1
	}
	current.arc = f.fullWidthRadians() * itemArcFraction

	var empty fanParams
	if f.last == empty {
		f.last = current
	} else if f.last != current {

		if !f.animatedLastFrame {
			f.Start(gtx.Now)
		}
		progress := f.Progress(gtx)
		if f.animatedLastFrame && progress >= 1 {
			f.last = current
		}
		f.animatedLastFrame = false
		if f.Animating(gtx) {
			f.animatedLastFrame = true
			gtx.Execute(op.InvalidateCmd{})
		}
		current.arc = f.last.arc - (f.last.arc-current.arc)*progress
		current.radius = f.last.radius - (f.last.radius-current.radius)*progress
	}

	visible := f.itemsCache[:min(f.last.len, current.len)]
	for i := range visible {
		if !f.itemsCache[i].elevated {
			f.layoutItem(gtx, i, current)
		}
	}
	for i := range visible {
		if f.itemsCache[i].elevated {
			f.layoutItem(gtx, i, current)
		}
	}
	return layout.Dimensions{
		Size: gtx.Constraints.Max,
	}

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (f *Fan) layoutItem(gtx layout.Context, index int, params fanParams) layout.Dimensions {
	arc := params.arc
	radius := params.radius
	arc = arc*float32(index) + f.offsetRadians()
	var transform f32.Affine2D
	transform = transform.Rotate(f32.Point{}, -math.Pi/2).
		Offset(f32.Pt(-radius, float32(f.itemsCache[index].Dimensions.Size.X/2))).
		Rotate(f32.Point{}, arc)
	defer op.Affine(transform).Push(gtx.Ops).Pop()
	f.itemsCache[index].Add(gtx.Ops)
	return layout.Dimensions{}
}
