package lib

/*
#cgo darwin,!ios,amd64 LDFLAGS: -L${SRCDIR}/amd64 -lwgpu_native
#cgo darwin,!ios,arm64 LDFLAGS: -L${SRCDIR}/arm64 -lwgpu_native

#cgo darwin LDFLAGS: -framework Metal -framework QuartzCore
*/
import "C"
