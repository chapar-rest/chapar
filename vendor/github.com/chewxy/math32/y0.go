package math32

import "math"

// Y0 returns the order-zero Bessel function of the second kind.
//
// Special cases are:
//
//	Y0(+Inf) = 0
//	Y0(0) = -Inf
//	Y0(x < 0) = NaN
//	Y0(NaN) = NaN
func Y0(x float32) float32 {
	return float32(math.Y0(float64(x)))
}
