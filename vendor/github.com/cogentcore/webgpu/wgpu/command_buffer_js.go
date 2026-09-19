//go:build js

package wgpu

import "syscall/js"

// CommandBuffer as described:
// https://gpuweb.github.io/gpuweb/#gpucommandbuffer
type CommandBuffer struct {
	jsValue js.Value
}

func (g CommandBuffer) toJS() any {
	return g.jsValue
}

func (g CommandBuffer) Release() {} // no-op
