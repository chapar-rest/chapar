// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fixed

import (
	"fmt"
	"strconv"

	"golang.org/x/image/math/fixed"
)

// Int12_20 is a signed 12.20 fixed-point number.
//
// The integer part ranges from -2048 to 2047, inclusive. The
// fractional part has 20 bits of precision.
type Int12_20 uint32

// I12_20 returns the integer value i as an Int12_20.
//
// For example, passing the integer value 2 yields Int12_20(2097152).
func I12_20(v int) Int12_20 {
	return Int12_20(v << 20)
}

// ParseInt12_20 converts the string s to a signed 12.20 fixed-point number.
func ParseInt12_20(s string) (Int12_20, error) {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, err
	}
	return Int12_20(int32(f * (1 << 20))), nil
}

// Float64 converts the 12.20 fixed-point number to a floating point one.
func (x Int12_20) Float64() float64 {
	v := int32(x)
	return float64(v) / (1 << 20)
}

// String returns a human-readable representation of a 12.20 fixed-point number.
func (x Int12_20) String() string {
	const (
		shift = 12
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
	return "-2048:00" // The minimum value is -(1<<(12-1)).
}

// ToInt26_6 converts the 12.20 fixed-point number to a 26.6 one.
func (x Int12_20) ToInt26_6() fixed.Int26_6 {
	f := x.Float64()
	return fixed.Int26_6(f * (1 << 6))
}
