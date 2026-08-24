// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tfm

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"math"

	"golang.org/x/image/font"
	xfix "golang.org/x/image/math/fixed"

	"star-tex.org/x/tex/font/fixed"
	"star-tex.org/x/tex/internal/iobuf"
)

type fontFamily string

const (
	fontTypeVanilla fontFamily = "TEX TEXT"
	fontTypeMathSym fontFamily = "TEX MATH SYMBOLS"
	fontTypeMathExt fontFamily = "TEX MATH EXTENSION"
)

// Font is a TeX Font metrics.
type Font struct {
	hdr  fileHeader
	body fileBody

	metSet  bool // metSet tells whether the font metrics have been computed.
	metrics font.Metrics
}

type fileHeader struct {
	lf uint16 // length of the entire file, in words
	lh uint16 // length of the header data, in words
	bc uint16 // smallest character code in the font
	ec uint16 // largest character code in the font
	nw uint16 // number of words in the width table
	nh uint16 // number of words in the height table
	nd uint16 // number of words in the depth table
	ni uint16 // number of words in the italic correction table
	nl uint16 // number of words in the lig/kern table
	nk uint16 // number of words in the kern table
	ne uint16 // number of words in the extensible character table
	np uint16 // number of font parameter words
}

// Parse parses a TFM file.
func Parse(r io.Reader) (Font, error) {
	var (
		fnt     Font
		rr, err = newReader(r)
	)
	if err != nil {
		return fnt, fmt.Errorf("could not read TFM file: %w", err)
	}

	err = fnt.readHeader(rr)
	if err != nil {
		return fnt, fmt.Errorf("could not parse TFM file header: %w", err)
	}

	err = fnt.readBody(rr)
	if err != nil {
		return fnt, fmt.Errorf("could not parse TFM file body: %w", err)
	}

	return fnt, nil
}

func (fnt *Font) DesignSize() fixed.Int12_20 {
	return fnt.body.header.designSize
}

func (fnt *Font) CodingScheme() string {
	return fnt.body.header.codingScheme
}

func (fnt *Font) Name() string {
	return fnt.body.header.fontID
}

// Checksum returns the checksum of the TeX font metrics file.
func (fnt *Font) Checksum() uint32 {
	return fnt.body.header.chksum
}

// NumGlyphs returns the number of glyphs in this font.
func (fnt *Font) NumGlyphs() int {
	return int(fnt.hdr.ec) + 1 - int(fnt.hdr.bc)
}

// index returns the glyphIndex for the given rune.
//
// index returns -1 if there is no such rune.
func (fnt *Font) index(r rune) glyphIndex {
	i := int(r)
	if !(int(fnt.hdr.bc) <= i && i <= int(fnt.hdr.ec)) {
		panic(fmt.Errorf("glyph out of range"))
	}
	i -= int(fnt.hdr.bc)
	return glyphIndex(i)
}

func (fnt *Font) glyph(x glyphIndex) glyphInfo {
	return fnt.body.glyphs[x]
}

// Box returns the width, height and depth of r's glyph.
//
// It returns !ok if the face does not contain a glyph for r.
func (fnt *Font) Box(r rune) (w, h, d fixed.Int12_20, ok bool) {
	i := int(r)
	if !(int(fnt.hdr.bc) <= i && i <= int(fnt.hdr.ec)) {
		return 0, 0, 0, false
	}
	i -= int(fnt.hdr.bc)
	g := fnt.body.glyphs[i]
	w = fnt.body.width[g.wd()] // FIXME(sbinet): apply italic correction?
	h = fnt.body.height[g.ht()]
	d = fnt.body.depth[g.dp()]
	return w, h, d, true
}

// GlyphAdvance returns the advance width of r's glyph.
//
// It returns !ok if the face does not contain a glyph for r.
func (fnt *Font) GlyphAdvance(r rune) (xfix.Int26_6, bool) {
	i := int(r)
	if !(int(fnt.hdr.bc) <= i && i <= int(fnt.hdr.ec)) {
		return 0, false
	}
	i -= int(fnt.hdr.bc)
	var (
		g  = fnt.body.glyphs[i]
		ic = fnt.body.italic[g.ic()]
		w  = fnt.body.width[g.wd()] + ic
	)
	return w.ToInt26_6(), true
}

