//go:build nogpu

package render

// Renderer is a no-op stub for headless builds.
type Renderer struct {
	ClearColor Color
}

func NewRenderer(_ any, _, _, _, _ int, _ *FontAtlas) (*Renderer, error) {
	return &Renderer{ClearColor: RGBA8(24, 24, 29, 255)}, nil
}

func (r *Renderer) Resize(_, _, _, _ int) {}
func (r *Renderer) Render(_ *DrawList) error { return nil }
func (r *Renderer) UpdateAtlas(_ *FontAtlas) error { return nil }
func (r *Renderer) UpdateAtlasRegion(_ DirtyRect) error { return nil }
func (r *Renderer) Destroy() {}
