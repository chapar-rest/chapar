//go:build windows

package wgpuglfw

import (
	"unsafe"

	"github.com/cogentcore/webgpu/wgpu"
	"github.com/go-gl/glfw/v3.3/glfw"
)

/*

#include <windows.h>

*/
import "C"

func GetSurfaceDescriptor(w *glfw.Window) *wgpu.SurfaceDescriptor {
	return &wgpu.SurfaceDescriptor{
		WindowsHWND: &wgpu.SurfaceDescriptorFromWindowsHWND{
			Hwnd:      unsafe.Pointer(w.GetWin32Window()),
			Hinstance: unsafe.Pointer(C.GetModuleHandle(nil)),
		},
	}
}
