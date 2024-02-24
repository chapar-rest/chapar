package component

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Scrim implments a clickable translucent overlay. It can animate appearing
// and disappearing as a fade-in, fade-out transition from zero opacity
// to a fixed maximum opacity.
type Scrim struct {
	// FinalAlpha is the final opacity of the scrim on a scale from 0 to 255.
	FinalAlpha uint8
	widget.Clickable
}

// Layout draws the scrim using the provided animation. If the animation indicates
// that the scrim is not visible, this is a no-op.
func (s *Scrim) Layout(gtx layout.Context, th *material.Theme, anim *VisibilityAnimation) layout.Dimensions {
	return s.Clickable.Layout(gtx, func(gtx C) D {
		if !anim.Visible() {
			return layout.Dimensions{}
		}
		gtx.Constraints.Min = gtx.Constraints.Max
		currentAlpha := s.FinalAlpha
		if anim.Animating() {
			revealed := anim.Revealed(gtx)
			currentAlpha = uint8(float32(s.FinalAlpha) * revealed)
		}
		color := th.Fg
		color.A = currentAlpha
		fill := WithAlpha(color, currentAlpha)
		paintRect(gtx, gtx.Constraints.Max, fill)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

// ScrimState defines persistent state for a scrim.
type ScrimState struct {
	widget.Clickable
	VisibilityAnimation
}

// ScrimStyle defines how to layout a scrim.
type ScrimStyle struct {
	*ScrimState
	Color      color.NRGBA
	FinalAlpha uint8
}

// NewScrim allocates a ScrimStyle.
// Alpha is the final alpha of a fully "appeared" scrim.
func NewScrim(th *material.Theme, scrim *ScrimState, alpha uint8) ScrimStyle {
	return ScrimStyle{
		ScrimState: scrim,
		Color:      th.Fg,
		FinalAlpha: alpha,
	}
}

func (scrim ScrimStyle) Layout(gtx C) D {
	return scrim.Clickable.Layout(gtx, func(gtx C) D {
		if !scrim.Visible() {
			return D{}
		}
		gtx.Constraints.Min = gtx.Constraints.Max
		alpha := scrim.FinalAlpha
		if scrim.Animating() {
			alpha = uint8(float32(scrim.FinalAlpha) * scrim.Revealed(gtx))
		}
		return Rect{
			Color: WithAlpha(scrim.Color, alpha),
			Size:  gtx.Constraints.Max,
		}.Layout(gtx)
	})
}
