//go:build !js

package wgpu

/*

#include <stdlib.h>
#include "./lib/wgpu.h"

*/
import "C"

type ComputePipeline struct {
	ref C.WGPUComputePipeline
}

func (p *ComputePipeline) GetBindGroupLayout(groupIndex uint32) *BindGroupLayout {
	ref := C.wgpuComputePipelineGetBindGroupLayout(p.ref, C.uint32_t(groupIndex))
	if ref == nil {
		panic("Failed to accquire BindGroupLayout")
	}

	return &BindGroupLayout{ref}
}

func (p *ComputePipeline) Release() {
	C.wgpuComputePipelineRelease(p.ref)
}
