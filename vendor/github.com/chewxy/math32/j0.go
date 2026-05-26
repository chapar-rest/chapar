package math32

import "math"

// J0 returns the order-zero Bessel function of the first kind.
//
// Special cases are:
//
//	J0(±Inf) = 0
//	J0(0) = 1
//	J0(NaN) = NaN
func J0(x float32) float32 {
	return float32(math.J0(float64(x)))
}
