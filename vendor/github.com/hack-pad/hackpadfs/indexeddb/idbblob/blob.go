//go:build wasm
// +build wasm

// Package idbblob contains a JavaScript implementation of blob.Blob.
package idbblob

import (
	"errors"
	"fmt"
	"sync/atomic"
	"syscall/js"

	"github.com/hack-pad/hackpadfs/internal/jswrapper"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
	"github.com/hack-pad/safejs"
)

var uint8Array safejs.Value

func init() {
	var err error
	uint8Array, err = safejs.Global().Get("Uint8Array")
	if err != nil {
		panic(err)
	}
}

var (
	_ interface {
		blob.Blob
		blob.ViewBlob
		blob.SliceBlob
		blob.SetBlob
		blob.GrowBlob
		blob.TruncateBlob
	} = &Blob{}
)

// Blob is a blob.Blob for JS environments, optimized for reading and writing to a Uint8Array without en/decoding back and forth.
type Blob struct {
	bytes   atomic.Value // *blob.Bytes
	jsValue atomic.Value // safejs.Value
	length  int64
}

// New creates a Blob wrapping the given JS Uint8Array buffer.
func New(unsafeBuf js.Value) (*Blob, error) {
	buf := safejs.Safe(unsafeBuf)
	truthy, err := buf.Truthy()
	if err != nil {
		return nil, err
	}
	if !truthy {
		return FromBlob(blob.NewBytes(nil)), nil
	}
	instanceOf, err := buf.InstanceOf(uint8Array)
	if err != nil {
		return nil, err
	}
	if !instanceOf {
		return nil, fmt.Errorf("invalid JS array type: %v", buf)
	}
	return newBlob(buf)
}

// NewLength creates a zero-value Blob with 'length' bytes.
func NewLength(length int) (*Blob, error) {
	jsBuf, err := uint8Array.New(length)
	if err != nil {
		return nil, err
	}
	return newBlob(jsBuf)
}

func newBlob(buf safejs.Value) (*Blob, error) {
	b := &Blob{}
	b.jsValue.Store(buf)
	length, err := buf.Length()
	if err != nil {
		return nil, err
	}
	atomic.StoreInt64(&b.length, int64(length))
	return b, nil
}

// FromBlob creates a Blob from the given blob.Blob, either wrapping the JS value or copying the bytes if incompatible.
func FromBlob(b blob.Blob) *Blob {
	blob, err := fromBlob(b)
	if err != nil {
		panic(err)
	}
	return blob
}

func fromBlob(b blob.Blob) (*Blob, error) {
	if b, ok := b.(jswrapper.Wrapper); ok {
		value := safejs.Safe(b.JSValue())
		return newBlob(value)
	}
	buf := b.Bytes()
	jsBuf, err := uint8Array.New(len(buf))
	if err != nil {
		return nil, err
	}
	_, err = safejs.CopyBytesToJS(jsBuf, buf)
	if err != nil {
		return nil, err
	}
	return newBlob(jsBuf)
}

func (b *Blob) currentBytes() *blob.Bytes {
	buf := b.bytes.Load()
	if buf != nil {
		return buf.(*blob.Bytes)
	}
	return nil
}

// Bytes implememts blob.Blob
func (b *Blob) Bytes() []byte {
	buf, err := b.getBytes()
	if err != nil {
		panic(err)
	}
	return buf
}

func (b *Blob) getBytes() ([]byte, error) {
	if buf := b.currentBytes(); buf != nil {
		return buf.Bytes(), nil
	}
	jsBuf := b.jsValue.Load().(safejs.Value)
	length, err := jsBuf.Length()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, length)
	_, err = safejs.CopyBytesToGo(buf, jsBuf)
	if err != nil {
		return nil, err
	}
	b.bytes.Store(blob.NewBytes(buf))
	return buf, nil
}

