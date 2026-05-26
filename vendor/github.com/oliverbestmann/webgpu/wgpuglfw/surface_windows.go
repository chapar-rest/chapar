//go:build windows

package wgpuglfw

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/oliverbestmann/webgpu/wgpu"
)

/*

#include <windows.h>

*/
import "C"

func GetSurfaceDescriptor(w *glfw.Window) *wgpu.SurfaceDescriptor {
	return &wgpu.SurfaceDescriptor{
		WindowsHWND: &wgpu.SurfaceSourceWindowsHWND{
			Hwnd:      unsafe.Pointer(w.GetWin32Window()),
			Hinstance: unsafe.Pointer(C.GetModuleHandle(nil)),
		},
	}
}
