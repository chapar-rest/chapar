package math32

import "math"

// Erfc returns the complementary error function of x.
//
// Special cases are:
//
//	Erfc(+Inf) = 0
//	Erfc(-Inf) = 2
//	Erfc(NaN) = NaN
func Erfc(x float32) float32 {
	return float32(math.Erfc(float64(x)))
}

// Erfcinv returns the inverse of [Erfc](x).
//
// Special cases are:
//
//	Erfcinv(0) = +Inf
//	Erfcinv(2) = -Inf
//	Erfcinv(x) = NaN if x < 0 or x > 2
//	Erfcinv(NaN) = NaN
func Erfcinv(x float32) float32 {
	return float32(math.Erfcinv(float64(x)))
}