// JSValue implements jswrapper.Wrapper
func (b *Blob) JSValue() js.Value {
	value := b.jsValue.Load().(safejs.Value)
	return safejs.Unsafe(value)
}

// Len implements blob.Blob
func (b *Blob) Len() int {
	return int(atomic.LoadInt64(&b.length))
}

// View implements blob.ViewBlob
func (b *Blob) View(start, end int64) (blob.Blob, error) {
	if start == 0 && end == atomic.LoadInt64(&b.length) {
		return b, nil
	}

	var newBlob *Blob
	value := safejs.Safe(b.JSValue())
	subarray, err := value.Call("subarray", start, end)
	if err != nil {
		return nil, err
	}
	newBlob, err = New(safejs.Unsafe(subarray))
	if err != nil {
		return nil, err
	}

	if buf := b.currentBytes(); buf != nil {
		newBytesBlob, err := b.currentBytes().View(start, end)
		if err != nil {
			return nil, err
		}
		newBlob.bytes.Store(newBytesBlob)
	}
	return newBlob, nil
}

// Slice implements blob.SliceBlob
func (b *Blob) Slice(start, end int64) (blob.Blob, error) {
	if start < 0 || start > int64(b.Len()) {
		return nil, fmt.Errorf("Start index out of bounds: %d", start)
	}
	if end < 0 || end > int64(b.Len()) {
		return nil, fmt.Errorf("End index out of bounds: %d", end)
	}

	value := safejs.Safe(b.JSValue())
	slice, err := value.Call("slice", start, end)
	if err != nil {
		return nil, err
	}
	newBlob, err := New(safejs.Unsafe(slice))
	if err != nil {
		return nil, err
	}
	if buf := b.currentBytes(); buf != nil {
		newBytes, err := buf.Slice(start, end)
		if err != nil {
			return nil, err
		}
		newBlob.bytes.Store(newBytes)
	}
	return newBlob, nil
}

// Set implements blob.SetBlob
func (b *Blob) Set(src blob.Blob, destStart int64) (n int, err error) {
	if destStart < 0 {
		return 0, errors.New("negative offset")
	}
	if destStart >= int64(b.Len()) && destStart == 0 && src.Len() > 0 {
		return 0, fmt.Errorf("Offset out of bounds: %d", destStart)
	}

	bValue := safejs.Safe(b.JSValue())
	srcValue := safejs.Safe(FromBlob(src).JSValue())
	_, err = bValue.Call("set", srcValue, destStart)
	if err != nil {
		return 0, err
	}
	n = src.Len()

	if buf := b.currentBytes(); buf != nil {
		_, err := buf.Set(src, destStart)
		if err != nil {
			return 0, err
		}
	}
	return n, nil
}

// Grow implements blob.GrowBlob
func (b *Blob) Grow(off int64) error {
	newLength := atomic.LoadInt64(&b.length) + off

	buf := b.jsValue.Load().(safejs.Value)
	biggerBuf, err := uint8Array.New(newLength)
	if err != nil {
		return err
	}
	_, err = biggerBuf.Call("set", buf, 0)
	if err != nil {
		return err
	}
	b.jsValue.Store(biggerBuf)
	atomic.StoreInt64(&b.length, newLength)

	if buf := b.currentBytes(); buf != nil {
		err := buf.Grow(off)
		if err != nil {
			return err
		}
	}
	return nil
}

// Truncate implements TruncateBlob
func (b *Blob) Truncate(size int64) error {
	if atomic.LoadInt64(&b.length) < size {
		return nil
	}

	value := safejs.Safe(b.JSValue())
	smallerBuf, err := value.Call("slice", 0, size)
	if err != nil {
		return err
	}
	b.jsValue.Store(smallerBuf)
	atomic.StoreInt64(&b.length, size)

	if buf := b.currentBytes(); buf != nil {
		err := buf.Truncate(size)
		if err != nil {
			return err
		}
	}
	return nil
}
