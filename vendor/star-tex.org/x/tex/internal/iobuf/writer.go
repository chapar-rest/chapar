// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iobuf

import (
	"encoding/binary"
	"io"
)

type Writer struct {
	w io.Writer

	buf []byte
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:   w,
		buf: make([]byte, 4),
	}
}

func (w *Writer) Write(v []byte) (int, error) {
	return w.w.Write(v)
}

func (w *Writer) WriteU8(v uint8) {
	w.buf[0] = v
	w.w.Write(w.buf[:1])
}

func (w *Writer) WriteU16(v uint16) {
	binary.BigEndian.PutUint16(w.buf, v)
	w.w.Write(w.buf[:2])
}

func (w *Writer) WriteU24(v uint32) {
	w.buf[0] = uint8(v >> 16)
	w.buf[1] = uint8(v >> 8)
	w.buf[2] = uint8(v)
	w.w.Write(w.buf[:3])
}

func (w *Writer) WriteU32(v uint32) {
	binary.BigEndian.PutUint32(w.buf, v)
	w.w.Write(w.buf[:4])
}

func (w *Writer) WriteI8(v int8) {
	w.WriteU8(uint8(v))
}

func (w *Writer) WriteI16(v int16) {
	w.WriteU16(uint16(v))
}

func (w *Writer) WriteI24(v int32) {
	w.buf[0] = uint8(v >> 16)
	w.buf[1] = uint8(v >> 8)
	w.buf[2] = uint8(v)
	w.w.Write(w.buf[:3])
}

func (w *Writer) WriteI32(v int32) {
	w.WriteU32(uint32(v))
}

func (w *Writer) WriteBuf(v []byte) {
	w.w.Write(v)
}

var (
	_ io.Writer = (*Writer)(nil)
)
