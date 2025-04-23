package list

import (
	"image"
	"image/color"

	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type InteractiveLabel struct {
	itemClick  gesture.Click
	isSelected bool
	hovering   bool
	Radius     unit.Dp
}

func (l *InteractiveLabel) IsSelected() bool {
	return l.isSelected
}

func (l *InteractiveLabel) Select() {
	l.isSelected = true
}

func (l *InteractiveLabel) Unselect() {
	l.isSelected = false
}

// follow the new Update API to handle events before layout in the current frame.
func (l *InteractiveLabel) Update(gtx layout.Context) bool {
	for {
		event, ok := gtx.Event(
			pointer.Filter{Target: l, Kinds: pointer.Enter | pointer.Leave},
		)
		if !ok {
			break
		}

		switch event := event.(type) {
		case pointer.Event:
			switch event.Kind {
			case pointer.Enter:
				l.hovering = true
			case pointer.Leave:
				l.hovering = false
			case pointer.Cancel:
				l.hovering = false
			}
		}
	}

	var clicked bool
	for {
		e, ok := l.itemClick.Update(gtx.Source)
		if !ok {
			break
		}
		if e.Kind == gesture.KindClick {
			l.isSelected = true
			clicked = true
		}
	}

	return clicked
}

func (l *InteractiveLabel) layoutBackground(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	if !l.isSelected && !l.hovering {
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}

	var fill color.NRGBA
	if l.hovering {
		fill = misc.WithAlpha(th.Palette.Fg, th.HoverAlpha)
	} else if l.isSelected {
		fill = misc.WithAlpha(th.Palette.Fg, th.SelectedAlpha)
	}
	rr := gtx.Dp(l.Radius)
	rect := clip.RRect{
		Rect: image.Rectangle{
			Max: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Min.Y},
		},
		NE: rr,
		SE: rr,
		NW: rr,
		SW: rr,
	}
	paint.FillShape(gtx.Ops, fill, rect.Op(gtx.Ops))
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func (l *InteractiveLabel) Layout(gtx C, th *theme.Theme, w func(gtx C, textColor color.NRGBA) D) D {
	l.Update(gtx)

	contentColor := th.Palette.Fg
	if l.hovering {
		contentColor = th.Palette.Fg
	} else if l.isSelected {
		contentColor = th.Palette.Fg
	}

	macro := op.Record(gtx.Ops)
	dims := layout.Background{}.Layout(gtx,
		func(gtx C) D { return l.layoutBackground(gtx, th) },
		func(gtx C) D { return w(gtx, contentColor) },
	)

	itemOps := macro.Stop()

	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rectangle{
		Max: dims.Size,
	}).Push(gtx.Ops).Pop()

	l.itemClick.Add(gtx.Ops)

	// register tag
	event.Op(gtx.Ops, l)

	itemOps.Add(gtx.Ops)

	return dims
}
