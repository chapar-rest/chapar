//go:build !js

package wgpu

/*

#include <stdlib.h>
#include "./lib/wgpu.h"

extern void gowebgpu_error_callback_c(WGPUErrorType type, char const * message, void * userdata);
extern void gowebgpu_buffer_map_callback_c(WGPUBufferMapAsyncStatus status, void *userdata);

static inline void gowebgpu_buffer_map_async(WGPUBuffer buffer, WGPUMapModeFlags mode, size_t offset, size_t size, WGPUBufferMapAsyncCallback callback, void * userdata, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuBufferMapAsync(buffer, mode, offset, size, callback, userdata);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
}

static inline void gowebgpu_buffer_unmap(WGPUBuffer buffer, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuBufferUnmap(buffer);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
}

static inline void gowebgpu_buffer_release(WGPUBuffer buffer, WGPUDevice device) {
	wgpuDeviceRelease(device);
	wgpuBufferRelease(buffer);
}

*/
import "C"
import (
	"errors"
	"runtime/cgo"
	"unsafe"
)

type Buffer struct {
	deviceRef C.WGPUDevice
	ref       C.WGPUBuffer
}

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
func gowebgpu_buffer_map_callback_go(status C.WGPUBufferMapAsyncStatus, userdata unsafe.Pointer) {
	handle := *(*cgo.Handle)(userdata)
	defer handle.Delete()

	cb, ok := handle.Value().(BufferMapCallback)
	if ok {
		cb(BufferMapAsyncStatus(status))
	}
}

func (p *Buffer) MapAsync(mode MapMode, offset uint64, size uint64, callback BufferMapCallback) (err error) {
	callbackHandle := cgo.NewHandle(callback)

	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Buffer).MapAsync(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	C.gowebgpu_buffer_map_async(
		p.ref,
		C.WGPUMapModeFlags(mode),
		C.size_t(offset),
		C.size_t(size),
		(C.WGPUBufferMapAsyncCallback)(C.gowebgpu_buffer_map_callback_c),
		unsafe.Pointer(&callbackHandle),
		p.deviceRef,
		unsafe.Pointer(&errorCallbackHandle),
	)
	return
}

func (p *Buffer) Unmap() (err error) {
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Buffer).Unmap(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	C.gowebgpu_buffer_unmap(
		p.ref,
		p.deviceRef,
		unsafe.Pointer(&errorCallbackHandle),
	)
	return
}

func (p *Buffer) Release() {
	C.gowebgpu_buffer_release(p.ref, p.deviceRef)
}
