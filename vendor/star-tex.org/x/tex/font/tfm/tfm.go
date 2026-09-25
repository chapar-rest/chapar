// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tfm implements a decoder for TFM (TeX Font Metrics) files.
package tfm // import "star-tex.org/x/tex/font/tfm"

import (
	"fmt"
)

type glyphKind uint8

const (
	gkInvalid glyphKind = iota
	gkVanilla
	gkLigKern
	gkGlyphList
	gkExtensible
)

func (ck glyphKind) String() string {
	switch ck {
	case gkVanilla:
		return "vanilla"
	case gkLigKern:
		return "ligkern"
	case gkGlyphList:
		return "glyphlist"
	case gkExtensible:
		return "extglyph"
	case gkInvalid:
		return "invalid"
	default:
		return fmt.Sprintf("glyphKind(%d)", int(ck))
	}
}

// glyphIndex is a glyph index in a Font.
type glyphIndex int

// glyphInfo provides informations about a glyph.
type glyphInfo struct {
	raw [4]uint8
}

func (g glyphInfo) wd() int {
	return int(g.raw[0])
}

func (g glyphInfo) ht() int {
	return int((g.raw[1] & 0b1111_0000) >> 4)
}

func (g glyphInfo) dp() int {
	return int(g.raw[1] & 0b0000_1111)
}

func (g glyphInfo) ic() int {
	return int((g.raw[2] & 0b1111_1100) >> 2)
}

func (g glyphInfo) kind() (glyphKind, int) {
	i := int(g.raw[3])
	switch g.raw[2] & 0b0000_0011 {
	case 0:
		return gkVanilla, i
	case 1:
		return gkLigKern, i
	case 2:
		return gkGlyphList, i
	case 3:
		return gkExtensible, i
	default:
		return gkInvalid, -i
	}
}

type ligKernCmd struct {
	raw [4]uint8
}

func (lk ligKernCmd) skipByte() bool {
	return lk.raw[0] > 128
}

func (lk ligKernCmd) nextChar() int {
	return int(lk.raw[1])
}

func (lk ligKernCmd) op() ligKernOp {
	if lk.raw[2] < 128 {
		return ligCmd
	}
	return krnCmd
}

func (lk ligKernCmd) nextIndex() int {
	return (int(lk.raw[2])-128)*256 + int(lk.raw[3])
}

type ligKernOp uint8

const (
	lkUknown ligKernOp = iota
	ligCmd
	krnCmd
)

func (lk ligKernOp) String() string {
	switch lk {
	default:
		return "invalid"
	case ligCmd:
		return "LIG"
	case krnCmd:
		return "KRN"
	}
}

type extensible struct {
	raw [4]uint8
}
