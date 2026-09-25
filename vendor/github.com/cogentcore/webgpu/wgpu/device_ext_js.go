//go:build js

package wgpu

import (
	"syscall/js"
)

// TODO(kai): this only needs to be separate for js because
// [Buffer.GetMappedRange] does not work correctly without GopherJS.
func (p *Device) CreateBufferInit(descriptor *BufferInitDescriptor) (*Buffer, error) {
	if descriptor == nil {
		panic("got nil descriptor")
	}

	if len(descriptor.Contents) == 0 {
		return p.CreateBuffer(&BufferDescriptor{
			Label:            descriptor.Label,
			Size:             0,
			Usage:            descriptor.Usage,
			MappedAtCreation: false,
		})
	}

	unpaddedSize := len(descriptor.Contents)
	const alignMask = CopyBufferAlignment - 1
	paddedSize := max(((unpaddedSize + alignMask) & ^alignMask), CopyBufferAlignment)

	buffer, err := p.CreateBuffer(&BufferDescriptor{
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
	buffer.Unmap()

	return buffer, nil
}
