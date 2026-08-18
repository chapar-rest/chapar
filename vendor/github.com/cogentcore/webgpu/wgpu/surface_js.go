//go:build js

package wgpu

import (
	"syscall/js"
)

// Surface as described:
// https://gpuweb.github.io/gpuweb/#gpucanvascontext
// (CanvasContext is the closest equivalent to Surface in js)
type Surface struct {
	jsValue js.Value
}

func (g Surface) GetCapabilities(adapter *Adapter) (ret SurfaceCapabilities) {
	// Based on https://developer.mozilla.org/en-US/docs/Web/API/GPUCanvasContext/configure
	ret.Formats = []TextureFormat{TextureFormatBGRA8Unorm, TextureFormatRGBA8Unorm, TextureFormatRGBA16Float}
	ret.AlphaModes = []CompositeAlphaMode{CompositeAlphaModeOpaque, CompositeAlphaModePremultiplied}
	ret.PresentModes = []PresentMode{PresentModeImmediate}
	return
}

func (g Surface) Configure(adapter *Adapter, device *Device, config *SurfaceConfiguration) {
	jsConfig := pointerToJS(config).(map[string]any)
	jsConfig["device"] = pointerToJS(device)
	g.jsValue.Call("configure", jsConfig)
}

func (g Surface) GetCurrentTexture() (*Texture, error) {
	texture := g.jsValue.Call("getCurrentTexture")
	return &Texture{texture}, nil
}

func (g Surface) Present() {} // no-op

func (g Surface) Release() {} // no-op
