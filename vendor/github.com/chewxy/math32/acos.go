package math32

// Acos returns the arccosine, in radians, of x.
//
// Special case is:
//
//	Acos(x) = NaN if x < -1 or x > 1
func Acos(x float32) float32 {
	return Pi/2 - Asin(x)
}
