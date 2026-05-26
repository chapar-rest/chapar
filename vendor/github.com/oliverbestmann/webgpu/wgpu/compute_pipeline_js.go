//go:build js

package wgpu

// ComputePipelineDescriptor as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpucomputepipelinedescriptor
type ComputePipelineDescriptor struct {
	Layout  *PipelineLayout
	Compute ProgrammableStageDescriptor
}

func (g ComputePipelineDescriptor) toJS() any {
	result := make(map[string]any)
	if g.Layout != nil {
		result["layout"] = pointerToJS(g.Layout)
	} else {
		result["layout"] = "auto"
	}
	result["compute"] = g.Compute.toJS()
	return result
}

func (g *ComputePipeline) GetBindGroupLayout(idx int) *BindGroupLayout {
	jsValue := g.jsValue.Call("getBindGroupLayout", idx)
	return &BindGroupLayout{jsValue}
}
