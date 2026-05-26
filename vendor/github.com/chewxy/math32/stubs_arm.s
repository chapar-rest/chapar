//go:build !tinygo && !noasm
#include "textflag.h"

// func archExp(x float32) float32
TEXT ·archExp(SB),NOSPLIT,$0
	B ·exp(SB)

// func archExp2(x float32) float32
TEXT ·archExp2(SB),NOSPLIT,$0
	B ·exp2(SB)

// func archLog(x float32) float32
TEXT ·archLog(SB),NOSPLIT,$0
	B ·log(SB)

// func archRemainder(x, y float32) float32
TEXT ·archRemainder(SB),NOSPLIT,$0
	B ·remainder(SB)

// func archSqrt(x float32) float32
TEXT ·archSqrt(SB),NOSPLIT,$0
	B ·sqrt(SB)
