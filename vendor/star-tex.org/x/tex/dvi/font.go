// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dvi

import (
	"star-tex.org/x/tex/font/fixed"
	"star-tex.org/x/tex/font/tfm"
)

// Font describes a DVI font, with TeX Font Metrics and its
// associated font glyph data.
type Font struct {
	name  string
	font  tfm.Font
	scale fixed.Int12_20
}

// Name returns the name of the DVI font.
// ex: "cmr10", "cmmi10" or "cmti10".
func (fnt *Font) Name() string {
	return fnt.name
}

// Size returns the DVI font size.
func (fnt *Font) Size() fixed.Int12_20 {
	return fnt.scale
}

// Metrics returns the associated TeX Font Metrics.
func (fnt *Font) Metrics() *tfm.Font {
	return &fnt.font
}

func (fnt *Font) advance(r rune) (fixed.Int12_20, bool) {
	w, _, _, ok := fnt.font.Box(r)
	if !ok {
		return 0, ok
	}
	return fixed.Int12_20((int64(w) * int64(fnt.scale)) >> 20), true
}
