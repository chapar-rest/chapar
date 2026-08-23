package wgpu

import "strconv"

// This file contains common types and constants

const (
	ArrayLayerCountUndefined        = 0xffffffff
	CopyStrideUndefined             = 0xffffffff
	LimitU32Undefined        uint32 = 0xffffffff
	LimitU64Undefined        uint64 = 0xffffffffffffffff
	MipLevelCountUndefined          = 0xffffffff
	WholeMapSize                    = ^uint(0)
	WholeSize                       = 0xffffffffffffffff
)

type Version uint32

func (v Version) String() string {
	return "0x" + strconv.FormatUint(uint64(v), 8)
}

type Limits struct {
	MaxTextureDimension1D                     uint32
	MaxTextureDimension2D                     uint32
	MaxTextureDimension3D                     uint32
	MaxTextureArrayLayers                     uint32
	MaxBindGroups                             uint32
	MaxBindingsPerBindGroup                   uint32
	MaxDynamicUniformBuffersPerPipelineLayout uint32
	MaxDynamicStorageBuffersPerPipelineLayout uint32
	MaxSampledTexturesPerShaderStage          uint32
	MaxSamplersPerShaderStage                 uint32
	MaxStorageBuffersPerShaderStage           uint32
	MaxStorageTexturesPerShaderStage          uint32
	MaxUniformBuffersPerShaderStage           uint32
	MaxUniformBufferBindingSize               uint64
	MaxStorageBufferBindingSize               uint64
	MinUniformBufferOffsetAlignment           uint32
	MinStorageBufferOffsetAlignment           uint32
	MaxVertexBuffers                          uint32
	MaxBufferSize                             uint64
	MaxVertexAttributes                       uint32
	MaxVertexBufferArrayStride                uint32
	MaxInterStageShaderComponents             uint32
	MaxInterStageShaderVariables              uint32
	MaxColorAttachments                       uint32
	MaxColorAttachmentBytesPerSample          uint32
	MaxComputeWorkgroupStorageSize            uint32
	MaxComputeInvocationsPerWorkgroup         uint32
	MaxComputeWorkgroupSizeX                  uint32
	MaxComputeWorkgroupSizeY                  uint32
	MaxComputeWorkgroupSizeZ                  uint32
	MaxComputeWorkgroupsPerDimension          uint32

	MaxPushConstantSize uint32
}

type SupportedLimits struct {
	Limits Limits
}

// Color as described:
// https://gpuweb.github.io/gpuweb/#typedefdef-gpucolor
type Color struct {
	R, G, B, A float64
}

type Origin3D struct {
	X, Y, Z uint32
}

// SurfaceConfiguration, corresponding to GPUCanvasConfiguration:
// https://gpuweb.github.io/gpuweb/#dictdef-gpucanvasconfiguration
type SurfaceConfiguration struct {
	Usage       TextureUsage
	Format      TextureFormat
	Width       uint32
	Height      uint32
	PresentMode PresentMode
	AlphaMode   CompositeAlphaMode
	ViewFormats []TextureFormat
}

type ImageCopyTexture struct {
	Texture  *Texture
	MipLevel uint32
	Origin   Origin3D
	Aspect   TextureAspect
}

type TextureDataLayout struct {
	Offset       uint64
	BytesPerRow  uint32
	RowsPerImage uint32
}

type Extent3D struct {
	Width              uint32
	Height             uint32
	DepthOrArrayLayers uint32
}

type InstanceDescriptor struct {
	Backends           InstanceBackend
	Dx12ShaderCompiler Dx12Compiler
	DxilPath           string
	DxcPath            string
}

type InstanceEnumerateAdapterOptons struct {
	Backends InstanceBackend
}

// RequestAdapterOptions as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpurequestadapteroptions
type RequestAdapterOptions struct {
	CompatibleSurface    *Surface
	PowerPreference      PowerPreference
	ForceFallbackAdapter bool
	BackendType          BackendType
}

type SurfaceCapabilities struct {
	Formats      []TextureFormat
	PresentModes []PresentMode
	AlphaModes   []CompositeAlphaMode
}

type ShaderModuleWGSLDescriptor struct {
	Code string
}

type RequiredLimits struct {
	Limits Limits
}

type DeviceDescriptor struct {
	Label              string
	RequiredFeatures   []FeatureName
	RequiredLimits     *RequiredLimits
	DeviceLostCallback DeviceLostCallback
	TracePath          string
}

