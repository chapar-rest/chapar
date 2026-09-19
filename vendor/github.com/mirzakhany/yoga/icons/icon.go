package icons

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math"
)

// BakePx is the device-pixel size icons are rasterized at during generation
// (20 logical px at 2× scale).
const BakePx = 40

// SourceVersion is the Lucide release baked into the generated catalog.
const SourceVersion = "1.33.0"

// Icon is a pre-baked Lucide glyph. Zero value means no icon.
type Icon struct {
	Name string
	pix  []byte // gzip-compressed BakePx×BakePx alpha mask
}

// Empty reports whether the icon carries no drawable data.
func (i Icon) Empty() bool {
	return i.Name == "" && len(i.pix) == 0
}

func newIcon(name string, pix []byte) Icon {
	return Icon{Name: name, pix: pix}
}

// Alpha returns the baked coverage mask, scaled to dstPx if needed.
func (i Icon) Alpha(dstPx int) (*AlphaMask, error) {
	if i.Empty() {
		return nil, fmt.Errorf("icons: empty icon")
	}
	raw, err := gunzip(i.pix)
	if err != nil {
		return nil, fmt.Errorf("icons: %s: %w", i.Name, err)
	}
	if len(raw) != BakePx*BakePx {
		return nil, fmt.Errorf("icons: %s: want %d bytes, got %d", i.Name, BakePx*BakePx, len(raw))
	}
	if dstPx <= 0 || dstPx == BakePx {
		return &AlphaMask{W: BakePx, H: BakePx, Pix: raw}, nil
	}
	return scaleAlpha(raw, BakePx, BakePx, dstPx, dstPx), nil
}

// AlphaMask is an 8-bit coverage bitmap.
type AlphaMask struct {
	W, H int
	Pix  []byte
}

func gunzip(b []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func scaleAlpha(src []byte, sw, sh, dw, dh int) *AlphaMask {
	if sw == dw && sh == dh {
		out := make([]byte, len(src))
		copy(out, src)
		return &AlphaMask{W: dw, H: dh, Pix: out}
	}
	out := make([]byte, dw*dh)
	if dw < sw || dh < sh {
		// Area average: each destination pixel is the coverage-weighted mean
		// of the source pixels it spans, so thin strokes stay smooth instead
		// of dropping or doubling as with nearest-neighbour picks.
		fx, fy := float64(sw)/float64(dw), float64(sh)/float64(dh)
		for y := 0; y < dh; y++ {
			y0, y1 := float64(y)*fy, float64(y+1)*fy
			for x := 0; x < dw; x++ {
				x0, x1 := float64(x)*fx, float64(x+1)*fx
				var sum, area float64
				for sy := int(y0); sy < sh && float64(sy) < y1; sy++ {
					wy := min(y1, float64(sy+1)) - max(y0, float64(sy))
					for sx := int(x0); sx < sw && float64(sx) < x1; sx++ {
						w := wy * (min(x1, float64(sx+1)) - max(x0, float64(sx)))
						sum += w * float64(src[sy*sw+sx])
						area += w
					}
				}
				if area > 0 {
					out[y*dw+x] = uint8(sum/area + 0.5)
				}
			}
		}
		return &AlphaMask{W: dw, H: dh, Pix: out}
	}
	// Upscale: bilinear between source pixel centers.
	at := func(x, y int) float64 {
		x = min(max(x, 0), sw-1)
		y = min(max(y, 0), sh-1)
		return float64(src[y*sw+x])
	}
	for y := 0; y < dh; y++ {
		syf := (float64(y)+0.5)*float64(sh)/float64(dh) - 0.5
		sy := int(math.Floor(syf))
		ty := syf - float64(sy)
		for x := 0; x < dw; x++ {
			sxf := (float64(x)+0.5)*float64(sw)/float64(dw) - 0.5
			sx := int(math.Floor(sxf))
			tx := sxf - float64(sx)
			top := at(sx, sy)*(1-tx) + at(sx+1, sy)*tx
			bot := at(sx, sy+1)*(1-tx) + at(sx+1, sy+1)*tx
			out[y*dw+x] = uint8(top*(1-ty) + bot*ty + 0.5)
		}
	}
	return &AlphaMask{W: dw, H: dh, Pix: out}
}
