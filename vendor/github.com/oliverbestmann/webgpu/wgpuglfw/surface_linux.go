//go:build (linux || freebsd || netbsd || openbsd) && !android && ((!x11 && !wayland) || (x11 && wayland))

package wgpuglfw

import "C"
import (
	"unsafe"

	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/oliverbestmann/webgpu/wgpu"
)

/*
extern void* glfwGetWaylandDisplay();
extern void* glfwGetWaylandWindow(void *win);
*/
import "C"

func GetSurfaceDescriptor(w *glfw.Window) *wgpu.SurfaceDescriptor {
	switch glfw.GetPlatform() {
	case glfw.PlatformX11:
		return &wgpu.SurfaceDescriptor{
			XlibWindow: &wgpu.SurfaceSourceXlibWindow{
				Display: unsafe.Pointer(glfw.GetX11Display()),
				Window:  uint32(w.GetX11Window()),
			},
		}

	case glfw.PlatformWayland:
		return &wgpu.SurfaceDescriptor{
			WaylandSurface: &wgpu.SurfaceSourceWaylandSurface{
				// TODO this is the proper way once the fix is merged:
				//  https://github.com/go-gl/glfw/pull/420
				// Display: unsafe.Pointer(glfw.GetWaylandDisplay()),
				// Surface: unsafe.Pointer(w.GetWaylandWindow()),
				Display: unsafe.Pointer(C.glfwGetWaylandDisplay()),
				Surface: unsafe.Pointer(C.glfwGetWaylandWindow(w.Handle())),
			},
		}

	default:
		panic("unsupported glfw platform")
	}
}
