//go:build !js

package wgpu

/*

#include <stdlib.h>
#include "./lib/wgpu.h"

extern void gowebgpu_error_callback_c(WGPUErrorType type, char const * message, void * userdata);

static inline WGPUBindGroup gowebgpu_device_create_bind_group(WGPUDevice device, WGPUBindGroupDescriptor const * descriptor, void * error_userdata) {
	WGPUBindGroup ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateBindGroup(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUBindGroupLayout gowebgpu_device_create_bind_group_layout(WGPUDevice device, WGPUBindGroupLayoutDescriptor const * descriptor, void * error_userdata) {
	WGPUBindGroupLayout ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateBindGroupLayout(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUBuffer gowebgpu_device_create_buffer(WGPUDevice device, WGPUBufferDescriptor const * descriptor, void * error_userdata) {
	WGPUBuffer ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateBuffer(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUCommandEncoder gowebgpu_device_create_command_encoder(WGPUDevice device, WGPUCommandEncoderDescriptor const * descriptor, void * error_userdata) {
	WGPUCommandEncoder ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateCommandEncoder(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUComputePipeline gowebgpu_device_create_compute_pipeline(WGPUDevice device, WGPUComputePipelineDescriptor const * descriptor, void * error_userdata) {
	WGPUComputePipeline ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateComputePipeline(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUPipelineLayout gowebgpu_device_create_pipeline_layout(WGPUDevice device, WGPUPipelineLayoutDescriptor const * descriptor, void * error_userdata) {
	WGPUPipelineLayout ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreatePipelineLayout(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUQuerySet gowebgpu_device_create_query_set(WGPUDevice device, WGPUQuerySetDescriptor const * descriptor, void * error_userdata) {
	WGPUQuerySet ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateQuerySet(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPURenderPipeline gowebgpu_device_create_render_pipeline(WGPUDevice device, WGPURenderPipelineDescriptor const * descriptor, void * error_userdata) {
	WGPURenderPipeline ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateRenderPipeline(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUSampler gowebgpu_device_create_sampler(WGPUDevice device, WGPUSamplerDescriptor const * descriptor, void * error_userdata) {
	WGPUSampler ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateSampler(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUShaderModule gowebgpu_device_create_shader_module(WGPUDevice device, WGPUShaderModuleDescriptor const * descriptor, void * error_userdata) {
	WGPUShaderModule ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateShaderModule(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline WGPUTexture gowebgpu_device_create_texture(WGPUDevice device, WGPUTextureDescriptor const * descriptor, void * error_userdata) {
	WGPUTexture ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateTexture(device, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

*/
import "C"
import (
	"errors"
	"runtime/cgo"
	"unsafe"
)

type Device struct {
	ref C.WGPUDevice
}

type errorCallback func(typ ErrorType, message string)

//export gowebgpu_error_callback_go
func gowebgpu_error_callback_go(_type C.WGPUErrorType, message *C.char, userdata unsafe.Pointer) {
	handle := *(*cgo.Handle)(userdata)
	cb, ok := handle.Value().(errorCallback)
	if ok {
		cb(ErrorType(_type), C.GoString(message))
	}
}

func (p *Device) Release() { C.wgpuDeviceRelease(p.ref) }

