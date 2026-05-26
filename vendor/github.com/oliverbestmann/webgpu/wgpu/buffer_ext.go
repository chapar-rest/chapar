package wgpu

import "unsafe"

func FromBytes[E any](src []byte) []E {
	l := uintptr(len(src))
	if l == 0 {
		return nil
	}

	var zero E
	elmSize := unsafe.Sizeof(zero)
	if l%elmSize != 0 {
		panic("invalid src")
	}

	return unsafe.Slice((*E)(unsafe.Pointer(&src[0])), l/elmSize)
}

func ToBytes[E any](src []E) []byte {
	l := uintptr(len(src))
	if l == 0 {
		return nil
	}

	elmSize := unsafe.Sizeof(src[0])
	return unsafe.Slice((*byte)(unsafe.Pointer(&src[0])), l*elmSize)
}
