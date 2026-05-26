//go:build js

package wgpu

import (
	"fmt"
)

// GetFormat as described:
// https://gpuweb.github.io/gpuweb/#dom-gputexture-format
func (g *Texture) GetFormat() TextureFormat {
	jsFormat := g.jsValue.Get("format").String()

	for tf := TextureFormatUndefined + 1; tf.String() != ""; tf++ {
		if tf.String() == jsFormat {
			return tf
		}
	}

	panic(fmt.Sprintf("unknown texture format %q", jsFormat))
}

// GetDepthOrArrayLayers as described:
// https://gpuweb.github.io/gpuweb/#dom-gputexture-depthorarraylayers
func (g *Texture) GetDepthOrArrayLayers() uint32 {
	return uint32(g.jsValue.Get("depthOrArrayLayers").Int())
}

// GetMipLevelCount as described:
// https://gpuweb.github.io/gpuweb/#dom-gputexture-miplevelcount
func (g *Texture) GetMipLevelCount() uint32 {
	return uint32(g.jsValue.Get("mipLevelCount").Int())
}

// GetWidth as described:
// https://gpuweb.github.io/gpuweb/#dom-gputexture-width
func (g *Texture) GetWidth() uint32 {
	return uint32(g.jsValue.Get("width").Int())
}

// GetHeight as described:
// https://gpuweb.github.io/gpuweb/#dom-gputexture-height
func (g *Texture) GetHeight() uint32 {
	return uint32(g.jsValue.Get("height").Int())
}

// GetSampleCount as described:
// https://gpuweb.github.io/gpuweb/#dom-gputexture-samplecount
func (g *Texture) GetSampleCount() uint32 {
	return uint32(g.jsValue.Get("sampleCount").Int())
}

// TryCreateView as described:
// https://gpuweb.github.io/gpuweb/#dom-gputexture-createview
func (g *Texture) TryCreateView(descriptor *TextureViewDescriptor) (_ *TextureView, err error) {
	defer handleJsException(&err)

	jsView := g.jsValue.Call("createView", pointerToJS(descriptor))
	return &TextureView{
		jsValue: jsView,
	}, nil
}
