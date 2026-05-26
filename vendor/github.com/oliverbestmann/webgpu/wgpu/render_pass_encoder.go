//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

extern void gowebgpu_error_callback_c(enum WGPUPopErrorScopeStatus status, WGPUErrorType type, WGPUStringView message, void * userdata, void * userdata2);

static inline void gowebgpu_render_pass_encoder_end(WGPURenderPassEncoder renderPassEncoder, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuRenderPassEncoderEnd(renderPassEncoder);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

*/
import "C"
import (
	"unsafe"
)

func (p *RenderPassEncoder) BeginOcclusionQuery(queryIndex uint32) {
	C.wgpuRenderPassEncoderBeginOcclusionQuery(p.ref, C.uint32_t(queryIndex))
}

func (p *RenderPassEncoder) BeginPipelineStatisticsQuery(querySet *QuerySet, queryIndex uint32) {
	C.wgpuRenderPassEncoderBeginPipelineStatisticsQuery(p.ref, querySet.ref, C.uint32_t(queryIndex))
}

func (p *RenderPassEncoder) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	C.wgpuRenderPassEncoderDraw(p.ref,
		C.uint32_t(vertexCount),
		C.uint32_t(instanceCount),
		C.uint32_t(firstVertex),
		C.uint32_t(firstInstance),
	)
}

func (p *RenderPassEncoder) DrawIndexed(indexCount uint32, instanceCount uint32, firstIndex uint32, baseVertex int32, firstInstance uint32) {
	C.wgpuRenderPassEncoderDrawIndexed(p.ref,
		C.uint32_t(indexCount),
		C.uint32_t(instanceCount),
		C.uint32_t(firstIndex),
		C.int32_t(baseVertex),
		C.uint32_t(firstInstance),
	)
}

func (p *RenderPassEncoder) DrawIndexedIndirect(indirectBuffer *Buffer, indirectOffset uint64) {
	C.wgpuRenderPassEncoderDrawIndexedIndirect(p.ref, indirectBuffer.ref, C.uint64_t(indirectOffset))
}

func (p *RenderPassEncoder) DrawIndirect(indirectBuffer *Buffer, indirectOffset uint64) {
	C.wgpuRenderPassEncoderDrawIndirect(p.ref, indirectBuffer.ref, C.uint64_t(indirectOffset))
}

// TryEnd ends the current render pass encoder.
// This will also release the instance, so it must not be used afterwards
func (p *RenderPassEncoder) TryEnd() error {
	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_render_pass_encoder_end(
		p.ref,
		p.device,
		errh.ToPointer(),
	)

	// also release the render pass at this point
	p.Release()

	return errh.err
}

func (p *RenderPassEncoder) EndOcclusionQuery() {
	C.wgpuRenderPassEncoderEndOcclusionQuery(p.ref)
}

func (p *RenderPassEncoder) EndPipelineStatisticsQuery() {
	C.wgpuRenderPassEncoderEndPipelineStatisticsQuery(p.ref)
}

func (p *RenderPassEncoder) ExecuteBundles(bundles ...*RenderBundle) {
	bundlesCount := len(bundles)
	if bundlesCount == 0 {
		C.wgpuRenderPassEncoderExecuteBundles(p.ref, 0, nil)
		return
	}

	bundlesPtr := C.calloc(C.size_t(bundlesCount), C.size_t(unsafe.Sizeof(C.WGPURenderBundle(nil))))
	defer C.free(bundlesPtr)

	bundlesSlice := unsafe.Slice((*C.WGPURenderBundle)(bundlesPtr), bundlesCount)
	for i, v := range bundles {
		bundlesSlice[i] = v.ref
	}

	C.wgpuRenderPassEncoderExecuteBundles(p.ref, C.size_t(bundlesCount), (*C.WGPURenderBundle)(bundlesPtr))
}

func (p *RenderPassEncoder) InsertDebugMarker(markerLabel string) {
	markerLabelStr := C.CString(markerLabel)
	defer C.free(unsafe.Pointer(markerLabelStr))

	C.wgpuRenderPassEncoderInsertDebugMarker(p.ref, C.WGPUStringView{
		data:   markerLabelStr,
		length: C.WGPU_STRLEN,
	})
}

func (p *RenderPassEncoder) PopDebugGroup() {
	C.wgpuRenderPassEncoderPopDebugGroup(p.ref)
}

func (p *RenderPassEncoder) PushDebugGroup(groupLabel string) {
	groupLabelStr := C.CString(groupLabel)
	defer C.free(unsafe.Pointer(groupLabelStr))

	C.wgpuRenderPassEncoderPushDebugGroup(p.ref, C.WGPUStringView{
		data:   groupLabelStr,
		length: C.WGPU_STRLEN,
	})
}

