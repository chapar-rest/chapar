//go:build js

package wgpu

import (
	"syscall/js"

	"github.com/oliverbestmann/webgpu/jsx"
)

// Destroy as described:
// https://gpuweb.github.io/gpuweb/#dom-gpubuffer-destroy
func (g *Buffer) Destroy() {
	g.jsValue.Call("destroy")
}

func (g *Buffer) GetMappedRange(offset, size uint) []byte {
	// TODO(kai): this does not work for writing because it does not get
	//  the actual pointer to the byte data; this is only really
	//  possible with GopherJS.
	buf := g.jsValue.Call("getMappedRange", offset, size)
	src := js.Global().Get("Uint8ClampedArray").New(buf)
	dst := make([]byte, src.Length())
	js.CopyBytesToGo(dst, src)
	return dst
}

func (p *Buffer) GetSize() uint64 {
	return uint64(p.jsValue.Get("size").Int())
}

func (g *Buffer) TryMapAsync(mode MapMode, offset uint64, size uint64, callback BufferMapCallback) (err error) {
	defer handleJsException(&err)

	// mapAsync will just return a promise
	promise := g.jsValue.Call("mapAsync", uint32(mode), offset, size)

	// TODO probably better to use .Then(...) on the promise
	//  to schedule the callback some time in the future and not to block here
	_, ok := jsx.Await(promise)
	if !ok {
		callback(MapAsyncStatusError)
		return
	}

	callback(MapAsyncStatusSuccess)
	return
}

func (g *Buffer) TryUnmap() (err error) {
	defer handleJsException(&err)

	g.jsValue.Call("unmap")
	return
}
