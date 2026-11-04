package theme

import "sync"

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