func (p *RenderPassEncoder) SetBindGroup(groupIndex uint32, group *BindGroup, dynamicOffsets []uint32) {
	dynamicOffsetCount := len(dynamicOffsets)
	if dynamicOffsetCount == 0 {
		C.wgpuRenderPassEncoderSetBindGroup(
			p.ref,
			C.uint32_t(groupIndex),
			group.ref,
			0,
			nil,
		)
	} else {
		C.wgpuRenderPassEncoderSetBindGroup(
			p.ref,
			C.uint32_t(groupIndex),
			group.ref,
			C.size_t(dynamicOffsetCount),
			(*C.uint32_t)(unsafe.Pointer(&dynamicOffsets[0])),
		)
	}
}

func (p *RenderPassEncoder) SetBlendConstant(color *Color) {
	c := C.WGPUColor{
		r: C.double(color.R),
		g: C.double(color.G),
		b: C.double(color.B),
		a: C.double(color.A),
	}
	C.wgpuRenderPassEncoderSetBlendConstant(p.ref, &c)
}

func (p *RenderPassEncoder) SetIndexBuffer(buffer *Buffer, format IndexFormat, offset uint64, size uint64) {
	C.wgpuRenderPassEncoderSetIndexBuffer(
		p.ref,
		buffer.ref,
		C.WGPUIndexFormat(format),
		C.uint64_t(offset),
		C.uint64_t(size),
	)
}

func (p *RenderPassEncoder) SetPipeline(pipeline *RenderPipeline) {
	C.wgpuRenderPassEncoderSetPipeline(p.ref, pipeline.ref)
}

func (p *RenderPassEncoder) SetScissorRect(x, y, width, height uint32) {
	C.wgpuRenderPassEncoderSetScissorRect(
		p.ref,
		C.uint32_t(x),
		C.uint32_t(y),
		C.uint32_t(width),
		C.uint32_t(height),
	)
}

func (p *RenderPassEncoder) SetStencilReference(reference uint32) {
	C.wgpuRenderPassEncoderSetStencilReference(p.ref, C.uint32_t(reference))
}

func (p *RenderPassEncoder) SetVertexBuffer(slot uint32, buffer *Buffer, offset uint64, size uint64) {
	C.wgpuRenderPassEncoderSetVertexBuffer(
		p.ref,
		C.uint32_t(slot),
		buffer.ref,
		C.uint64_t(offset),
		C.uint64_t(size),
	)
}

func (p *RenderPassEncoder) SetViewport(x, y, width, height, minDepth, maxDepth float32) {
	C.wgpuRenderPassEncoderSetViewport(
		p.ref,
		C.float(x),
		C.float(y),
		C.float(width),
		C.float(height),
		C.float(minDepth),
		C.float(maxDepth),
	)
}

func (p *RenderPassEncoder) SetImmediates(offset uint32, data []byte) {
	size := len(data)
	if size == 0 {
		C.wgpuRenderPassEncoderSetImmediates(
			p.ref,
			C.uint32_t(offset),
			0,
			nil,
		)
		return
	}

	C.wgpuRenderPassEncoderSetImmediates(
		p.ref,
		C.uint32_t(offset),
		C.uint32_t(size),
		unsafe.Pointer(&data[0]),
	)
}

func (p *RenderPassEncoder) MultiDrawIndirect(encoder *RenderPassEncoder, buffer Buffer, offset uint64, count uint32) {
	C.wgpuRenderPassEncoderMultiDrawIndirect(
		encoder.ref,
		buffer.ref,
		C.uint64_t(offset),
		C.uint32_t(count),
	)
}

func (p *RenderPassEncoder) MultiDrawIndexedIndirect(encoder *RenderPassEncoder, buffer Buffer, offset uint64, count uint32) {
	C.wgpuRenderPassEncoderMultiDrawIndexedIndirect(
		encoder.ref,
		buffer.ref,
		C.uint64_t(offset),
		C.uint32_t(count),
	)
}

func (p *RenderPassEncoder) MultiDrawIndirectCount(encoder *RenderPassEncoder, buffer Buffer, offset uint64, countBuffer Buffer, countBufferOffset uint64, maxCount uint32) {
	C.wgpuRenderPassEncoderMultiDrawIndirectCount(
		encoder.ref,
		buffer.ref,
		C.uint64_t(offset),
		countBuffer.ref,
		C.uint64_t(countBufferOffset),
		C.uint32_t(maxCount),
	)
}

func (p *RenderPassEncoder) MultiDrawIndexedIndirectCount(encoder *RenderPassEncoder, buffer Buffer, offset uint64, countBuffer Buffer, countBufferOffset uint64, maxCount uint32) {
	C.wgpuRenderPassEncoderMultiDrawIndexedIndirectCount(
		encoder.ref,
		buffer.ref,
		C.uint64_t(offset),
		countBuffer.ref,
		C.uint64_t(countBufferOffset),
		C.uint32_t(maxCount),
	)
}
