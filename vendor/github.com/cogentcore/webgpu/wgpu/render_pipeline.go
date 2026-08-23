//go:build !js

package wgpu

/*

#include <stdlib.h>
#include "./lib/wgpu.h"

*/
import "C"

type RenderPipeline struct {
	ref C.WGPURenderPipeline
}

func (p *RenderPipeline) GetBindGroupLayout(groupIndex uint32) *BindGroupLayout {
	ref := C.wgpuRenderPipelineGetBindGroupLayout(p.ref, C.uint32_t(groupIndex))
	if ref == nil {
		panic("Failed to accquire BindGroupLayout")
	}

	return &BindGroupLayout{ref}
}

func (p *RenderPipeline) Release() {
	C.wgpuRenderPipelineRelease(p.ref)
}
