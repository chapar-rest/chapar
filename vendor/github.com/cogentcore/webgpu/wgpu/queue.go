//go:build !js

package wgpu

/*

#include <stdlib.h>
#include "./lib/wgpu.h"

extern void gowebgpu_error_callback_c(WGPUErrorType type, char const * message, void * userdata);
extern void gowebgpu_queue_work_done_callback_c(WGPUQueueWorkDoneStatus status, void * userdata);

static inline void gowebgpu_queue_write_buffer(WGPUQueue queue, WGPUBuffer buffer, uint64_t bufferOffset, void const * data, size_t size, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuQueueWriteBuffer(queue, buffer, bufferOffset, data, size);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
}

static inline void gowebgpu_queue_write_texture(WGPUQueue queue, WGPUImageCopyTexture const * destination, void const * data, size_t dataSize, WGPUTextureDataLayout const * dataLayout, WGPUExtent3D const * writeSize, WGPUDevice device, void * error_userdata) {
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	wgpuQueueWriteTexture(queue, destination, data, dataSize, dataLayout, writeSize);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
}

static inline void gowebgpu_queue_release(WGPUQueue queue, WGPUDevice device) {
	wgpuDeviceRelease(device);
	wgpuQueueRelease(queue);
}

*/
import "C"
import (
	"errors"
	"runtime/cgo"
	"unsafe"
)

type Queue struct {
	deviceRef C.WGPUDevice
	ref       C.WGPUQueue
}

//export gowebgpu_queue_work_done_callback_go
func gowebgpu_queue_work_done_callback_go(status C.WGPUQueueWorkDoneStatus, userdata unsafe.Pointer) {
	handle := *(*cgo.Handle)(userdata)
	defer handle.Delete()

	cb, ok := handle.Value().(QueueWorkDoneCallback)
	if ok {
		cb(QueueWorkDoneStatus(status))
	}
}

func (p *Queue) OnSubmittedWorkDone(callback QueueWorkDoneCallback) {
	handle := cgo.NewHandle(callback)

	C.wgpuQueueOnSubmittedWorkDone(p.ref, C.WGPUQueueOnSubmittedWorkDoneCallback(C.gowebgpu_queue_work_done_callback_c), unsafe.Pointer(&handle))
}

func (p *Queue) Submit(commands ...*CommandBuffer) (submissionIndex SubmissionIndex) {
	commandCount := len(commands)
	if commandCount == 0 {
		r := C.wgpuQueueSubmitForIndex(p.ref, 0, nil)
		return SubmissionIndex(r)
	}

	commandRefs := C.malloc(C.size_t(commandCount) * C.size_t(unsafe.Sizeof(C.WGPUCommandBuffer(nil))))
	defer C.free(commandRefs)

	commandRefsSlice := unsafe.Slice((*C.WGPUCommandBuffer)(commandRefs), commandCount)
	for i, v := range commands {
		commandRefsSlice[i] = v.ref
	}

	r := C.wgpuQueueSubmitForIndex(
		p.ref,
		C.size_t(commandCount),
		(*C.WGPUCommandBuffer)(commandRefs),
	)
	return SubmissionIndex(r)
}

func (p *Queue) WriteBuffer(buffer *Buffer, bufferOffset uint64, data []byte) (err error) {
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Queue).WriteBuffer(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	size := len(data)
	if size == 0 {
		C.gowebgpu_queue_write_buffer(
			p.ref,
			buffer.ref,
			C.uint64_t(bufferOffset),
			nil,
			0,
			p.deviceRef,
			unsafe.Pointer(&errorCallbackHandle),
		)
		return
	}

	C.gowebgpu_queue_write_buffer(
		p.ref,
		buffer.ref,
		C.uint64_t(bufferOffset),
		unsafe.Pointer(&data[0]),
		C.size_t(size),
		p.deviceRef,
		unsafe.Pointer(&errorCallbackHandle),
	)
	return
}

func (p *Queue) WriteTexture(destination *ImageCopyTexture, data []byte, dataLayout *TextureDataLayout, writeSize *Extent3D) (err error) {
	var dst C.WGPUImageCopyTexture
	if destination != nil {
		dst = C.WGPUImageCopyTexture{
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

	var layout C.WGPUTextureDataLayout
	if dataLayout != nil {
		layout = C.WGPUTextureDataLayout{
			offset:       C.uint64_t(dataLayout.Offset),
			bytesPerRow:  C.uint32_t(dataLayout.BytesPerRow),
			rowsPerImage: C.uint32_t(dataLayout.RowsPerImage),
		}
	}

	var writeExtent C.WGPUExtent3D
	if writeSize != nil {
		writeExtent = C.WGPUExtent3D{
			width:              C.uint32_t(writeSize.Width),
			height:             C.uint32_t(writeSize.Height),
			depthOrArrayLayers: C.uint32_t(writeSize.DepthOrArrayLayers),
		}
	}

	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Queue).WriteTexture(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	size := len(data)
	if size == 0 {
		C.gowebgpu_queue_write_texture(
			p.ref,
			&dst,
			nil,
			0,
			&layout,
			&writeExtent,
			p.deviceRef,
			unsafe.Pointer(&errorCallbackHandle),
		)
		return
	}

	C.gowebgpu_queue_write_texture(
		p.ref,
		&dst,
		unsafe.Pointer(&data[0]),
		C.size_t(size),
		&layout,
		&writeExtent,
		p.deviceRef,
		unsafe.Pointer(&errorCallbackHandle),
	)
	return
}

func (p *Queue) Release() {
	C.gowebgpu_queue_release(p.ref, p.deviceRef)
}
