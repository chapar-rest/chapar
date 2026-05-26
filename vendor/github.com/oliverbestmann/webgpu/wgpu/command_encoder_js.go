//go:build js

package wgpu

// BeginRenderPass as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-beginrenderpass
func (g *CommandEncoder) BeginRenderPass(descriptor *RenderPassDescriptor) *RenderPassEncoder {
	jsRenderPass := g.jsValue.Call("beginRenderPass", pointerToJS(descriptor))
	return &RenderPassEncoder{
		jsValue: jsRenderPass,
	}
}

// BeginComputePass as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-begincomputepass
func (g *CommandEncoder) BeginComputePass(descriptor *ComputePassDescriptor) *ComputePassEncoder {
	params := make([]any, 1)
	params[0] = pointerToJS(descriptor)
	jsComputePass := g.jsValue.Call("beginComputePass", params...)
	return &ComputePassEncoder{
		jsValue: jsComputePass,
	}
}

// TryCopyBufferToBuffer as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-copybuffertobuffer
func (g *CommandEncoder) TryCopyBufferToBuffer(source *Buffer, sourceOffset uint64, destination *Buffer, destinationOffset uint64, size uint64) (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("copyBufferToBuffer", pointerToJS(source), sourceOffset, pointerToJS(destination), destinationOffset, size)
	return nil
}

// TryCopyBufferToTexture as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-copybuffertotexture
func (g *CommandEncoder) TryCopyBufferToTexture(source *TexelCopyBufferInfo, destination *TexelCopyTextureInfo, copySize *Extent3D) (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("copyBufferToTexture", pointerToJS(source), pointerToJS(destination), pointerToJS(copySize))
	return nil
}

// TryCopyTextureToBuffer as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-copytexturetobuffer
func (g *CommandEncoder) TryCopyTextureToBuffer(source *TexelCopyTextureInfo, destination *TexelCopyBufferInfo, copySize *Extent3D) (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("copyTextureToBuffer", pointerToJS(source), pointerToJS(destination), pointerToJS(copySize))
	return nil
}

// TryCopyTextureToTexture as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-copytexturetotexture
func (g *CommandEncoder) TryCopyTextureToTexture(source *TexelCopyTextureInfo, destination *TexelCopyTextureInfo, copySize *Extent3D) (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("copyTextureToTexture", pointerToJS(source), pointerToJS(destination), pointerToJS(copySize))
	return nil
}

// TryFinish as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-finish
func (g *CommandEncoder) TryFinish(descriptor *CommandBufferDescriptor) (_ *CommandBuffer, err error) {
	defer handleJsException(&err)
	jsBuffer := g.jsValue.Call("finish", pointerToJS(descriptor))
	return &CommandBuffer{
		jsValue: jsBuffer,
	}, nil
}

func (g *CommandEncoder) TryClearBuffer(buffer *Buffer, offset uint64, size uint64) (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("clearBuffer", buffer.toJS(), offset, size)
	return nil
}

func (g *CommandEncoder) TryInsertDebugMarker(label string) (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("insertDebugMarker", label)
	return nil
}

func (g *CommandEncoder) TryPopDebugGroup() (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("popDebugGroup")
	return nil
}

func (g *CommandEncoder) TryPushDebugGroup(label string) (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("pushDebugGroup", label)
	return nil
}

func (g *CommandEncoder) TryResolveQuerySet(querySet *QuerySet, query uint32, count uint32, destination *Buffer, offset uint64) (err error) {
	defer handleJsException(&err)
	g.jsValue.Call("resolveQuerySet", querySet.toJS(), query, count, destination.toJS(), offset)
	return nil
}
