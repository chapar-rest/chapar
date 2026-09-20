//go:build !darwin

package yoga

import "time"

// scrollNormalizer maps platform scroll deltas onto wheel units. Only macOS
// mixes notches and pixel-precise deltas on one callback, so elsewhere the
// platform delta already is the unit.
type scrollNormalizer struct{}

func (s *scrollNormalizer) normalize(dx, dy float64, _ time.Time) (float32, float32) {
	return float32(dx), float32(dy)
}
