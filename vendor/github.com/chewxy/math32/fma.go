package math32

import "math/bits"

// stickyRSh right-shifts x by s bits, setting the LSB if any shifted-out bits
// were nonzero (the "sticky" bit used for correct IEEE 754 rounding).
func stickyRSh(x uint64, s uint) uint64 {
	if s >= 64 {
		if x != 0 {
			return 1
		}
		return 0
	}
	result := x >> s
	if result<<s != x {
		result |= 1
	}
	return result
}

// FMA returns x * y + z, computed with only one rounding.
// (That is, FMA returns the fused multiply-add of x, y, and z.)
//
// Ported from fma_model_ieee in
// https://github.com/tenstorrent/tt-isa-documentation/blob/main/Miscellaneous/FMA/fma.c
// (Apache-2.0).
func FMA(x, y, z float32) float32 {
	bx := Float32bits(x)
	by := Float32bits(y)
	bz := Float32bits(z)

	// Unpack x: biased exponent and 24-bit mantissa (with implicit leading 1).
	// Subnormals are normalised so the rest of the algorithm is uniform.
	xE := int32(bx>>23) & 0xFF
	xM := uint64(bx&0x7FFFFF) ^ 0x800000
	if xE == 0 {
		xM ^= 0x800000
		xE = 9 - int32(bits.LeadingZeros32(uint32(xM)|1))
		xM <<= uint(1 - xE)
	}

	// Unpack y.
	yE := int32(by>>23) & 0xFF
	yM := uint64(by&0x7FFFFF) ^ 0x800000
	if yE == 0 {
		yM ^= 0x800000
		yE = 9 - int32(bits.LeadingZeros32(uint32(yM)|1))
		yM <<= uint(1 - yE)
	}

	// Unpack z (keep original sign separately; it is not used in the mantissa).
	zE := int32(bz>>23) & 0xFF
	zM := uint64(bz&0x7FFFFF) ^ 0x800000
	zSign := bz & 0x80000000
	if zE == 0 {
		zM ^= 0x800000
		zE = 9 - int32(bits.LeadingZeros32(uint32(zM)|1))
		zM <<= uint(1 - zE)
	}

	// Compute p = x * y.
	pSign := (bx ^ by) & 0x80000000
	pM := xM * yM // 48-bit exact product of two 24-bit mantissas
	pE := xE + yE - 23 - 127

	// Append three guard/round/sticky bits to both mantissas so the final
	// round-to-nearest-even step can inspect them cheaply.
	pM <<= 3
	zM <<= 3

	// Handle NaN and Inf inputs.  At this point zM has been left-shifted by 3,
	// so Inf maps to 0x4000000 (0x800000 << 3).
	if xE == 255 || yE == 255 || zE == 255 {
		isNaN := (xE == 255 && (xM != 0x800000 || yM == 0)) || // x is NaN, or Inf*0
			(yE == 255 && (yM != 0x800000 || xM == 0)) || // y is NaN, or 0*Inf
			(zE == 255 && zM != 0x4000000) || // z is NaN
			(zE == 255 && (xE == 255 || yE == 255) && zSign != pSign) // ±Inf−Inf
		if isNaN {
			return Float32frombits(uvnan)
		}
		if zE == 255 {
			return z
		}
		return Float32frombits(pSign | uvinf)
	}

	// Realign zM so that it shares the same exponent scale as pM.
	// After this, zE is directly comparable to pE for alignment shifts.
	zM <<= 23
	zE -= 23

	// If x or y was zero, pM is zero; return z (or signed zero).
	if pM == 0 {
		if zM != 0 {
			return z
		}
		return Float32frombits(zSign & pSign)
	}

	// Bring p and z to a common exponent rE = max(pE, zE), discarding low bits
	// of the smaller operand into the sticky bit.
	rE := pE
	if zE > pE {
		rE = zE
	}
	if pE < rE {
		pM = stickyRSh(pM, uint(rE-pE))
	}
	if zE < rE {
		zM = stickyRSh(zM, uint(rE-zE))
	}

	// The result sign follows the operand with larger magnitude.
	rSign := pSign
	if pM < zM {
		rSign = zSign
	}

	// Perform signed addition via two's complement: negate whichever operand
	// has the opposite sign to the result (one's complement + carry-in below).
	if zSign != rSign {
		zM = ^zM
	}
	if pSign != rSign {
		pM = ^pM
	}
	carryIn := uint64(0)
	if pSign != zSign {
		carryIn = 1 // completes two's complement when signs differ
	}
	rM := zM + pM + carryIn

	// Exact cancellation → signed zero.
	if rM == 0 {
		return Float32frombits(zSign & pSign)
	}

	// Normalise rM to the target layout:
	//   [63..27] = 37 leading zero bits
	//   [26]     = implicit leading 1
	//   [25..3]  = 23 mantissa bits
	//   [2..0]   = 3 GRS bits
	n := int32(37) - int32(bits.LeadingZeros64(rM))
	rE += n

	if rE >= 255 {
		return Float32frombits(rSign | uvinf)
	}

	if rE <= 0 { // result is subnormal; shift mantissa further right
		n += 1 - rE
		rE = 0
	}

	if n <= 0 {
		rM <<= uint(-n)
	} else {
		rM = stickyRSh(rM, uint(n))
	}

	// Pack exponent and mantissa (strip the implicit 1 and the 3 GRS bits).
	r := uint32(rE)<<23 | (uint32(rM>>3) & 0x7FFFFF)

	// Round to nearest, ties to even: round up when (GRS + LSB_of_mantissa) > 4.
	if (rM&7)+uint64(r&1) > 4 {
		r++ // carry may propagate naturally into the exponent field
	}

	return Float32frombits(rSign | r)
}
