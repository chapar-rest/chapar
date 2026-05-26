//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

extern void gowebgpu_error_callback_c(enum WGPUPopErrorScopeStatus status, WGPUErrorType type, WGPUStringView message, void * userdata, void * userdata2);

static inline void gowebgpu_compute_pass_encoder_end(WGPUComputePassEncoder computePassEncoder, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuComputePassEncoderEnd(computePassEncoder);

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

func (p *ComputePassEncoder) BeginPipelineStatisticsQuery(querySet *QuerySet, queryIndex uint32) {
	C.wgpuComputePassEncoderBeginPipelineStatisticsQuery(p.ref, querySet.ref, C.uint32_t(queryIndex))
}

func (p *ComputePassEncoder) DispatchWorkgroups(workgroupCountX, workgroupCountY, workgroupCountZ uint32) {
	C.wgpuComputePassEncoderDispatchWorkgroups(p.ref, C.uint32_t(workgroupCountX), C.uint32_t(workgroupCountY), C.uint32_t(workgroupCountZ))
}

func (p *ComputePassEncoder) DispatchWorkgroupsIndirect(indirectBuffer *Buffer, indirectOffset uint64) {
	C.wgpuComputePassEncoderDispatchWorkgroupsIndirect(p.ref, indirectBuffer.ref, C.uint64_t(indirectOffset))
}

func (p *ComputePassEncoder) TryEnd() error {
	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_compute_pass_encoder_end(p.ref, p.device, errh.ToPointer())
	return errh.err
}

func (p *ComputePassEncoder) EndPipelineStatisticsQuery() {
	C.wgpuComputePassEncoderEndPipelineStatisticsQuery(p.ref)
}

func (p *ComputePassEncoder) InsertDebugMarker(markerLabel string) {
	markerLabelStr := C.CString(markerLabel)
	defer C.free(unsafe.Pointer(markerLabelStr))

	C.wgpuComputePassEncoderInsertDebugMarker(p.ref, C.WGPUStringView{
		data:   markerLabelStr,
		length: C.WGPU_STRLEN,
	})
}

func (p *ComputePassEncoder) PopDebugGroup() {
	C.wgpuComputePassEncoderPopDebugGroup(p.ref)
}

func (p *ComputePassEncoder) PushDebugGroup(groupLabel string) {
	groupLabelStr := C.CString(groupLabel)
	defer C.free(unsafe.Pointer(groupLabelStr))

	C.wgpuComputePassEncoderPushDebugGroup(p.ref, C.WGPUStringView{
		data:   groupLabelStr,
		length: C.WGPU_STRLEN,
	})
}

func (p *ComputePassEncoder) SetBindGroup(groupIndex uint32, group *BindGroup, dynamicOffsets []uint32) {
	dynamicOffsetCount := len(dynamicOffsets)
	if dynamicOffsetCount == 0 {
		C.wgpuComputePassEncoderSetBindGroup(p.ref, C.uint32_t(groupIndex), group.ref, 0, nil)
	} else {
		C.wgpuComputePassEncoderSetBindGroup(
			p.ref, C.uint32_t(groupIndex), group.ref,
			C.size_t(dynamicOffsetCount), (*C.uint32_t)(unsafe.Pointer(&dynamicOffsets[0])),
		)
	}
}

func (p *ComputePassEncoder) SetPipeline(pipeline *ComputePipeline) {
	C.wgpuComputePassEncoderSetPipeline(p.ref, pipeline.ref)
}
