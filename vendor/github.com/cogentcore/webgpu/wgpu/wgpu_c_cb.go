//go:build !js

package wgpu

/*

#include "./lib/wgpu.h"

void gowebgpu_buffer_map_callback_c(WGPUBufferMapAsyncStatus status, void *userdata) {
  extern void gowebgpu_buffer_map_callback_go(WGPUBufferMapAsyncStatus status, void *userdata);
  gowebgpu_buffer_map_callback_go(status, userdata);
}

void gowebgpu_request_adapter_callback_c(WGPURequestAdapterStatus status, WGPUAdapter adapter, char const *message, void *userdata) {
  extern void gowebgpu_request_adapter_callback_go(WGPURequestAdapterStatus status, WGPUAdapter adapter, char const *message, void *userdata);
  gowebgpu_request_adapter_callback_go(status, adapter, message, userdata);
}

void gowebgpu_request_device_callback_c(WGPURequestDeviceStatus status, WGPUDevice device, char const *message, void *userdata) {
  extern void gowebgpu_request_device_callback_go(WGPURequestDeviceStatus status, WGPUDevice device, char const *message, void *userdata);
  gowebgpu_request_device_callback_go(status, device, message, userdata);
}

void gowebgpu_device_lost_callback_c(WGPUDeviceLostReason reason, char const * message, void * userdata) {
  extern void gowebgpu_device_lost_callback_go(WGPUDeviceLostReason reason, char const * message, void * userdata);
  gowebgpu_device_lost_callback_go(reason, message, userdata);
}

void gowebgpu_error_callback_c(WGPUErrorType type, char const * message, void * userdata) {
  if (type == WGPUErrorType_NoError) {
    return;
  }

  extern void gowebgpu_error_callback_go(WGPUErrorType type, char const * message, void * userdata);
  gowebgpu_error_callback_go(type, message, userdata);
}

void gowebgpu_queue_work_done_callback_c(WGPUQueueWorkDoneStatus status, void * userdata) {
  extern void gowebgpu_queue_work_done_callback_go(WGPUQueueWorkDoneStatus status, void * userdata);
  gowebgpu_queue_work_done_callback_go(status, userdata);
}

*/
import "C"
