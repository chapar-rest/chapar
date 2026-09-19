// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tfm

import (
	"io"

	"star-tex.org/x/tex/font/fixed"
	"star-tex.org/x/tex/internal/iobuf"
)

func newReader(r io.Reader) (*iobuf.Reader, error) {
	p, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return iobuf.NewReader(p), nil
}

func readStr(r *iobuf.Reader, max int) string {
	n := int(r.ReadU8())
	if n > max {
		return ""
	}

	raw := r.ReadBuf(max - 1)
	return string(raw[:n])
}

func readHeader(r *iobuf.Reader, hdr *fileHeader) error {
	const hdrSize = 12 * 2
	if len(r.Bytes()) < hdrSize {
		return io.ErrUnexpectedEOF
	}

	hdr.lf = r.ReadU16()
	hdr.lh = r.ReadU16()
	hdr.bc = r.ReadU16()
	hdr.ec = r.ReadU16()
	hdr.nw = r.ReadU16()
	hdr.nh = r.ReadU16()
	hdr.nd = r.ReadU16()
	hdr.ni = r.ReadU16()
	hdr.nl = r.ReadU16()
	hdr.nk = r.ReadU16()
	hdr.ne = r.ReadU16()
	hdr.np = r.ReadU16()
	return nil
}

func readCharInfos(r *iobuf.Reader, n int) []glyphInfo {
	if len(r.Bytes()) < n*4 {
		return nil
	}

	out := make([]glyphInfo, n)
	for i := range out {
		out[i].raw[0] = r.ReadU8()
		out[i].raw[1] = r.ReadU8()
		out[i].raw[2] = r.ReadU8()
		out[i].raw[3] = r.ReadU8()
	}
	return out
}

func readFWs(r *iobuf.Reader, n int) []fixed.Int12_20 {
	if len(r.Bytes()) < n*4 {
		return nil
	}

	out := make([]fixed.Int12_20, n)
	for i := range out {
		out[i] = fixed.Int12_20(r.ReadU32())
	}
	return out
}

func readLigKerns(r *iobuf.Reader, n int) []ligKernCmd {
	if len(r.Bytes()) < n*4 {
		return nil
	}

	out := make([]ligKernCmd, n)
	for i := range out {
		out[i].raw[0] = r.ReadU8()
		out[i].raw[1] = r.ReadU8()
		out[i].raw[2] = r.ReadU8()
		out[i].raw[3] = r.ReadU8()
	}
	return out
}

func readExtens(r *iobuf.Reader, n int) []extensible {
	if len(r.Bytes()) < n*4 {
		return nil
	}

	out := make([]extensible, n)
	for i := range out {
		out[i].raw[0] = r.ReadU8()
		out[i].raw[1] = r.ReadU8()
		out[i].raw[2] = r.ReadU8()
		out[i].raw[3] = r.ReadU8()
	}
	return out
}
