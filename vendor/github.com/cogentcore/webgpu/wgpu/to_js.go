//go:build js

package wgpu

import (
	"fmt"
	"syscall/js"
)

// enumToJS converts the given non-bit-flag enum value to a type that
// can be passed as an argument to JavaScript. Bit flag enums should be
// passed as a uint.
func enumToJS(s fmt.Stringer) any {
	ss := s.String()
	if ss == "undefined" {
		return js.Undefined()
	}
	return ss
}

// pointerToJS converts the given pointer value to a type that can be
// passed as an argument to JavaScript. It must implement a toJS method.
func pointerToJS[T any, P interface {
	*T
	toJS() any
}](v P) any {
	if v == nil {
		return js.Undefined()
	}
	return v.toJS()
}

// uint32ToJS converts the given uint32 value to a type that can be
// passed as an argument to JavaScript.
func uint32ToJS(v uint32) any {
	if v == LimitU32Undefined {
		return js.Undefined()
	}
	return v
}

// uint64ToJS converts the given uint64 value to a type that can be
// passed as an argument to JavaScript.
func uint64ToJS(v uint64) any {
	if v == LimitU64Undefined {
		return js.Undefined()
	}
	return v
}

func (g Color) toJS() any {
	return []any{g.R, g.G, g.B, g.A}
}

func (g Extent3D) toJS() any {
	return []any{g.Width, g.Height, g.DepthOrArrayLayers}
}

func (g Origin3D) toJS() any {
	return []any{g.X, g.Y, g.Z}
}

func (g *RequestAdapterOptions) toJS() any {
	result := make(map[string]any)
	result["powerPreference"] = enumToJS(g.PowerPreference)
	result["forceFallbackAdapter"] = g.ForceFallbackAdapter
	return result
}

func (g *DeviceDescriptor) toJS() any {
	result := make(map[string]any)
	result["label"] = g.Label
	result["requiredFeatures"] = mapSlice(g.RequiredFeatures, func(f FeatureName) any { return f })
	// result["requiredLimits"] = // TODO(kai): convert requiredLimits to JS
	return result
}

func (g *SurfaceConfiguration) toJS() any {
	result := make(map[string]any)
	result["usage"] = uint32(g.Usage)
	result["format"] = enumToJS(g.Format)
	result["alphaMode"] = enumToJS(g.AlphaMode)
	result["viewFormats"] = mapSlice(g.ViewFormats, func(f TextureFormat) any {
		return enumToJS(f)
	})
	return result
}

func (g *TextureDescriptor) toJS() any {
	return map[string]any{
		"label":         g.Label,
		"usage":         uint32(g.Usage),
		"dimension":     enumToJS(g.Dimension),
		"size":          g.Size.toJS(),
		"format":        enumToJS(g.Format),
		"mipLevelCount": g.MipLevelCount,
		"sampleCount":   g.SampleCount,
	}
}

func (g *TextureViewDescriptor) toJS() any {
	return map[string]any{
		"label":           g.Label,
		"format":          enumToJS(g.Format),
		"dimension":       enumToJS(g.Dimension),
		"baseMipLevel":    g.BaseMipLevel,
		"mipLevelCount":   g.MipLevelCount,
		"baseArrayLayer":  g.BaseArrayLayer,
		"arrayLayerCount": g.ArrayLayerCount,
		"aspect":          enumToJS(g.Aspect),
	}
}

func (g *CommandEncoderDescriptor) toJS() any {
	return map[string]any{"label": g.Label}
}

func (g *CommandBufferDescriptor) toJS() any {
	return map[string]any{"label": g.Label}
}

func (g BufferDescriptor) toJS() any {
	return map[string]any{
		"label":            g.Label,
		"size":             g.Size,
		"usage":            uint32(g.Usage),
		"mappedAtCreation": g.MappedAtCreation,
	}
}

func (g *ImageCopyBuffer) toJS() any {
	return map[string]any{
		"buffer":       pointerToJS(g.Buffer),
		"offset":       g.Layout.Offset,
		"bytesPerRow":  g.Layout.BytesPerRow,
		"rowsPerImage": uint32ToJS(g.Layout.RowsPerImage),
	}
}

func (g *ImageCopyTexture) toJS() any {
	return map[string]any{
		"texture":  pointerToJS(g.Texture),
		"mipLevel": g.MipLevel,
		"origin":   g.Origin.toJS(),
		"aspect":   enumToJS(g.Aspect),
	}
}

func (g *TextureDataLayout) toJS() any {
	return map[string]any{
		"offset":       g.Offset,
		"bytesPerRow":  g.BytesPerRow,
		"rowsPerImage": g.RowsPerImage,
	}
}

func (g *RenderPassDescriptor) toJS() any {
	result := make(map[string]any)
	result["colorAttachments"] = mapSlice(g.ColorAttachments, func(attachment RenderPassColorAttachment) any {
		return attachment.toJS()
	})
	result["depthStencilAttachment"] = pointerToJS(g.DepthStencilAttachment)
	return result
}

func (g *RenderPassColorAttachment) toJS() any {
	result := make(map[string]any)
	result["view"] = g.View.jsValue
	result["loadOp"] = enumToJS(g.LoadOp)
	result["storeOp"] = enumToJS(g.StoreOp)
	result["clearValue"] = g.ClearValue.toJS()
	result["resolveTarget"] = pointerToJS(g.ResolveTarget)
	return result
}

