package render

// SpriteSheet maps named sprites to normalized UV regions inside a texture
// atlas. It is the generic mechanism behind the Icon component: a sprite sheet
// is just a grid (or free-form set of regions) packed into the same texture the
// glyphs live in, so icons cost nothing extra at draw time.
//
// Here it is backed by the FontAtlas's baked icon row, but the type is
// deliberately independent of how the atlas was produced: point it at any
// texture-coordinate provider and it will emit the right quads.
type SpriteSheet struct {
	atlas *FontAtlas
}

// NewSpriteSheet returns a sheet that resolves icon names against the atlas.
func NewSpriteSheet(atlas *FontAtlas) *SpriteSheet {
	return &SpriteSheet{atlas: atlas}
}

// Region returns the UV rectangle for a named sprite.
func (s *SpriteSheet) Region(name string) (Rect, bool) {
	return s.atlas.IconUV(name)
}

// Draw appends a textured quad that stretches the named sprite over dst, tinted
// by c. Because the source is an 8-bit coverage mask, the tint color fully
// controls the icon's appearance (same path as glyph rendering).
func (s *SpriteSheet) Draw(dl *DrawList, name string, dst Rect, c Color) bool {
	uv, ok := s.atlas.IconUV(name)
	if !ok {
		return false
	}
	dl.AddTexQuad(dst, uv, c)
	return true
}
