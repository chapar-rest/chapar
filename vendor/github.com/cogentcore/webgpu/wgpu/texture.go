//go:build !js

package wgpu

/*

#include <stdlib.h>
#include "./lib/wgpu.h"

extern void gowebgpu_error_callback_c(WGPUErrorType type, char const * message, void * userdata);

static inline WGPUTextureView gowebgpu_texture_create_view(WGPUTexture texture, WGPUTextureViewDescriptor const * descriptor, WGPUDevice device, void * error_userdata) {
	WGPUTextureView ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuTextureCreateView(texture, descriptor);
	wgpuDevicePopErrorScope(device, gowebgpu_error_callback_c, error_userdata);
	return ref;
}

static inline void gowebgpu_texture_release(WGPUTexture texture, WGPUDevice device) {
	wgpuDeviceRelease(device);
	wgpuTextureRelease(texture);
}

*/
import "C"
import (
	"errors"
	"runtime/cgo"
	"unsafe"
)

type Texture struct {
	deviceRef C.WGPUDevice
	ref       C.WGPUTexture
}

func (p *Texture) CreateView(descriptor *TextureViewDescriptor) (*TextureView, error) {
	var desc *C.WGPUTextureViewDescriptor

	if descriptor != nil {
		desc = &C.WGPUTextureViewDescriptor{
			format:          C.WGPUTextureFormat(descriptor.Format),
			dimension:       C.WGPUTextureViewDimension(descriptor.Dimension),
			baseMipLevel:    C.uint32_t(descriptor.BaseMipLevel),
			mipLevelCount:   C.uint32_t(descriptor.MipLevelCount),
			baseArrayLayer:  C.uint32_t(descriptor.BaseArrayLayer),
			arrayLayerCount: C.uint32_t(descriptor.ArrayLayerCount),
			aspect:          C.WGPUTextureAspect(descriptor.Aspect),
		}

		if descriptor.Label != "" {
			label := C.CString(descriptor.Label)
			defer C.free(unsafe.Pointer(label))

			desc.label = label
		}
	}

	var err error = nil
	var cb errorCallback = func(_ ErrorType, message string) {
		err = errors.New("wgpu.(*Texture).CreateView(): " + message)
	}
	errorCallbackHandle := cgo.NewHandle(cb)
	defer errorCallbackHandle.Delete()

	ref := C.gowebgpu_texture_create_view(
		p.ref,
		desc,
		p.deviceRef,
		unsafe.Pointer(&errorCallbackHandle),
	)
	if err != nil {
		C.wgpuTextureViewRelease(ref)
		return nil, err
	}

	return &TextureView{ref}, nil
}

func (p *Texture) Destroy() {
	C.wgpuTextureDestroy(p.ref)
}

func (p *Texture) GetDepthOrArrayLayers() uint32 {
	return uint32(C.wgpuTextureGetDepthOrArrayLayers(p.ref))
}

func (p *Texture) GetDimension() TextureDimension {
	return TextureDimension(C.wgpuTextureGetDimension(p.ref))
}

func (p *Texture) GetFormat() TextureFormat {
	return TextureFormat(C.wgpuTextureGetFormat(p.ref))
}

func (p *Texture) GetHeight() uint32 {
	return uint32(C.wgpuTextureGetHeight(p.ref))
}

func (p *Texture) GetMipLevelCount() uint32 {
	return uint32(C.wgpuTextureGetMipLevelCount(p.ref))
}

func (p *Texture) GetSampleCount() uint32 {
	return uint32(C.wgpuTextureGetSampleCount(p.ref))
}

func (p *Texture) GetUsage() TextureUsage {
	return TextureUsage(C.wgpuTextureGetUsage(p.ref))
}

func (p *Texture) GetWidth() uint32 {
	return uint32(C.wgpuTextureGetWidth(p.ref))
}

func (p *Texture) Release() {
	C.gowebgpu_texture_release(p.ref, p.deviceRef)
}
