//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

extern void gowebgpu_error_callback_c(enum WGPUPopErrorScopeStatus status, WGPUErrorType type, WGPUStringView message, void * userdata, void * userdata2);

extern void gowebgpu_buffer_map_callback_c(WGPUMapAsyncStatus status, WGPUStringView message, void *userdata, void *userdata2);

static inline void gowebgpu_buffer_map_async(WGPUBuffer buffer, WGPUMapMode mode, size_t offset, size_t size, WGPUBufferMapCallbackInfo callback, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuBufferMapAsync(buffer, mode, offset, size, callback);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);
}

static inline void gowebgpu_buffer_unmap(WGPUBuffer buffer, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuBufferUnmap(buffer);

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

func (p *Buffer) Destroy() {
	C.wgpuBufferDestroy(p.ref)
}

func (p *Buffer) GetMappedRange(offset, size uint) []byte {
	buf := C.wgpuBufferGetMappedRange(p.ref, C.size_t(offset), C.size_t(size))
	return unsafe.Slice((*byte)(buf), size)
}

func (p *Buffer) GetSize() uint64 {
	return uint64(C.wgpuBufferGetSize(p.ref))
}

func (p *Buffer) GetUsage() BufferUsage {
	return BufferUsage(C.wgpuBufferGetUsage(p.ref))
}

//export gowebgpu_buffer_map_callback_go
func gowebgpu_buffer_map_callback_go(status C.WGPUMapAsyncStatus, userdata unsafe.Pointer) {
	handle := lookupHandle(userdata)
	defer handle.Delete()

	cb, ok := handle.Value().(BufferMapCallback)
	if ok {
		cb(MapAsyncStatus(status))
	}
}

func (p *Buffer) TryMapAsync(mode MapMode, offset uint64, size uint64, callback BufferMapCallback) error {
	callbackHandle := newHandle(callback)

	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_buffer_map_async(
		p.ref,
		C.WGPUMapMode(mode),
		C.size_t(offset),
		C.size_t(size),
		C.WGPUBufferMapCallbackInfo{
			callback:  C.WGPUBufferMapCallback(C.gowebgpu_buffer_map_callback_c),
			userdata1: callbackHandle.ToPointer(),
		},
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}

func (p *Buffer) TryUnmap() error {
	errh := acquireErrorCallback()
	defer errh.Done()

	C.gowebgpu_buffer_unmap(
		p.ref,
		p.device,
		errh.ToPointer(),
	)

	return errh.err
}