func (p *Device) CreateBindGroup(descriptor *BindGroupDescriptor) (*BindGroup, error) {
	var desc C.WGPUBindGroupDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}

		if descriptor.Layout != nil {
			desc.layout = descriptor.Layout.ref
		}

		entryCount := len(descriptor.Entries)
		if entryCount > 0 {
			entries := C.malloc(C.size_t(entryCount) * C.size_t(unsafe.Sizeof(C.WGPUBindGroupEntry{})))
			defer C.free(entries)

			entriesSlice := unsafe.Slice((*C.WGPUBindGroupEntry)(entries), entryCount)

			for i, v := range descriptor.Entries {
				entry := C.WGPUBindGroupEntry{
					binding: C.uint32_t(v.Binding),
					offset:  C.uint64_t(v.Offset),
					size:    C.uint64_t(v.Size),
				}

				if v.Buffer != nil {
					entry.buffer = v.Buffer.ref
				}
				if v.Sampler != nil {
					entry.sampler = v.Sampler.ref
				}
				if v.TextureView != nil {
					entry.textureView = v.TextureView.ref
				}

				entriesSlice[i] = entry
			}

			desc.entryCount = C.size_t(entryCount)
			desc.entries = (*C.WGPUBindGroupEntry)(entries)
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateBindGroup(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_bind_group(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuBindGroupRelease(ref)
		return nil, err
	}

	return &BindGroup{ref}, nil
}

type BufferBindingLayout struct {
	Type             BufferBindingType
	HasDynamicOffset bool
	MinBindingSize   uint64
}

type SamplerBindingLayout struct {
	Type SamplerBindingType
}

type TextureBindingLayout struct {
	SampleType    TextureSampleType
	ViewDimension TextureViewDimension
	Multisampled  bool
}

type StorageTextureBindingLayout struct {
	Access        StorageTextureAccess
	Format        TextureFormat
	ViewDimension TextureViewDimension
}

type BindGroupLayoutEntry struct {
	Binding        uint32
	Visibility     ShaderStage
	Buffer         BufferBindingLayout
	Sampler        SamplerBindingLayout
	Texture        TextureBindingLayout
	StorageTexture StorageTextureBindingLayout
}

func (p *Device) CreateBindGroupLayout(descriptor *BindGroupLayoutDescriptor) (*BindGroupLayout, error) {
	var desc C.WGPUBindGroupLayoutDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}

		entryCount := len(descriptor.Entries)
		if entryCount > 0 {
			entries := C.malloc(C.size_t(entryCount) * C.size_t(unsafe.Sizeof(C.WGPUBindGroupLayoutEntry{})))
			defer C.free(entries)

			entriesSlice := unsafe.Slice((*C.WGPUBindGroupLayoutEntry)(entries), entryCount)

			for i, v := range descriptor.Entries {
				entriesSlice[i] = C.WGPUBindGroupLayoutEntry{
					nextInChain: nil,
					binding:     C.uint32_t(v.Binding),
					visibility:  C.WGPUShaderStageFlags(v.Visibility),
					buffer: C.WGPUBufferBindingLayout{
						nextInChain:      nil,
						_type:            C.WGPUBufferBindingType(v.Buffer.Type),
						hasDynamicOffset: cBool(v.Buffer.HasDynamicOffset),
						minBindingSize:   C.uint64_t(v.Buffer.MinBindingSize),
					},
					sampler: C.WGPUSamplerBindingLayout{
						nextInChain: nil,
						_type:       C.WGPUSamplerBindingType(v.Sampler.Type),
					},
					texture: C.WGPUTextureBindingLayout{
						nextInChain:   nil,
						sampleType:    C.WGPUTextureSampleType(v.Texture.SampleType),
						viewDimension: C.WGPUTextureViewDimension(v.Texture.ViewDimension),
						multisampled:  cBool(v.Texture.Multisampled),
					},
					storageTexture: C.WGPUStorageTextureBindingLayout{
						nextInChain:   nil,
						access:        C.WGPUStorageTextureAccess(v.StorageTexture.Access),
						format:        C.WGPUTextureFormat(v.StorageTexture.Format),
						viewDimension: C.WGPUTextureViewDimension(v.StorageTexture.ViewDimension),
					},
				}
			}

			desc.entryCount = C.size_t(entryCount)
			desc.entries = (*C.WGPUBindGroupLayoutEntry)(entries)
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateBindGroupLayout(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_bind_group_layout(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuBindGroupLayoutRelease(ref)
		return nil, err
	}

	return &BindGroupLayout{ref}, nil
}

func (p *Device) CreateBuffer(descriptor *BufferDescriptor) (*Buffer, error) {
	var desc C.WGPUBufferDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}

		desc.usage = C.WGPUBufferUsageFlags(descriptor.Usage)
		desc.size = C.uint64_t(descriptor.Size)
		desc.mappedAtCreation = cBool(descriptor.MappedAtCreation)
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateBuffer(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_buffer(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuBufferRelease(ref)
		return nil, err
	}

	C.wgpuDeviceReference(p.ref)
	return &Buffer{deviceRef: p.ref, ref: ref}, nil
}

func (p *Device) CreateCommandEncoder(descriptor *CommandEncoderDescriptor) (*CommandEncoder, error) {
	var desc *C.WGPUCommandEncoderDescriptor

	if descriptor != nil && descriptor.Label != "" {
		label := C.CString(descriptor.Label)
		defer C.free(unsafe.Pointer(label))

		desc = &C.WGPUCommandEncoderDescriptor{
			label: label,
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateCommandEncoder(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_command_encoder(
		p.ref,
		desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuCommandEncoderRelease(ref)
		return nil, err
	}

	C.wgpuDeviceReference(p.ref)
	return &CommandEncoder{deviceRef: p.ref, ref: ref}, nil
}

type ConstantEntry struct {
	Key   string
	Value float64
}

type ComputePipelineDescriptor struct {
	Label   string
	Layout  *PipelineLayout
	Compute ProgrammableStageDescriptor
}

func (p *Device) CreateComputePipeline(descriptor *ComputePipelineDescriptor) (*ComputePipeline, error) {
	var desc C.WGPUComputePipelineDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}

		if descriptor.Layout != nil {
			desc.layout = descriptor.Layout.ref
		}

		var compute C.WGPUProgrammableStageDescriptor
		if descriptor.Compute.Module != nil {
			compute.module = descriptor.Compute.Module.ref
		}
		if descriptor.Compute.EntryPoint != "" {
			entryPoint := C.CString(descriptor.Compute.EntryPoint)
			defer C.free(unsafe.Pointer(entryPoint))

			compute.entryPoint = entryPoint
		}
		desc.compute = compute
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateComputePipeline(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_compute_pipeline(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuComputePipelineRelease(ref)
		return nil, err
	}

	return &ComputePipeline{ref}, nil
}

type PushConstantRange struct {
	Stages ShaderStage
	Start  uint32
	End    uint32
}

type PipelineLayoutDescriptor struct {
	Label              string
	BindGroupLayouts   []*BindGroupLayout
	PushConstantRanges []PushConstantRange
}

func (p *Device) CreatePipelineLayout(descriptor *PipelineLayoutDescriptor) (*PipelineLayout, error) {
	var desc C.WGPUPipelineLayoutDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}

		bindGroupLayoutCount := len(descriptor.BindGroupLayouts)
		if bindGroupLayoutCount > 0 {
			bindGroupLayouts := C.malloc(C.size_t(bindGroupLayoutCount) * C.size_t(unsafe.Sizeof(C.WGPUBindGroupLayout(nil))))
			defer C.free(bindGroupLayouts)

			bindGroupLayoutsSlice := unsafe.Slice((*C.WGPUBindGroupLayout)(bindGroupLayouts), bindGroupLayoutCount)

			for i, v := range descriptor.BindGroupLayouts {
				bindGroupLayoutsSlice[i] = v.ref
			}

			desc.bindGroupLayoutCount = C.size_t(bindGroupLayoutCount)
			desc.bindGroupLayouts = (*C.WGPUBindGroupLayout)(bindGroupLayouts)
		}

		if len(descriptor.PushConstantRanges) > 0 {
			pipelineLayoutExtras := (*C.WGPUPipelineLayoutExtras)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUPipelineLayoutExtras{}))))
			defer C.free(unsafe.Pointer(pipelineLayoutExtras))

			pipelineLayoutExtras.chain.next = nil
			pipelineLayoutExtras.chain.sType = C.WGPUSType_PipelineLayoutExtras

			pushConstantRangeCount := len(descriptor.PushConstantRanges)
			pushConstantRanges := C.malloc(C.size_t(pushConstantRangeCount) * C.size_t(unsafe.Sizeof(C.WGPUPushConstantRange{})))
			defer C.free(pushConstantRanges)

			pushConstantRangesSlice := unsafe.Slice((*C.WGPUPushConstantRange)(pushConstantRanges), pushConstantRangeCount)

			for i, v := range descriptor.PushConstantRanges {
				pushConstantRangesSlice[i] = C.WGPUPushConstantRange{
					stages: C.WGPUShaderStageFlags(v.Stages),
					start:  C.uint32_t(v.Start),
					end:    C.uint32_t(v.End),
				}
			}

			pipelineLayoutExtras.pushConstantRangeCount = C.size_t(pushConstantRangeCount)
			pipelineLayoutExtras.pushConstantRanges = (*C.WGPUPushConstantRange)(pushConstantRanges)

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(pipelineLayoutExtras))
		} else {
			desc.nextInChain = nil
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreatePipelineLayout(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_pipeline_layout(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuPipelineLayoutRelease(ref)
		return nil, err
	}

	return &PipelineLayout{ref}, nil
}

type QuerySetDescriptor struct {
	Label              string
	Type               QueryType
	Count              uint32
	PipelineStatistics []PipelineStatisticName
}

func (p *Device) CreateQuerySet(descriptor *QuerySetDescriptor) (*QuerySet, error) {
	var desc C.WGPUQuerySetDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))
			desc.label = label
		}

		desc._type = C.WGPUQueryType(descriptor.Type)
		desc.count = C.uint32_t(descriptor.Count)

		// TODO: no longer present in C API
		// pipelineStatisticCount := len(descriptor.PipelineStatistics)
		// if pipelineStatisticCount > 0 {
		// 	pipelineStatistics := C.malloc(C.size_t(pipelineStatisticCount) * C.size_t(unsafe.Sizeof(C.WGPUPipelineStatisticName(0))))
		// 	defer C.free(pipelineStatistics)

		// 	pipelineStatisticsSlice := unsafe.Slice((*PipelineStatisticName)(pipelineStatistics), pipelineStatisticCount)
		// 	copy(pipelineStatisticsSlice, descriptor.PipelineStatistics)

		// 	desc.pipelineStatisticCount = C.size_t(pipelineStatisticCount)
		// 	desc.pipelineStatistics = (*C.WGPUPipelineStatisticName)(pipelineStatistics)
		// }
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateQuerySet(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_query_set(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuQuerySetRelease(ref)
		return nil, err
	}

	return &QuerySet{ref: ref}, nil
}

