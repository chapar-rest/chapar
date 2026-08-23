//go:build js

package wgpu

import (
	"syscall/js"
)

// CommandEncoder as described:
// https://gpuweb.github.io/gpuweb/#gpucommandencoder
type CommandEncoder struct {
	jsValue js.Value
}

func (g CommandEncoder) toJS() any {
	return g.jsValue
}

// BeginRenderPass as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-beginrenderpass
func (g CommandEncoder) BeginRenderPass(descriptor *RenderPassDescriptor) *RenderPassEncoder {
	jsRenderPass := g.jsValue.Call("beginRenderPass", pointerToJS(descriptor))
	return &RenderPassEncoder{
		jsValue: jsRenderPass,
	}
}

// BeginComputePass as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-begincomputepass
func (g CommandEncoder) BeginComputePass(descriptor *ComputePassDescriptor) *ComputePassEncoder {
	params := make([]any, 1)
	params[0] = pointerToJS(descriptor)
	jsComputePass := g.jsValue.Call("beginComputePass", params...)
	return &ComputePassEncoder{
		jsValue: jsComputePass,
	}
}

// CopyBufferToBuffer as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-copybuffertobuffer
func (g CommandEncoder) CopyBufferToBuffer(source *Buffer, sourceOffset uint64, destination *Buffer, destinationOffset uint64, size uint64) (err error) {
	g.jsValue.Call("copyBufferToBuffer", pointerToJS(source), sourceOffset, pointerToJS(destination), destinationOffset, size)
	return nil
}

// CopyBufferToTexture as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-copybuffertotexture
func (g CommandEncoder) CopyBufferToTexture(source *ImageCopyBuffer, destination *ImageCopyTexture, copySize *Extent3D) (err error) {
	g.jsValue.Call("copyBufferToTexture", pointerToJS(source), pointerToJS(destination), pointerToJS(copySize))
	return nil
}

// CopyTextureToBuffer as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-copytexturetobuffer
func (g CommandEncoder) CopyTextureToBuffer(source *ImageCopyTexture, destination *ImageCopyBuffer, copySize *Extent3D) (err error) {
	g.jsValue.Call("copyTextureToBuffer", pointerToJS(source), pointerToJS(destination), pointerToJS(copySize))
	return nil
}

// CopyTextureToTexture as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-copytexturetotexture
func (g CommandEncoder) CopyTextureToTexture(source *ImageCopyTexture, destination *ImageCopyTexture, copySize *Extent3D) (err error) {
	g.jsValue.Call("copyTextureToTexture", pointerToJS(source), pointerToJS(destination), pointerToJS(copySize))
	return nil
}

// Finish as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucommandencoder-finish
func (g CommandEncoder) Finish(descriptor *CommandBufferDescriptor) (*CommandBuffer, error) {
	jsBuffer := g.jsValue.Call("finish", pointerToJS(descriptor))
	return &CommandBuffer{
		jsValue: jsBuffer,
	}, nil
}

func (g CommandEncoder) Release() {} // no-op
