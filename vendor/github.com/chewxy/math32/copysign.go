package math32

import "math"

// Copysign returns a value with the magnitude of f and the sign of sign.
func Copysign(x, y float32) float32 {
	const sign = 1 << 31
	return math.Float32frombits(math.Float32bits(x)&^sign | math.Float32bits(y)&sign)
}
