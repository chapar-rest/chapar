//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

*/
import "C"

func (g *RenderPipeline) GetBindGroupLayout(groupIndex uint32) *BindGroupLayout {
	ref := C.wgpuRenderPipelineGetBindGroupLayout(g.ref, C.uint32_t(groupIndex))
	if ref == nil {
		panic("Failed to acquire BindGroupLayout")
	}

	return releaseOnGC(&BindGroupLayout{ref: ref})
}
