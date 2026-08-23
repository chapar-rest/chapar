//go:build !js

package wgpu

/*

#include <stdlib.h>
#include "./lib/wgpu.h"

extern void gowebgpu_request_device_callback_c(WGPURequestDeviceStatus status, WGPUDevice device, char const *message, void *userdata);
extern void gowebgpu_device_lost_callback_c(WGPUDeviceLostReason reason, char const * message, void * userdata);

*/
import "C"
import (
	"errors"
	"runtime/cgo"
	"unsafe"
)

type Adapter struct {
	ref C.WGPUAdapter
}

func (p *Adapter) EnumerateFeatures() []FeatureName {
	size := C.wgpuAdapterEnumerateFeatures(p.ref, nil)
	if size == 0 {
		return nil
	}

	features := make([]FeatureName, size)
	C.wgpuAdapterEnumerateFeatures(p.ref, (*C.WGPUFeatureName)(unsafe.Pointer(&features[0])))
	return features
}

func (p *Adapter) GetLimits() SupportedLimits {
	var supportedLimits C.WGPUSupportedLimits

	extras := (*C.WGPUSupportedLimitsExtras)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUSupportedLimitsExtras{}))))
	defer C.free(unsafe.Pointer(extras))
	supportedLimits.nextInChain = (*C.WGPUChainedStructOut)(unsafe.Pointer(extras))

	C.wgpuAdapterGetLimits(p.ref, &supportedLimits)

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
			MaxColorAttachmentBytesPerSample:          uint32(limits.maxColorAttachmentBytesPerSample),
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

func (p *Adapter) GetInfo() AdapterInfo {
	var info C.WGPUAdapterInfo

	C.wgpuAdapterGetInfo(p.ref, &info)

	return AdapterInfo{
		VendorId:          uint32(info.vendorID),
		VendorName:        C.GoString(info.vendor),
		Architecture:      C.GoString(info.architecture),
		DeviceId:          uint32(info.deviceID),
		Name:              C.GoString(info.device),
		DriverDescription: C.GoString(info.description),
		AdapterType:       AdapterType(info.adapterType),
		BackendType:       BackendType(info.backendType),
	}
}

func (p *Adapter) HasFeature(feature FeatureName) bool {
	hasFeature := C.wgpuAdapterHasFeature(p.ref, C.WGPUFeatureName(feature))
	return goBool(hasFeature)
}

type requestDeviceCb func(status RequestDeviceStatus, device *Device, message string)

//export gowebgpu_request_device_callback_go
func gowebgpu_request_device_callback_go(status C.WGPURequestDeviceStatus, device C.WGPUDevice, message *C.char, userdata unsafe.Pointer) {
	handle := *(*cgo.Handle)(userdata)
	defer handle.Delete()

	cb, ok := handle.Value().(requestDeviceCb)
	if ok {
		cb(RequestDeviceStatus(status), &Device{ref: device}, C.GoString(message))
	}
}

//export gowebgpu_device_lost_callback_go
func gowebgpu_device_lost_callback_go(reason C.WGPUDeviceLostReason, message *C.char, userdata unsafe.Pointer) {
	handle := *(*cgo.Handle)(userdata)
	defer handle.Delete()

	cb, ok := handle.Value().(DeviceLostCallback)
	if ok {
		cb(DeviceLostReason(reason), C.GoString(message))
	}
}

