package math32

import "math"

// Lgamma returns the natural logarithm and sign (-1 or +1) of [Gamma](x).
//
// Special cases are:
//
//	Lgamma(+Inf) = +Inf
//	Lgamma(0) = +Inf
//	Lgamma(-integer) = +Inf
//	Lgamma(-Inf) = -Inf
//	Lgamma(NaN) = NaN
func Lgamma(x float32) (lgamma float32, sign int) {
	lg, sign := math.Lgamma(float64(x))
	return float32(lg), sign
}
