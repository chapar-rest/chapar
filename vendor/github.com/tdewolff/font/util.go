package font

import (
	"encoding/binary"
	"fmt"
)

// MaxMemory is the maximum memory that can be allocated by a font.
var MaxMemory uint32 = 30 * 1024 * 1024

// ErrExceedsMemory is returned if the font is malformed.
var ErrExceedsMemory = fmt.Errorf("memory limit exceded")

// ErrInvalidFontData is returned if the font is malformed.
var ErrInvalidFontData = fmt.Errorf("invalid font data")

func calcChecksum(b []byte) uint32 {
	if len(b)%4 != 0 {
		panic("data not multiple of four bytes")
	}
	var sum uint32
	for i := 0; i < len(b); i += 4 {
		sum += binary.BigEndian.Uint32(b[i : i+4])
	}
	return sum
}

// Uint8ToFlags converts a uint8 in 8 booleans from least to most significant.
func Uint8ToFlags(v uint8) (flags [8]bool) {
	for i := 0; i < 8; i++ {
		flags[i] = v&(1<<i) != 0
	}
	return
}

// Uint16ToFlags converts a uint16 in 16 booleans from least to most significant.
func Uint16ToFlags(v uint16) (flags [16]bool) {
	for i := 0; i < 16; i++ {
		flags[i] = v&(1<<i) != 0
	}
	return
}

func flagsToUint8(flags [8]bool) (v uint8) {
	for i := 0; i < 8; i++ {
		if flags[i] {
			v |= 1 << i
		}
	}
	return
}

func uint32ToString(v uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return string(b)
}
