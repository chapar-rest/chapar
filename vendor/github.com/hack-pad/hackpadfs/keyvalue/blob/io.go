package blob

import "io"

// Reader reads a Blob of up to 'length' bytes.
type Reader interface {
	ReadBlob(length int) (blob Blob, n int, err error)
}

// ReaderAt reads a Blob of up to 'length' bytes starting from this reader's 'srcOffset'.
type ReaderAt interface {
	ReadBlobAt(length int, srcOffset int64) (blob Blob, n int, err error)
}

// Writer writes 'src' to this writer.
type Writer interface {
	WriteBlob(src Blob) (n int, err error)
}

// WriterAt writes 'src' to this writer starting at 'destOffset'.
type WriterAt interface {
	WriteBlobAt(src Blob, destOffset int64) (n int, err error)
}

// Read reads 'src' into a new Blob up to length bytes. Attempts to use an optimized ReadBlob if available.
func Read(src io.Reader, length int) (blob Blob, n int, err error) {
	if src, ok := src.(Reader); ok {
		return src.ReadBlob(length)
	}
	buf := make([]byte, length)
	n, err = src.Read(buf)
	return NewBytes(buf), n, err
}

// ReadAt reads 'src' into a new Blob up to length bytes starting at 'srcOffset'. Attempts to use an optimized ReadBlobAt if available.
func ReadAt(src io.ReaderAt, length int, srcOffset int64) (blob Blob, n int, err error) {
	if src, ok := src.(ReaderAt); ok {
		return src.ReadBlobAt(length, srcOffset)
	}
	buf := make([]byte, length)
	n, err = src.ReadAt(buf, srcOffset)
	return NewBytes(buf), n, err
}

// Write writes 'src' into 'dest'. Attempts to use an optimized WriteBlob if available.
func Write(dest io.Writer, src Blob) (n int, err error) {
	if dest, ok := dest.(Writer); ok {
		return dest.WriteBlob(src)
	}
	return dest.Write(src.Bytes())
}

// WriteAt writes 'src' into 'dest' starting at 'destOffset'. Attempts to use an optimized WriteBlobAt if available.
func WriteAt(dest io.WriterAt, src Blob, destOffset int64) (n int, err error) {
	if dest, ok := dest.(WriterAt); ok {
		return dest.WriteBlobAt(src, destOffset)
	}
	return dest.WriteAt(src.Bytes(), destOffset)
}
