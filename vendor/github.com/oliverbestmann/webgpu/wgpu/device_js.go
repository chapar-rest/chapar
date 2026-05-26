//go:build js

package wgpu

// GetQueue returns a Queue as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-queue
func (g *Device) GetQueue() *Queue {
	jsQueue := g.jsValue.Get("queue")
	return &Queue{jsValue: jsQueue}
}

func (g *Device) TryCreateQuerySet(descriptor *QuerySetDescriptor) (_ *QuerySet, err error) {
	defer handleJsException(&err)

	jsQuerySet := g.jsValue.Call("createQuerySet", pointerToJS(descriptor))
	return &QuerySet{jsQuerySet}, nil
}

// TryCreateCommandEncoder as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createcommandencoder
func (g *Device) TryCreateCommandEncoder(descriptor *CommandEncoderDescriptor) (_ *CommandEncoder, err error) {
	defer handleJsException(&err)

	jsEncoder := g.jsValue.Call("createCommandEncoder", pointerToJS(descriptor))
	return &CommandEncoder{jsValue: jsEncoder}, nil
}

// TryCreateBuffer as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createbuffer
func (g *Device) TryCreateBuffer(descriptor *BufferDescriptor) (_ *Buffer, err error) {
	defer handleJsException(&err)

	jsBuffer := g.jsValue.Call("createBuffer", pointerToJS(descriptor))
	return &Buffer{jsValue: jsBuffer}, nil
}

// TryCreateShaderModule as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createshadermodule
func (g *Device) TryCreateShaderModule(desc *ShaderModuleDescriptor) (_ *ShaderModule, err error) {
	defer handleJsException(&err)

	jsShader := g.jsValue.Call("createShaderModule", pointerToJS(desc))
	return &ShaderModule{jsValue: jsShader}, nil
}

// TryCreateRenderPipeline as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createrenderpipeline
func (g *Device) TryCreateRenderPipeline(descriptor *RenderPipelineDescriptor) (_ *RenderPipeline, err error) {
	defer handleJsException(&err)

	jsPipeline := g.jsValue.Call("createRenderPipeline", pointerToJS(descriptor))
	return &RenderPipeline{jsValue: jsPipeline}, nil
}

// TryCreateBindGroup as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createbindgroup
func (g *Device) TryCreateBindGroup(descriptor *BindGroupDescriptor) (_ *BindGroup, err error) {
	defer handleJsException(&err)

	jsBindGroup := g.jsValue.Call("createBindGroup", pointerToJS(descriptor))
	return &BindGroup{jsValue: jsBindGroup}, nil
}

// TryCreateBindGroupLayout as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createbindgrouplayout
func (g *Device) TryCreateBindGroupLayout(descriptor *BindGroupLayoutDescriptor) (_ *BindGroupLayout, err error) {
	defer handleJsException(&err)

	jsLayout := g.jsValue.Call("createBindGroupLayout", pointerToJS(descriptor))
	return &BindGroupLayout{jsValue: jsLayout}, nil
}

// TryCreatePipelineLayout as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createpipelinelayout
func (g *Device) TryCreatePipelineLayout(descriptor *PipelineLayoutDescriptor) (_ *PipelineLayout, err error) {
	defer handleJsException(&err)

	jsLayout := g.jsValue.Call("createPipelineLayout", pointerToJS(descriptor))
	return &PipelineLayout{jsValue: jsLayout}, nil
}

// TryCreateComputePipeline as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createcomputepipeline
func (g *Device) TryCreateComputePipeline(descriptor *ComputePipelineDescriptor) (_ *ComputePipeline, err error) {
	defer handleJsException(&err)

	jsPipeline := g.jsValue.Call("createComputePipeline", pointerToJS(descriptor))
	return &ComputePipeline{jsValue: jsPipeline}, nil
}

// TryCreateTexture as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createtexture
func (g *Device) TryCreateTexture(descriptor *TextureDescriptor) (_ *Texture, err error) {
	defer handleJsException(&err)

	jsTexture := g.jsValue.Call("createTexture", pointerToJS(descriptor))
	return &Texture{jsValue: jsTexture}, nil
}

// TryCreateSampler as described:
// https://gpuweb.github.io/gpuweb/#dom-gpudevice-createsampler
func (g *Device) TryCreateSampler(descriptor *SamplerDescriptor) (_ *Sampler, err error) {
	defer handleJsException(&err)

	jsSampler := g.jsValue.Call("createSampler", pointerToJS(descriptor))
	return &Sampler{jsValue: jsSampler}, nil
}

func (g *Device) TryCreateRenderBundleEncoder(descriptor *RenderBundleEncoderDescriptor) (_ *RenderBundleEncoder, err error) {
	defer handleJsException(&err)

	jsValue := g.jsValue.Call("createRenderBundleEncoder", descriptor.toJS())
	return &RenderBundleEncoder{jsValue: jsValue}, nil
}

func (g *Device) GetLimits() Limits {
	return limitsFromJS(g.jsValue.Get("limits"))
}

func (g *Device) Poll(wait bool, wrappedSubmissionIndex *uint64) (queueEmpty bool) {
	return false // no-op
}
