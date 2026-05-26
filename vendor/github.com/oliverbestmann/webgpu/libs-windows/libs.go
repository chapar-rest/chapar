package lib

/*
#cgo windows,amd64 LDFLAGS: -L${SRCDIR}/amd64 -lwgpu_native
#cgo windows,arm64 LDFLAGS: -L${SRCDIR}/arm64 -lwgpu_native
#cgo windows,386 LDFLAGS: -L${SRCDIR}/386 -lwgpu_native

#cgo windows LDFLAGS: -lopengl32 -lgdi32 -ld3dcompiler_47 -lws2_32 -luserenv -lbcrypt -lntdll -loleaut32 -lole32 -lpropsys -lruntimeobject
*/
import "C"
