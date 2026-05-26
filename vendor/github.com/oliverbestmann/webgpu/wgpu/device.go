//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

extern void gowebgpu_error_callback_c(enum WGPUPopErrorScopeStatus status, WGPUErrorType type, WGPUStringView message, void * userdata, void * userdata2);

static inline WGPUBindGroup gowebgpu_device_create_bind_group(WGPUDevice device, WGPUBindGroupDescriptor const * descriptor, void * error_userdata) {
	WGPUBindGroup ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateBindGroup(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUBindGroupLayout gowebgpu_device_create_bind_group_layout(WGPUDevice device, WGPUBindGroupLayoutDescriptor const * descriptor, void * error_userdata) {
	WGPUBindGroupLayout ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateBindGroupLayout(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUBuffer gowebgpu_device_create_buffer(WGPUDevice device, WGPUBufferDescriptor const * descriptor, void * error_userdata) {
	WGPUBuffer ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateBuffer(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUCommandEncoder gowebgpu_device_create_command_encoder(WGPUDevice device, WGPUCommandEncoderDescriptor const * descriptor, void * error_userdata) {
	WGPUCommandEncoder ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateCommandEncoder(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUComputePipeline gowebgpu_device_create_compute_pipeline(WGPUDevice device, WGPUComputePipelineDescriptor const * descriptor, void * error_userdata) {
	WGPUComputePipeline ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateComputePipeline(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUPipelineLayout gowebgpu_device_create_pipeline_layout(WGPUDevice device, WGPUPipelineLayoutDescriptor const * descriptor, void * error_userdata) {
	WGPUPipelineLayout ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreatePipelineLayout(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUQuerySet gowebgpu_device_create_query_set(WGPUDevice device, WGPUQuerySetDescriptor const * descriptor, void * error_userdata) {
	WGPUQuerySet ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateQuerySet(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPURenderPipeline gowebgpu_device_create_render_pipeline(WGPUDevice device, WGPURenderPipelineDescriptor const * descriptor, void * error_userdata) {
	WGPURenderPipeline ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateRenderPipeline(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUSampler gowebgpu_device_create_sampler(WGPUDevice device, WGPUSamplerDescriptor const * descriptor, void * error_userdata) {
	WGPUSampler ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateSampler(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUShaderModule gowebgpu_device_create_shader_module(WGPUDevice device, WGPUShaderModuleDescriptor const * descriptor, void * error_userdata) {
	WGPUShaderModule ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateShaderModule(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline WGPUTexture gowebgpu_device_create_texture(WGPUDevice device, WGPUTextureDescriptor const * descriptor, void * error_userdata) {
	WGPUTexture ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuDeviceCreateTexture(device, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

*/
import "C"
import (
	"unsafe"
)

func (g *Device) TryCreateBindGroup(descriptor *BindGroupDescriptor) (*BindGroup, error) {
	var desc *C.WGPUBindGroupDescriptor = allocWGPUBindGroupDescriptor.GetZeroed()
	defer allocWGPUBindGroupDescriptor.Put(desc)

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		if descriptor.Layout != nil {
			desc.layout = descriptor.Layout.ref
		}

		entryCount := len(descriptor.Entries)
		if entryCount > 0 {
			entries := C.calloc(C.size_t(entryCount), C.size_t(unsafe.Sizeof(C.WGPUBindGroupEntry{})))
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

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_bind_group(
		g.ref,
		desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuBindGroupRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&BindGroup{ref: ref}), nil
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

func (g *Device) TryCreateBindGroupLayout(descriptor *BindGroupLayoutDescriptor) (*BindGroupLayout, error) {
	var desc C.WGPUBindGroupLayoutDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		entryCount := len(descriptor.Entries)
		if entryCount > 0 {
			entries := C.calloc(C.size_t(entryCount), C.size_t(unsafe.Sizeof(C.WGPUBindGroupLayoutEntry{})))
			defer C.free(entries)

			entriesSlice := unsafe.Slice((*C.WGPUBindGroupLayoutEntry)(entries), entryCount)

			for i, v := range descriptor.Entries {
				entriesSlice[i] = C.WGPUBindGroupLayoutEntry{
					nextInChain: nil,
					binding:     C.uint32_t(v.Binding),
					visibility:  C.WGPUShaderStage(v.Visibility),
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

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_bind_group_layout(
		g.ref,
		&desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuBindGroupLayoutRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&BindGroupLayout{ref: ref}), nil
}

func (g *Device) TryCreateBuffer(descriptor *BufferDescriptor) (*Buffer, error) {
	var desc C.WGPUBufferDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		desc.usage = C.WGPUBufferUsage(descriptor.Usage)
		desc.size = C.uint64_t(descriptor.Size)
		desc.mappedAtCreation = cBool(descriptor.MappedAtCreation)
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_buffer(
		g.ref,
		&desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuBufferRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&Buffer{device: g.addRef(), ref: ref}), nil
}

func (g *Device) TryCreateCommandEncoder(descriptor *CommandEncoderDescriptor) (*CommandEncoder, error) {
	var desc *C.WGPUCommandEncoderDescriptor

	if descriptor != nil && descriptor.Label != "" {
		label := C.CString(descriptor.Label)
		defer C.free(unsafe.Pointer(label))

		desc = allocWGPUCommandEncoderDescriptor.GetZeroed()
		defer allocWGPUCommandEncoderDescriptor.Put(desc)

		desc.label = C.WGPUStringView{
			data:   label,
			length: C.WGPU_STRLEN,
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_command_encoder(
		g.ref,
		desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuCommandEncoderRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&CommandEncoder{device: g.addRef(), ref: ref}), nil
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

func (g *Device) TryCreateComputePipeline(descriptor *ComputePipelineDescriptor) (*ComputePipeline, error) {
	var desc C.WGPUComputePipelineDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		if descriptor.Layout != nil {
			desc.layout = descriptor.Layout.ref
		}

		var compute C.WGPUComputeState
		if descriptor.Compute.Module != nil {
			compute.module = descriptor.Compute.Module.ref
		}
		if descriptor.Compute.EntryPoint != "" {
			entryPoint := C.CString(descriptor.Compute.EntryPoint)
			defer C.free(unsafe.Pointer(entryPoint))

			compute.entryPoint.data = entryPoint
			compute.entryPoint.length = C.WGPU_STRLEN
		}
		desc.compute = compute
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_compute_pipeline(
		g.ref,
		&desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuComputePipelineRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&ComputePipeline{ref: ref}), nil
}

type PushConstantRange struct {
	Stages ShaderStage
	Start  uint32
	End    uint32
}

type PipelineLayoutDescriptor struct {
	Label            string
	BindGroupLayouts []*BindGroupLayout

	// enable support for immediates
	EnableImmediates bool
}

func (g *Device) TryCreatePipelineLayout(descriptor *PipelineLayoutDescriptor) (*PipelineLayout, error) {
	var desc C.WGPUPipelineLayoutDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		bindGroupLayoutCount := len(descriptor.BindGroupLayouts)
		if bindGroupLayoutCount > 0 {
			bindGroupLayouts := C.calloc(C.size_t(bindGroupLayoutCount), C.size_t(unsafe.Sizeof(C.WGPUBindGroupLayout(nil))))
			defer C.free(bindGroupLayouts)

			bindGroupLayoutsSlice := unsafe.Slice((*C.WGPUBindGroupLayout)(bindGroupLayouts), bindGroupLayoutCount)

			for i, v := range descriptor.BindGroupLayouts {
				bindGroupLayoutsSlice[i] = v.ref
			}

			desc.bindGroupLayoutCount = C.size_t(bindGroupLayoutCount)
			desc.bindGroupLayouts = (*C.WGPUBindGroupLayout)(bindGroupLayouts)
		}

		if descriptor.EnableImmediates {
			pipelineLayoutExtras := (*C.WGPUPipelineLayoutExtras)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUPipelineLayoutExtras{}))))
			defer C.free(unsafe.Pointer(pipelineLayoutExtras))

			pipelineLayoutExtras.chain.next = nil
			pipelineLayoutExtras.chain.sType = C.WGPUSType_PipelineLayoutExtras

			pipelineLayoutExtras.immediateDataSize = 4

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(pipelineLayoutExtras))
		} else {
			desc.nextInChain = nil
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_pipeline_layout(
		g.ref,
		&desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuPipelineLayoutRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&PipelineLayout{ref: ref}), nil
}

func (g *Device) TryCreateQuerySet(descriptor *QuerySetDescriptor) (*QuerySet, error) {
	var desc C.WGPUQuerySetDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		desc._type = C.WGPUQueryType(descriptor.Type)
		desc.count = C.uint32_t(descriptor.Count)
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_query_set(
		g.ref,
		&desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuQuerySetRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&QuerySet{ref: ref}), nil
}

func (g *Device) TryCreateRenderBundleEncoder(descriptor *RenderBundleEncoderDescriptor) (*RenderBundleEncoder, error) {
	var desc C.WGPURenderBundleEncoderDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		colorFormatCount := len(descriptor.ColorFormats)
		if colorFormatCount > 0 {
			colorFormats := C.calloc(C.size_t(colorFormatCount), C.size_t(unsafe.Sizeof(C.WGPUTextureFormat(0))))
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

	ref := C.wgpuDeviceCreateRenderBundleEncoder(g.ref, &desc)

	return releaseOnGC(&RenderBundleEncoder{ref: ref}), nil
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
	DepthWriteEnabled   OptionalBool
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

func (g *Device) TryCreateRenderPipeline(descriptor *RenderPipelineDescriptor) (*RenderPipeline, error) {
	var desc C.WGPURenderPipelineDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
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

				vert.entryPoint.data = entryPoint
				vert.entryPoint.length = C.WGPU_STRLEN
			}

			bufferCount := len(vertex.Buffers)
			if bufferCount > 0 {
				buffers := C.calloc(C.size_t(bufferCount), C.size_t(unsafe.Sizeof(C.WGPUVertexBufferLayout{})))
				defer C.free(buffers)

				buffersSlice := unsafe.Slice((*C.WGPUVertexBufferLayout)(buffers), bufferCount)

				for i, v := range vertex.Buffers {
					buffer := C.WGPUVertexBufferLayout{
						arrayStride: C.uint64_t(v.ArrayStride),
						stepMode:    C.WGPUVertexStepMode(v.StepMode),
					}

					attributeCount := len(v.Attributes)
					if attributeCount > 0 {
						attributes := C.calloc(C.size_t(attributeCount), C.size_t(unsafe.Sizeof(C.WGPUVertexAttribute{})))
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

			ds := (*C.WGPUDepthStencilState)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUDepthStencilState{}))))
			defer C.free(unsafe.Pointer(ds))

			ds.nextInChain = nil
			ds.format = C.WGPUTextureFormat(depthStencil.Format)
			ds.depthWriteEnabled = C.WGPUOptionalBool(depthStencil.DepthWriteEnabled)
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

			frag := (*C.WGPUFragmentState)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUFragmentState{}))))
			defer C.free(unsafe.Pointer(frag))

			frag.nextInChain = nil
			if fragment.EntryPoint != "" {
				entryPoint := C.CString(fragment.EntryPoint)
				defer C.free(unsafe.Pointer(entryPoint))

				frag.entryPoint.data = entryPoint
				frag.entryPoint.length = C.WGPU_STRLEN
			}

			if fragment.Module != nil {
				frag.module = fragment.Module.ref
			}

			targetCount := len(fragment.Targets)
			if targetCount > 0 {
				targets := C.calloc(C.size_t(targetCount), C.size_t(unsafe.Sizeof(C.WGPUColorTargetState{})))
				defer C.free(targets)

				targetsSlice := unsafe.Slice((*C.WGPUColorTargetState)(targets), targetCount)

				for i, v := range fragment.Targets {
					target := C.WGPUColorTargetState{
						format:    C.WGPUTextureFormat(v.Format),
						writeMask: C.WGPUColorWriteMask(v.WriteMask),
					}

					if v.Blend != nil {
						blend := (*C.WGPUBlendState)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUBlendState{}))))
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
			frag.constants = nil   // even though wgpu doesn't even support it.

			desc.fragment = frag
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_render_pipeline(
		g.ref,
		&desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuRenderPipelineRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&RenderPipeline{ref: ref}), nil
}

func (g *Device) TryCreateSampler(descriptor *SamplerDescriptor) (*Sampler, error) {
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

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_sampler(
		g.ref,
		desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuSamplerRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&Sampler{ref: ref}), nil
}

type ShaderSourceGLSL struct {
	Code        string
	Defines     map[string]string
	ShaderStage ShaderStage
}

type ShaderModuleDescriptor struct {
	Label       string
	SPIRVSource *ShaderSourceSPIRV
	WGSLSource  *ShaderSourceWGSL
	GLSLSource  *ShaderSourceGLSL
}

func (g *Device) TryCreateShaderModule(descriptor *ShaderModuleDescriptor) (*ShaderModule, error) {
	var desc C.WGPUShaderModuleDescriptor

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		switch {
		case descriptor.SPIRVSource != nil:
			spirv := (*C.WGPUShaderSourceSPIRV)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUShaderSourceSPIRV{}))))
			defer C.free(unsafe.Pointer(spirv))

			codeSize := len(descriptor.SPIRVSource.Code)
			if codeSize > 0 {
				code := C.CBytes(descriptor.SPIRVSource.Code)
				defer C.free(code)

				spirv.codeSize = C.uint32_t(codeSize)
				spirv.code = (*C.uint32_t)(code)
			} else {
				spirv.code = nil
				spirv.codeSize = 0
			}

			spirv.chain.next = nil
			spirv.chain.sType = C.WGPUSType_ShaderSourceSPIRV

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(spirv))

		case descriptor.WGSLSource != nil:
			wgsl := (*C.WGPUShaderSourceWGSL)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUShaderSourceWGSL{}))))
			defer C.free(unsafe.Pointer(wgsl))

			if descriptor.WGSLSource.Code != "" {
				code := C.CString(descriptor.WGSLSource.Code)
				defer C.free(unsafe.Pointer(code))

				wgsl.code.data = code
				wgsl.code.length = C.WGPU_STRLEN
			} else {
				wgsl.code.data = nil
				wgsl.code.length = 0
			}

			wgsl.chain.next = nil
			wgsl.chain.sType = C.WGPUSType_ShaderSourceWGSL

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(wgsl))

		case descriptor.GLSLSource != nil:
			glsl := (*C.WGPUShaderSourceGLSL)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUShaderSourceGLSL{}))))
			defer C.free(unsafe.Pointer(glsl))

			if descriptor.GLSLSource.Code != "" {
				code := C.CString(descriptor.GLSLSource.Code)
				defer C.free(unsafe.Pointer(code))

				glsl.code.data = code
				glsl.code.length = C.WGPU_STRLEN
			} else {
				glsl.code.data = nil
				glsl.code.length = 0
			}

			defineCount := len(descriptor.GLSLSource.Defines)
			if defineCount > 0 {
				shaderDefines := C.calloc(C.size_t(unsafe.Sizeof(C.WGPUShaderDefine{})), C.size_t(defineCount))
				defer C.free(shaderDefines)

				shaderDefinesSlice := unsafe.Slice((*C.WGPUShaderDefine)(shaderDefines), defineCount)
				index := 0

				for name, value := range descriptor.GLSLSource.Defines {
					namePtr := C.CString(name)
					defer C.free(unsafe.Pointer(namePtr))
					valuePtr := C.CString(value)
					defer C.free(unsafe.Pointer(valuePtr))

					shaderDefinesSlice[index] = C.WGPUShaderDefine{
						name:  C.WGPUStringView{data: namePtr, length: C.WGPU_STRLEN},
						value: C.WGPUStringView{data: valuePtr, length: C.WGPU_STRLEN},
					}
					index++
				}

				glsl.defineCount = C.uint32_t(defineCount)
				glsl.defines = (*C.WGPUShaderDefine)(shaderDefines)
			} else {
				glsl.defineCount = 0
				glsl.defines = nil
			}

			glsl.stage = C.WGPUShaderStage(descriptor.GLSLSource.ShaderStage)
			glsl.chain.next = nil
			glsl.chain.sType = C.WGPUSType_ShaderSourceGLSL

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(glsl))
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_shader_module(
		g.ref,
		&desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuShaderModuleRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&ShaderModule{ref: ref}), nil
}

func (g *Device) TryCreateTexture(descriptor *TextureDescriptor) (*Texture, error) {
	var desc C.WGPUTextureDescriptor

	if descriptor != nil {
		desc = C.WGPUTextureDescriptor{
			usage:     C.WGPUTextureUsage(descriptor.Usage),
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

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_device_create_texture(
		g.ref,
		&desc,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuTextureRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&Texture{device: g.addRef(), ref: ref}), nil
}

func (g *Device) GetFeatures() []FeatureName {
	var supportedFeatures C.WGPUSupportedFeatures
	C.wgpuDeviceGetFeatures(g.ref, (*C.WGPUSupportedFeatures)(unsafe.Pointer(&supportedFeatures)))
	defer C.free(unsafe.Pointer(supportedFeatures.features))

	features := make([]FeatureName, supportedFeatures.featureCount)

	for i := range int(supportedFeatures.featureCount) {
		offset := uintptr(i) * unsafe.Sizeof(C.WGPUFeatureName(0))
		features[i] = FeatureName(*(*C.WGPUFeatureName)(unsafe.Pointer(uintptr(unsafe.Pointer(supportedFeatures.features)) + offset)))
	}

	return features
}

func (g *Device) GetLimits() Limits {
	var limits C.WGPULimits

	nativeLimits := (*C.WGPUNativeLimits)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUNativeLimits{}))))
	defer C.free(unsafe.Pointer(&nativeLimits))
	limits.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(&nativeLimits))

	C.wgpuDeviceGetLimits(g.ref, &limits)

	return Limits{
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
		MaxInterStageShaderVariables:              uint32(limits.maxInterStageShaderVariables),
		MaxColorAttachments:                       uint32(limits.maxColorAttachments),
		MaxComputeWorkgroupStorageSize:            uint32(limits.maxComputeWorkgroupStorageSize),
		MaxComputeInvocationsPerWorkgroup:         uint32(limits.maxComputeInvocationsPerWorkgroup),
		MaxComputeWorkgroupSizeX:                  uint32(limits.maxComputeWorkgroupSizeX),
		MaxComputeWorkgroupSizeY:                  uint32(limits.maxComputeWorkgroupSizeY),
		MaxComputeWorkgroupSizeZ:                  uint32(limits.maxComputeWorkgroupSizeZ),
		MaxComputeWorkgroupsPerDimension:          uint32(limits.maxComputeWorkgroupsPerDimension),

		MaxImmediateSize:      uint32(nativeLimits.maxImmediateSize),
		MaxNonSamplerBindings: uint32(nativeLimits.maxNonSamplerBindings),
	}
}

func (g *Device) GetQueue() *Queue {
	ref := C.wgpuDeviceGetQueue(g.ref)
	return releaseOnGC(&Queue{device: g.addRef(), ref: ref})
}

func (g *Device) HasFeature(feature FeatureName) bool {
	hasFeature := C.wgpuDeviceHasFeature(g.ref, C.WGPUFeatureName(feature))
	return goBool(hasFeature)
}

func (g *Device) Poll(wait bool, submissionIndex *uint64) (queueEmpty bool) {
	return goBool(C.wgpuDevicePoll(g.ref, cBool(wait), (*C.WGPUSubmissionIndex)(submissionIndex)))
}
