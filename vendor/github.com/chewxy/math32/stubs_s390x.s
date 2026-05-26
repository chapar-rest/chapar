//go:build !tinygo && !noasm
#include "textflag.h"

// func archExp(x float32) float32
TEXT ·archExp(SB),NOSPLIT,$0
	BR ·exp(SB)

// func archLog(x float32) float32
TEXT ·archLog(SB),NOSPLIT,$0
	BR ·log(SB)

// func archRemainder(x, y float32) float32
TEXT ·archRemainder(SB),NOSPLIT,$0
	BR ·remainder(SB)

// func archSqrt(x float32) float32
TEXT ·archSqrt(SB),NOSPLIT,$0
	BR ·sqrt(SB)
