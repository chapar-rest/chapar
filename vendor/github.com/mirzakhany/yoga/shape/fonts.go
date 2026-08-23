package shape

import (
	"bytes"
	"log"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	"github.com/mirzakhany/yoga/render"
)

const logicalFontPx = 14.0

// Font weights used by UI chrome. Values >= WeightSemiBold select Inter SemiBold.
const (
	WeightRegular  = 400
	WeightSemiBold = 600
)

// FontSystem resolves faces (UI primary + mono editor + system fallback) for shaping.
type FontSystem struct {
	fontMap       *fontscan.FontMap
	primary       *font.Face // UI sans-serif Regular
	primaryStrong *font.Face // UI sans-serif SemiBold (Weight >= 600)
	mono          *font.Face // editor monospace face
	scale         float32

	// Per-role shaping sizes, letter spacing, and line-height multipliers.
	uiPixelSize    fixed.Int26_6
	monoPixelSize  fixed.Int26_6
	uiSpacing      float32 // extra logical px per UI glyph
	monoSpacing    float32 // extra logical px per editor glyph
	uiLineFactor   float32 // multiplier of (ascent+descent); 0 = natural
	monoLineFactor float32
	tabCols        int
	fontGen        uint64 // bumped on every SetFont so consumers can refresh

	faceID  map[*font.Face]uint32
	idFace  map[uint32]*font.Face
	nextID  uint32
	segment shaping.Segmenter
	shaper  shaping.HarfbuzzShaper

	metrics          Metrics
	metricsCache     map[render.Px]Metrics
	monoMetrics      Metrics
	monoMetricsCache map[render.Px]Metrics
}

// Metrics describes line layout in logical pixels.
type Metrics struct {
	Ascent     render.Px
	Descent    render.Px
	LineHeight render.Px
}

// NewFontSystem loads Inter (UI) and JetBrains Mono (editor mono),
// optionally indexing system fonts for fallback.
func NewFontSystem(scale float32, useSystemFonts bool) (*FontSystem, error) {
	if scale < 1 {
		scale = 1
	}
	px := int(logicalFontPx * float64(scale) + 0.5)
	if px < 1 {
		px = 1
	}

	primary, err := font.ParseTTF(bytes.NewReader(render.InterTTF))
	if err != nil {
		return nil, err
	}
	primary.SetPpem(uint16(px), uint16(px))

	primaryStrong, err := font.ParseTTF(bytes.NewReader(render.InterSemiBoldTTF))
	if err != nil {
		return nil, err
	}
	primaryStrong.SetPpem(uint16(px), uint16(px))

	mono, err := font.ParseTTF(bytes.NewReader(render.JetBrainsMonoTTF))
	if err != nil {
		return nil, err
	}
	mono.SetPpem(uint16(px), uint16(px))

	fm := fontscan.NewFontMap(log.Default())
	uiMD := primary.Describe()
	fm.AddFace(primary, fontscan.Location{File: "Inter-Regular.ttf"}, uiMD)
	strongMD := primaryStrong.Describe()
	fm.AddFace(primaryStrong, fontscan.Location{File: "Inter-SemiBold.ttf"}, strongMD)
	monoMD := mono.Describe()
	fm.AddFace(mono, fontscan.Location{File: "JetBrainsMono-Regular.ttf"}, monoMD)
	fm.SetQuery(fontscan.Query{
		Families: []string{"Inter", "sans-serif", "monospace"},
		Aspect:   font.Aspect{},
	})
	if useSystemFonts {
		_ = fm.UseSystemFonts("") // degrade to embedded faces when unavailable
	}

	fs := &FontSystem{
		fontMap:          fm,
		primary:          primary,
		primaryStrong:    primaryStrong,
		mono:             mono,
		scale:            scale,
		uiPixelSize:      fixed.I(px),
		monoPixelSize:    fixed.I(px),
		tabCols:          defaultTabCols,
		faceID:           make(map[*font.Face]uint32),
		idFace:           make(map[uint32]*font.Face),
		metricsCache:     make(map[render.Px]Metrics),
		monoMetricsCache: make(map[render.Px]Metrics),
	}
	fs.registerFace(primary)
	fs.registerFace(primaryStrong)
	fs.registerFace(mono)
	fs.metrics = fs.computeMetrics(primary, fs.uiPixelSize, fs.uiLineFactor)
	fs.monoMetrics = fs.computeMetrics(mono, fs.monoPixelSize, fs.monoLineFactor)
	return fs, nil
}

const defaultTabCols = 4

