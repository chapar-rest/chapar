// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package afm

import (
	"fmt"
	"image"
	"io"
	"math"

	"golang.org/x/image/font"
	"star-tex.org/x/tex/font/fixed"
)

type direction struct {
	// underlinePosition is the distance from the baseline for centering
	// underlining strokes.
	underlinePosition fixed.Int16_16

	// underlineThickness is the stroke width for underlining.
	underlineThickness fixed.Int16_16

	// italicAngle is the angle (in degrees counter-clockwise from the vertical)
	// of the dominant vertical stroke of the font.
	italicAngle fixed.Int16_16

	// charWidth is the width vector of this font's program characters.
	charWidth charWidth

	// isFixedPitch indicates whether the program is a fixed pitch (monospace) font.
	isFixedPitch bool
}

type charWidth struct {
	x fixed.Int16_16 // x component of the width vector of a font's program characters.
	y fixed.Int16_16 // y component of the width vector of a font's program characters.
}

// CharMetric represents an individual character's metric.
type CharMetric struct {
	// c is the decimal value of default character code.
	// c is -1 if the character is not encoded.
	c int

	// name is the PostScript name of this character.
	name string

	// w0 is the character width vector for writing direction 0.
	w0 charWidth

	// w1 is the character width vector for writing direction 1.
	w1 charWidth

	// vvector holds the components of a vector from origin 0 to origin 1.
	// origin 0 is the origin for writing direction 0.
	// origin 1 is the origin for writing direction 1.
	vv [2]fixed.Int16_16

	// bbox is the character bounding box.
	bbox bbox

	// ligs is a ligature sequence.
	ligs []lig
}

// Code returns the decimal value of this character, or -1 if this character
// is not encoded.
func (cm *CharMetric) Code() int { return cm.c }

// Name returns the PostScript name of this character.
func (cm *CharMetric) Name() string { return cm.name }

type bbox struct {
	llx, lly fixed.Int16_16
	urx, ury fixed.Int16_16
}

// lig is a ligature.
type lig struct {
	// succ is the name of the successor
	succ string
	// name is the name of the composite ligature, consisting
	// of the current character and the successor.
	name string
}

// Font is an Adobe Font metrics.
type Font struct {
	// metricsSets defines the writing direction.
	// 0: direction 0 only.
	// 1: direction 1 only.
	// 2: both directions.
	metricsSets int

	fontName   string // fontName is the name of the font program as presented to the PostScript language 'findfont' operator.
	fullName   string // fullName is the full text name of the font.
	familyName string // familyName is the name of the typeface family to which the font belongs.
	weight     string // weight is the weight of the font (ex: Regular, Bold, Light).
	bbox       bbox   // bbox is the font bounding box.
	version    string // version is the font program version identifier.
	notice     string // notice contains the font name trademark or copyright notice.

	// encodingScheme specifies the default encoding vector used for this font
	// program (ex: AdobeStandardEncoding, JIS12-88-CFEncoding, ...)
	// Special font program might state FontSpecific.
	encodingScheme string
	mappingScheme  int
	escChar        int
	characterSet   string // characterSet describes the character set (glyph complement) of this font program.
	characters     int    // characters describes the number of characters defined in this font program.
	isBaseFont     bool   // isBaseFont indicates whether this font is a base font program.

	// vvector holds the components of a vector from origin 0 to origin 1.
	// origin 0 is the origin for writing direction 0.
	// origin 1 is the origin for writing direction 1.
	// vvector is required when metricsSet is 2.
	vvector [2]fixed.Int16_16

	isFixedV  bool // isFixedV indicates whether vvector is the same for every character in this font.
	isCIDFont bool // isCIDFont indicates whether the font is a CID-keyed font.

	capHeight fixed.Int16_16 // capHeight is usually the y-value of the top of the capital 'H'.
	xHeight   fixed.Int16_16 // xHeight is typically the y-value of the top of the lowercase 'x'.
	ascender  fixed.Int16_16 // ascender is usually the y-value of the top of the lowercase 'd'.
	descender fixed.Int16_16 // descender is typically the y-value of the bottom of the lowercase 'p'.
	stdHW     fixed.Int16_16 // stdHW specifies the dominant width of horizontal stems.
	stdVW     fixed.Int16_16 // stdVW specifies the dominant width of vertical stems.

	//	blendAxisTypes       []string
	//	blendDesignPositions [][]fixed.Int16_16
	//	blendDesignMap       [][][]fixed.Int16_16
	//	weightVector         []fixed.Int16_16

	direction   [3]direction
	charMetrics []CharMetric
	composites  []composite

	tkerns []trackKern
	pkerns []kernPair
}

// EncodingScheme returns the font encoding scheme.
// EncodingScheme returns "FontSpecific" for special font programs.
func (fnt *Font) EncodingScheme() string {
	return fnt.encodingScheme
}

func (fnt *Font) CharMetrics() []CharMetric {
	return fnt.charMetrics
}

func newFont() *Font {
	return &Font{
		isBaseFont: true,
	}
}

// Parse parses an AFM file.
func Parse(r io.Reader) (*Font, error) {
	var (
		fnt = newFont()
		p   = newParser(r)
	)
	err := p.parse(fnt)
	if err != nil {
		return fnt, fmt.Errorf("could not parse AFM file: %w", err)
	}
	return fnt, nil
}

// Metrics returns the metrics for this Face.
func (f *Font) Metrics() font.Metrics {
	met := font.Metrics{
		Height:     (f.bbox.ury - f.bbox.lly).ToInt26_6(),
		Ascent:     f.ascender.ToInt26_6(),
		Descent:    -f.descender.ToInt26_6(),
		XHeight:    f.xHeight.ToInt26_6(),
		CapHeight:  f.capHeight.ToInt26_6(),
		CaretSlope: slopeFrom(f.direction[0].italicAngle.Float64()), // FIXME(sbinet): check direction.
	}
	return met
}

func slopeFrom(angle float64) image.Point {
	if angle == 0 {
		return image.Pt(0, 1)
	}
	const (
		factor  = 1e6
		deg2rad = math.Pi / 180.0
	)
	radians := angle*deg2rad + 0.5*math.Pi
	y, x := math.Sincos(radians)

	return image.Pt(int(factor*x/y), factor)
}