// GlyphBounds returns the bounding box of r's glyph, drawn at a dot equal
// to the origin, and that glyph's advance width.
//
// It returns !ok if the face does not contain a glyph for r.
//
// The glyph's ascent and descent equal -bounds.Min.Y and +bounds.Max.Y. A
// visual depiction of what these metrics are is at
// https://developer.apple.com/library/mac/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png
func (fnt *Font) GlyphBounds(r rune) (bounds xfix.Rectangle26_6, advance xfix.Int26_6, ok bool) {
	i := fnt.index(r)
	if i < 0 {
		return
	}
	ok = true
	g := fnt.body.glyphs[i]
	var (
		ic = fnt.body.italic[g.ic()]
		w  = fnt.body.width[g.wd()] + ic
		h  = fnt.body.height[g.ht()]
		d  = fnt.body.depth[g.dp()]
	)

	bounds = xfix.Rectangle26_6{
		Min: xfix.Point26_6{
			X: 0, // bearing?
			Y: -h.ToInt26_6(),
		},
		Max: xfix.Point26_6{
			X: w.ToInt26_6(),
			Y: d.ToInt26_6(),
		},
	}
	advance = w.ToInt26_6()

	return
}

// Kern returns the horizontal adjustment for the kerning pair (r0, r1). A
// positive kern means to move the glyphs further apart.
func (fnt *Font) Kern(r0, r1 rune) xfix.Int26_6 {
	i0 := fnt.index(r0)
	if i0 < 0 {
		return 0
	}
	i1 := fnt.index(r1)
	if i1 < 0 {
		return 0
	}

	g0 := fnt.glyph(i0)
	c1 := int(r1)
	if g0.raw[2]&3 == 1 {
		ii := int(g0.raw[3])
		rr := fnt.body.ligKern[ii]
		if rr.raw[0] > 128 {
			ii = int(rr.raw[2])*256 + int(rr.raw[3])
		}
		for {
			lk := fnt.body.ligKern[ii]
			switch {
			case lk.raw[2] >= 128:
				if int(lk.raw[1]) == c1 {
					rr := lk.nextIndex()
					kern := fnt.body.kern[rr]
					return fixed.Int12_20(kern).ToInt26_6()
				}
			default:
				// ok.
			}

			switch {
			case lk.raw[0] >= 128:
				ii = len(fnt.body.ligKern)
			default:
				ii += int(lk.raw[0]) + 1
			}

			if ii >= len(fnt.body.ligKern) {
				return 0
			}
		}
	}

	// FIXME(sbinet): implement it.
	return 0
}

// Metrics returns the metrics for this Face.
func (fnt *Font) Metrics() font.Metrics {
	if !fnt.metSet {
		fnt.metSet = true
		fnt.computeMetrics()
	}
	return fnt.metrics
}

func (fnt *Font) computeMetrics() {
	slant := fnt.body.param[0].Float64()
	slope := slopeFrom(slant)

	var (
		caph fixed.Int12_20
		asc  fixed.Int12_20
		desc fixed.Int12_20
	)
	for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		idx := fnt.index(r)
		if idx < 0 {
			continue
		}
		g := fnt.glyph(idx)
		h := fnt.body.height[g.ht()]
		if h >= caph {
			caph = h
		}
	}

	// FIXME(sbinet): ascender and descender do not seem to be properly inferred
	// when using these heuristics.
	//	for _, r := range "abcdefghijklmnopqrstuvwxyz" {
	//		idx := fnt.index(r)
	//		if idx < 0 {
	//			continue
	//		}
	//		g := fnt.glyph(idx)
	//		h := fnt.body.height[g.ht()]
	//		d := fnt.body.depth[g.dp()]
	//		if h > asc {
	//			asc = h
	//		}
	//		if d > desc {
	//			desc = d
	//		}
	//	}

	fnt.metrics = font.Metrics{
		Ascent:     asc.ToInt26_6(),
		Descent:    desc.ToInt26_6(),
		XHeight:    fnt.body.param[4].ToInt26_6(),
		CapHeight:  caph.ToInt26_6(),
		CaretSlope: slope,
	}
}

