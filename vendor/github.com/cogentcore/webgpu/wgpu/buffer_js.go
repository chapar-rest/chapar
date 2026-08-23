//go:build js

package wgpu

import (
	"syscall/js"

	"github.com/cogentcore/webgpu/jsx"
)

// Buffer as described:
// https://gpuweb.github.io/gpuweb/#gpubuffer
type Buffer struct {
	jsValue js.Value
}

func (g Buffer) toJS() any {
	return g.jsValue
}

// Destroy as described:
// https://gpuweb.github.io/gpuweb/#dom-gpubuffer-destroy
func (g Buffer) Destroy() {
	g.jsValue.Call("destroy")
}

func (g Buffer) GetMappedRange(offset, size uint) []byte {
	// TODO(kai): this does not work because it does not get
	// the actual pointer to the byte data; this is only really
	// possible with GopherJS.
	buf := g.jsValue.Call("getMappedRange", offset, size)
	src := js.Global().Get("Uint8ClampedArray").New(buf)
	dst := make([]byte, src.Length())
	js.CopyBytesToGo(dst, src)
	return dst
}

func (g Buffer) MapAsync(mode MapMode, offset uint64, size uint64, callback BufferMapCallback) (err error) {
	jsx.Await(g.jsValue.Call("mapAsync", uint32(mode), offset, size))
	callback(BufferMapAsyncStatusSuccess) // TODO(kai): is this the right thing to do?
	return
}

func (g Buffer) Unmap() (err error) {
	g.jsValue.Call("unmap")
	return
}

func (g Buffer) Release() {} // no-op
