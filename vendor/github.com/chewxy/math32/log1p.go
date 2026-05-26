package math32

import "math"

// Log1p returns the natural logarithm of 1 plus its argument x.
// It is more accurate than [Log](1 + x) when x is near zero.
//
// Special cases are:
//
//	Log1p(+Inf) = +Inf
//	Log1p(±0) = ±0
//	Log1p(-1) = -Inf
//	Log1p(x < -1) = NaN
//	Log1p(NaN) = NaN
func Log1p(x float32) float32 {
	return float32(math.Log1p(float64(x)))
}
