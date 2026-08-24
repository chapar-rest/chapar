package shape

import (
	"sort"
	"unicode/utf8"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	"github.com/mirzakhany/yoga/render"
)

// monoNoLigatures disables ligatures and contextual alternates for editor mono
// shaping. Fonts like JetBrains Mono substitute connected glyphs for sequences
// such as "//", ":=", and "==" (via calt/liga), which visually merges adjacent
// characters and breaks the one-cell-per-character model a code editor needs.
var monoNoLigatures = []shaping.FontFeature{
	{Tag: ot.MustNewTag("liga"), Value: 0}, // standard ligatures
	{Tag: ot.MustNewTag("clig"), Value: 0}, // contextual ligatures
	{Tag: ot.MustNewTag("calt"), Value: 0}, // contextual alternates (JetBrains Mono code ligatures)
	{Tag: ot.MustNewTag("dlig"), Value: 0}, // discretionary ligatures
}

// Glyph is one visually-placed glyph in a shaped line.
//
// X is the pen/dot (advance cursor) used for caret, hit-testing, and selection.
// Ink is drawn at X+BearingX (and Y already encodes the ink top via YBearing).
type Glyph struct {
	FaceID      uint32
	GID         font.GID
	X, Y        float32 // pen X; ink-top Y in line coordinates (logical px)
	BearingX    float32 // XOffset+XBearing: ink left relative to pen
	W, H        float32
	Advance     float32
	ClusterByte int // byte offset in line of cluster start
	ClusterLen  int // byte length of cluster in line
	Color       bool
}

// Line is a fully shaped, visually ordered text line.
type Line struct {
	Glyphs      []Glyph
	Width       float32
	Runes       []rune    // original line runes (without trailing newline)
	LogicalSize render.Px // size used for shaping; 0 = face default
	Weight      int       // CSS-like weight; >=600 uses SemiBold UI face
	Mono        bool      // shaped with the editor mono face
}

// Shaper shapes single logical lines (no soft wrap).
type Shaper struct {
	fs         *FontSystem
	lastGlyphs []Glyph
}

// NewShaper returns a shaper bound to fonts.
func NewShaper(fs *FontSystem) *Shaper { return &Shaper{fs: fs} }

// ShapeLine shapes s as one LTR/RTL-aware UI line with tab expansion.
func (s *Shaper) ShapeLine(text string) Line {
	return s.shapeLineFaceAt(text, false, 0, WeightRegular)
}

// ShapeLineMono shapes s as one LTR/RTL-aware editor mono line with tab expansion.
func (s *Shaper) ShapeLineMono(text string) Line {
	return s.shapeLineFaceAt(text, true, 0, WeightRegular)
}

// ShapeLineFace shapes s with the UI face (mono=false) or editor mono face (mono=true).
func (s *Shaper) ShapeLineFace(text string, mono bool) Line {
	return s.shapeLineFaceAt(text, mono, 0, WeightRegular)
}

// ShapeLineAt shapes UI text at logicalSize (HarfBuzz at that ppem).
func (s *Shaper) ShapeLineAt(text string, logicalSize render.Px) Line {
	return s.shapeLineFaceAt(text, false, logicalSize, WeightRegular)
}

// ShapeLineAtWeight shapes UI text at logicalSize with the given CSS-like weight.
// Weight >= WeightSemiBold uses Inter SemiBold.
func (s *Shaper) ShapeLineAtWeight(text string, logicalSize render.Px, weight int) Line {
	return s.shapeLineFaceAt(text, false, logicalSize, weight)
}

// ShapeLineMonoAt shapes editor mono text at logicalSize.
func (s *Shaper) ShapeLineMonoAt(text string, logicalSize render.Px) Line {
	return s.shapeLineFaceAt(text, true, logicalSize, WeightRegular)
}

func (s *Shaper) shapeLineFaceAt(text string, mono bool, logicalSize render.Px, weight int) Line {
	if text == "" {
		return Line{LogicalSize: logicalSize, Weight: weight, Mono: mono}
	}
	runes := []rune(text)
	var out Line
	out.Runes = runes
	out.LogicalSize = logicalSize
	out.Weight = weight
	out.Mono = mono
	x := float32(0)

	segStart := 0
	flush := func(end int, tabStop bool) {
		if end > segStart {
			segRunes := runes[segStart:end]
			byteBase := len(string(runes[:segStart]))
			w := s.shapeSegment(segRunes, byteBase, x, mono, logicalSize, weight)
			x += w
			out.Glyphs = append(out.Glyphs, s.lastGlyphs...)
		}
		if tabStop {
			cw := s.cellWidthAt(mono, logicalSize, weight)
			tabCols := s.fs.TabCols()
			col := int(x / cw)
			nextCol := (col/tabCols + 1) * tabCols
			x = float32(nextCol) * cw
		}
		segStart = end
	}

	for i, r := range runes {
		if r == '\t' {
			flush(i, true)
			segStart = i + 1
		}
	}
	flush(len(runes), false)

	out.Width = x
	return out
}

