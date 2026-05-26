//go:build js

package wgpu

import (
	"syscall/js"
)

// TODO(kai): this only needs to be separate for js because
//
//	[Buffer.GetMappedRange] does not work correctly without GopherJS.
func (g *Device) TryCreateBufferInit(descriptor *BufferInitDescriptor) (_ *Buffer, err error) {
	defer handleJsException(&err)

	if len(descriptor.Contents) == 0 {
		return g.TryCreateBuffer(&BufferDescriptor{
			Label:            descriptor.Label,
			Size:             0,
			Usage:            descriptor.Usage,
			MappedAtCreation: false,
		})
	}

	unpaddedSize := len(descriptor.Contents)
	const alignMask = CopyBufferAlignment - 1
	paddedSize := max(((unpaddedSize + alignMask) & ^alignMask), CopyBufferAlignment)

	buffer, err := g.TryCreateBuffer(&BufferDescriptor{
		Label:            descriptor.Label,
		Size:             uint64(paddedSize),
		Usage:            descriptor.Usage,
		MappedAtCreation: true,
	})
	if err != nil {
		return nil, err
	}

	// TODO(kai): this is a temporary workaround as per the method comment.
	buf := buffer.jsValue.Call("getMappedRange", 0, uint(paddedSize))
	array := js.Global().Get("Uint8ClampedArray").New(buf)
	js.CopyBytesToJS(array, descriptor.Contents)

	if err := buffer.TryUnmap(); err != nil {
		return nil, err
	}

	return buffer, nil
}