type RenderBundleEncoderDescriptor struct {
	Label              string
	ColorFormats       []TextureFormat
	DepthStencilFormat TextureFormat
	SampleCount        uint32
	DepthReadOnly      bool
	StencilReadOnly    bool
}

func (p *Device) CreateRenderBundleEncoder(descriptor *RenderBundleEncoderDescriptor) (*RenderBundleEncoder, error) {
	var desc C.WGPURenderBundleEncoderDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))
			desc.label = label
		}

		colorFormatCount := len(descriptor.ColorFormats)
		if colorFormatCount > 0 {
			colorFormats := C.malloc(C.size_t(colorFormatCount) * C.size_t(unsafe.Sizeof(C.WGPUTextureFormat(0))))
			defer C.free(colorFormats)

			colorFormatsSlice := unsafe.Slice((*TextureFormat)(colorFormats), colorFormatCount)
			copy(colorFormatsSlice, descriptor.ColorFormats)

			desc.colorFormatCount = C.size_t(colorFormatCount)
			desc.colorFormats = (*C.WGPUTextureFormat)(colorFormats)
		}

		desc.depthStencilFormat = C.WGPUTextureFormat(descriptor.DepthStencilFormat)
		desc.sampleCount = C.uint32_t(descriptor.SampleCount)
		desc.depthReadOnly = cBool(descriptor.DepthReadOnly)
		desc.stencilReadOnly = cBool(descriptor.StencilReadOnly)
	}

	ref := C.wgpuDeviceCreateRenderBundleEncoder(p.ref, &desc)

	return &RenderBundleEncoder{ref}, nil
}

