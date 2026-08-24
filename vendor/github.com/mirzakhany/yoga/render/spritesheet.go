package render

import (
	"image"

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
	uv, ok := s.atlas.EnsureIcon(icon)
	if !ok {
		return false
	}
	dl.AddTexQuad(dst, uv, c)
	return true
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
