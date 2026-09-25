//go:build js

package wgpu

import (
	"fmt"
	"syscall/js"

	"github.com/cogentcore/webgpu/jsx"
)

// Adapter as described:
// https://gpuweb.github.io/gpuweb/#gpuadapter
type Adapter struct {
	jsValue js.Value
}

func (g Adapter) RequestDevice(descriptor *DeviceDescriptor) (*Device, error) {
	device, ok := jsx.Await(g.jsValue.Call("requestDevice", pointerToJS(descriptor)))
	if !ok || !device.Truthy() {
		return nil, fmt.Errorf("no WebGPU device avaliable")
	}
	return &Device{jsValue: device}, nil
}

func (g Adapter) GetInfo() AdapterInfo {
	return AdapterInfo{} // TODO(kai): implement?
}

func (g Adapter) GetLimits() SupportedLimits {
	return SupportedLimits{limitsFromJS(g.jsValue.Get("limits"))}
}

func (g Adapter) Release() {} // no-op