type BlendComponent struct {
	Operation BlendOperation
	SrcFactor BlendFactor
	DstFactor BlendFactor
}

type BlendState struct {
	Color BlendComponent
	Alpha BlendComponent
}

type ColorTargetState struct {
	Format    TextureFormat
	Blend     *BlendState
	WriteMask ColorWriteMask
}

type FragmentState struct {
	Module     *ShaderModule
	EntryPoint string
	Targets    []ColorTargetState

	// unused in wgpu
	// Constants  []ConstantEntry
}

type VertexAttribute struct {
	Format         VertexFormat
	Offset         uint64
	ShaderLocation uint32
}

type VertexBufferLayout struct {
	ArrayStride uint64
	StepMode    VertexStepMode
	Attributes  []VertexAttribute
}

type VertexState struct {
	Module     *ShaderModule
	EntryPoint string
	Buffers    []VertexBufferLayout

	// unused in wgpu
	// Constants  []ConstantEntry
}

type PrimitiveState struct {
	Topology         PrimitiveTopology
	StripIndexFormat IndexFormat
	FrontFace        FrontFace
	CullMode         CullMode
}

type StencilFaceState struct {
	Compare     CompareFunction
	FailOp      StencilOperation
	DepthFailOp StencilOperation
	PassOp      StencilOperation
}

type DepthStencilState struct {
	Format              TextureFormat
	DepthWriteEnabled   bool
	DepthCompare        CompareFunction
	StencilFront        StencilFaceState
	StencilBack         StencilFaceState
	StencilReadMask     uint32
	StencilWriteMask    uint32
	DepthBias           int32
	DepthBiasSlopeScale float32
	DepthBiasClamp      float32
}

type MultisampleState struct {
	Count                  uint32
	Mask                   uint32
	AlphaToCoverageEnabled bool
}