func (p *Adapter) RequestDevice(descriptor *DeviceDescriptor) (*Device, error) {
	var desc *C.WGPUDeviceDescriptor = nil

	if descriptor != nil {
		desc = &C.WGPUDeviceDescriptor{}

		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}

		requiredFeatureCount := len(descriptor.RequiredFeatures)
		if requiredFeatureCount != 0 {
			requiredFeatures := C.malloc(C.size_t(requiredFeatureCount) * C.size_t(unsafe.Sizeof(C.WGPUFeatureName(0))))
			defer C.free(requiredFeatures)

			requiredFeaturesSlice := unsafe.Slice((*FeatureName)(requiredFeatures), requiredFeatureCount)
			copy(requiredFeaturesSlice, descriptor.RequiredFeatures)

			desc.requiredFeatures = (*C.WGPUFeatureName)(requiredFeatures)
			desc.requiredFeatureCount = C.size_t(requiredFeatureCount)
		}

		if descriptor.RequiredLimits != nil {
			requiredLimits := (*C.WGPURequiredLimits)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPURequiredLimits{}))))
			defer C.free(unsafe.Pointer(requiredLimits))

			l := descriptor.RequiredLimits.Limits
			requiredLimits.limits = C.WGPULimits{
				maxTextureDimension1D:                     C.uint32_t(l.MaxTextureDimension1D),
				maxTextureDimension2D:                     C.uint32_t(l.MaxTextureDimension2D),
				maxTextureDimension3D:                     C.uint32_t(l.MaxTextureDimension3D),
				maxTextureArrayLayers:                     C.uint32_t(l.MaxTextureArrayLayers),
				maxBindGroups:                             C.uint32_t(l.MaxBindGroups),
				maxBindingsPerBindGroup:                   C.uint32_t(l.MaxBindingsPerBindGroup),
				maxDynamicUniformBuffersPerPipelineLayout: C.uint32_t(l.MaxDynamicUniformBuffersPerPipelineLayout),
				maxDynamicStorageBuffersPerPipelineLayout: C.uint32_t(l.MaxDynamicStorageBuffersPerPipelineLayout),
				maxSampledTexturesPerShaderStage:          C.uint32_t(l.MaxSampledTexturesPerShaderStage),
				maxSamplersPerShaderStage:                 C.uint32_t(l.MaxSamplersPerShaderStage),
				maxStorageBuffersPerShaderStage:           C.uint32_t(l.MaxStorageBuffersPerShaderStage),
				maxStorageTexturesPerShaderStage:          C.uint32_t(l.MaxStorageTexturesPerShaderStage),
				maxUniformBuffersPerShaderStage:           C.uint32_t(l.MaxUniformBuffersPerShaderStage),
				maxUniformBufferBindingSize:               C.uint64_t(l.MaxUniformBufferBindingSize),
				maxStorageBufferBindingSize:               C.uint64_t(l.MaxStorageBufferBindingSize),
				minUniformBufferOffsetAlignment:           C.uint32_t(l.MinUniformBufferOffsetAlignment),
				minStorageBufferOffsetAlignment:           C.uint32_t(l.MinStorageBufferOffsetAlignment),
				maxVertexBuffers:                          C.uint32_t(l.MaxVertexBuffers),
				maxBufferSize:                             C.uint64_t(l.MaxBufferSize),
				maxVertexAttributes:                       C.uint32_t(l.MaxVertexAttributes),
				maxVertexBufferArrayStride:                C.uint32_t(l.MaxVertexBufferArrayStride),
				maxInterStageShaderComponents:             C.uint32_t(l.MaxInterStageShaderComponents),
				maxInterStageShaderVariables:              C.uint32_t(l.MaxInterStageShaderVariables),
				maxColorAttachments:                       C.uint32_t(l.MaxColorAttachments),
				maxComputeWorkgroupStorageSize:            C.uint32_t(l.MaxComputeWorkgroupStorageSize),
				maxComputeInvocationsPerWorkgroup:         C.uint32_t(l.MaxComputeInvocationsPerWorkgroup),
				maxComputeWorkgroupSizeX:                  C.uint32_t(l.MaxComputeWorkgroupSizeX),
				maxComputeWorkgroupSizeY:                  C.uint32_t(l.MaxComputeWorkgroupSizeY),
				maxComputeWorkgroupSizeZ:                  C.uint32_t(l.MaxComputeWorkgroupSizeZ),
				maxComputeWorkgroupsPerDimension:          C.uint32_t(l.MaxComputeWorkgroupsPerDimension),
			}
			desc.requiredLimits = requiredLimits

			requiredLimitsExtras := (*C.WGPURequiredLimitsExtras)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPURequiredLimitsExtras{}))))
			defer C.free(unsafe.Pointer(requiredLimitsExtras))

			requiredLimitsExtras.chain.next = nil
			requiredLimitsExtras.chain.sType = C.WGPUSType_RequiredLimitsExtras
			requiredLimitsExtras.limits.maxPushConstantSize = C.uint32_t(l.MaxPushConstantSize)

			desc.requiredLimits.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(requiredLimitsExtras))
		}

		if descriptor.DeviceLostCallback != nil {
			handle := cgo.NewHandle(descriptor.DeviceLostCallback)

			desc.deviceLostCallback = C.WGPUDeviceLostCallback(C.gowebgpu_device_lost_callback_c)
			desc.deviceLostUserdata = unsafe.Pointer(&handle)
		}

		if descriptor.TracePath != "" {
			deviceExtras := (*C.WGPUDeviceExtras)(C.malloc(C.size_t(unsafe.Sizeof(C.WGPUDeviceExtras{}))))
			defer C.free(unsafe.Pointer(deviceExtras))

			deviceExtras.chain.next = nil
			deviceExtras.chain.sType = C.WGPUSType_DeviceExtras

			tracePath := C.CString(descriptor.TracePath)
			defer C.free(unsafe.Pointer(tracePath))

			deviceExtras.tracePath = tracePath

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(deviceExtras))
		}
	}

	var status RequestDeviceStatus
	var device *Device

	var cb requestDeviceCb = func(s RequestDeviceStatus, d *Device, _ string) {
		status = s
		device = d
	}
	handle := cgo.NewHandle(cb)
	C.wgpuAdapterRequestDevice(p.ref, desc, C.WGPUAdapterRequestDeviceCallback(C.gowebgpu_request_device_callback_c), unsafe.Pointer(&handle))

	if status != RequestDeviceStatusSuccess {
		return nil, errors.New("failed to request device")
	}

	return device, nil
}

func (p *Adapter) Release() {
	C.wgpuAdapterRelease(p.ref)
}
