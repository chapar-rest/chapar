package math32

import "math"

// Gamma returns the Gamma function of x.
//
// Special cases are:
//
//	Gamma(+Inf) = +Inf
//	Gamma(+0) = +Inf
//	Gamma(-0) = -Inf
//	Gamma(x) = NaN for integer x < 0
//	Gamma(-Inf) = NaN
//	Gamma(NaN) = NaN
func Gamma(x float32) float32 {
	return float32(math.Gamma(float64(x)))
}