func (p *Device) CreateRenderPipeline(descriptor *RenderPipelineDescriptor) (*RenderPipeline, error) {
	var desc C.WGPURenderPipelineDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}

		if descriptor.Layout != nil {
			desc.layout = descriptor.Layout.ref
		}

		// vertex
		{
			vertex := descriptor.Vertex

			var vert C.WGPUVertexState

			if vertex.Module != nil {
				vert.module = vertex.Module.ref
			}

			if vertex.EntryPoint != "" {
				entryPoint := C.CString(vertex.EntryPoint)
				defer C.free(unsafe.Pointer(entryPoint))

				vert.entryPoint = entryPoint
			}

			bufferCount := len(vertex.Buffers)
			if bufferCount > 0 {
				buffers := C.malloc(C.size_t(bufferCount) * C.size_t(unsafe.Sizeof(C.WGPUVertexBufferLayout{})))
				defer C.free(buffers)

				buffersSlice := unsafe.Slice((*C.WGPUVertexBufferLayout)(buffers), bufferCount)

				for i, v := range vertex.Buffers {
					buffer := C.WGPUVertexBufferLayout{
						arrayStride: C.uint64_t(v.ArrayStride),
						stepMode:    C.WGPUVertexStepMode(v.StepMode),
					}

					attributeCount := len(v.Attributes)
					if attributeCount > 0 {
						attributes := C.malloc(C.size_t(attributeCount) * C.size_t(unsafe.Sizeof(C.WGPUVertexAttribute{})))
						defer C.free(attributes)

						attributesSlice := unsafe.Slice((*C.WGPUVertexAttribute)(attributes), attributeCount)

						for j, attribute := range v.Attributes {
							attributesSlice[j] = C.WGPUVertexAttribute{
								format:         C.WGPUVertexFormat(attribute.Format),
								offset:         C.uint64_t(attribute.Offset),
								shaderLocation: C.uint32_t(attribute.ShaderLocation),
							}
						}

						buffer.attributeCount = C.size_t(attributeCount)
						buffer.attributes = (*C.WGPUVertexAttribute)(attributes)
					}

					buffersSlice[i] = buffer
				}

				vert.bufferCount = C.size_t(bufferCount)
				vert.buffers = (*C.WGPUVertexBufferLayout)(buffers)
			}

			desc.vertex = vert
		}

		desc.primitive = C.WGPUPrimitiveState{
			topology:         C.WGPUPrimitiveTopology(descriptor.Primitive.Topology),
			stripIndexFormat: C.WGPUIndexFormat(descriptor.Primitive.StripIndexFormat),
			frontFace:        C.WGPUFrontFace(descriptor.Primitive.FrontFace),
			cullMode:         C.WGPUCullMode(descriptor.Primitive.CullMode),
		}

		if descriptor.DepthStencil != nil {
			depthStencil := descriptor.DepthStencil

			ds := (*C.WGPUDepthStencilState)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUDepthStencilState{}))))
			defer C.free(unsafe.Pointer(ds))

			ds.nextInChain = nil
			ds.format = C.WGPUTextureFormat(depthStencil.Format)
			ds.depthWriteEnabled = cBool(depthStencil.DepthWriteEnabled)
			ds.depthCompare = C.WGPUCompareFunction(depthStencil.DepthCompare)
			ds.stencilFront = C.WGPUStencilFaceState{
				compare:     C.WGPUCompareFunction(depthStencil.StencilFront.Compare),
				failOp:      C.WGPUStencilOperation(depthStencil.StencilFront.FailOp),
				depthFailOp: C.WGPUStencilOperation(depthStencil.StencilFront.DepthFailOp),
				passOp:      C.WGPUStencilOperation(depthStencil.StencilFront.PassOp),
			}
			ds.stencilBack = C.WGPUStencilFaceState{
				compare:     C.WGPUCompareFunction(depthStencil.StencilBack.Compare),
				failOp:      C.WGPUStencilOperation(depthStencil.StencilBack.FailOp),
				depthFailOp: C.WGPUStencilOperation(depthStencil.StencilBack.DepthFailOp),
				passOp:      C.WGPUStencilOperation(depthStencil.StencilBack.PassOp),
			}
			ds.stencilReadMask = C.uint32_t(depthStencil.StencilReadMask)
			ds.stencilWriteMask = C.uint32_t(depthStencil.StencilWriteMask)
			ds.depthBias = C.int32_t(depthStencil.DepthBias)
			ds.depthBiasSlopeScale = C.float(depthStencil.DepthBiasSlopeScale)
			ds.depthBiasClamp = C.float(depthStencil.DepthBiasClamp)

			desc.depthStencil = ds
		}

		desc.multisample = C.WGPUMultisampleState{
			count:                  C.uint32_t(descriptor.Multisample.Count),
			mask:                   C.uint32_t(descriptor.Multisample.Mask),
			alphaToCoverageEnabled: cBool(descriptor.Multisample.AlphaToCoverageEnabled),
		}

		if descriptor.Fragment != nil {
			fragment := descriptor.Fragment

			frag := (*C.WGPUFragmentState)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUFragmentState{}))))
			defer C.free(unsafe.Pointer(frag))

			frag.nextInChain = nil
			if fragment.EntryPoint != "" {
				entryPoint := C.CString(fragment.EntryPoint)
				defer C.free(unsafe.Pointer(entryPoint))

				frag.entryPoint = entryPoint
			}

			if fragment.Module != nil {
				frag.module = fragment.Module.ref
			}

			targetCount := len(fragment.Targets)
			if targetCount > 0 {
				targets := C.malloc(C.size_t(targetCount) * C.size_t(unsafe.Sizeof(C.WGPUColorTargetState{})))
				defer C.free(targets)

				targetsSlice := unsafe.Slice((*C.WGPUColorTargetState)(targets), targetCount)

				for i, v := range fragment.Targets {
					target := C.WGPUColorTargetState{
						format:    C.WGPUTextureFormat(v.Format),
						writeMask: C.WGPUColorWriteMaskFlags(v.WriteMask),
					}

					if v.Blend != nil {
						blend := (*C.WGPUBlendState)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUBlendState{}))))
						defer C.free(unsafe.Pointer(blend))

						blend.color = C.WGPUBlendComponent{
							operation: C.WGPUBlendOperation(v.Blend.Color.Operation),
							srcFactor: C.WGPUBlendFactor(v.Blend.Color.SrcFactor),
							dstFactor: C.WGPUBlendFactor(v.Blend.Color.DstFactor),
						}
						blend.alpha = C.WGPUBlendComponent{
							operation: C.WGPUBlendOperation(v.Blend.Alpha.Operation),
							srcFactor: C.WGPUBlendFactor(v.Blend.Alpha.SrcFactor),
							dstFactor: C.WGPUBlendFactor(v.Blend.Alpha.DstFactor),
						}

						target.blend = blend
					}

					targetsSlice[i] = target
				}

				frag.targetCount = C.size_t(targetCount)
				frag.targets = (*C.WGPUColorTargetState)(targets)
			} else {
				frag.targetCount = 0
				frag.targets = nil
			}
			frag.constantCount = 0 // note: crashes on linux arm64 without setting this to 0
			frag.constants = nil // even though wgpu doesn't even support it.

			desc.fragment = frag
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateRenderPipeline(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_render_pipeline(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuRenderPipelineRelease(ref)
		return nil, err
	}

	return &RenderPipeline{ref}, nil
}