func (s *Shaper) runSize(mono bool, logicalSize render.Px) fixed.Int26_6 {
	if logicalSize > 0 {
		return s.fs.ppem(logicalSize)
	}
	if mono {
		return s.fs.monoPixelSize
	}
	return s.fs.uiPixelSize
}

func (s *Shaper) cellWidthAt(mono bool, logicalSize render.Px, weight int) float32 {
	// approximate monospace tab width from 'm' advance plus letter spacing,
	// so tab stops align with the spaced character cells.
	size := s.runSize(mono, logicalSize)
	face := s.fs.uiFaceForWeight(weight)
	if mono {
		face = s.fs.mono
	}
	in := s.fs.baseInputFace([]rune("m"), face, size)
	out := s.fs.shaper.Shape(in)
	return toLogical(s.fs, out.Advance) + s.letterSpacing(mono)
}

// letterSpacing returns the extra logical px added per glyph for the role.
func (s *Shaper) letterSpacing(mono bool) float32 {
	if mono {
		return s.fs.monoSpacing
	}
	return s.fs.uiSpacing
}

func (s *Shaper) lineMetricsAt(mono bool, logicalSize render.Px) Metrics {
	if mono {
		return s.fs.MonoMetricsAt(logicalSize)
	}
	return s.fs.MetricsAt(logicalSize)
}

func (s *Shaper) shapeSegment(runes []rune, byteBase int, startX float32, mono bool, logicalSize render.Px, weight int) float32 {
	if len(runes) == 0 {
		s.lastGlyphs = nil
		return 0
	}
	runSize := s.runSize(mono, logicalSize)
	face := s.fs.uiFaceForWeight(weight)
	if mono {
		face = s.fs.mono
	}
	var fm shaping.Fontmap = s.fs
	switch {
	case mono:
		fm = monoFontmap{s.fs}
	case weight >= WeightSemiBold:
		fm = strongFontmap{s.fs}
	}
	in := s.fs.baseInputFace(runes, face, runSize)
	runs := s.fs.segment.Split(in, fm)
	if len(runs) == 0 {
		s.lastGlyphs = nil
		return 0
	}

	spacing := s.letterSpacing(mono)
	outputs := make([]shaping.Output, len(runs))
	for i, run := range runs {
		run.Size = runSize
		if mono {
			run.FontFeatures = monoNoLigatures
		}
		outputs[i] = s.fs.shaper.Shape(run)
	}
	computeBidiOrdering(di.DirectionLTR, outputs)

	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].VisualIndex < outputs[j].VisualIndex
	})

	metrics := s.lineMetricsAt(mono, logicalSize)
	ppem := uint16(runSize.Round())
	var glyphs []Glyph
	x := startX
	for _, out := range outputs {
		faceID := s.fs.FaceID(out.Face)
		if out.Face != nil {
			out.Face.SetPpem(ppem, ppem)
		}
		for _, g := range out.Glyphs {
			// Pen stays at the advance cursor; ink is offset by XOffset+XBearing
			// (HarfBuzz / go-text convention). Y already uses YBearing for ink top.
			gy := metrics.Ascent + toLogical(s.fs, g.YOffset)
			w := toLogical(s.fs, g.Width)
			h := toLogical(s.fs, g.Height)
			if w <= 0 {
				w = toLogical(s.fs, g.Advance)
			}
			if h <= 0 {
				h = metrics.Ascent + metrics.Descent
			}
			runeIdx := g.TextIndex()
			if runeIdx < 0 || runeIdx >= len(runes) {
				runeIdx = 0
			}
			clusterByte := byteBase + len(string(runes[:runeIdx]))
			clusterLen := utf8.RuneLen(runes[runeIdx])
			if clusterLen < 1 {
				clusterLen = 1
			}
			color := isColorGlyph(out.Face, g.GlyphID)
			adv := toLogical(s.fs, g.Advance) + spacing
			glyphs = append(glyphs, Glyph{
				FaceID: faceID, GID: g.GlyphID,
				X: x, Y: gy - toLogical(s.fs, g.YBearing),
				BearingX:    toLogical(s.fs, g.XOffset+g.XBearing),
				W:           w, H: h,
				Advance:     adv,
				ClusterByte: clusterByte,
				ClusterLen:  clusterLen,
				Color:       color,
			})
			x += adv
		}
	}
	s.lastGlyphs = glyphs
	return x - startX
}

