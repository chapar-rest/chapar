package libs

/*
#cgo android,amd64 LDFLAGS: -L${SRCDIR}/amd64 -lwgpu_native
#cgo android,386 LDFLAGS: -L${SRCDIR}/386 -lwgpu_native
#cgo android,arm64 LDFLAGS: -L${SRCDIR}/arm64 -lwgpu_native
#cgo android,arm LDFLAGS: -L${SRCDIR}/arm -lwgpu_native

#cgo android LDFLAGS: -landroid -lm -llog
*/
import "C"
