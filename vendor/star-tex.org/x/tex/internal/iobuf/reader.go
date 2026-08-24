// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iobuf

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Reader struct {
	p []byte
	c int
}

func NewReader(p []byte) *Reader {
	return &Reader{p: p}
}

func (r *Reader) Len() int     { return len(r.p) }
func (r *Reader) Pos() int     { return r.c }
func (r *Reader) SetPos(p int) { r.c = p }

func (r *Reader) Bytes() []byte { return r.p[r.c:] }

func (r *Reader) PeekU8() uint8 {
	return r.p[r.c]
}

func (r *Reader) Read(p []byte) (int, error) {
	if r.c >= len(r.p) {
		return 0, io.EOF
	}
	n := copy(p, r.p[r.c:])
	r.c += n
	return n, nil
}

func (r *Reader) ReadByte() (byte, error) {
	if r.c >= len(r.p) {
		return 0, io.EOF
	}
	v := r.p[r.c]
	r.c++
	return v, nil
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.c = int(offset)
	case io.SeekCurrent:
		r.c += int(offset)
	case io.SeekEnd:
		r.c = len(r.p) - int(offset)
	default:
		return 0, fmt.Errorf("rbytes: invalid whence")
	}
	if r.c < 0 {
		return 0, fmt.Errorf("rbytes: negative position")
	}
	return int64(r.c), nil
}

func (r *Reader) ReadU8() uint8 {
	c := r.c
	r.c++
	return r.p[c]
}

func (r *Reader) ReadU16() uint16 {
	var (
		beg = r.c
		end = r.c + 2
		buf = r.p[beg:end]
	)
	r.c = end
	return binary.BigEndian.Uint16(buf)
}

func (r *Reader) ReadU24() uint32 {
	var (
		beg = r.c
		end = r.c + 3
		buf = r.p[beg:end]
	)
	r.c = end
	return uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])
}

func (r *Reader) ReadU32() uint32 {
	var (
		beg = r.c
		end = r.c + 4
		buf = r.p[beg:end]
	)
	r.c = end
	return binary.BigEndian.Uint32(buf)
}

func (r *Reader) ReadI8() int8 {
	return int8(r.ReadU8())
}

func (r *Reader) ReadI16() int16 {
	return int16(r.ReadU16())
}

func (r *Reader) ReadI24() int32 {
	var (
		beg = r.c
		end = r.c + 3
		buf = r.p[beg:end]
	)
	r.c = end
	if buf[0] < 128 {
		return int32(uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2]))
	}
	return int32((uint32(buf[0])-256)<<16 | uint32(buf[1])<<8 | uint32(buf[2]))
}

func (r *Reader) ReadI32() int32 {
	return int32(r.ReadU32())
}

func (r *Reader) ReadBuf(n int) []byte {
	var (
		beg = r.c
		end = r.c + n
		buf = r.p[beg:end]
	)
	r.c = end
	return buf
}

var (
	_ io.Reader     = (*Reader)(nil)
	_ io.ByteReader = (*Reader)(nil)
	_ io.Seeker     = (*Reader)(nil)
)
