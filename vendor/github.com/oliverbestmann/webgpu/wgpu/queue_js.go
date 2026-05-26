//go:build js

package wgpu

import (
	"runtime"
	"syscall/js"
	"unsafe"

	"github.com/oliverbestmann/webgpu/jsx"
)

// Submit as described:
// https://gpuweb.github.io/gpuweb/#dom-gpuqueue-submit
func (g *Queue) Submit(commandBuffers ...*CommandBuffer) SubmissionIndex {
	jsSequence := mapSlice(commandBuffers, func(buffer *CommandBuffer) any {
		return pointerToJS(buffer)
	})
	g.jsValue.Call("submit", jsSequence)
	return SubmissionIndex(0)
}

// TryWriteBuffer as described:
// https://gpuweb.github.io/gpuweb/#dom-gpuqueue-writebuffer
func (g *Queue) TryWriteBuffer(buffer *Buffer, offset uint64, data []byte) (err error) {
	if len(data) == 0 {
		return nil
	}

	defer handleJsException(&err)
	address := uintptr(unsafe.Pointer(&data[0]))
	queueWriteBuffer.Invoke(g.jsValue, pointerToJS(buffer), offset, address, uint64(0), len(data))
	runtime.KeepAlive(data)
	return
}

// TryWriteTexture as described:
// https://gpuweb.github.io/gpuweb/#dom-gpuqueue-writetexture
func (g *Queue) TryWriteTexture(destination *TexelCopyTextureInfo, data []byte, dataLayout *TexelCopyBufferLayout, writeSize *Extent3D) (err error) {
	if len(data) == 0 {
		return nil
	}

	defer handleJsException(&err)
	address := uintptr(unsafe.Pointer(&data[0]))
	queueWriteTexture.Invoke(g.jsValue, pointerToJS(destination), address, len(data), pointerToJS(dataLayout), pointerToJS(writeSize))
	runtime.KeepAlive(data)
	return
}

// OnSubmittedWorkDone as described:
// https://gpuweb.github.io/gpuweb/#dom-gpuqueue-onsubmittedworkdone
func (g *Queue) OnSubmittedWorkDone(callback QueueWorkDoneCallback) {
	// TODO should probably just schedule the callback using .then
	jsx.Await(g.jsValue.Call("onSubmittedWorkDone"))
	callback(QueueWorkDoneStatusSuccess)
}

var queueWriteBuffer js.Value
var queueWriteTexture js.Value

func init() {
	queueWriteBuffer = js.Global().Call("eval", `
		(queue, buf, offset, addr, x, n) => {
			const mem = wasm.instance.exports.mem.buffer;
			const data = new Uint8ClampedArray(mem, addr, n);
			return queue.writeBuffer(buf, offset, data, x, n);
		} 
	`)

	queueWriteTexture = js.Global().Call("eval", `
		(queue, tex, addr, n, layout, writeSize) => {
			const mem = wasm.instance.exports.mem.buffer;
			const data = new Uint8ClampedArray(mem, addr, n);
			return queue.writeTexture(tex, data, layout, writeSize);
		} 
	`)
}
