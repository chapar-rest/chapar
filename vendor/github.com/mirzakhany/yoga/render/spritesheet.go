package render

import (
	"image"
	"math"

	"github.com/mirzakhany/yoga/icons"
)

// SpriteSheet maps named sprites to normalized UV regions inside a texture
// atlas. Icons are packed lazily on first draw via FontAtlas.EnsureIcon.
type SpriteSheet struct {
	atlas *FontAtlas
}

// NewSpriteSheet returns a sheet backed by the font atlas icon cache.
func NewSpriteSheet(atlas *FontAtlas) *SpriteSheet {
	return &SpriteSheet{atlas: atlas}
}

// Atlas returns the backing font/image atlas.
func (s *SpriteSheet) Atlas() *FontAtlas { return s.atlas }

// Region returns the UV rectangle for an icon already packed in the atlas.
func (s *SpriteSheet) Region(icon icons.Icon) (Rect, bool) {
	if icon.Empty() {
		return Rect{}, false
	}
	return s.atlas.IconUV(icon.Name)
}

// Draw appends a textured quad that stretches the icon over dst, tinted by c.
func (s *SpriteSheet) Draw(dl *DrawList, icon icons.Icon, dst Rect, c Color) bool {
	if icon.Empty() {
		return false
	}
	// Bake at the drawn device size and centre it on whole pixels so the
	// mask maps 1:1 onto the screen instead of being resampled.
	sc := s.atlas.Scale()
	px := int(math.Round(float64(min(dst.W, dst.H) * sc)))
	if px < 1 {
		return false
	}
	uv, ok := s.atlas.EnsureIconPx(icon, px)
	if !ok {
		return false
	}
	size := float32(px) / sc
	dl.AddTexQuad(Rect{
		X: snapPx(dst.X+(dst.W-size)/2, sc),
		Y: snapPx(dst.Y+(dst.H-size)/2, sc),
		W: size, H: size,
	}, uv, c)
	return true
}

// snapPx rounds a logical coordinate to the nearest device pixel.
func snapPx(v, scale float32) float32 {
	return float32(math.Round(float64(v*scale))) / scale
}

// DrawImage appends a color-atlas quad for src packed under key.
func (s *SpriteSheet) DrawImage(dl *DrawList, key string, src *image.RGBA, dst Rect) bool {
	if s == nil || s.atlas == nil || src == nil {
		return false
	}
	entry, ok := s.atlas.EnsureImage(key, src)
	if !ok {
		return false
	}
	dl.AddGlyphQuad(dst, entry.UV, PageColor, Color{R: 1, G: 1, B: 1, A: 1})
	return true
}

// DrawImageEntryPixelAligned draws a packed image at its baked pixel size,
// centred in dst and snapped to whole device pixels. Use it for bitmaps
// rasterized for their on-screen size (SVGs) so they are not resampled.
func (s *SpriteSheet) DrawImageEntryPixelAligned(dl *DrawList, key string, dst Rect) bool {
	if s == nil || s.atlas == nil {
		return false
	}
	entry, ok := s.atlas.ImageUV(key)
	if !ok {
		return false
	}
	sc := s.atlas.Scale()
	w, h := float32(entry.physW)/sc, float32(entry.physH)/sc
	dl.AddGlyphQuad(Rect{
		X: snapPx(dst.X+(dst.W-w)/2, sc),
		Y: snapPx(dst.Y+(dst.H-h)/2, sc),
		W: w, H: h,
	}, entry.UV, PageColor, Color{R: 1, G: 1, B: 1, A: 1})
	return true
}

// DrawImageEntry draws a previously packed image by atlas key only.
func (s *SpriteSheet) DrawImageEntry(dl *DrawList, key string, dst Rect) bool {
	if s == nil || s.atlas == nil {
		return false
	}
	entry, ok := s.atlas.ImageUV(key)
	if !ok {
		return false
	}
	dl.AddGlyphQuad(dst, entry.UV, PageColor, Color{R: 1, G: 1, B: 1, A: 1})
	return true
}