// SetFont reconfigures both faces, sizes, spacing, line height, and tab width.
// Faces are reloaded fresh so the atlas re-bakes glyphs at the new size (sharp,
// not stretched). An explicit FaceConfig.File that fails to load returns an
// error and leaves the previous configuration unchanged.
func (fs *FontSystem) SetFont(cfg FontConfig) error {
	uiFace, err := fs.loadFace(cfg.UI, render.InterTTF)
	if err != nil {
		return err
	}
	strongFace, err := font.ParseTTF(bytes.NewReader(render.InterSemiBoldTTF))
	if err != nil {
		return err
	}
	monoFace, err := fs.loadFace(cfg.Mono, render.JetBrainsMonoTTF)
	if err != nil {
		return err
	}

	uiSize := cfg.UI.resolvedSize()
	monoSize := cfg.Mono.resolvedSize()
	uiPx := fs.ppem(uiSize)
	monoPx := fs.ppem(monoSize)
	uiFace.SetPpem(uint16(uiPx.Round()), uint16(uiPx.Round()))
	strongFace.SetPpem(uint16(uiPx.Round()), uint16(uiPx.Round()))
	monoFace.SetPpem(uint16(monoPx.Round()), uint16(monoPx.Round()))

	fs.primary = uiFace
	fs.primaryStrong = strongFace
	fs.mono = monoFace
	fs.registerFace(uiFace)
	fs.registerFace(strongFace)
	fs.registerFace(monoFace)
	fs.uiPixelSize = uiPx
	fs.monoPixelSize = monoPx
	fs.uiSpacing = cfg.UI.LetterSpacing
	fs.monoSpacing = cfg.Mono.LetterSpacing
	fs.uiLineFactor = cfg.UI.LineHeight
	fs.monoLineFactor = cfg.Mono.LineHeight
	if cfg.TabWidth > 0 {
		fs.tabCols = cfg.TabWidth
	} else {
		fs.tabCols = defaultTabCols
	}

	// Refresh the query so non-ASCII fallback prefers the chosen families.
	fs.fontMap.AddFace(uiFace, fontscan.Location{File: "ui"}, uiFace.Describe())
	fs.fontMap.AddFace(strongFace, fontscan.Location{File: "ui-strong"}, strongFace.Describe())
	fs.fontMap.AddFace(monoFace, fontscan.Location{File: "mono"}, monoFace.Describe())

	fs.metrics = fs.computeMetrics(uiFace, fs.uiPixelSize, fs.uiLineFactor)
	fs.monoMetrics = fs.computeMetrics(monoFace, fs.monoPixelSize, fs.monoLineFactor)
	fs.metricsCache = make(map[render.Px]Metrics)
	fs.monoMetricsCache = make(map[render.Px]Metrics)
	fs.fontGen++
	return nil
}

// ppem converts a logical size to a device-pixel shaping size.
func (fs *FontSystem) ppem(logicalSize float32) fixed.Int26_6 {
	px := int(logicalSize*fs.scale + 0.5)
	if px < 1 {
		px = 1
	}
	return fixed.I(px)
}

// FontGen returns a counter bumped on each SetFont; consumers compare it to
// detect when cached font-derived state must be refreshed.
func (fs *FontSystem) FontGen() uint64 { return fs.fontGen }

// TabCols returns the configured tab width in columns.
func (fs *FontSystem) TabCols() int { return fs.tabCols }

func (fs *FontSystem) registerFace(face *font.Face) uint32 {
	if id, ok := fs.faceID[face]; ok {
		return id
	}
	id := fs.nextID
	fs.nextID++
	fs.faceID[face] = id
	fs.idFace[id] = face
	return id
}

// FaceID returns a stable atlas key for face.
func (fs *FontSystem) FaceID(face *font.Face) uint32 { return fs.registerFace(face) }

// Face returns the face for id.
func (fs *FontSystem) Face(id uint32) *font.Face { return fs.idFace[id] }

// ResolveFace implements shaping.Fontmap.
func (fs *FontSystem) ResolveFace(r rune) *font.Face {
	f := fs.fontMap.ResolveFace(r)
	if f == nil {
		return fs.primary
	}
	fs.registerFace(f)
	return f
}

// SetScript implements shaping.FontmapScript.
func (fs *FontSystem) SetScript(s language.Script) { fs.fontMap.SetScript(s) }

// monoFontmap is a face resolver that prefers the editor monospace face for any
// rune it can render, falling back to the regular resolution chain (UI face +
// system fallback) only for runes the mono face lacks. The default fontMap query
// ranks the proportional UI face ahead of mono, so without this the segmenter
// would shape ASCII code in the UI face and break monospace alignment.
type monoFontmap struct{ fs *FontSystem }

// ResolveFace implements shaping.Fontmap.
func (m monoFontmap) ResolveFace(r rune) *font.Face {
	if mono := m.fs.mono; mono != nil {
		if _, has := mono.NominalGlyph(r); has {
			m.fs.registerFace(mono)
			return mono
		}
	}
	return m.fs.ResolveFace(r)
}

// SetScript implements shaping.FontmapScript.
func (m monoFontmap) SetScript(s language.Script) { m.fs.SetScript(s) }

// strongFontmap prefers Inter SemiBold for runes it can render, then falls back
// to the normal UI/system resolution chain.
type strongFontmap struct{ fs *FontSystem }

