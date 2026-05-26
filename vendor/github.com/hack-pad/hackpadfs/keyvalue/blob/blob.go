// Package blob defines a common data interchange type for keyvalue FS's.
package blob

// Blob is a binary blob of data that can support platform-optimized mutations for better performance.
type Blob interface {
	// Bytes returns the byte slice equivalent to the data in this Blob.
	Bytes() []byte
	// Len returns the number of bytes contained in this blob.
	// Can be used to avoid unnecessary conversions and allocations from len(Bytes()).
	Len() int
}

// ViewBlob is a Blob that can return a view into the same underlying data.
// Mutating the returned Blob also mutates the original.
type ViewBlob interface {
	Blob
	View(start, end int64) (Blob, error)
}

// SliceBlob is a Blob that can return a copy of the data between start and end.
type SliceBlob interface {
	Blob
	Slice(start, end int64) (Blob, error)
}

// SetBlob is a Blob that can copy 'src' into itself starting at the given offset into this Blob.
// Use View() on 'src' to control the maximum that is copied into this Blob.
type SetBlob interface {
	Blob
	Set(src Blob, offset int64) (n int, err error)
}

// GrowBlob is a Blob that can increase it's size by allocating offset bytes at the end.
type GrowBlob interface {
	Blob
	Grow(offset int64) error
}

// TruncateBlob is a Blob that can cut off bytes from the end until it is 'size' bytes long.
type TruncateBlob interface {
	Blob
	Truncate(size int64) error
}

// View attempts to call an optimized blob.View(), falls back to copying into Bytes and running Bytes.View().
func View(b Blob, start, end int64) (Blob, error) {
	if b, ok := b.(ViewBlob); ok {
		return b.View(start, end)
	}
	return NewBytes(b.Bytes()).View(start, end)
}

// Slice attempts to call an optimized blob.Slice(), falls back to copying into Bytes and running Bytes.Slice().
func Slice(b Blob, start, end int64) (Blob, error) {
	if b, ok := b.(SliceBlob); ok {
		return b.Slice(start, end)
	}
	return NewBytes(b.Bytes()).Slice(start, end)
}

// Set attempts to call an optimized blob.Set(), falls back to copying into Bytes and running Bytes.Set().
func Set(dest Blob, src Blob, offset int64) (n int, err error) {
	if dest, ok := dest.(SetBlob); ok {
		return dest.Set(src, offset)
	}
	return NewBytes(dest.Bytes()).Set(src, offset)
}

// Grow attempts to call an optimized blob.Grow(), falls back to copying into Bytes and running Bytes.Grow().
func Grow(b Blob, offset int64) error {
	if b, ok := b.(GrowBlob); ok {
		return b.Grow(offset)
	}
	return NewBytes(b.Bytes()).Grow(offset)
}

// Truncate attempts to call an optimized blob.Truncate(), falls back to copying into Bytes and running Bytes.Truncate().
func Truncate(b Blob, size int64) error {
	if b, ok := b.(TruncateBlob); ok {
		return b.Truncate(size)
	}
	return NewBytes(b.Bytes()).Truncate(size)
}
