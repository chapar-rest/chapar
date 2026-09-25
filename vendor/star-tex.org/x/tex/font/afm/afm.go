// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package afm implements a decoder for AFM (Adobe Font Metrics) files.
//
// See:
//   - https://adobe-type-tools.github.io/font-tech-notes/pdfs/5004.AFM_Spec.pdf
//
// for more informations.
package afm // import "star-tex.org/x/tex/font/afm"

import (
	"star-tex.org/x/tex/font/fixed"
)

type trackKern struct {
	degree    int
	minPtSize fixed.Int16_16
	minKern   fixed.Int16_16
	maxPtSize fixed.Int16_16
	maxKern   fixed.Int16_16
}

type kernPair struct {
	n1 string         // name of the first character of this pair.
	n2 string         // name of the second character of this pair.
	x  fixed.Int16_16 // x component of the kerning vector.
	y  fixed.Int16_16 // y component of the kerning vector.
}

type composite struct {
	name  string
	parts []part
}

type part struct {
	name string
	x, y fixed.Int16_16 // (x,y) displacement from the origin.
}
