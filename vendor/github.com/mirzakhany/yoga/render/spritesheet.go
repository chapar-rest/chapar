package render

import "github.com/mirzakhany/yoga/icons"

// SpriteSheet maps named sprites to normalized UV regions inside a texture
// atlas. Icons are packed lazily on first draw via FontAtlas.EnsureIcon.
type SpriteSheet struct {
	atlas *FontAtlas
}

// NewSpriteSheet returns a sheet backed by the font atlas icon cache.
func NewSpriteSheet(atlas *FontAtlas) *SpriteSheet {
	return &SpriteSheet{atlas: atlas}
}

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
