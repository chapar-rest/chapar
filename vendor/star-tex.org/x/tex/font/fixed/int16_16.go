// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fixed

import (
	"fmt"
	"strconv"

	"golang.org/x/image/math/fixed"
)

// Int16_16 is a signed 16.16 fixed-point number.
//
// The integer part ranges from -32768 to 32767, inclusive. The
// fractional part has 16 bits of precision.
type Int16_16 uint32

// I16_16 returns the integer value i as an Int16_16.
//
// For example, passing the integer value 2 yields Int16_16(131072).
func I16_16(v int) Int16_16 {
	return Int16_16(v << 16)
}

// ParseInt16_16 converts the string s to a signed 16.16 fixed-point number.
func ParseInt16_16(s string) (Int16_16, error) {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, err
	}
	return Int16_16(int32(f * (1 << 16))), nil
}

// Float64 converts the 16.16 fixed-point number to a floating point one.
func (x Int16_16) Float64() float64 {
	v := int32(x)
	return float64(v) / (1 << 16)
}

// String returns a human-readable representation of a 16.16 fixed-point number.
func (x Int16_16) String() string {
	const (
		shift = 16
		mask  = 1<<shift - 1
	)
	xx := int32(x)
	if xx >= 0 {
		return fmt.Sprintf("%d:%02d", int32(xx>>shift), int32(xx&mask))
	}
	xx = -xx
	if xx >= 0 {
		return fmt.Sprintf("-%d:%02d", int32(xx>>shift), int32(xx&mask))
	}
	return "-32768:00" // The minimum value is -(1<<(16-1)).
}

// ToInt26_6 converts the 16.16 fixed-point number to a 26.6 one.
func (x Int16_16) ToInt26_6() fixed.Int26_6 {
	f := x.Float64()
	return fixed.Int26_6(f * (1 << 6))
}
