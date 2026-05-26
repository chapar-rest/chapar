//go:build (linux || freebsd || netbsd || openbsd) && !android && !x11 && wayland

package wgpuglfw

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func GetSurfaceDescriptor(w *glfw.Window) *wgpu.SurfaceDescriptor {
	switch glfw.GetPlatform() {

	case glfw.PlatformWayland:
		return &wgpu.SurfaceDescriptor{
			WaylandSurface: &wgpu.SurfaceSourceWaylandSurface{
				Display: unsafe.Pointer(glfw.GetWaylandDisplay()),
				Surface: unsafe.Pointer(w.GetWaylandWindow()),
			},
		}

	default:
		panic("Unsupported glfw platform. To support both x11 and wayland, build with --tags wayland,x11")
	}
}
