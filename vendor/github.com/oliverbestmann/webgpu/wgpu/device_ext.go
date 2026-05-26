//go:build !js

package wgpu

func (g *Device) TryCreateBufferInit(descriptor *BufferInitDescriptor) (*Buffer, error) {
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
	paddedSize := max((unpaddedSize+alignMask) & ^alignMask, CopyBufferAlignment)

	buffer, err := g.TryCreateBuffer(&BufferDescriptor{
		Label:            descriptor.Label,
		Size:             uint64(paddedSize),
		Usage:            descriptor.Usage,
		MappedAtCreation: true,
	})
	if err != nil {
		return nil, err
	}

	buf := buffer.GetMappedRange(0, uint(paddedSize))
	copy(buf, descriptor.Contents)

	if err := buffer.TryUnmap(); err != nil {
		return nil, err
	}

	return buffer, nil
}
