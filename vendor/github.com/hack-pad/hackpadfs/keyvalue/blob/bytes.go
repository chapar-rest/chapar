package blob

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	// ensure Bytes conforms to these interfaces:
	_ interface {
		Blob
		ViewBlob
		SliceBlob
		SetBlob
		GrowBlob
		TruncateBlob
	} = (*Bytes)(nil)
)

// Bytes is a Blob that wraps a byte slice.
type Bytes struct {
	mu     *sync.Mutex // mutex can be shared when the byte slice is shared
	bytes  []byte
	length int64
}

// NewBytes returns a Blob that wraps the given byte slice.
// Mutations to this Blob are reflected in the original slice.
func NewBytes(buf []byte) *Bytes {
	return &Bytes{
		bytes:  buf,
		length: int64(len(buf)),
		mu:     new(sync.Mutex),
	}
}

// NewBytesLength returns a new Bytes with the given length of zero-byte data.
func NewBytesLength(length int) *Bytes {
	return NewBytes(make([]byte, length))
}

// Bytes implements Blob.
func (b *Bytes) Bytes() []byte {
	// always return a copy of the bytes, to avoid concurrent modification
	newB, err := b.Slice(0, int64(b.Len()))
	if err != nil {
		panic(err)
	}
	return newB.(*Bytes).bytes
}

// Len implements Blob.
func (b *Bytes) Len() int {
	return int(atomic.LoadInt64(&b.length))
}

// View implements Blob.
func (b *Bytes) View(start, end int64) (Blob, error) {
	if start < 0 || start > int64(b.Len()) {
		return nil, fmt.Errorf("Start index out of bounds: %d", start)
	}
	if end < 0 || end > int64(b.Len()) {
		return nil, fmt.Errorf("End index out of bounds: %d", end)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	newB := NewBytes(b.bytes[start:end])
	newB.mu = b.mu
	return newB, nil
}

// Slice implements Blob.
func (b *Bytes) Slice(start, end int64) (Blob, error) {
	if start < 0 || start > int64(b.Len()) {
		return nil, fmt.Errorf("Start index out of bounds: %d", start)
	}
	if end < 0 || end > int64(b.Len()) {
		return nil, fmt.Errorf("End index out of bounds: %d", end)
	}
	buf := make([]byte, end-start)
	b.mu.Lock()
	copy(buf, b.bytes)
	b.mu.Unlock()
	return NewBytes(buf), nil
}

// Set implements Blob.
func (b *Bytes) Set(src Blob, destStart int64) (n int, err error) {
	if destStart < 0 {
		return 0, errors.New("negative offset")
	}
	if destStart >= int64(b.Len()) && destStart == 0 && src.Len() > 0 {
		return 0, fmt.Errorf("Offset out of bounds: %d", destStart)
	}
	b.mu.Lock()
	n = copy(b.bytes[destStart:], src.Bytes())
	b.mu.Unlock()
	return n, nil
}

// Grow implements Blob.
func (b *Bytes) Grow(offset int64) error {
	b.mu.Lock()
	b.bytes = append(b.bytes, make([]byte, offset)...)
	atomic.StoreInt64(&b.length, int64(len(b.bytes)))
	b.mu.Unlock()
	return nil
}

// Truncate implements Blob.
func (b *Bytes) Truncate(size int64) error {
	if int64(b.Len()) < size {
		return nil
	}
	b.mu.Lock()
	if int64(b.Len()) >= size {
		b.bytes = b.bytes[:size]
		atomic.StoreInt64(&b.length, int64(len(b.bytes)))
	}
	b.mu.Unlock()
	return nil
}