func isColorGlyph(face *font.Face, gid font.GID) bool {
	if data := face.GlyphData(gid); data != nil {
		_, isColor := data.(font.GlyphColor)
		return isColor
	}
	return false
}

// Width returns shaped width of text.
func (s *Shaper) Width(text string) float32 {
	return s.ShapeLine(text).Width
}

// Measure returns width and line height for a single-line UI string.
func (s *Shaper) Measure(text string) (w, h float32) {
	return s.ShapeLine(text).Width, s.fs.metrics.LineHeight
}

// MeasureMono returns width and line height for a single-line editor mono string.
func (s *Shaper) MeasureMono(text string) (w, h float32) {
	return s.ShapeLineMono(text).Width, s.fs.monoMetrics.LineHeight
}

// MeasureAt returns width and line height at logicalSize for UI text.
func (s *Shaper) MeasureAt(text string, logicalSize render.Px) (w, h render.Px) {
	return s.MeasureAtWeight(text, logicalSize, WeightRegular)
}

// MeasureAtWeight returns width and line height at logicalSize for UI text at weight.
func (s *Shaper) MeasureAtWeight(text string, logicalSize render.Px, weight int) (w, h render.Px) {
	return s.ShapeLineAtWeight(text, logicalSize, weight).Width, s.fs.MetricsAt(logicalSize).LineHeight
}

// MeasureMonoAt returns width and line height at logicalSize for editor mono text.
func (s *Shaper) MeasureMonoAt(text string, logicalSize render.Px) (w, h render.Px) {
	return s.ShapeLineMonoAt(text, logicalSize).Width, s.fs.MonoMetricsAt(logicalSize).LineHeight
}

// XForByte returns the x coordinate for a byte offset within the line text.
func (l Line) XForByte(byteOff int) float32 {
	if byteOff <= 0 {
		return 0
	}
	for _, g := range l.Glyphs {
		end := g.ClusterByte + g.ClusterLen
		if byteOff <= g.ClusterByte {
			return g.X
		}
		if byteOff < end {
			return g.X + g.W*0.5
		}
	}
	return l.Width
}

// ByteForX maps a pixel x to the nearest byte offset (cluster boundary).
func (l Line) ByteForX(x float32) int {
	if x <= 0 || len(l.Glyphs) == 0 {
		return 0
	}
	for _, g := range l.Glyphs {
		mid := g.X + g.Advance*0.5
		if x < mid {
			return g.ClusterByte
		}
	}
	last := l.Glyphs[len(l.Glyphs)-1]
	return last.ClusterByte + last.ClusterLen
}

// PrevCluster returns the byte offset before off (grapheme/cluster aware).
func (l Line) PrevCluster(off int) int {
	if off <= 0 {
		return 0
	}
	prev := 0
	for _, g := range l.Glyphs {
		if g.ClusterByte >= off {
			break
		}
		prev = g.ClusterByte
	}
	return prev
}

// NextCluster returns the byte offset after off.
func (l Line) NextCluster(off int, lineLen int) int {
	for _, g := range l.Glyphs {
		end := g.ClusterByte + g.ClusterLen
		if end > off {
			return end
		}
	}
	return lineLen
}

// SelectionRects returns highlight rectangles for byte range [a,b) in line coords.
func (l Line) SelectionRects(a, b int, y, lineH float32) [][4]float32 {
	if a >= b {
		return nil
	}
	var rects [][4]float32
	for _, g := range l.Glyphs {
		gStart := g.ClusterByte
		gEnd := g.ClusterByte + g.ClusterLen
		if gEnd <= a || gStart >= b {
			continue
		}
		rects = append(rects, [4]float32{g.X, y, g.Advance, lineH})
	}
	return rects
}

// computeBidiOrdering resolves VisualIndex for shaped runs (from go-text).
func computeBidiOrdering(dir di.Direction, finalLine []shaping.Output) {
	bidiStart := -1
	for idx, run := range finalLine {
		basePosition := idx
		if dir.Progression() == di.TowardTopLeft {
			basePosition = len(finalLine) - 1 - idx
		}
		finalLine[idx].VisualIndex = int32(basePosition)
		if run.Direction == dir {
			if bidiStart != -1 {
				swapVisualOrder(finalLine[bidiStart:idx])
				bidiStart = -1
			}
		} else if bidiStart == -1 {
			bidiStart = idx
		}
	}
	if bidiStart != -1 {
		swapVisualOrder(finalLine[bidiStart:])
	}
}

func swapVisualOrder(subline []shaping.Output) {
	L := len(subline)
	for i := 0; i < L/2; i++ {
		j := (L - i) - 1
		subline[i].VisualIndex, subline[j].VisualIndex = subline[j].VisualIndex, subline[i].VisualIndex
	}
}
