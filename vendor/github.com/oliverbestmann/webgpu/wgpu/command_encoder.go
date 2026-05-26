//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

extern void gowebgpu_error_callback_c(enum WGPUPopErrorScopeStatus status, WGPUErrorType type, WGPUStringView message, void * userdata, void * userdata2);

static inline void gowebgpu_command_encoder_clear_buffer(WGPUCommandEncoder commandEncoder, WGPUBuffer buffer, uint64_t offset, uint64_t size, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderClearBuffer(commandEncoder, buffer, offset, size);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_command_encoder_copy_buffer_to_buffer(WGPUCommandEncoder commandEncoder, WGPUBuffer source, uint64_t sourceOffset, WGPUBuffer destination, uint64_t destinationOffset, uint64_t size, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderCopyBufferToBuffer(commandEncoder, source, sourceOffset, destination, destinationOffset, size);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_command_encoder_copy_buffer_to_texture(WGPUCommandEncoder commandEncoder, WGPUTexelCopyBufferInfo const * source, WGPUTexelCopyTextureInfo const * destination, WGPUExtent3D const * copySize, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderCopyBufferToTexture(commandEncoder, source, destination, copySize);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_command_encoder_copy_texture_to_buffer(WGPUCommandEncoder commandEncoder, WGPUTexelCopyTextureInfo const * source, WGPUTexelCopyBufferInfo const * destination, WGPUExtent3D const * copySize, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderCopyTextureToBuffer(commandEncoder, source, destination, copySize);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_command_encoder_copy_texture_to_texture(WGPUCommandEncoder commandEncoder, WGPUTexelCopyTextureInfo const * source, WGPUTexelCopyTextureInfo const * destination, WGPUExtent3D const * copySize, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderCopyTextureToTexture(commandEncoder, source, destination, copySize);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline WGPUCommandBuffer gowebgpu_command_encoder_finish(WGPUCommandEncoder commandEncoder, WGPUCommandBufferDescriptor const * descriptor, WGPUDevice device, void * error_userdata) {
	WGPUCommandBuffer ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuCommandEncoderFinish(commandEncoder, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

static inline void gowebgpu_command_encoder_insert_debug_marker(WGPUCommandEncoder commandEncoder, char const * markerLabel, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderInsertDebugMarker(commandEncoder, (WGPUStringView) {markerLabel, WGPU_STRLEN});

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_command_encoder_pop_debug_group(WGPUCommandEncoder commandEncoder, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderPopDebugGroup(commandEncoder);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_command_encoder_push_debug_group(WGPUCommandEncoder commandEncoder, char const * groupLabel, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderPushDebugGroup(commandEncoder, (WGPUStringView) {groupLabel, WGPU_STRLEN});

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_command_encoder_resolve_query_set(WGPUCommandEncoder commandEncoder, WGPUQuerySet querySet, uint32_t firstQuery, uint32_t queryCount, WGPUBuffer destination, uint64_t destinationOffset, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderResolveQuerySet(commandEncoder, querySet, firstQuery, queryCount, destination, destinationOffset);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_command_encoder_write_timestamp(WGPUCommandEncoder commandEncoder, WGPUQuerySet querySet, uint32_t queryIndex, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuCommandEncoderWriteTimestamp(commandEncoder, querySet, queryIndex);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

*/
import "C"
import (
	"errors"
	"unsafe"
)

type ComputePassDescriptor struct {
	Label string

	// unused in wgpu
	// TimestampWrites []ComputePassTimestampWrite
}

func (p *CommandEncoder) BeginComputePass(descriptor *ComputePassDescriptor) *ComputePassEncoder {
	var desc *C.WGPUComputePassDescriptor

	if descriptor != nil && descriptor.Label != "" {
		label := C.CString(descriptor.Label)
		defer C.free(unsafe.Pointer(label))

		desc = allocWGPUComputePassDescriptor.GetZeroed()
		defer allocWGPUComputePassDescriptor.Put(desc)

		desc.label = C.WGPUStringView{data: label, length: C.WGPU_STRLEN}
	}

	ref := C.wgpuCommandEncoderBeginComputePass(p.ref, desc)
	if ref == nil {
		err := errors.New("failed to acquire ComputePassEncoder")
		panic(wrap(err, ""))
	}

	C.wgpuDeviceAddRef(p.device)
	return releaseOnGC(&ComputePassEncoder{device: p.device, ref: ref})
}

func (p *CommandEncoder) BeginRenderPass(descriptor *RenderPassDescriptor) *RenderPassEncoder {
	var desc *C.WGPURenderPassDescriptor = allocWGPURenderPassDescriptor.GetZeroed()
	defer allocWGPURenderPassDescriptor.Put(desc)

	if descriptor != nil {
		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		colorAttachmentCount := len(descriptor.ColorAttachments)
		if colorAttachmentCount > 0 {
			colorAttachments := C.calloc(C.size_t(unsafe.Sizeof(C.WGPURenderPassColorAttachment{})), C.size_t(colorAttachmentCount))
			defer C.free(colorAttachments)

			colorAttachmentsSlice := unsafe.Slice((*C.WGPURenderPassColorAttachment)(colorAttachments), colorAttachmentCount)

			for i, v := range descriptor.ColorAttachments {
				colorAttachment := C.WGPURenderPassColorAttachment{
					loadOp:     C.WGPULoadOp(v.LoadOp),
					storeOp:    C.WGPUStoreOp(v.StoreOp),
					depthSlice: C.WGPU_DEPTH_SLICE_UNDEFINED,
					clearValue: C.WGPUColor{
						r: C.double(v.ClearValue.R),
						g: C.double(v.ClearValue.G),
						b: C.double(v.ClearValue.B),
						a: C.double(v.ClearValue.A),
					},
				}
				if v.View != nil {
					colorAttachment.view = v.View.ref
				}
				if v.ResolveTarget != nil {
					colorAttachment.resolveTarget = v.ResolveTarget.ref
				}

				colorAttachmentsSlice[i] = colorAttachment
			}

			desc.colorAttachmentCount = C.size_t(colorAttachmentCount)
			desc.colorAttachments = (*C.WGPURenderPassColorAttachment)(colorAttachments)
		}

		if descriptor.DepthStencilAttachment != nil {
			depthStencilAttachment := (*C.WGPURenderPassDepthStencilAttachment)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPURenderPassDepthStencilAttachment{}))))
			defer C.free(unsafe.Pointer(depthStencilAttachment))

			if descriptor.DepthStencilAttachment.View != nil {
				depthStencilAttachment.view = descriptor.DepthStencilAttachment.View.ref
			}
			depthStencilAttachment.depthLoadOp = C.WGPULoadOp(descriptor.DepthStencilAttachment.DepthLoadOp)
			depthStencilAttachment.depthStoreOp = C.WGPUStoreOp(descriptor.DepthStencilAttachment.DepthStoreOp)
			depthStencilAttachment.depthClearValue = C.float(descriptor.DepthStencilAttachment.DepthClearValue)
			depthStencilAttachment.depthReadOnly = cBool(descriptor.DepthStencilAttachment.DepthReadOnly)
			depthStencilAttachment.stencilLoadOp = C.WGPULoadOp(descriptor.DepthStencilAttachment.StencilLoadOp)
			depthStencilAttachment.stencilStoreOp = C.WGPUStoreOp(descriptor.DepthStencilAttachment.StencilStoreOp)
			depthStencilAttachment.stencilClearValue = C.uint32_t(descriptor.DepthStencilAttachment.StencilClearValue)
			depthStencilAttachment.stencilReadOnly = cBool(descriptor.DepthStencilAttachment.DepthReadOnly)

			desc.depthStencilAttachment = depthStencilAttachment
		}
	}

	ref := C.wgpuCommandEncoderBeginRenderPass(p.ref, desc)
	if ref == nil {
		err := errors.New("failed to acquire RenderPassEncoder")
		panic(wrap(err, ""))
	}

	C.wgpuDeviceAddRef(p.device)
	return releaseOnGC(&RenderPassEncoder{device: p.device, ref: ref})
}

func (p *CommandEncoder) TryClearBuffer(buffer *Buffer, offset uint64, size uint64) error {
	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_clear_buffer(
		p.ref,
		buffer.ref,
		C.uint64_t(offset),
		C.uint64_t(size),
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *CommandEncoder) TryCopyBufferToBuffer(source *Buffer, sourceOffset uint64, destination *Buffer, destinationOffset uint64, size uint64) error {
	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_copy_buffer_to_buffer(
		p.ref,
		source.ref,
		C.uint64_t(sourceOffset),
		destination.ref,
		C.uint64_t(destinationOffset),
		C.uint64_t(size),
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *CommandEncoder) TryCopyBufferToTexture(source *TexelCopyBufferInfo, destination *TexelCopyTextureInfo, copySize *Extent3D) error {
	var src C.WGPUTexelCopyBufferInfo
	if source != nil {
		if source.Buffer != nil {
			src.buffer = source.Buffer.ref
		}
		src.layout = C.WGPUTexelCopyBufferLayout{
			offset:       C.uint64_t(source.Layout.Offset),
			bytesPerRow:  C.uint32_t(source.Layout.BytesPerRow),
			rowsPerImage: C.uint32_t(source.Layout.RowsPerImage),
		}
	}

	var dst C.WGPUTexelCopyTextureInfo
	if destination != nil {
		dst = C.WGPUTexelCopyTextureInfo{
			mipLevel: C.uint32_t(destination.MipLevel),
			origin: C.WGPUOrigin3D{
				x: C.uint32_t(destination.Origin.X),
				y: C.uint32_t(destination.Origin.Y),
				z: C.uint32_t(destination.Origin.Z),
			},
			aspect: C.WGPUTextureAspect(destination.Aspect),
		}
		if destination.Texture != nil {
			dst.texture = destination.Texture.ref
		}
	}

	var cpySize C.WGPUExtent3D
	if copySize != nil {
		cpySize = C.WGPUExtent3D{
			width:              C.uint32_t(copySize.Width),
			height:             C.uint32_t(copySize.Height),
			depthOrArrayLayers: C.uint32_t(copySize.DepthOrArrayLayers),
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_copy_buffer_to_texture(
		p.ref,
		&src,
		&dst,
		&cpySize,
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *CommandEncoder) TryCopyTextureToBuffer(source *TexelCopyTextureInfo, destination *TexelCopyBufferInfo, copySize *Extent3D) error {
	var src C.WGPUTexelCopyTextureInfo
	if source != nil {
		src = C.WGPUTexelCopyTextureInfo{
			mipLevel: C.uint32_t(source.MipLevel),
			origin: C.WGPUOrigin3D{
				x: C.uint32_t(source.Origin.X),
				y: C.uint32_t(source.Origin.Y),
				z: C.uint32_t(source.Origin.Z),
			},
			aspect: C.WGPUTextureAspect(source.Aspect),
		}
		if source.Texture != nil {
			src.texture = source.Texture.ref
		}
	}

	var dst C.WGPUTexelCopyBufferInfo
	if destination != nil {
		if destination.Buffer != nil {
			dst.buffer = destination.Buffer.ref
		}
		dst.layout = C.WGPUTexelCopyBufferLayout{
			offset:       C.uint64_t(destination.Layout.Offset),
			bytesPerRow:  C.uint32_t(destination.Layout.BytesPerRow),
			rowsPerImage: C.uint32_t(destination.Layout.RowsPerImage),
		}
	}

	var cpySize C.WGPUExtent3D
	if copySize != nil {
		cpySize = C.WGPUExtent3D{
			width:              C.uint32_t(copySize.Width),
			height:             C.uint32_t(copySize.Height),
			depthOrArrayLayers: C.uint32_t(copySize.DepthOrArrayLayers),
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_copy_texture_to_buffer(
		p.ref,
		&src,
		&dst,
		&cpySize,
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *CommandEncoder) TryCopyTextureToTexture(source *TexelCopyTextureInfo, destination *TexelCopyTextureInfo, copySize *Extent3D) error {
	var src C.WGPUTexelCopyTextureInfo
	if source != nil {
		src = C.WGPUTexelCopyTextureInfo{
			mipLevel: C.uint32_t(source.MipLevel),
			origin: C.WGPUOrigin3D{
				x: C.uint32_t(source.Origin.X),
				y: C.uint32_t(source.Origin.Y),
				z: C.uint32_t(source.Origin.Z),
			},
			aspect: C.WGPUTextureAspect(source.Aspect),
		}
		if source.Texture != nil {
			src.texture = source.Texture.ref
		}
	}

	var dst C.WGPUTexelCopyTextureInfo
	if destination != nil {
		dst = C.WGPUTexelCopyTextureInfo{
			mipLevel: C.uint32_t(destination.MipLevel),
			origin: C.WGPUOrigin3D{
				x: C.uint32_t(destination.Origin.X),
				y: C.uint32_t(destination.Origin.Y),
				z: C.uint32_t(destination.Origin.Z),
			},
			aspect: C.WGPUTextureAspect(destination.Aspect),
		}
		if destination.Texture != nil {
			dst.texture = destination.Texture.ref
		}
	}

	var cpySize C.WGPUExtent3D
	if copySize != nil {
		cpySize = C.WGPUExtent3D{
			width:              C.uint32_t(copySize.Width),
			height:             C.uint32_t(copySize.Height),
			depthOrArrayLayers: C.uint32_t(copySize.DepthOrArrayLayers),
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_copy_texture_to_texture(
		p.ref,
		&src,
		&dst,
		&cpySize,
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *CommandEncoder) TryFinish(descriptor *CommandBufferDescriptor) (*CommandBuffer, error) {
	var desc *C.WGPUCommandBufferDescriptor

	if descriptor != nil && descriptor.Label != "" {
		label := C.CString(descriptor.Label)
		defer C.free(unsafe.Pointer(label))

		desc = &C.WGPUCommandBufferDescriptor{
			label: C.WGPUStringView{data: label, length: C.WGPU_STRLEN},
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_command_encoder_finish(
		p.ref,
		desc,
		p.device,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuCommandBufferRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&CommandBuffer{ref: ref}), nil
}

func (p *CommandEncoder) TryInsertDebugMarker(markerLabel string) error {
	markerLabelStr := C.CString(markerLabel)
	defer C.free(unsafe.Pointer(markerLabelStr))

	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_insert_debug_marker(
		p.ref,
		markerLabelStr,
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *CommandEncoder) TryPopDebugGroup() error {
	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_pop_debug_group(
		p.ref,
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *CommandEncoder) TryPushDebugGroup(groupLabel string) error {
	groupLabelStr := C.CString(groupLabel)
	defer C.free(unsafe.Pointer(groupLabelStr))

	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_push_debug_group(
		p.ref,
		groupLabelStr,
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *CommandEncoder) TryResolveQuerySet(querySet *QuerySet, firstQuery uint32, queryCount uint32, destination *Buffer, destinationOffset uint64) error {
	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_command_encoder_resolve_query_set(
		p.ref,
		querySet.ref,
		C.uint32_t(firstQuery),
		C.uint32_t(queryCount),
		destination.ref,
		C.uint64_t(destinationOffset),
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}
