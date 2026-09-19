//go:build js

package wgpu

import (
	"syscall/js"
)

// BufferBindingLayout as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpubufferbindinglayout
type BufferBindingLayout struct {
	Type             BufferBindingType
	HasDynamicOffset bool
	MinBindingSize   uint64
}

func (g BufferBindingLayout) toJS() any {
	result := make(map[string]any)
	result["type"] = enumToJS(g.Type)
	result["hasDynamicOffset"] = g.HasDynamicOffset
	result["minBindingSize"] = g.MinBindingSize
	return result
}

// SamplerBindingLayout as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpusamplerbindinglayout
type SamplerBindingLayout struct {
	Type SamplerBindingType
}

func (g SamplerBindingLayout) toJS() any {
	result := make(map[string]any)
	result["type"] = enumToJS(g.Type)
	return result
}

// TextureBindingLayout as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gputexturebindinglayout
type TextureBindingLayout struct {
	SampleType    TextureSampleType
	ViewDimension TextureViewDimension
	Multisampled  bool
}

func (g TextureBindingLayout) toJS() any {
	result := make(map[string]any)
	result["sampleType"] = enumToJS(g.SampleType)
	result["viewDimension"] = enumToJS(g.ViewDimension)
	result["multisampled"] = g.Multisampled
	return result
}

// StorageTextureBindingLayout as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpustoragetexturebindinglayout
type StorageTextureBindingLayout struct {
	Access        StorageTextureAccess
	Format        TextureFormat
	ViewDimension TextureViewDimension
}

func (g StorageTextureBindingLayout) toJS() any {
	result := make(map[string]any)
	result["access"] = enumToJS(g.Access)
	result["format"] = enumToJS(g.Format)
	result["viewDimension"] = enumToJS(g.ViewDimension)
	return result
}

// ExternalTextureBindingLayout as described:
type ExternalTextureBindingLayout struct {
	jsValue js.Value
}

func (g ExternalTextureBindingLayout) toJS() any {
	return g.jsValue
}

// BindGroupLayoutEntry as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpubindgrouplayoutentry
type BindGroupLayoutEntry struct {
	Binding         uint32
	Visibility      ShaderStage
	Buffer          BufferBindingLayout
	Sampler         SamplerBindingLayout
	Texture         TextureBindingLayout
	StorageTexture  StorageTextureBindingLayout
	ExternalTexture ExternalTextureBindingLayout
}

func (g BindGroupLayoutEntry) toJS() any {
	result := make(map[string]any)
	result["binding"] = g.Binding
	result["visibility"] = uint32(g.Visibility)
	switch {
	case g.Buffer != BufferBindingLayout{}:
		result["buffer"] = g.Buffer.toJS()
	case g.Sampler != SamplerBindingLayout{}:
		result["sampler"] = g.Sampler.toJS()
	case g.Texture != TextureBindingLayout{}:
		result["texture"] = g.Texture.toJS()
	case g.StorageTexture != StorageTextureBindingLayout{}:
		result["storageTexture"] = g.StorageTexture.toJS()
	case !g.ExternalTexture.jsValue.IsUndefined():
		result["externalTexture"] = g.ExternalTexture.toJS()
	}
	return result
}

func (g BindGroupLayoutDescriptor) toJS() any {
	return map[string]any{
		"entries": mapSlice(g.Entries, func(entry BindGroupLayoutEntry) any {
			return entry.toJS()
		}),
	}
}

// BindGroupLayout as described:
// https://gpuweb.github.io/gpuweb/#gpubindgrouplayout
type BindGroupLayout struct {
	jsValue js.Value
}

func (g BindGroupLayout) toJS() any {
	return g.jsValue
}

func (g BindGroupLayout) Release() {} // no-op

func (g BindGroupEntry) toJS() any {
	result := make(map[string]any)
	result["binding"] = g.Binding
	switch {
	case g.Sampler != nil:
		result["resource"] = pointerToJS(g.Sampler)
	case g.TextureView != nil:
		result["resource"] = pointerToJS(g.TextureView)
	default:
		result["resource"] = map[string]any{
			"buffer": pointerToJS(g.Buffer),
			"offset": g.Offset,
			"size":   uint64ToJS(g.Size),
		}
	}
	return result
}

func (g BindGroupDescriptor) toJS() any {
	return map[string]any{
		"layout": pointerToJS(g.Layout),
		"entries": mapSlice(g.Entries, func(entry BindGroupEntry) any {
			return entry.toJS()
		}),
	}
}

// BindGroup as described:
// https://gpuweb.github.io/gpuweb/#gpubindgroup
type BindGroup struct {
	jsValue js.Value
}

func (g BindGroup) Release() {} // no-op

func (g BindGroup) toJS() any {
	return g.jsValue
}
