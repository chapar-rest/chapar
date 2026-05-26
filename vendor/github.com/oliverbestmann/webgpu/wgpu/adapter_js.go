//go:build js

package wgpu

import (
	"fmt"

	"github.com/oliverbestmann/webgpu/jsx"
)

func (g *Adapter) RequestDevice(descriptor *DeviceDescriptor) (*Device, error) {
	promise := g.jsValue.Call("requestDevice", pointerToJS(descriptor))

	device, ok := jsx.Await(promise)
	if !ok || !device.Truthy() {
		return nil, fmt.Errorf("no WebGPU device avaliable")
	}
	return &Device{jsValue: device}, nil
}

func (g *Adapter) GetInfo() AdapterInfo {
	return AdapterInfo{} // TODO(kai): implement?
}

func (g *Adapter) GetLimits() Limits {
	return limitsFromJS(g.jsValue.Get("limits"))
}
