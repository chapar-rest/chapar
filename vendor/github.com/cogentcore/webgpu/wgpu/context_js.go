//go:build js

package wgpu

import "syscall/js"

// NewCanvasContext creates a new GPUCanvasContext using the specified
// JavaScript reference as the underlying context.
func NewCanvasContext(jsValue js.Value) CanvasContext {
	return CanvasContext{
		jsValue: jsValue,
	}
}

// CanvasContext as described:
// https://gpuweb.github.io/gpuweb/#gpucanvascontext
type CanvasContext struct {
	jsValue js.Value
}

func (g CanvasContext) toJS() any {
	return g.jsValue
}

// GetCurrentTexture as described:
// https://gpuweb.github.io/gpuweb/#dom-gpucanvascontext-getcurrenttexture
func (g CanvasContext) GetCurrentTexture() Texture {
	jsTexture := g.jsValue.Call("getCurrentTexture")
	return Texture{
		jsValue: jsTexture,
	}
}
