package math32

import "math"

// Erf returns the error function of x.
//
// Special cases are:
//
//	Erf(+Inf) = 1
//	Erf(-Inf) = -1
//	Erf(NaN) = NaN
func Erf(x float32) float32 {
	return float32(math.Erf(float64(x)))
}

// Erfinv returns the inverse error function of x.
//
// Special cases are:
//
//	Erfinv(1) = +Inf
//	Erfinv(-1) = -Inf
//	Erfinv(x) = NaN if x < -1 or x > 1
//	Erfinv(NaN) = NaN
func Erfinv(x float32) float32 {
	return float32(math.Erfinv(float64(x)))
}
