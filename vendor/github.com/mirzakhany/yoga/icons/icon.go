package icons

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
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
	for y := 0; y < dh; y++ {
		sy := y * sh / dh
		if sy >= sh {
			sy = sh - 1
		}
		for x := 0; x < dw; x++ {
			sx := x * sw / dw
			if sx >= sw {
				sx = sw - 1
			}
			out[y*dw+x] = src[sy*sw+sx]
		}
	}
	return &AlphaMask{W: dw, H: dh, Pix: out}
}
