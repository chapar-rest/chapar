//go:build js

package wgpu

func (g *Surface) GetCapabilities(adapter *Adapter) (ret SurfaceCapabilities) {
	// Based on https://developer.mozilla.org/en-US/docs/Web/API/GPUCanvasContext/configure
	ret.Formats = []TextureFormat{TextureFormatBGRA8Unorm, TextureFormatRGBA8Unorm, TextureFormatRGBA16Float}
	ret.AlphaModes = []CompositeAlphaMode{CompositeAlphaModeOpaque, CompositeAlphaModePremultiplied}
	ret.PresentModes = []PresentMode{PresentModeImmediate}
	return
}

func (g *Surface) Configure(device *Device, config *SurfaceConfiguration) {
	jsConfig := pointerToJS(config).(map[string]any)
	jsConfig["device"] = pointerToJS(device)
	g.jsValue.Call("configure", jsConfig)
}

func (g *Surface) TryGetCurrentTexture() (_ SurfaceTexture, err error) {
	defer handleJsException(&err)

	surfaceTexture := SurfaceTexture{
		Status: SurfaceGetCurrentTextureStatusSuccessOptimal,
		Texture: &Texture{
			jsValue: g.jsValue.Call("getCurrentTexture"),
		},
	}

	return surfaceTexture, nil
}

// Present is a no-op on javascript. The surface is automatically presented at the end
// of the requestAnimationFrame callback.
func (g *Surface) Present() {}
