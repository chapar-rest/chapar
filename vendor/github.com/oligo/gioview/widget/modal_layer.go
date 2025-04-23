package widget

import (
	"image"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

// ModalLayer is a widget drawn on top of the normal UI that can be populated
// by other components with dismissble modal dialogs.
type ModalLayer struct {
	component.VisibilityAnimation
	Widget        func(gtx layout.Context, th *material.Theme, anim *component.VisibilityAnimation) layout.Dimensions
	justDismissed bool
}

const defaultModalAnimationDuration = time.Millisecond * 250

// NewModal creates an initializes a modal layer.
func NewModal() *ModalLayer {
	m := ModalLayer{}
	m.VisibilityAnimation.State = component.Invisible
	m.VisibilityAnimation.Duration = defaultModalAnimationDuration
	return &m
}

// Layout renders the modal layer. Unless a modal widget has been triggered,
// this will do nothing.
func (m *ModalLayer) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	m.Update(gtx)
	if !m.Visible() {
		return D{}
	}

	// Lay out a transparent scrim to block input to things beneath the
	// contextual widget.
	suppressionScrim := func() op.CallOp {
		macro := op.Record(gtx.Ops)
		pr := clip.Rect(image.Rectangle{Min: image.Point{-1e6, -1e6}, Max: image.Point{1e6, 1e6}})
		stack := pr.Push(gtx.Ops)
		event.Op(gtx.Ops, m)
		stack.Pop()
		return macro.Stop()
	}()
	op.Defer(gtx.Ops, suppressionScrim)

	gtx.Constraints.Min = gtx.Constraints.Max
	if m.Widget != nil {
		macro := op.Record(gtx.Ops)
		dims := m.Widget(gtx, th, &m.VisibilityAnimation)
		contentOps := macro.Stop()

		modalAreaOps := func() op.CallOp {
			macro := op.Record(gtx.Ops)
			var modalArea clip.Rect
			if m.Animating() {
				revealed := m.Revealed(gtx)
				modalArea = clip.Rect{Max: image.Point{dims.Size.X, int(float32(dims.Size.Y) * revealed)}}
			} else {
				modalArea = clip.Rect{Max: image.Point{dims.Size.X, dims.Size.Y}}
			}
			stack := modalArea.Push(gtx.Ops)
			contentOps.Add(gtx.Ops)
			stack.Pop()
			return macro.Stop()
		}()
		op.Defer(gtx.Ops, modalAreaOps)
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}

}

func (m *ModalLayer) Update(gtx C) {
	// Dismiss the contextual widget if the user clicked outside of it.
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: m,
			Kinds:  pointer.Press,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Kind == pointer.Press {
			m.Disappear(gtx.Now)
		}
	}
}

func (m *ModalLayer) Disappear(now time.Time) {
	if m.Visible() && !m.Animating() {
		m.justDismissed = true
	}
	m.VisibilityAnimation.Disappear(now)
}

func (m *ModalLayer) ToggleVisibility(now time.Time) {
	if m.Visible() {
		m.Disappear(now)
	} else {
		m.Appear(now)
	}
}

func (m *ModalLayer) Dismissed() bool {
	defer func() {
		m.justDismissed = false
	}()
	return m.justDismissed
}