func (p *Device) CreateSampler(descriptor *SamplerDescriptor) (*Sampler, error) {
	var desc *C.WGPUSamplerDescriptor

	if descriptor != nil {
		desc = &C.WGPUSamplerDescriptor{
			addressModeU:  C.WGPUAddressMode(descriptor.AddressModeU),
			addressModeV:  C.WGPUAddressMode(descriptor.AddressModeV),
			addressModeW:  C.WGPUAddressMode(descriptor.AddressModeW),
			magFilter:     C.WGPUFilterMode(descriptor.MagFilter),
			minFilter:     C.WGPUFilterMode(descriptor.MinFilter),
			mipmapFilter:  C.WGPUMipmapFilterMode(descriptor.MipmapFilter),
			lodMinClamp:   C.float(descriptor.LodMinClamp),
			lodMaxClamp:   C.float(descriptor.LodMaxClamp),
			compare:       C.WGPUCompareFunction(descriptor.Compare),
			maxAnisotropy: C.uint16_t(descriptor.MaxAnisotropy),
		}

		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateSampler(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_sampler(
		p.ref,
		desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuSamplerRelease(ref)
		return nil, err
	}

	return &Sampler{ref}, nil
}

type ShaderModuleSPIRVDescriptor struct {
	Code []byte
}

type ShaderModuleGLSLDescriptor struct {
	Code        string
	Defines     map[string]string
	ShaderStage ShaderStage
}

type ShaderModuleDescriptor struct {
	Label           string
	SPIRVDescriptor *ShaderModuleSPIRVDescriptor
	WGSLDescriptor  *ShaderModuleWGSLDescriptor
	GLSLDescriptor  *ShaderModuleGLSLDescriptor
}

func (p *Device) CreateShaderModule(descriptor *ShaderModuleDescriptor) (*ShaderModule, error) {
	var desc C.WGPUShaderModuleDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}

		switch {
		case descriptor.SPIRVDescriptor != nil:
			spirv := (*C.WGPUShaderModuleSPIRVDescriptor)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUShaderModuleSPIRVDescriptor{}))))
			defer C.free(unsafe.Pointer(spirv))

			codeSize := len(descriptor.SPIRVDescriptor.Code)
			if codeSize > 0 {
				code := C.CBytes(descriptor.SPIRVDescriptor.Code)
				defer C.free(code)

				spirv.codeSize = C.uint32_t(codeSize)
				spirv.code = (*C.uint32_t)(code)
			} else {
				spirv.code = nil
				spirv.codeSize = 0
			}

			spirv.chain.next = nil
			spirv.chain.sType = C.WGPUSType_ShaderModuleSPIRVDescriptor

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(spirv))

		case descriptor.WGSLDescriptor != nil:
			wgsl := (*C.WGPUShaderModuleWGSLDescriptor)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUShaderModuleWGSLDescriptor{}))))
			defer C.free(unsafe.Pointer(wgsl))

			if descriptor.WGSLDescriptor.Code != "" {
				code := C.CString(descriptor.WGSLDescriptor.Code)
				defer C.free(unsafe.Pointer(code))

				wgsl.code = code
			} else {
				wgsl.code = nil
			}

			wgsl.chain.next = nil
			wgsl.chain.sType = C.WGPUSType_ShaderModuleWGSLDescriptor

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(wgsl))

		case descriptor.GLSLDescriptor != nil:
			glsl := (*C.WGPUShaderModuleGLSLDescriptor)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUShaderModuleGLSLDescriptor{}))))
			defer C.free(unsafe.Pointer(glsl))

			if descriptor.GLSLDescriptor.Code != "" {
				code := C.CString(descriptor.GLSLDescriptor.Code)
				defer C.free(unsafe.Pointer(code))

				glsl.code = code
			} else {
				glsl.code = nil
			}

			defineCount := len(descriptor.GLSLDescriptor.Defines)
			if defineCount > 0 {
				shaderDefines := C.malloc(C.size_t(unsafe.Sizeof(C.WGPUShaderDefine{})) * C.size_t(defineCount))
				defer C.free(shaderDefines)

				shaderDefinesSlice := unsafe.Slice((*C.WGPUShaderDefine)(shaderDefines), defineCount)
				index := 0

				for name, value := range descriptor.GLSLDescriptor.Defines {
					namePtr := C.CString(name)
					defer C.free(unsafe.Pointer(namePtr))
					valuePtr := C.CString(value)
					defer C.free(unsafe.Pointer(valuePtr))

					shaderDefinesSlice[index] = C.WGPUShaderDefine{
						name:  namePtr,
						value: valuePtr,
					}
					index++
				}

				glsl.defineCount = C.uint32_t(defineCount)
				glsl.defines = (*C.WGPUShaderDefine)(shaderDefines)
			} else {
				glsl.defineCount = 0
				glsl.defines = nil
			}

			glsl.stage = C.WGPUShaderStage(descriptor.GLSLDescriptor.ShaderStage)
			glsl.chain.next = nil
			glsl.chain.sType = C.WGPUSType_ShaderModuleGLSLDescriptor

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(glsl))
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateShaderModule(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_shader_module(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuShaderModuleRelease(ref)
		return nil, err
	}

	return &ShaderModule{ref}, nil
}

