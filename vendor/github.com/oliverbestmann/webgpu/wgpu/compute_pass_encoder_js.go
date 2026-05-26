//go:build js

package wgpu

import (
	"syscall/js"
)

// ComputePassDescriptor as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpucomputepassdescriptor
type ComputePassDescriptor struct {
	Label string
}

func (g *ComputePassDescriptor) toJS() any {
	return map[string]any{"label": g.Label}
}

// SetPipeline as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucomputepassencoder-setpipeline
func (g *ComputePassEncoder) SetPipeline(pipeline *ComputePipeline) {
	g.jsValue.Call("setPipeline", pointerToJS(pipeline))
}

// SetBindGroup as described:
// https://gpuweb.github.io/gpuweb/#dom-gpubindingcommandsmixin-setbindgroup
func (g *ComputePassEncoder) SetBindGroup(index uint32, bindGroup *BindGroup, dynamicOffsets []uint32) {
	params := make([]any, 3)
	params[0] = index
	params[1] = pointerToJS(bindGroup)
	params[2] = mapSlice(dynamicOffsets, func(offset uint32) any {
		return offset
	})
	g.jsValue.Call("setBindGroup", params...)
}

// DispatchWorkgroups as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucomputepassencoder-dispatchworkgroups
func (g *ComputePassEncoder) DispatchWorkgroups(workgroupCountX, workgroupCountY, workgroupCountZ uint32) {
	params := make([]any, 3)
	params[0] = workgroupCountX
	if workgroupCountY > 0 {
		params[1] = workgroupCountY
	} else {
		params[1] = js.Undefined()
	}
	if workgroupCountZ > 0 {
		params[2] = workgroupCountZ
	} else {
		params[2] = js.Undefined()
	}
	g.jsValue.Call("dispatchWorkgroups", params...)
}

// TryEnd as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucomputepassencoder-end
func (g *ComputePassEncoder) TryEnd() (err error) {
	defer handleJsException(&err)

	g.jsValue.Call("end")
	return nil
}
