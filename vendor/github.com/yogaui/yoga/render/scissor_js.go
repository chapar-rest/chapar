//go:build !nogpu && js

package render

import (
	"syscall/js"
	"unsafe"

	"github.com/cogentcore/webgpu/wgpu"
)

// renderPassJS mirrors the js build of wgpu.RenderPassEncoder (single js.Value).
type renderPassJS struct {
	jsValue js.Value
}

func (r *Renderer) setScissor(pass *wgpu.RenderPassEncoder, clip Rect) {
	x, y, w, h := r.scissorRect(clip)
	p := (*renderPassJS)(unsafe.Pointer(pass))
	p.jsValue.Call("setScissorRect", x, y, w, h)
}

func (r *Renderer) scissorRect(clip Rect) (x, y, w, h uint32) {
	fbW, fbH := r.config.Width, r.config.Height
	if clip.W < 0 || clip.H < 0 {
		return 0, 0, fbW, fbH
	}
	x0 := clampU32(clip.X*r.scaleX, fbW)
	y0 := clampU32(clip.Y*r.scaleY, fbH)
	x1 := clampU32((clip.X+clip.W)*r.scaleX, fbW)
	y1 := clampU32((clip.Y+clip.H)*r.scaleY, fbH)
	return x0, y0, x1 - x0, y1 - y0
}

func clampU32(v float32, hi uint32) uint32 {
	if v <= 0 {
		return 0
	}
	u := uint32(v + 0.5)
	if u > hi {
		return hi
	}
	return u
}
