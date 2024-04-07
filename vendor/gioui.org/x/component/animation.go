package component

import (
	"fmt"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
)

// VisibilityAnimation holds the animation state for animations that transition between a
// "visible" and "invisible" state for a fixed duration of time.
type VisibilityAnimation struct {
	// How long does the animation last
	time.Duration
	State   VisibilityAnimationState
	Started time.Time
}

// Revealed returns the fraction of the animated entity that should be revealed at the current
// time in the animation. This fraction is computed with linear interpolation.
//
// Revealed should be invoked during every frame that v.Animating() returns true.
//
// If the animation reaches its end this frame, Revealed will transition it to a non-animating
// state automatically.
//
// If the animation is in the process of animating, calling Revealed will automatically add
// an InvalidateOp to the provided layout.Context to ensure that the next frame will be generated
// promptly.
func (v *VisibilityAnimation) Revealed(gtx layout.Context) float32 {
	if v.Animating() {
		gtx.Execute(op.InvalidateCmd{})
	}
	if v.Duration == time.Duration(0) {
		v.Duration = time.Second
	}
	progress := float32(gtx.Now.Sub(v.Started).Milliseconds()) / float32(v.Milliseconds())
	if progress >= 1 {
		if v.State == Appearing {
			v.State = Visible
		} else if v.State == Disappearing {
			v.State = Invisible
		}
	}
	switch v.State {
	case Visible:
		return 1
	case Invisible:
		return 0
	case Appearing:
		return progress
	case Disappearing:
		return 1 - progress
	}
	return progress
}

// Visible() returns whether any part of the animated entity should be visible during the
// current animation frame.
func (v VisibilityAnimation) Visible() bool {
	return v.State != Invisible
}

// Animating() returns whether the animation is either in the process of appearsing or
// disappearing.
func (v VisibilityAnimation) Animating() bool {
	return v.State == Appearing || v.State == Disappearing
}

// Appear triggers the animation to begin becoming visible at the provided time. It is
// a no-op if the animation is already visible.
func (v *VisibilityAnimation) Appear(now time.Time) {
	if !v.Visible() && !v.Animating() {
		v.State = Appearing
		v.Started = now
	}
}

// Disappear triggers the animation to begin becoming invisible at the provided time.
// It is a no-op if the animation is already invisible.
func (v *VisibilityAnimation) Disappear(now time.Time) {
	if v.Visible() && !v.Animating() {
		v.State = Disappearing
		v.Started = now
	}
}

// ToggleVisibility will make an invisible animation begin the process of becoming
// visible and a visible animation begin the process of disappearing.
func (v *VisibilityAnimation) ToggleVisibility(now time.Time) {
	if v.Visible() {
		v.Disappear(now)
	} else {
		v.Appear(now)
	}
}

func (v *VisibilityAnimation) String(gtx layout.Context) string {
	return fmt.Sprintf(
		"State: %v, Revealed: %f, Duration: %v, Started: %v",
		v.State,
		v.Revealed(gtx),
		v.Duration,
		v.Started.Local(),
	)
}

// VisibilityAnimationState represents possible states that a VisibilityAnimation can
// be in.
type VisibilityAnimationState int

const (
	Visible VisibilityAnimationState = iota
	Disappearing
	Appearing
	Invisible
)

func (v VisibilityAnimationState) String() string {
	switch v {
	case Visible:
		return "visible"
	case Disappearing:
		return "disappearing"
	case Appearing:
		return "appearing"
	case Invisible:
		return "invisible"
	default:
		return "invalid VisibilityAnimationState"
	}
}

// Progress is an animation primitive that tracks progress of time over a fixed
// duration as a float between [0, 1].
//
// Progress is reversable.
//
// Widgets map async UI events to state changes: stop, forward, reverse.
// Widgets then interpolate visual data based on progress value.
//
// Update method must be called every tick to update the progress value.
type Progress struct {
	progress  float32
	duration  time.Duration
	began     time.Time
	direction ProgressDirection
	active    bool
}

// ProgressDirection specifies how to update progress every tick.
type ProgressDirection int

const (
	// Forward progresses from 0 to 1.
	Forward ProgressDirection = iota
	// Reverse progresses from 1 to 0.
	Reverse
)

// Progress reports the current progress as a float between [0, 1].
func (p Progress) Progress() float32 {
	if p.progress < 0.0 {
		return 0.0
	}
	if p.progress > 1.0 {
		return 1.0
	}
	return p.progress
}

// Absolute reports the absolute progress, ignoring direction.
func (p Progress) Absolute() float32 {
	if p.direction == Forward {
		return p.Progress()
	}
	return 1 - p.Progress()
}

// Direction reports the current direction.
func (p Progress) Direction() ProgressDirection {
	return p.direction
}

// Started reports true if progression has started.
func (p Progress) Started() bool {
	return p.active
}

func (p Progress) Finished() bool {
	switch p.direction {
	case Forward:
		return p.progress >= 1.0
	case Reverse:
		return p.progress <= 0.0
	}
	return false
}

// Start the progress in the given direction over the given duration.
func (p *Progress) Start(began time.Time, direction ProgressDirection, duration time.Duration) {
	if !p.active {
		p.active = true
		p.began = began
		p.direction = direction
		p.duration = duration
		p.Update(began)
	}
}

// Stop the progress.
func (p *Progress) Stop() {
	p.active = false
}

func (p *Progress) Update(now time.Time) {
	if !p.Started() || p.Finished() {
		p.Stop()
		return
	}
	var (
		elapsed = now.Sub(p.began).Milliseconds()
		total   = p.duration.Milliseconds()
	)
	switch p.direction {
	case Forward:
		p.progress = float32(elapsed) / float32(total)
	case Reverse:
		p.progress = 1 - float32(elapsed)/float32(total)
	}
}

func (d ProgressDirection) String() string {
	switch d {
	case Forward:
		return "forward"
	case Reverse:
		return "reverse"
	}
	return "unknown"
}
