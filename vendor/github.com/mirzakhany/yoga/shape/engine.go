package shape

import (
	"github.com/mirzakhany/yoga/render"
)

// Engine bundles fonts, atlas, shaping, and line cache for UI text.
type Engine struct {
	Fonts  *FontSystem
	Atlas  *render.FontAtlas
	Shaper *Shaper
	Cache  *LineCache
}

// NewEngine creates a text engine at the given device scale.
func NewEngine(scale float32, useSystemFonts bool) (*Engine, error) {
	fonts, err := NewFontSystem(scale, useSystemFonts)
	if err != nil {
		return nil, err
	}
	atlas := render.NewAtlasScale(scale)
	shaper := NewShaper(fonts)
	return &Engine{
		Fonts:  fonts,
		Atlas:  atlas,
		Shaper: shaper,
		Cache:  NewLineCache(shaper),
	}, nil
}

// SetFont reconfigures the UI and editor faces, sizes, letter spacing, line
// height, and tab width, then invalidates the shaped-line cache so subsequent
// text uses the new configuration.
func (e *Engine) SetFont(cfg FontConfig) error {
	if err := e.Fonts.SetFont(cfg); err != nil {
		return err
	}
	e.Cache.Invalidate()
	return nil
}

// FontGen returns a counter bumped on each SetFont; consumers compare it to
// detect when font-derived state must be refreshed.
func (e *Engine) FontGen() uint64 { return e.Fonts.FontGen() }

// TabWidth returns the configured tab width in columns.
func (e *Engine) TabWidth() int { return e.Fonts.TabCols() }

// Metrics returns UI line metrics in logical pixels.
func (e *Engine) Metrics() Metrics { return e.Fonts.Metrics() }

// MetricsMono returns editor mono line metrics in logical pixels.
func (e *Engine) MetricsMono() Metrics { return e.Fonts.MonoMetrics() }

// Line shapes and caches a single UI line.
func (e *Engine) Line(text string) Line { return e.Cache.Get(text) }

// LineMono shapes and caches a single editor mono line.
func (e *Engine) LineMono(text string) Line { return e.Cache.GetMono(text) }

// Measure returns width and height for a single-line UI string.
func (e *Engine) Measure(s string) (w, h float32) {
	return e.Shaper.Measure(s)
}

// MeasureMono returns width and height for a single-line editor mono string.
func (e *Engine) MeasureMono(s string) (w, h float32) {
	return e.Shaper.MeasureMono(s)
}

// LineAt returns a shaped line at logicalSize (0 = default UI size).
func (e *Engine) LineAt(text string, logicalSize render.Px) Line {
	return e.Cache.GetAt(text, logicalSize)
}

// LineAtWeight returns a shaped UI line at logicalSize and CSS-like weight.
func (e *Engine) LineAtWeight(text string, logicalSize render.Px, weight int) Line {
	return e.Cache.GetAtWeight(text, logicalSize, weight)
}

// MeasureAt returns width and height for a single-line string at logicalSize.
func (e *Engine) MeasureAt(s string, logicalSize render.Px) (w, h render.Px) {
	return e.Shaper.MeasureAt(s, logicalSize)
}

// MeasureAtWeight returns width and height at logicalSize for the given weight.
func (e *Engine) MeasureAtWeight(s string, logicalSize render.Px, weight int) (w, h render.Px) {
	return e.Shaper.MeasureAtWeight(s, logicalSize, weight)
}

// DrawStringTop draws UI text with top-left y (convenience for UI chrome).
func (e *Engine) DrawStringTop(dl *render.DrawList, s string, x, topY float32, c render.Color) float32 {
	return e.DrawString(dl, s, x, topY+e.Metrics().Ascent, c)
}

// DrawStringTopMono draws editor mono text with top-left y.
func (e *Engine) DrawStringTopMono(dl *render.DrawList, s string, x, topY float32, c render.Color) float32 {
	m := e.MetricsMono()
	return e.DrawStringMono(dl, s, x, topY+m.Ascent, c)
}

// DrawStringTopAt draws at logicalSize with top-left y (Regular weight).
func (e *Engine) DrawStringTopAt(dl *render.DrawList, s string, x, topY float32, c render.Color, logicalSize render.Px) float32 {
	return e.DrawStringTopAtWeight(dl, s, x, topY, c, logicalSize, WeightRegular)
}

