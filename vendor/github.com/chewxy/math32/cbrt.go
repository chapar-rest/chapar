package math32

import "math"

// Cbrt returns the cube root of x.
//
// Special cases are:
//
//	Cbrt(±0) = ±0
//	Cbrt(±Inf) = ±Inf
//	Cbrt(NaN) = NaN
func Cbrt(x float32) float32 {
	return float32(math.Cbrt(float64(x)))
}
