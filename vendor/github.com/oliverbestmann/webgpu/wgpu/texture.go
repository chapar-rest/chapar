//go:build !js

package wgpu

/*

#include <stdlib.h>
#include <wgpu.h>

extern void gowebgpu_error_callback_c(enum WGPUPopErrorScopeStatus status, WGPUErrorType type, WGPUStringView message, void * userdata, void * userdata2);

static inline WGPUTextureView gowebgpu_texture_create_view(WGPUTexture texture, WGPUTextureViewDescriptor const * descriptor, WGPUDevice device, void * error_userdata) {
	WGPUTextureView ref = NULL;
	wgpuDevicePushErrorScope(device, WGPUErrorFilter_Validation);
	ref = wgpuTextureCreateView(texture, descriptor);

	WGPUPopErrorScopeCallbackInfo const err_cb = {
		.callback = gowebgpu_error_callback_c,
		.userdata1 = error_userdata,
	};

	wgpuDevicePopErrorScope(device, err_cb);

	return ref;
}

*/
import "C"
import (
	"unsafe"
)

func (g *Texture) TryCreateView(descriptor *TextureViewDescriptor) (*TextureView, error) {
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

			desc.label.data = label
			desc.label.length = C.WGPU_STRLEN
		}
	}

	errh := acquireErrorCallback()
	defer errh.Done()

	ref := C.gowebgpu_texture_create_view(
		g.ref,
		desc,
		g.device,
		errh.ToPointer(),
	)
	if errh.err != nil {
		C.wgpuTextureViewRelease(ref)
		return nil, errh.err
	}

	return releaseOnGC(&TextureView{ref: ref}), nil
}

func (g *Texture) Destroy() {
	C.wgpuTextureDestroy(g.ref)
}

func (g *Texture) GetDepthOrArrayLayers() uint32 {
	return uint32(C.wgpuTextureGetDepthOrArrayLayers(g.ref))
}

func (g *Texture) GetDimension() TextureDimension {
	return TextureDimension(C.wgpuTextureGetDimension(g.ref))
}

func (g *Texture) GetFormat() TextureFormat {
	return TextureFormat(C.wgpuTextureGetFormat(g.ref))
}

func (g *Texture) GetHeight() uint32 {
	return uint32(C.wgpuTextureGetHeight(g.ref))
}

func (g *Texture) GetMipLevelCount() uint32 {
	return uint32(C.wgpuTextureGetMipLevelCount(g.ref))
}

func (g *Texture) GetSampleCount() uint32 {
	return uint32(C.wgpuTextureGetSampleCount(g.ref))
}

func (g *Texture) GetUsage() TextureUsage {
	return TextureUsage(C.wgpuTextureGetUsage(g.ref))
}

func (g *Texture) GetWidth() uint32 {
	return uint32(C.wgpuTextureGetWidth(g.ref))
}
