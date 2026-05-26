//go:build !tinygo && !noasm
#include "textflag.h"

// func archLog(x float64) float64
TEXT ·archLog(SB),NOSPLIT,$0
	B ·log(SB)

// func archRemainder(x, y float32) float32 // TODO
// TEXT ·archRemainderTODO(SB),NOSPLIT,$0
//	B ·remainder(SB)
