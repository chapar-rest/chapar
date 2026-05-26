package math32

import "math"

// Log10 returns the decimal logarithm of x.
// The special cases are the same as for [Log].
func Log10(x float32) float32 {
	return float32(math.Log10(float64(x)))
}