// ResolveFace implements shaping.Fontmap.
func (m strongFontmap) ResolveFace(r rune) *font.Face {
	if strong := m.fs.primaryStrong; strong != nil {
		if _, has := strong.NominalGlyph(r); has {
			m.fs.registerFace(strong)
			return strong
		}
	}
	return m.fs.ResolveFace(r)
}

// SetScript implements shaping.FontmapScript.
func (m strongFontmap) SetScript(s language.Script) { m.fs.SetScript(s) }

// Metrics returns UI line metrics in logical pixels at the default size.
func (fs *FontSystem) Metrics() Metrics { return fs.metrics }

// MonoMetrics returns editor mono line metrics at the default size.
func (fs *FontSystem) MonoMetrics() Metrics { return fs.monoMetrics }

// MetricsAt returns UI line metrics at logicalSize (logical px), shaped at
// that size's ppem rather than linearly scaling the default metrics.
func (fs *FontSystem) MetricsAt(logicalSize render.Px) Metrics {
	return fs.metricsAtSize(logicalSize, fs.primary, fs.metrics, fs.uiLineFactor, fs.uiPixelSize, fs.metricsCache)
}

// MonoMetricsAt returns editor mono line metrics at logicalSize.
func (fs *FontSystem) MonoMetricsAt(logicalSize render.Px) Metrics {
	return fs.metricsAtSize(logicalSize, fs.mono, fs.monoMetrics, fs.monoLineFactor, fs.monoPixelSize, fs.monoMetricsCache)
}

func (fs *FontSystem) metricsAtSize(logicalSize render.Px, face *font.Face, base Metrics, lineFactor float32, basePx fixed.Int26_6, cache map[render.Px]Metrics) Metrics {
	if logicalSize <= 0 {
		return base
	}
	baseLogical := float32(basePx.Round()) / fs.scale
	if abs32(logicalSize-baseLogical) < 0.01 {
		return base
	}
	if m, ok := cache[logicalSize]; ok {
		return m
	}
	m := fs.computeMetrics(face, fs.ppem(logicalSize), lineFactor)
	cache[logicalSize] = m
	return m
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// DefaultLogicalSize is the base UI font size in logical pixels.
func DefaultLogicalSize() render.Px { return render.Px(logicalFontPx) }

// Scale returns the device pixel scale.
func (fs *FontSystem) Scale() float32 { return fs.scale }

// PixelSize returns the UI shaping size in pixels.
func (fs *FontSystem) PixelSize() fixed.Int26_6 { return fs.uiPixelSize }

// Primary returns the UI sans-serif Regular face.
func (fs *FontSystem) Primary() *font.Face { return fs.primary }

// PrimaryStrong returns the UI sans-serif SemiBold face (Weight >= 600).
func (fs *FontSystem) PrimaryStrong() *font.Face { return fs.primaryStrong }

// Mono returns the editor monospace face.
func (fs *FontSystem) Mono() *font.Face { return fs.mono }

// uiFaceForWeight returns Regular or SemiBold for the given CSS-like weight.
func (fs *FontSystem) uiFaceForWeight(weight int) *font.Face {
	if weight >= WeightSemiBold && fs.primaryStrong != nil {
		return fs.primaryStrong
	}
	return fs.primary
}

func (fs *FontSystem) computeMetrics(face *font.Face, size fixed.Int26_6, lineFactor float32) Metrics {
	in := fs.baseInputFace([]rune("Ag"), face, size)
	out := fs.shaper.Shape(in)
	a := toLogical(fs, out.LineBounds.Ascent)
	d := -toLogical(fs, out.LineBounds.Descent)
	if d < 0 {
		d = -d
	}
	var lh float32
	if lineFactor > 0 {
		lh = (a + d) * lineFactor
	} else {
		lh = a + d + 3/fs.scale
	}
	return Metrics{Ascent: a, Descent: d, LineHeight: lh}
}

func fixedToFloat(v fixed.Int26_6) float32 { return float32(v) / 64 }

func toLogical(fs *FontSystem, px fixed.Int26_6) float32 {
	return fixedToFloat(px) / fs.scale
}

// baseInput builds a shaping input for the UI face.
func (fs *FontSystem) baseInput(text []rune) shaping.Input {
	return fs.baseInputFace(text, fs.primary, fs.uiPixelSize)
}

// baseInputMono builds a shaping input for the editor mono face.
func (fs *FontSystem) baseInputMono(text []rune) shaping.Input {
	return fs.baseInputFace(text, fs.mono, fs.monoPixelSize)
}

// baseInputFace builds a shaping input for a rune slice with default LTR paragraph direction.
func (fs *FontSystem) baseInputFace(text []rune, face *font.Face, size fixed.Int26_6) shaping.Input {
	return shaping.Input{
		Text:      text,
		RunStart:  0,
		RunEnd:    len(text),
		Direction: di.DirectionLTR,
		Face:      face,
		Size:      size,
		Script:    language.Latin,
		Language:  language.NewLanguage("en"),
	}
}