// DrawStringTopAtWeight draws at logicalSize and weight with top-left y.
func (e *Engine) DrawStringTopAtWeight(dl *render.DrawList, s string, x, topY float32, c render.Color, logicalSize render.Px, weight int) float32 {
	m := e.Fonts.MetricsAt(logicalSize)
	return e.DrawStringAtWeight(dl, s, x, topY+m.Ascent, c, logicalSize, weight)
}

// DrawStringAt draws a single line at baseline y and logicalSize (Regular).
func (e *Engine) DrawStringAt(dl *render.DrawList, s string, x, baselineY float32, c render.Color, logicalSize render.Px) float32 {
	return e.DrawStringAtWeight(dl, s, x, baselineY, c, logicalSize, WeightRegular)
}

// DrawStringAtWeight draws a single UI line at baseline y, size, and weight.
func (e *Engine) DrawStringAtWeight(dl *render.DrawList, s string, x, baselineY float32, c render.Color, logicalSize render.Px, weight int) float32 {
	ln := e.LineAtWeight(s, logicalSize, weight)
	topY := baselineY - e.Fonts.MetricsAt(logicalSize).Ascent
	e.drawLineGlyphsTint(dl, ln, x, topY, func(int) render.Color { return c })
	return ln.Width
}

// DrawString draws a single UI line at baseline y.
func (e *Engine) DrawString(dl *render.DrawList, s string, x, baselineY float32, c render.Color) float32 {
	ln := e.Line(s)
	topY := baselineY - e.Metrics().Ascent
	e.drawLineGlyphsTint(dl, ln, x, topY, func(int) render.Color { return c })
	return ln.Width
}

// DrawStringMono draws a single editor mono line at baseline y.
func (e *Engine) DrawStringMono(dl *render.DrawList, s string, x, baselineY float32, c render.Color) float32 {
	ln := e.LineMono(s)
	topY := baselineY - e.MetricsMono().Ascent
	e.drawLineGlyphsTint(dl, ln, x, topY, func(int) render.Color { return c })
	return ln.Width
}

// DrawLineGlyphs draws an already-shaped line with per-glyph colors from tintAt byte offset.
func (e *Engine) DrawLineGlyphs(dl *render.DrawList, ln Line, x, topY float32, tint func(byteOff int) render.Color) {
	e.drawLineGlyphsTint(dl, ln, x, topY, tint)
}

// drawLineGlyphsTint places atlas quads so ink aligns with HarfBuzz bearings:
// ink-left = pen + BearingX, then subtract atlas Pad so the padded bitmap
// does not shift glyphs right/down.
func (e *Engine) drawLineGlyphsTint(dl *render.DrawList, ln Line, x, topY float32, tint func(byteOff int) render.Color) {
	ppem := e.glyphPpem(ln)
	for _, g := range ln.Glyphs {
		face := e.Fonts.Face(g.FaceID)
		entry := e.Atlas.EnsureGlyph(g.FaceID, face, g.GID, ppem)
		dst := render.Rect{
			X: x + g.X + g.BearingX - entry.Pad,
			Y: topY + g.Y - entry.Pad,
			W: entry.W,
			H: entry.H,
		}
		col := tint(g.ClusterByte)
		if entry.Color {
			dl.AddGlyphQuad(dst, entry.UV, render.PageColor, col)
		} else {
			dl.AddGlyphQuad(dst, entry.UV, render.PageMono, col)
		}
	}
}

// glyphPpem returns the device-pixel size used to bake glyphs for ln.
func (e *Engine) glyphPpem(ln Line) uint16 {
	if ln.LogicalSize > 0 {
		return uint16(e.Fonts.ppem(ln.LogicalSize).Round())
	}
	if ln.Mono {
		return uint16(e.Fonts.monoPixelSize.Round())
	}
	return uint16(e.Fonts.uiPixelSize.Round())
}

// FlushAtlas uploads dirty atlas regions to the GPU renderer when present.
func (e *Engine) FlushAtlas(r *render.Renderer) error {
	if r == nil {
		return nil
	}
	// A full rebuild re-uploads both pages, so skip (and drop) any pending
	// per-region uploads; otherwise the stale dirty rects would be re-uploaded
	// on every subsequent frame and the dirty list would grow without bound.
	if e.Atlas.NeedsFullRebuild() {
		defer e.Atlas.ClearDirty()
		return r.UpdateAtlas(e.Atlas)
	}
	for _, d := range e.Atlas.DirtyRects() {
		if err := r.UpdateAtlasRegion(d); err != nil {
			return err
		}
	}
	e.Atlas.ClearDirty()
	return nil
}
