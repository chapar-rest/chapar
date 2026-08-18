//go:build js

package wgpu

import "syscall/js"

// Sampler as described:
// https://gpuweb.github.io/gpuweb/#gpusampler
type Sampler struct {
	jsValue js.Value
}

func (g Sampler) toJS() any {
	return g.jsValue
}

func (g Sampler) Release() {} // no-op
