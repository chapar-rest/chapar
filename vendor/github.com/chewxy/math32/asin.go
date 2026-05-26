package math32

func Asin(x float32) float32 {
	if x == 0 {
		return x // special case
	}
	sign := false
	if x < 0 {
		x = -x
		sign = true
	}
	if x > 1 {
		return NaN() // special case
	}

	temp := Sqrt(1 - x*x)
	if x > 0.7 {
		temp = Pi/2 - satan(temp/x)
	} else {
		temp = satan(x / temp)
	}

	if sign {
		temp = -temp
	}
	return temp
}
