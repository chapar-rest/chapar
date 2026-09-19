package render

import (
	"bytes"
	"fmt"
	"image"
	"math"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

// CSS pixels per canvas millimetre (96 dpi), matching tdewolff/canvas's SVG parser.
const svgCSSPxPerMM = 96.0 / 25.4

// maxSVGRaster is the longest side in device pixels we will rasterize.
const maxSVGRaster = 1024

// SVGDoc is a parsed SVG ready to rasterize at any pixel size.
type SVGDoc struct {
	c          *canvas.Canvas
	cssW, cssH float32
}

// ParseSVGDoc parses svg, substituting currentColor with tint (or black if tint is zero-alpha).
func ParseSVGDoc(svg []byte, tint Color) (doc *SVGDoc, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("parse svg panic: %v", r)
			doc = nil
		}
	}()
	data := rewriteSVGColorTo(svg, colorHex(tint))
	c, perr := canvas.ParseSVG(bytes.NewReader(data))
	if perr != nil {
		return nil, perr
	}
	if c == nil {
		return nil, fmt.Errorf("parse svg: empty document")
	}
	w, h := float32(c.W*svgCSSPxPerMM), float32(c.H*svgCSSPxPerMM)
	if w <= 0 || h <= 0 {
		w, h = 24, 24
	}
	return &SVGDoc{c: c, cssW: w, cssH: h}, nil
}

// Size returns the SVG's intrinsic size in CSS pixels.
func (d *SVGDoc) Size() (w, h float32) {
	if d == nil {
		return 0, 0
	}
	return d.cssW, d.cssH
}

// Rasterize draws the SVG into an RGBA bitmap of approximately pxW×pxH.
// The result keeps the document aspect; pxH is used only to pick resolution
// when pxW is zero.
func (d *SVGDoc) Rasterize(pxW, pxH int) (img *image.RGBA, err error) {
	if d == nil || d.c == nil {
		return nil, fmt.Errorf("nil svg")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("rasterize panic: %v", r)
		}
	}()
	pxW, pxH = clampSVGRaster(pxW, pxH, d.cssW, d.cssH)
	if d.c.W <= 0 || d.c.H <= 0 {
		return image.NewRGBA(image.Rect(0, 0, pxW, pxH)), nil
	}
	dpm := canvas.DPMM(float64(pxW) / d.c.W)
	img = rasterizer.Draw(d.c, dpm, canvas.DefaultColorSpace)
	return img, err
}

// RasterizeSVGRGBA parses and rasterizes svg in one shot.
func RasterizeSVGRGBA(svg []byte, pxW, pxH int, tint Color) (*image.RGBA, error) {
	doc, err := ParseSVGDoc(svg, tint)
	if err != nil {
		return nil, err
	}
	return doc.Rasterize(pxW, pxH)
}

func clampSVGRaster(pxW, pxH int, cssW, cssH float32) (int, int) {
	if pxW < 1 && pxH < 1 {
		pxW = int(cssW + 0.5)
		pxH = int(cssH + 0.5)
	} else if pxW < 1 {
		if cssH > 0 {
			pxW = int(float32(pxH)*cssW/cssH + 0.5)
		} else {
			pxW = pxH
		}
	} else if pxH < 1 {
		if cssW > 0 {
			pxH = int(float32(pxW)*cssH/cssW + 0.5)
		} else {
			pxH = pxW
		}
	}
	if pxW < 1 {
		pxW = 1
	}
	if pxH < 1 {
		pxH = 1
	}
	long := pxW
	if pxH > long {
		long = pxH
	}
	if long > maxSVGRaster {
		s := float64(maxSVGRaster) / float64(long)
		pxW = int(math.Max(1, math.Round(float64(pxW)*s)))
		pxH = int(math.Max(1, math.Round(float64(pxH)*s)))
	}
	return pxW, pxH
}

func colorHex(c Color) string {
	if c.A <= 0 && c.R == 0 && c.G == 0 && c.B == 0 {
		return "#000000"
	}
	r := hexByte(c.R)
	g := hexByte(c.G)
	b := hexByte(c.B)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func hexByte(v float32) byte {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 255
	}
	return byte(v*255 + 0.5)
}

// Scale returns the atlas device-pixel scale (1 = 96 dpi).
func (a *FontAtlas) Scale() float32 {
	if a == nil || a.scale < 1 {
		return 1
	}
	return a.scale
}
