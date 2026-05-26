package lib

/*
#cgo linux,!android,amd64 LDFLAGS: -L${SRCDIR}/amd64 -lwgpu_native
#cgo linux,!android,arm64 LDFLAGS: -L${SRCDIR}/arm64 -lwgpu_native
#cgo linux,!android LDFLAGS: -lm -ldl
*/
import "C"
