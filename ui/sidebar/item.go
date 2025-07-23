package sidebar

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type Item struct {
	Tag  any
	Name string
	Icon *widget.Icon
}

type renderItem struct {
	Item
	hovering bool
	selected bool
	widget.Clickable
	*AlphaPalette
}

type AlphaPalette struct {
	Hover, Selected uint8
}

func (r *renderItem) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	for {
		event, ok := gtx.Event(pointer.Filter{
			Target: r,
			Kinds:  pointer.Enter | pointer.Leave,
		})
		if !ok {
			break
		}
		switch event := event.(type) {
		case pointer.Event:
			switch event.Kind {
			case pointer.Enter:
				r.hovering = true
			case pointer.Leave, pointer.Cancel:
				r.hovering = false
			}
		}
	}
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rectangle{
		Max: gtx.Constraints.Max,
	}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, r)
	return layout.Inset{
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
		Left:   unit.Dp(4),
		Right:  unit.Dp(4),
	}.Layout(gtx, func(gtx C) D {
		return material.Clickable(gtx, &r.Clickable, func(gtx C) D {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx C) D { return r.layoutBackground(gtx, th) }),
				layout.Stacked(func(gtx C) D { return r.layoutContent(gtx, th) }),
			)
		})
	})
}

func (r *renderItem) layoutContent(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	contentColor := widgets.Disabled(th.SideBarTextColor)
	if r.selected {
		contentColor = th.SideBarTextColor
	}
	return layout.Inset{
		Left:  unit.Dp(2),
		Right: unit.Dp(2),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle, Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				if r.Item.Icon == nil {
					return layout.Dimensions{}
				}
				return layout.Inset{Bottom: unit.Dp(5), Top: unit.Dp(5)}.Layout(gtx,
					func(gtx C) D {
						iconSize := gtx.Dp(unit.Dp(24))
						gtx.Constraints = layout.Exact(image.Pt(iconSize, iconSize))
						return r.Item.Icon.Layout(gtx, contentColor)
					})
			}),
			layout.Rigid(func(gtx C) D {
				label := material.Label(th.Material(), unit.Sp(12), r.Name)
				label.Color = contentColor
				return layout.Center.Layout(gtx, label.Layout)
			}),
		)
	})
}

func (r *renderItem) layoutBackground(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	if !r.selected && !r.hovering {
		return layout.Dimensions{}
	}
	var fill color.NRGBA
	if r.hovering {
		fill = WithAlpha(th.Palette.Fg, r.AlphaPalette.Hover)
	} else if r.selected {
		fill = WithAlpha(th.Palette.ContrastBg, r.AlphaPalette.Selected)
	}
	rr := gtx.Dp(unit.Dp(5))
	defer clip.RRect{
		Rect: image.Rectangle{
			Max: gtx.Constraints.Max,
		},
		NE: rr,
		SE: rr,
		NW: rr,
		SW: rr,
	}.Push(gtx.Ops).Pop()
	paintRect(gtx, gtx.Constraints.Max, fill)
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

// WithAlpha returns the input color with the new alpha value.
func WithAlpha(c color.NRGBA, a uint8) color.NRGBA {
	return color.NRGBA{
		R: c.R,
		G: c.G,
		B: c.B,
		A: a,
	}
}

func paintRect(gtx layout.Context, size image.Point, fill color.NRGBA) {
	Rect{
		Color: fill,
		Size:  size,
	}.Layout(gtx)
}

type Rect struct {
	Color color.NRGBA
	Size  image.Point
	Radii int
}

func (r Rect) Layout(gtx C) D {
	paint.FillShape(
		gtx.Ops,
		r.Color,
		clip.UniformRRect(
			image.Rectangle{
				Max: r.Size,
			},
			r.Radii,
		).Op(gtx.Ops))
	return layout.Dimensions{Size: r.Size}
}
