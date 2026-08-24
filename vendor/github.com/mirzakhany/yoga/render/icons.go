package render

import (
	"bytes"
	"fmt"
	"image"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

// iconBakePx matches icons.BakePx — pre-rasterized Lucide icons are baked at this size.
const iconBakePx = 40

// svgOverrides maps icon names to raw SVG bytes registered at runtime via RegisterIcon.
var svgOverrides = map[string][]byte{}

// RegisterIcon adds or replaces a named icon from raw SVG bytes. The SVG is
// rasterized on first draw. Returns an Icon value for use with SpriteSheet.Draw.
func RegisterIcon(name string, svg []byte) {
	svgOverrides[name] = append([]byte(nil), svg...)
}

// HasSVGOverride reports whether name was registered via RegisterIcon.
func HasSVGOverride(name string) bool {
	_, ok := svgOverrides[name]
	return ok
}

// RasterizeSVG renders an SVG into a px-by-px 8-bit coverage mask. Stroke and
// fill SVGs (including Lucide) are supported via tdewolff/canvas.
func RasterizeSVG(svg []byte, px int) (mask *image.Alpha, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("rasterize panic: %v", r)
		}
	}()
	data := rewriteSVGColorTo(svg, "#000000")
	c, err := canvas.ParseSVG(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if c.W <= 0 || c.H <= 0 {
		return image.NewAlpha(image.Rect(0, 0, px, px)), nil
	}
	dpm := canvas.DPMM(float64(px) / c.W)
	img := rasterizer.Draw(c, dpm, canvas.DefaultColorSpace)
	b := img.Bounds()
	out := image.NewAlpha(image.Rect(0, 0, px, px))
	for y := 0; y < px; y++ {
		for x := 0; x < px; x++ {
			sx := b.Min.X + x*b.Dx()/px
			sy := b.Min.Y + y*b.Dy()/px
			if sx >= b.Max.X {
				sx = b.Max.X - 1
			}
			if sy >= b.Max.Y {
				sy = b.Max.Y - 1
			}
			_, _, _, a := img.At(sx, sy).RGBA()
			out.Pix[y*px+x] = uint8(a >> 8)
		}
	}
	return out, nil
}

func rewriteSVGColorTo(svg []byte, hex string) []byte {
	s := string(svg)
	s = strings.ReplaceAll(s, "currentColor", hex)
	s = strings.ReplaceAll(s, "currentcolor", hex)
	return []byte(s)
}

// rasterizeOverrideSVG rasterizes a runtime-registered SVG at the given px size.
func rasterizeOverrideSVG(name string, px int) (*image.Alpha, error) {
	data, ok := svgOverrides[name]
	if !ok {
		return nil, nil
	}
	return RasterizeSVG(data, px)
}
