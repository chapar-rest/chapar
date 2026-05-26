package math32

func Cosh(x float32) float32 {
	x = Abs(x)
	if x > 21 {
		return Exp(x) * 0.5
	}
	ex := Exp(x)
	return (ex + 1./ex) * 0.5
}
