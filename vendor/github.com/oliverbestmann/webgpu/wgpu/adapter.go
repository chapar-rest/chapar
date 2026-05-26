//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

extern void gowebgpu_request_device_callback_c(WGPURequestDeviceStatus status, WGPUDevice device, char const *message, void *userdata);
extern void gowebgpu_device_lost_callback_c(WGPUDeviceLostReason reason, char const * message, void * userdata);

*/
import "C"
import (
	"errors"
	"unsafe"
)

func (g *Adapter) GetFeatures() []FeatureName {
	var supportedFeatures C.WGPUSupportedFeatures
	C.wgpuAdapterGetFeatures(g.ref, (*C.WGPUSupportedFeatures)(unsafe.Pointer(&supportedFeatures)))
	defer C.free(unsafe.Pointer(supportedFeatures.features))

	features := make([]FeatureName, supportedFeatures.featureCount)

	for i := range int(supportedFeatures.featureCount) {
		offset := uintptr(i) * unsafe.Sizeof(C.WGPUFeatureName(0))
		features[i] = FeatureName(*(*C.WGPUFeatureName)(unsafe.Pointer(uintptr(unsafe.Pointer(supportedFeatures.features)) + offset)))
	}

	return features
}

func (g *Adapter) GetLimits() Limits {
	var limits C.WGPULimits

	nativeLimits := (*C.WGPUNativeLimits)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUNativeLimits{}))))
	defer C.free(unsafe.Pointer(nativeLimits))
	limits.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(nativeLimits))

	C.wgpuAdapterGetLimits(g.ref, &limits)

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
		MaxColorAttachmentBytesPerSample:          uint32(limits.maxColorAttachmentBytesPerSample),
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

func (g *Adapter) GetInfo() AdapterInfo {
	var info C.WGPUAdapterInfo

	C.wgpuAdapterGetInfo(g.ref, &info)

	return AdapterInfo{
		Vendor:       C.GoStringN(info.vendor.data, C.int(info.vendor.length)),
		Architecture: C.GoStringN(info.architecture.data, C.int(info.architecture.length)),
		Device:       C.GoStringN(info.device.data, C.int(info.device.length)),
		Description:  C.GoStringN(info.description.data, C.int(info.description.length)),
		AdapterType:  AdapterType(info.adapterType),
		BackendType:  BackendType(info.backendType),
		VendorId:     uint32(info.vendorID),
		DeviceId:     uint32(info.deviceID),
	}
}

func (g *Adapter) HasFeature(feature FeatureName) bool {
	hasFeature := C.wgpuAdapterHasFeature(g.ref, C.WGPUFeatureName(feature))
	return goBool(hasFeature)
}

type requestDeviceCb func(status RequestDeviceStatus, device *Device, message string)

//export gowebgpu_request_device_callback_go
func gowebgpu_request_device_callback_go(status C.WGPURequestDeviceStatus, device C.WGPUDevice, message C.WGPUStringView, userdata unsafe.Pointer) {
	handle := lookupHandle(userdata)
	defer handle.Delete()

	cb, ok := handle.Value().(requestDeviceCb)
	if ok {
		device := releaseOnGC(&Device{ref: device})
		cb(RequestDeviceStatus(status), device, C.GoStringN(message.data, C.int(message.length)))
	}
}

//export gowebgpu_device_lost_callback_go
func gowebgpu_device_lost_callback_go(reason C.WGPUDeviceLostReason, message *C.char, userdata unsafe.Pointer) {
	handle := lookupHandle(userdata)
	defer handle.Delete()

	cb, ok := handle.Value().(DeviceLostCallback)
	if ok {
		cb(DeviceLostReason(reason), C.GoString(message))
	}
}

func (g *Adapter) RequestDevice(descriptor *DeviceDescriptor) (*Device, error) {
	var desc *C.WGPUDeviceDescriptor = nil

	if descriptor != nil {
		desc = &C.WGPUDeviceDescriptor{}

		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}

		requiredFeatureCount := len(descriptor.RequiredFeatures)
		if requiredFeatureCount != 0 {
			requiredFeatures := C.calloc(C.size_t(requiredFeatureCount), C.size_t(unsafe.Sizeof(C.WGPUFeatureName(0))))
			defer C.free(requiredFeatures)

			requiredFeaturesSlice := unsafe.Slice((*FeatureName)(requiredFeatures), requiredFeatureCount)
			copy(requiredFeaturesSlice, descriptor.RequiredFeatures)

			desc.requiredFeatures = (*C.WGPUFeatureName)(requiredFeatures)
			desc.requiredFeatureCount = C.size_t(requiredFeatureCount)
		}

		if descriptor.RequiredLimits != nil {
			l := descriptor.RequiredLimits

			requiredLimits := (*C.WGPULimits)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPULimits{}))))
			defer C.free(unsafe.Pointer(requiredLimits))

			*requiredLimits = C.WGPULimits{
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

			nativeLimits := (*C.WGPUNativeLimits)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUNativeLimits{}))))
			defer C.free(unsafe.Pointer(nativeLimits))

			nativeLimits.chain.next = nil
			nativeLimits.chain.sType = C.WGPUSType_NativeLimits
			nativeLimits.maxImmediateSize = C.uint32_t(l.MaxImmediateSize)
			nativeLimits.maxNonSamplerBindings = C.uint32_t(l.MaxNonSamplerBindings)

			desc.requiredLimits.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(nativeLimits))
		}

		if descriptor.DeviceLostCallback != nil {
			handle := newHandle(descriptor.DeviceLostCallback)

			desc.deviceLostCallbackInfo = C.WGPUDeviceLostCallbackInfo{
				callback:  C.WGPUDeviceLostCallback(C.gowebgpu_device_lost_callback_c),
				userdata1: handle.ToPointer(),
			}
		}

		if descriptor.TracePath != "" {
			deviceExtras := (*C.WGPUDeviceExtras)(C.calloc(1, C.size_t(unsafe.Sizeof(C.WGPUDeviceExtras{}))))
			defer C.free(unsafe.Pointer(deviceExtras))

			deviceExtras.chain.next = nil
			deviceExtras.chain.sType = C.WGPUSType_DeviceExtras

			tracePath := C.CString(descriptor.TracePath)
			defer C.free(unsafe.Pointer(tracePath))

			deviceExtras.tracePath.data = tracePath
			deviceExtras.tracePath.length = C.WGPU_STRLEN

			desc.nextInChain = (*C.WGPUChainedStruct)(unsafe.Pointer(deviceExtras))
		}
	}

	var status RequestDeviceStatus
	var device *Device

	var cb requestDeviceCb = func(s RequestDeviceStatus, d *Device, _ string) {
		status = s
		device = d
	}
	handle := newHandle(cb)
	C.wgpuAdapterRequestDevice(g.ref, desc, C.WGPURequestDeviceCallbackInfo{
		callback:  C.WGPURequestDeviceCallback(C.gowebgpu_request_device_callback_c),
		userdata1: handle.ToPointer(),
	})

	if status != RequestDeviceStatusSuccess {
		return nil, errors.New("failed to request device")
	}

	return device, nil
}