type DeviceLostCallback func(reason DeviceLostReason, message string)

// TextureDescriptor as described:
// https://gpuweb.github.io/gpuweb/#gputexturedescriptor
type TextureDescriptor struct {
	Label         string
	Usage         TextureUsage
	Dimension     TextureDimension
	Size          Extent3D
	Format        TextureFormat
	MipLevelCount uint32
	SampleCount   uint32
}

// BufferDescriptor as described:
// https://gpuweb.github.io/gpuweb/#gpubufferdescriptor
type BufferDescriptor struct {
	Label            string
	Usage            BufferUsage
	Size             uint64
	MappedAtCreation bool
}

type BufferInitDescriptor struct {
	Label    string
	Contents []byte
	Usage    BufferUsage
}

type BufferMapCallback func(BufferMapAsyncStatus)

type QueueWorkDoneCallback func(QueueWorkDoneStatus)

// RenderPassDescriptor as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpurenderpassdescriptor
type RenderPassDescriptor struct {
	Label                  string
	ColorAttachments       []RenderPassColorAttachment
	DepthStencilAttachment *RenderPassDepthStencilAttachment

	// unused in wgpu
	// 	OcclusionQuerySet      QuerySet
	// 	TimestampWrites        []RenderPassTimestampWrite
}

type RenderPassDepthStencilAttachment struct {
	View              *TextureView
	DepthLoadOp       LoadOp
	DepthStoreOp      StoreOp
	DepthClearValue   float32
	DepthReadOnly     bool
	StencilLoadOp     LoadOp
	StencilStoreOp    StoreOp
	StencilClearValue uint32
	StencilReadOnly   bool
}

// RenderPassColorAttachment as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpurenderpasscolorattachment
type RenderPassColorAttachment struct {
	View          *TextureView
	ResolveTarget *TextureView
	LoadOp        LoadOp
	StoreOp       StoreOp
	ClearValue    Color
}

// RenderPipelineDescriptor as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpurenderpipelinedescriptor
type RenderPipelineDescriptor struct {
	Label        string
	Layout       *PipelineLayout
	Vertex       VertexState
	Primitive    PrimitiveState
	DepthStencil *DepthStencilState
	Multisample  MultisampleState
	Fragment     *FragmentState
}

type CommandEncoderDescriptor struct {
	Label string
}

type TextureViewDescriptor struct {
	Label           string
	Format          TextureFormat
	Dimension       TextureViewDimension
	BaseMipLevel    uint32
	MipLevelCount   uint32
	BaseArrayLayer  uint32
	ArrayLayerCount uint32
	Aspect          TextureAspect
}

type CommandBufferDescriptor struct {
	Label string
}

type WrappedSubmissionIndex struct {
	Queue           *Queue
	SubmissionIndex SubmissionIndex
}

type SubmissionIndex uint64

type ImageCopyBuffer struct {
	Layout TextureDataLayout
	Buffer *Buffer
}

type AdapterInfo struct {
	VendorId          uint32
	VendorName        string
	Architecture      string
	DeviceId          uint32
	Name              string
	DriverDescription string
	AdapterType       AdapterType
	BackendType       BackendType
}

type SamplerDescriptor struct {
	Label         string
	AddressModeU  AddressMode
	AddressModeV  AddressMode
	AddressModeW  AddressMode
	MagFilter     FilterMode
	MinFilter     FilterMode
	MipmapFilter  MipmapFilterMode
	LodMinClamp   float32
	LodMaxClamp   float32
	Compare       CompareFunction
	MaxAnisotropy uint16
}

// BindGroupLayoutDescriptor as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpubindgrouplayoutdescriptor
type BindGroupLayoutDescriptor struct {
	Label   string
	Entries []BindGroupLayoutEntry
}

// BindGroupDescriptor as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpubindgroupdescriptor
type BindGroupDescriptor struct {
	Label   string
	Layout  *BindGroupLayout
	Entries []BindGroupEntry
}

// BindGroupEntry as described:
// https://gpuweb.github.io/gpuweb/#dictdef-gpubindgroupentry
type BindGroupEntry struct {
	Binding     uint32
	Buffer      *Buffer
	Offset      uint64
	Size        uint64
	Sampler     *Sampler
	TextureView *TextureView
}

// ProgrammableStageDescriptor as described:
// https://gpuweb.github.io/gpuweb/#gpuprogrammablestage
type ProgrammableStageDescriptor struct {
	Module     *ShaderModule
	EntryPoint string
}
