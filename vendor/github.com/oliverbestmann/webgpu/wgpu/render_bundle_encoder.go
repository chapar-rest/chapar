//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

*/
import "C"
import "unsafe"

func (g *RenderBundleEncoder) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	C.wgpuRenderBundleEncoderDraw(
		g.ref,
		C.uint32_t(vertexCount),
		C.uint32_t(instanceCount),
		C.uint32_t(firstVertex),
		C.uint32_t(firstInstance),
	)
}

func (g *RenderBundleEncoder) DrawIndexed(indexCount, instanceCount, firstIndex, baseVertex, firstInstance uint32) {
	C.wgpuRenderBundleEncoderDrawIndexed(
		g.ref,
		C.uint32_t(indexCount),
		C.uint32_t(instanceCount),
		C.uint32_t(firstIndex),
		C.int32_t(baseVertex),
		C.uint32_t(firstInstance),
	)
}

func (g *RenderBundleEncoder) DrawIndexedIndirect(indirectBuffer *Buffer, indirectOffset uint64) {
	C.wgpuRenderBundleEncoderDrawIndexedIndirect(
		g.ref,
		indirectBuffer.ref,
		C.uint64_t(indirectOffset),
	)
}

func (g *RenderBundleEncoder) DrawIndirect(indirectBuffer *Buffer, indirectOffset uint64) {
	C.wgpuRenderBundleEncoderDrawIndirect(
		g.ref,
		indirectBuffer.ref,
		C.uint64_t(indirectOffset),
	)
}

type RenderBundleDescriptor struct {
	Label string
}

func (g *RenderBundleEncoder) Finish(descriptor *RenderBundleDescriptor) *RenderBundle {
	var desc *C.WGPURenderBundleDescriptor

	if descriptor != nil {
		label := C.CString(descriptor.Label)
		defer C.free(unsafe.Pointer(label))

		desc = &C.WGPURenderBundleDescriptor{
			label: C.WGPUStringView{data: label, length: C.WGPU_STRLEN},
		}
	}

	ref := C.wgpuRenderBundleEncoderFinish(g.ref, desc)
	if ref == nil {
		panic("Failed to accquire RenderBundle")
	}
	return releaseOnGC(&RenderBundle{ref: ref})
}

func (g *RenderBundleEncoder) InsertDebugMarker(markerLabel string) {
	markerLabelStr := C.CString(markerLabel)
	defer C.free(unsafe.Pointer(markerLabelStr))

	C.wgpuRenderBundleEncoderInsertDebugMarker(g.ref, C.WGPUStringView{
		data:   markerLabelStr,
		length: C.WGPU_STRLEN,
	})
}

func (g *RenderBundleEncoder) PopDebugGroup() {
	C.wgpuRenderBundleEncoderPopDebugGroup(g.ref)
}

func (g *RenderBundleEncoder) PushDebugGroup(groupLabel string) {
	groupLabelStr := C.CString(groupLabel)
	defer C.free(unsafe.Pointer(groupLabelStr))

	C.wgpuRenderBundleEncoderPushDebugGroup(g.ref, C.WGPUStringView{
		data:   groupLabelStr,
		length: C.WGPU_STRLEN,
	})
}

func (g *RenderBundleEncoder) SetBindGroup(groupIndex uint32, group *BindGroup, dynamicOffsets []uint32) {
	dynamicOffsetCount := len(dynamicOffsets)
	if dynamicOffsetCount == 0 {
		C.wgpuRenderBundleEncoderSetBindGroup(g.ref, C.uint32_t(groupIndex), group.ref, 0, nil)
	} else {
		C.wgpuRenderBundleEncoderSetBindGroup(
			g.ref, C.uint32_t(groupIndex), group.ref,
			C.size_t(dynamicOffsetCount), (*C.uint32_t)(unsafe.Pointer(&dynamicOffsets[0])),
		)
	}
}

func (g *RenderBundleEncoder) SetIndexBuffer(buffer *Buffer, format IndexFormat, offset uint64, size uint64) {
	C.wgpuRenderBundleEncoderSetIndexBuffer(
		g.ref,
		buffer.ref,
		C.WGPUIndexFormat(format),
		C.uint64_t(offset),
		C.uint64_t(size),
	)
}

func (g *RenderBundleEncoder) SetPipeline(pipeline *RenderPipeline) {
	C.wgpuRenderBundleEncoderSetPipeline(g.ref, pipeline.ref)
}

func (g *RenderBundleEncoder) SetVertexBuffer(slot uint32, buffer *Buffer, offset uint64, size uint64) {
	C.wgpuRenderBundleEncoderSetVertexBuffer(
		g.ref,
		C.uint32_t(slot),
		buffer.ref,
		C.uint64_t(offset),
		C.uint64_t(size),
	)
}