func (p *Device) CreateTexture(descriptor *TextureDescriptor) (*Texture, error) {
	var desc C.WGPUTextureDescriptor

	if descriptor != nil {
		desc = C.WGPUTextureDescriptor{
			usage:     C.WGPUTextureUsageFlags(descriptor.Usage),
			dimension: C.WGPUTextureDimension(descriptor.Dimension),
			size: C.WGPUExtent3D{
				width:              C.uint32_t(descriptor.Size.Width),
				height:             C.uint32_t(descriptor.Size.Height),
				depthOrArrayLayers: C.uint32_t(descriptor.Size.DepthOrArrayLayers),
			},
			format:        C.WGPUTextureFormat(descriptor.Format),
			mipLevelCount: C.uint32_t(descriptor.MipLevelCount),
			sampleCount:   C.uint32_t(descriptor.SampleCount),
		}

		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Device).CreateTexture(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_device_create_texture(
		p.ref,
		&desc,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuTextureRelease(ref)
		return nil, err
	}

	C.wgpuDeviceReference(p.ref)
	return &Texture{deviceRef: p.ref, ref: ref}, nil
}

func (p *Device) EnumerateFeatures() []FeatureName {
	size := C.wgpuDeviceEnumerateFeatures(p.ref, nil)
	if size == 0 {
		return nil
	}

	features := make([]FeatureName, size)
	C.wgpuDeviceEnumerateFeatures(p.ref, (*C.WGPUFeatureName)(unsafe.Pointer(&features[0])))
	return features
}

func (p *Device) GetLimits() SupportedLimits {
	var supportedLimits C.WGPUSupportedLimits

	extras := (*C.WGPUSupportedLimitsExtras)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUSupportedLimitsExtras{}))))
	defer C.free(unsafe.Pointer(extras))
	supportedLimits.nextInChain = (*C.WGPUChainedStructOut)(unsafe.Pointer(extras))

	C.wgpuDeviceGetLimits(p.ref, &supportedLimits)

	limits := supportedLimits.limits
	return SupportedLimits{
		Limits{
			MaxTextureDimension1D:                     uint32(limits.maxTextureDimension1D),
			MaxTextureDimension2D:                     uint32(limits.maxTextureDimension2D),
			MaxTextureDimension3D:                     uint32(limits.maxTextureDimension3D),
			MaxTextureArrayLayers:                     uint32(limits.maxTextureArrayLayers),
			MaxBindGroups:                             uint32(limits.maxBindGroups),
			MaxBindingsPerBindGroup:                   uint32(limits.maxBindingsPerBindGroup),
			MaxDynamicUniformBuffersPerPipelineLayout: uint32(limits.maxDynamicUniformBuffersPerPipelineLayout),
			MaxDynamicStorageBuffersPerPipelineLayout: uint32(limits.maxDynamicStorageBuffersPerPipelineLayout),
			MaxSampledTexturesPerShaderStage:          uint32(limits.maxSampledTexturesPerShaderStage),
			MaxSamplersPerShaderStage:                 uint32(limits.maxSamplersPerShaderStage),
			MaxStorageBuffersPerShaderStage:           uint32(limits.maxStorageBuffersPerShaderStage),
			MaxStorageTexturesPerShaderStage:          uint32(limits.maxStorageTexturesPerShaderStage),
			MaxUniformBuffersPerShaderStage:           uint32(limits.maxUniformBuffersPerShaderStage),
			MaxUniformBufferBindingSize:               uint64(limits.maxUniformBufferBindingSize),
			MaxStorageBufferBindingSize:               uint64(limits.maxStorageBufferBindingSize),
			MinUniformBufferOffsetAlignment:           uint32(limits.minUniformBufferOffsetAlignment),
			MinStorageBufferOffsetAlignment:           uint32(limits.minStorageBufferOffsetAlignment),
			MaxVertexBuffers:                          uint32(limits.maxVertexBuffers),
			MaxBufferSize:                             uint64(limits.maxBufferSize),
			MaxVertexAttributes:                       uint32(limits.maxVertexAttributes),
			MaxVertexBufferArrayStride:                uint32(limits.maxVertexBufferArrayStride),
			MaxInterStageShaderComponents:             uint32(limits.maxInterStageShaderComponents),
			MaxInterStageShaderVariables:              uint32(limits.maxInterStageShaderVariables),
			MaxColorAttachments:                       uint32(limits.maxColorAttachments),
			MaxComputeWorkgroupStorageSize:            uint32(limits.maxComputeWorkgroupStorageSize),
			MaxComputeInvocationsPerWorkgroup:         uint32(limits.maxComputeInvocationsPerWorkgroup),
			MaxComputeWorkgroupSizeX:                  uint32(limits.maxComputeWorkgroupSizeX),
			MaxComputeWorkgroupSizeY:                  uint32(limits.maxComputeWorkgroupSizeY),
			MaxComputeWorkgroupSizeZ:                  uint32(limits.maxComputeWorkgroupSizeZ),
			MaxComputeWorkgroupsPerDimension:          uint32(limits.maxComputeWorkgroupsPerDimension),

			MaxPushConstantSize: uint32(extras.limits.maxPushConstantSize),
		},
	}
}

func (p *Device) GetQueue() *Queue {
	ref := C.wgpuDeviceGetQueue(p.ref)
	C.wgpuDeviceReference(p.ref)
	return &Queue{deviceRef: p.ref, ref: ref}
}

func (p *Device) HasFeature(feature FeatureName) bool {
	hasFeature := C.wgpuDeviceHasFeature(p.ref, C.WGPUFeatureName(feature))
	return goBool(hasFeature)
}

func (p *Device) Poll(wait bool, wrappedSubmissionIndex *WrappedSubmissionIndex) (queueEmpty bool) {
	var index *C.WGPUWrappedSubmissionIndex
	if wrappedSubmissionIndex != nil {
		index = &C.WGPUWrappedSubmissionIndex{
			queue:           wrappedSubmissionIndex.Queue.ref,
			submissionIndex: C.WGPUSubmissionIndex(wrappedSubmissionIndex.SubmissionIndex),
		}
	}

	return goBool(C.wgpuDevicePoll(p.ref, cBool(wait), index))
}
