package yoga

import (
	"math"
	"time"

	"github.com/mirzakhany/yoga/ui"
)

// gestureIdle is how long a gap in the scroll stream ends a gesture. AppKit
// delivers a trackpad swing (and its momentum tail) as a continuous stream, so
// a pause this long means the next delta starts a fresh gesture.
const gestureIdle = 200 * time.Millisecond

// scrollNormalizer converts GLFW scroll deltas into wheel units.
//
// GLFW's cocoa backend reports a mouse wheel notch as ±1, but pre-scales a
// trackpad's precise deltas by 0.1 — so for a trackpad it hands back pixels/10.
// Taking those for notches scrolls ui.WheelPixelsPerUnit per 0.1px of finger
// travel, 4.2x too far.
//
// GLFW exposes no -[NSEvent hasPreciseScrollingDeltas], so a fractional delta
// stands in for it: notches arrive as whole numbers, pixel deltas rarely do. A
// precise gesture can still land on a whole number mid-swing, which would make
// one frame jump 4.2x, so the first fractional delta latches the whole gesture
// as precise until the stream goes idle.
type scrollNormalizer struct {
	precise bool
	last    time.Time
}

// normalize maps one platform delta onto wheel units. now is the event time.
func (s *scrollNormalizer) normalize(dx, dy float64, now time.Time) (float32, float32) {
	if now.Sub(s.last) > gestureIdle {
		s.precise = false
	}
	s.last = now
	if fractional(dx) || fractional(dy) {
		s.precise = true
	}
	if !s.precise {
		return float32(dx), float32(dy)
	}
	// Undo GLFW's 0.1 scale to recover pixels, then express those pixels in
	// wheel units so the ui layer's units*WheelPixelsPerUnit lands 1:1 with the
	// distance the fingers moved.
	const pxPerDelta = 10.0
	return float32(dx * pxPerDelta / ui.WheelPixelsPerUnit),
		float32(dy * pxPerDelta / ui.WheelPixelsPerUnit)
}

func fractional(v float64) bool { return v != math.Trunc(v) }
