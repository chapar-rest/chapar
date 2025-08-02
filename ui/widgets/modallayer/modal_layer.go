package modallayer

import (
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

// ModalLayer is a widget drawn on top of the normal UI that can be populated
// by other material components with dismissble modal dialogs. For instance,
// the App Bar can render its overflow menu within the modal layer, and the
// modal navigation drawer is entirely within the modal layer.
type ModalLayer struct {
	component.VisibilityAnimation
	component.Scrim
	Widget func(gtx layout.Context, th *material.Theme, anim *component.VisibilityAnimation) layout.Dimensions
}

const defaultModalAnimationDuration = time.Millisecond * 250

// NewModal creates an initializes a modal layer.
func NewModal() *ModalLayer {
	m := ModalLayer{}
	m.VisibilityAnimation.State = component.Invisible
	m.VisibilityAnimation.Duration = defaultModalAnimationDuration
	m.Scrim.FinalAlpha = 82 //default
	return &m
}

// Layout renders the modal layer. Unless a modal widget has been triggered,
// this will do nothing.
func (m *ModalLayer) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !m.Visible() {
		return layout.Dimensions{}
	}
	scrimDims := m.Scrim.Layout(gtx, th, &m.VisibilityAnimation)
	if m.Widget != nil {
		_ = m.Widget(gtx, th, &m.VisibilityAnimation)
	}
	return scrimDims
}

// ModalState defines persistent state for a modal.
type ModalState struct {
	component.ScrimState
	// content is the content widget to layout atop a scrim.
	// This is specified as a field because where the content is defined
	// is not where it is invoked.
	// Thus, the content widget becomes the state of the modal.
	content layout.Widget
}

// ModalStyle describes how to layout a modal.
// Modal content is layed centered atop a clickable scrim.
type ModalStyle struct {
	*ModalState
	Scrim component.ScrimStyle
}

// Modal lays out a content widget atop a clickable scrim.
// Clicking the scrim dismisses the modal.
func Modal(th *material.Theme, modal *ModalState) ModalStyle {
	return ModalStyle{
		ModalState: modal,
		Scrim:      component.NewScrim(th, &modal.ScrimState, 250),
	}
}

// Layout the scrim and content. The content is only laid out once
// the scrim is fully animated in, and is hidden on the first frame
// of the scrim's fade-out animation.
func (m ModalStyle) Layout(gtx layout.Context) layout.Dimensions {
	if m.content == nil || !m.Visible() {
		return layout.Dimensions{}
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return m.Scrim.Layout(gtx)
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			if m.Scrim.Visible() && !m.Scrim.Animating() {
				return m.content(gtx)
			}
			return layout.Dimensions{}
		}),
	)
	op.Defer(gtx.Ops, macro.Stop())
	return dims
}

// Show widget w in the modal, starting animation at now.
func (m *ModalState) Show(now time.Time, w layout.Widget) {
	m.content = w
	m.Appear(now)
}