func (g *RenderPassDepthStencilAttachment) toJS() any {
	return map[string]any{
		"view":            pointerToJS(g.View),
		"depthLoadOp":     enumToJS(g.DepthLoadOp),
		"depthStoreOp":    enumToJS(g.DepthStoreOp),
		"depthClearValue": g.DepthClearValue,
		"depthReadOnly":   g.DepthReadOnly,
		// TODO(kai): these cause errors if passed
		// "stencilLoadOp":     enumToJS(g.StencilLoadOp),
		// "stencilStoreOp":    enumToJS(g.StencilStoreOp),
		"stencilClearValue": g.StencilClearValue,
		"stencilReadOnly":   g.StencilReadOnly,
	}
}

func (g *RenderPipelineDescriptor) toJS() any {
	result := make(map[string]any)
	if g.Layout == nil {
		result["layout"] = "auto"
	} else {
		result["layout"] = pointerToJS(g.Layout)
	}
	result["vertex"] = g.Vertex.toJS()
	result["primitive"] = g.Primitive.toJS()
	result["depthStencil"] = pointerToJS(g.DepthStencil)
	result["multisample"] = g.Multisample.toJS()
	result["fragment"] = pointerToJS(g.Fragment)
	return result
}

func (g *SamplerDescriptor) toJS() any {
	result := make(map[string]any)
	result["addressModeU"] = enumToJS(g.AddressModeU)
	result["addressModeV"] = enumToJS(g.AddressModeV)
	result["addressModeW"] = enumToJS(g.AddressModeW)
	result["magFilter"] = enumToJS(g.MagFilter)
	result["minFilter"] = enumToJS(g.MinFilter)
	result["mipmapFilter"] = enumToJS(g.MipmapFilter)
	result["lodMinClamp"] = g.LodMinClamp
	result["lodMaxClamp"] = g.LodMaxClamp
	result["compare"] = enumToJS(g.Compare)
	result["maxAnisotropy"] = g.MaxAnisotropy
	return result
}

func (g *ProgrammableStageDescriptor) toJS() any {
	return map[string]any{
		"module":     pointerToJS(g.Module),
		"entryPoint": g.EntryPoint,
	}
}

func limitsFromJS(j js.Value) Limits {
	return Limits{
		MaxTextureDimension1D:                     uint32(j.Get("maxTextureDimension1D").Int()),
		MaxTextureDimension2D:                     uint32(j.Get("maxTextureDimension2D").Int()),
		MaxTextureDimension3D:                     uint32(j.Get("maxTextureDimension3D").Int()),
		MaxTextureArrayLayers:                     uint32(j.Get("maxTextureArrayLayers").Int()),
		MaxBindGroups:                             uint32(j.Get("maxBindGroups").Int()),
		MaxDynamicUniformBuffersPerPipelineLayout: uint32(j.Get("maxDynamicUniformBuffersPerPipelineLayout").Int()),
		MaxDynamicStorageBuffersPerPipelineLayout: uint32(j.Get("maxDynamicStorageBuffersPerPipelineLayout").Int()),
		MaxSampledTexturesPerShaderStage:          uint32(j.Get("maxSampledTexturesPerShaderStage").Int()),
		MaxSamplersPerShaderStage:                 uint32(j.Get("maxSamplersPerShaderStage").Int()),
		MaxStorageBuffersPerShaderStage:           uint32(j.Get("maxStorageBuffersPerShaderStage").Int()),
		MaxStorageTexturesPerShaderStage:          uint32(j.Get("maxStorageTexturesPerShaderStage").Int()),
		MaxUniformBuffersPerShaderStage:           uint32(j.Get("maxUniformBuffersPerShaderStage").Int()),
		MaxUniformBufferBindingSize:               uint64(j.Get("maxUniformBufferBindingSize").Int()),
		MaxStorageBufferBindingSize:               uint64(j.Get("maxStorageBufferBindingSize").Int()),
		MinUniformBufferOffsetAlignment:           uint32(j.Get("minUniformBufferOffsetAlignment").Int()),
		MinStorageBufferOffsetAlignment:           uint32(j.Get("minStorageBufferOffsetAlignment").Int()),
		MaxVertexBuffers:                          uint32(j.Get("maxVertexBuffers").Int()),
		MaxBufferSize:                             uint64(j.Get("maxBufferSize").Int()),
		MaxVertexAttributes:                       uint32(j.Get("maxVertexAttributes").Int()),
		MaxVertexBufferArrayStride:                uint32(j.Get("maxVertexBufferArrayStride").Int()),
		// MaxInterStageShaderComponents:             uint32(j.Get("maxInterStageShaderComponents").Int()), // no present on firefox
		MaxInterStageShaderVariables:      uint32(j.Get("maxInterStageShaderVariables").Int()),
		MaxColorAttachments:               uint32(j.Get("maxColorAttachments").Int()),
		MaxColorAttachmentBytesPerSample:  uint32(j.Get("maxColorAttachmentBytesPerSample").Int()),
		MaxComputeWorkgroupStorageSize:    uint32(j.Get("maxComputeWorkgroupStorageSize").Int()),
		MaxComputeInvocationsPerWorkgroup: uint32(j.Get("maxComputeInvocationsPerWorkgroup").Int()),
		MaxComputeWorkgroupSizeX:          uint32(j.Get("maxComputeWorkgroupSizeX").Int()),
		MaxComputeWorkgroupSizeY:          uint32(j.Get("maxComputeWorkgroupSizeY").Int()),
		MaxComputeWorkgroupSizeZ:          uint32(j.Get("maxComputeWorkgroupSizeZ").Int()),
		MaxComputeWorkgroupsPerDimension:  uint32(j.Get("maxComputeWorkgroupsPerDimension").Int()),
		MaxPushConstantSize:               128,
	}
}
