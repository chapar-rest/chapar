package theme

import (
	"sync"
	"time"
)

// Motion holds the durations of UI transitions, so they move at one pace
// across widgets: Fast for small state changes (a check mark, a fade in),
// Normal for things that open and close in place (a fold, a drawer, a menu),
// Slow for larger moves (a dialog or toast entering).
type Motion struct {
	Fast   time.Duration
	Normal time.Duration
	Slow   time.Duration
}

// DefaultMotion returns Yoga's transition durations.
func DefaultMotion() Motion {
	return Motion{
		Fast:   100 * time.Millisecond,
		Normal: 180 * time.Millisecond,
		Slow:   240 * time.Millisecond,
	}
}

// Duration returns d, or 0 when ReducedMotion is on: callers treat 0 as
// "jump to the end".
func (m Motion) Duration(d time.Duration) time.Duration {
	if ReducedMotion() {
		return 0
	}
	return d
}

// EaseEnter eases things appearing or opening: fast out of the start, gentle
// into place (cubic ease-out). t and the result run from 0 to 1.
func EaseEnter(t float32) float32 {
	u := 1 - clamp01(t)
	return 1 - u*u*u
}

// EaseExit eases things leaving or closing: a gentle start that speeds away
// (cubic ease-in).
func EaseExit(t float32) float32 {
	t = clamp01(t)
	return t * t * t
}

// EaseStandard eases a move between two on-screen positions (cubic
// ease-in-out).
func EaseStandard(t float32) float32 {
	t = clamp01(t)
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := -2*t + 2
	return 1 - u*u*u/2
}

func clamp01(t float32) float32 {
	return min(max(t, 0), 1)
}

var (
	motionOnce    sync.Once
	motionReduced bool
	motionSet     bool
)

// ReducedMotion reports whether animations should be skipped: the user asked
// the operating system to reduce motion, or the app said so with
// SetReducedMotion. The system setting is read once.
func ReducedMotion() bool {
	if motionSet {
		return motionReduced
	}
	motionOnce.Do(func() { motionReduced = osPrefersReducedMotion() })
	return motionReduced
}

// SetReducedMotion overrides the system's reduce-motion setting, for an
// app's own preference.
func SetReducedMotion(v bool) { motionReduced, motionSet = v, true }