func slopeFrom(slant float64) image.Point {
	if slant == 0 {
		return image.Pt(0, 1)
	}
	const epsilon = 1e-6
	var (
		v = math.Abs(slant)
		f = 1.0
	)
	for i := 0; i < 10; i++ {
		f = math.Pow10(i)
		r := math.Trunc(v * f)
		if math.Abs(r-v*f) < epsilon {
			break
		}
	}

	return image.Pt(int(f*slant), int(f))
}

func (fnt *Font) readHeader(r *iobuf.Reader) error {
	hdr := &fnt.hdr
	err := readHeader(r, hdr)
	if err != nil {
		return fmt.Errorf("could not parse TFM file header: %w", err)
	}

	if !(int32(hdr.bc)-1 <= int32(hdr.ec) && hdr.ec <= 255) {
		return fmt.Errorf("invalid TFM file header glyph code range")
	}

	if !(hdr.ne <= 256) {
		return fmt.Errorf("invalid TFM file header extensible character table")
	}

	sum := 6 + hdr.lh + (hdr.ec - hdr.bc + 1) + hdr.nw + hdr.nh + hdr.nd + hdr.ni + hdr.nl + hdr.nk + hdr.ne + hdr.np
	if hdr.lf != sum {
		return fmt.Errorf("invalid TFM file length")
	}

	return nil
}

type fileBody struct {
	header  header
	glyphs  []glyphInfo
	width   []fixed.Int12_20
	height  []fixed.Int12_20
	depth   []fixed.Int12_20
	italic  []fixed.Int12_20
	ligKern []ligKernCmd
	kern    []fixed.Int12_20
	exten   []extensible
	param   []fixed.Int12_20
}

type header struct {
	chksum       uint32
	designSize   fixed.Int12_20
	codingScheme string
	fontID       string
	sevenBitSafe bool
	face         byte

	extra []fixed.Int12_20
}

func (fnt *Font) readBody(r *iobuf.Reader) error {
	fnt.body.header.chksum = r.ReadU32()
	fnt.body.header.designSize = fixed.Int12_20(r.ReadU32())
	if fnt.hdr.lh > 2 {
		fnt.body.header.codingScheme = readStr(r, 40)
		fnt.body.header.fontID = readStr(r, 20)
		fnt.body.header.sevenBitSafe = r.ReadU8() == 0b1000_0000
		_ = r.ReadU16() // unused
		fnt.body.header.face = r.ReadU8()
	}
	if lh := int(fnt.hdr.lh); lh > 18 {
		n := lh - 18
		fnt.body.header.extra = readFWs(r, n)
	}

	fnt.body.glyphs = readCharInfos(r, int(fnt.hdr.ec-fnt.hdr.bc+1))
	fnt.body.width = readFWs(r, int(fnt.hdr.nw))
	fnt.body.height = readFWs(r, int(fnt.hdr.nh))
	fnt.body.depth = readFWs(r, int(fnt.hdr.nd))
	fnt.body.italic = readFWs(r, int(fnt.hdr.ni))
	fnt.body.ligKern = readLigKerns(r, int(fnt.hdr.nl))
	fnt.body.kern = readFWs(r, int(fnt.hdr.nk))
	fnt.body.exten = readExtens(r, int(fnt.hdr.ne))
	fnt.body.param = readFWs(r, int(fnt.hdr.np))

	return nil
}

func (fnt *Font) MarshalText() ([]byte, error) {
	var (
		buf = new(bytes.Buffer)
		err = newTextEncoder(buf).encode(fnt)
	)
	return buf.Bytes(), err
}
