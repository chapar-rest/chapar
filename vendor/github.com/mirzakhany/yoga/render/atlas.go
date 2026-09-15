package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/mirzakhany/yoga/icons"
	"golang.org/x/image/draw"
	"golang.org/x/image/vector"
)

// Page selects which atlas texture to sample.
type Page int

const (
	PageMono Page = iota + 1
	PageColor
)

type glyphKey struct {
	faceID uint32
	gid    font.GID
	ppem   uint16
}

// GlyphRasterPad is the device-pixel padding around outline ink in the atlas.
// Draw paths subtract PadLogical so the ink, not the pad, aligns with bearings.
const GlyphRasterPad = 2

// GlyphEntry describes a baked glyph in the atlas.
type GlyphEntry struct {
	Page  Page
	UV    Rect
	W, H  float32 // logical size (includes Pad on each side for outlines)
	Pad   float32 // logical px of raster pad around ink (0 for bitmaps)
	Color bool
	physW int
	physH int
	physX int
	physY int
}

// PadLogical converts device-pixel GlyphRasterPad to logical px for this atlas scale.
func (a *FontAtlas) PadLogical() float32 {
	if a.scale < 1 {
		return float32(GlyphRasterPad)
	}
	return float32(GlyphRasterPad) / a.scale
}

// ImageEntry describes a baked bitmap image in the color atlas page.
type ImageEntry struct {
	UV    Rect
	W, H  float32 // logical size
	physW int
	physH int
	physX int
	physY int
}

// IconEntry describes a packed icon mask in the mono atlas page.
type IconEntry struct {
	UV    Rect
	physW int
	physH int
	physX int
	physY int
}

// DirtyRect is a sub-rectangle that changed in an atlas page.
type DirtyRect struct {
	Page Page
	X, Y int
	W, H int
	Pix  []byte
}

// FontAtlas is a dynamic glyph cache with mono (R8) and color (RGBA) pages.
type FontAtlas struct {
	scale float32

	monoPix []byte
	monoW   int
	monoH   int

	colorPix []byte
	colorW   int
	colorH   int

	glyphs map[glyphKey]GlyphEntry
	icons  map[string]IconEntry
	images map[string]ImageEntry

	// iconFails remembers icons whose rasterization failed so we don't retry
	// (re-parse + re-rasterize SVG) on every single frame.
	iconFails map[string]bool

	monoShelf   shelf
	colorShelf  shelf
	dirty       []DirtyRect
	fullRebuild bool

	// drawList is optionally bound during paint so mid-paint page growth can
	// rescale V coords already written into the current frame's geometry.
	drawList *DrawList

	// Legacy monospace metrics (approximate, for gutter numbers etc.).
	CellW float32
	CellH float32

	// Legacy fields used by renderer during upload.
	W int
	H int
}

const (
	initialMonoW  = 512
	initialMonoH  = 512
	initialColorW = 512
	initialColorH = 512
	iconLogical   = 20.0
	logicalFontPx = 14.0
)

// NewAtlasScale creates an atlas at device scale (icons pre-baked).
func NewAtlasScale(scale float32) *FontAtlas {
	if scale < 1 {
		scale = 1
	}
	a := &FontAtlas{
		scale:     scale,
		monoW:     initialMonoW,
		monoH:     initialMonoH,
		colorW:    initialColorW,
		colorH:    initialColorH,
		W:         initialMonoW,
		H:         initialMonoH,
		glyphs:    make(map[glyphKey]GlyphEntry),
		icons:     make(map[string]IconEntry, 32),
		images:    make(map[string]ImageEntry, 8),
		iconFails: make(map[string]bool),
		monoShelf: shelf{pad: 1},
		colorShelf: shelf{pad: 1},
	}
	a.monoPix = make([]byte, a.monoW*a.monoH)
	a.colorPix = make([]byte, a.colorW*a.colorH*4)
	a.CellW = logicalFontPx * 0.62
	a.CellH = logicalFontPx + 3
	return a
}

// NewMonoAtlasScale is an alias for NewAtlasScale.
func NewMonoAtlasScale(scale float32) *FontAtlas { return NewAtlasScale(scale) }

// NewMonoAtlas creates a 1x atlas.
func NewMonoAtlas() *FontAtlas { return NewAtlasScale(1) }

// BindDrawList attaches the frame's draw list so atlas growth can rescale
// already-emitted textured quads. Pass nil to clear after paint.
func (a *FontAtlas) BindDrawList(dl *DrawList) {
	if a == nil {
		return
	}
	a.drawList = dl
}

func (a *FontAtlas) EnsureIcon(icon icons.Icon) (Rect, bool) {
	if icon.Empty() {
		return Rect{}, false
	}
	if e, ok := a.icons[icon.Name]; ok {
		return e.UV, true
	}
	if a.iconFails[icon.Name] {
		// Rasterization failed before; retrying every frame would burn CPU.
		return Rect{}, false
	}
	iconPx := int(iconLogical*a.scale + 0.5)
	if iconPx < 8 {
		iconPx = 8
	}
	var mask *image.Alpha
	var err error
	if HasSVGOverride(icon.Name) {
		mask, err = rasterizeOverrideSVG(icon.Name, iconPx)
	} else {
		if am, aerr := icon.Alpha(iconPx); aerr != nil {
			err = aerr
		} else {
			mask = image.NewAlpha(image.Rect(0, 0, am.W, am.H))
			copy(mask.Pix, am.Pix)
		}
	}
	if err != nil || mask == nil {
		a.iconFails[icon.Name] = true
		return Rect{}, false
	}
	return a.packIconMask(icon.Name, mask), true
}

func (a *FontAtlas) packIconMask(name string, mask *image.Alpha) Rect {
	w, h := mask.Bounds().Dx(), mask.Bounds().Dy()
	x, y, ok := a.allocMono(w, h)
	if !ok {
		// Cannot happen for icons (page is at least 512 wide), but never blit
		// out of bounds.
		return Rect{}
	}
	blitAlpha(a.monoPix, a.monoW, x, y, mask)
	a.markMonoDirty(x, y, w, h)
	uv := insetUV(x, y, w, h, a.monoW, a.monoH)
	a.icons[name] = IconEntry{
		UV:    uv,
		physW: w, physH: h, physX: x, physY: y,
	}
	return uv
}

// allocMono finds space in the mono shelf, growing the page height until the
// cell fits. The caller must guarantee w fits the fixed page width.
func (a *FontAtlas) allocMono(w, h int) (int, int, bool) {
	x, y, ok := a.monoShelf.alloc(a, w, h, true)
	for !ok {
		a.growMono(a.monoH * 2)
		x, y, ok = a.monoShelf.alloc(a, w, h, true)
	}
	return x, y, ok
}

// allocColor finds space in the color shelf, growing the page height until the
// cell fits. The caller must guarantee w fits the fixed page width.
func (a *FontAtlas) allocColor(w, h int) (int, int, bool) {
	x, y, ok := a.colorShelf.alloc(a, w, h, false)
	for !ok {
		a.growColor(a.colorH * 2)
		x, y, ok = a.colorShelf.alloc(a, w, h, false)
	}
	return x, y, ok
}

func (a *FontAtlas) growMono(newH int) {
	if newH <= a.monoH {
		newH = a.monoH * 2
	}
	oldH := a.monoH
	pix := make([]byte, a.monoW*newH)
	copy(pix, a.monoPix)
	a.monoPix = pix
	a.monoH = newH
	a.H = newH
	a.recomputeMonoUVs()
	if oldH > 0 {
		a.drawList.ScalePageUVY(PageMono, float32(oldH)/float32(newH))
	}
	a.fullRebuild = true
}

// EnsureGlyph returns a baked glyph entry, rasterizing on miss at the given
// device-pixel ppem. The face's ppem is set before baking so outlines match.
func (a *FontAtlas) EnsureGlyph(faceID uint32, face *font.Face, gid font.GID, ppem uint16) GlyphEntry {
	if ppem < 1 {
		ppem = 1
	}
	key := glyphKey{faceID: faceID, gid: gid, ppem: ppem}
	if e, ok := a.glyphs[key]; ok {
		return e
	}
	if face != nil {
		face.SetPpem(ppem, ppem)
	}
	e := a.bakeGlyph(face, gid)
	a.glyphs[key] = e
	return e
}

func (a *FontAtlas) bakeGlyph(face *font.Face, gid font.GID) GlyphEntry {
	data := face.GlyphData(gid)
	switch d := data.(type) {
	case font.GlyphColor:
		if bm, ok := face.GlyphData(gid).(font.GlyphBitmap); ok {
			if img := decodeBitmap(bm); img != nil {
				return a.packColor(img, true)
			}
		}
		_ = d
	case font.GlyphBitmap:
		if img := decodeBitmap(d); img != nil {
			if d.Format == font.PNG {
				return a.packColor(img, true)
			}
			return a.packMono(toAlpha(img))
		}
	case font.GlyphOutline:
		e := a.packMono(rasterizeOutline(face, d))
		e.Pad = a.PadLogical()
		return e
	case font.GlyphSVG:
		e := a.packMono(rasterizeOutline(face, d.Outline))
		e.Pad = a.PadLogical()
		return e
	}
	img := image.NewAlpha(image.Rect(0, 0, 1, 1))
	return a.packMono(img)
}

func (a *FontAtlas) packMono(src *image.Alpha) GlyphEntry {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	// The page width is fixed, so a cell wider than the page can never fit;
	// downscale it instead of writing out of bounds or looping forever.
	if maxW := a.monoW - 2*a.monoShelf.pad; w > maxW {
		nh := int(float64(h) * float64(maxW) / float64(w))
		if nh < 1 {
			nh = 1
		}
		dst := image.NewAlpha(image.Rect(0, 0, maxW, nh))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
		src, w, h = dst, maxW, nh
	}
	x, y, _ := a.allocMono(w, h)
	blitAlpha(a.monoPix, a.monoW, x, y, src)
	a.markMonoDirty(x, y, w, h)
	return GlyphEntry{
		Page: PageMono,
		UV:   insetUV(x, y, w, h, a.monoW, a.monoH),
		W:    float32(w) / a.scale, H: float32(h) / a.scale,
		physW: w, physH: h, physX: x, physY: y,
	}
}

func (a *FontAtlas) packColor(src *image.RGBA, isColor bool) GlyphEntry {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	// The page width is fixed, so a cell wider than the page can never fit;
	// downscale it instead of writing out of bounds or looping forever.
	if maxW := a.colorW - 2*a.colorShelf.pad; w > maxW {
		nh := int(float64(h) * float64(maxW) / float64(w))
		if nh < 1 {
			nh = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, maxW, nh))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
		src, w, h = dst, maxW, nh
	}
	x, y, _ := a.allocColor(w, h)
	blitRGBA(a.colorPix, a.colorW, x, y, src)
	a.markColorDirty(x, y, w, h)
	return GlyphEntry{
		Page: PageColor, Color: isColor,
		UV: insetUV(x, y, w, h, a.colorW, a.colorH),
		W:  float32(w) / a.scale, H: float32(h) / a.scale,
		physW: w, physH: h, physX: x, physY: y,
	}
}

type shelf struct {
	x, y int
	rowH int
	pad  int
}

// insetUV shrinks a packed cell's UV rect by half a texel on each side so the
// linear sampler never bleeds neighbouring shelf cells at quad edges. A 1px
// wide/tall cell gets a zero-span UV so it samples its single texel exactly.
func insetUV(x, y, w, h, pageW, pageH int) Rect {
	const half = 0.5
	uw := w - 1
	if uw < 0 {
		uw = 0
	}
	uh := h - 1
	if uh < 0 {
		uh = 0
	}
	return Rect{
		X: (float32(x) + half) / float32(pageW),
		Y: (float32(y) + half) / float32(pageH),
		W: float32(uw) / float32(pageW),
		H: float32(uh) / float32(pageH),
	}
}

func (s *shelf) alloc(a *FontAtlas, w, h int, mono bool) (int, int, bool) {
	pad := s.pad
	pw := pageW(a, mono)
	ph := pageH(a, mono)
	if s.x+pad+w+pad > pw {
		s.x = 0
		s.y += s.rowH + pad
		s.rowH = 0
	}
	if s.y+pad+h+pad > ph {
		return 0, 0, false
	}
	x := s.x + pad
	y := s.y + pad
	s.x += pad + w + pad
	if h > s.rowH {
		s.rowH = h
	}
	return x, y, true
}

func pageW(a *FontAtlas, mono bool) int {
	if mono {
		return a.monoW
	}
	return a.colorW
}

func pageH(a *FontAtlas, mono bool) int {
	if mono {
		return a.monoH
	}
	return a.colorH
}

func (a *FontAtlas) growColor(newH int) {
	if newH <= a.colorH {
		newH = a.colorH * 2
	}
	oldH := a.colorH
	pix := make([]byte, a.colorW*newH*4)
	copy(pix, a.colorPix)
	a.colorPix = pix
	a.colorH = newH
	a.recomputeColorUVs()
	if oldH > 0 {
		a.drawList.ScalePageUVY(PageColor, float32(oldH)/float32(newH))
	}
	a.fullRebuild = true
}

func (a *FontAtlas) recomputeMonoUVs() {
	for k, e := range a.glyphs {
		if e.Page != PageMono {
			continue
		}
		e.UV = insetUV(e.physX, e.physY, e.physW, e.physH, a.monoW, a.monoH)
		a.glyphs[k] = e
	}
	for k, e := range a.icons {
		e.UV = insetUV(e.physX, e.physY, e.physW, e.physH, a.monoW, a.monoH)
		a.icons[k] = e
	}
}

func (a *FontAtlas) recomputeColorUVs() {
	for k, e := range a.glyphs {
		if e.Page != PageColor {
			continue
		}
		e.UV = insetUV(e.physX, e.physY, e.physW, e.physH, a.colorW, a.colorH)
		a.glyphs[k] = e
	}
	for k, e := range a.images {
		e.UV = insetUV(e.physX, e.physY, e.physW, e.physH, a.colorW, a.colorH)
		a.images[k] = e
	}
}

// EnsureImage packs src into the color atlas under key, returning cached entry on hit.
func (a *FontAtlas) EnsureImage(key string, src *image.RGBA) (ImageEntry, bool) {
	if key == "" || src == nil {
		return ImageEntry{}, false
	}
	if e, ok := a.images[key]; ok {
		return e, true
	}
	prepared := a.prepareImage(src)
	if prepared == nil {
		return ImageEntry{}, false
	}
	e := a.packImage(key, prepared)
	if e.physW < 1 {
		return ImageEntry{}, false
	}
	return e, true
}

func (a *FontAtlas) prepareImage(src *image.RGBA) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w < 1 || h < 1 {
		return nil
	}
	maxW := a.colorW - 2*a.colorShelf.pad
	if maxW < 1 {
		maxW = 1
	}
	if w <= maxW {
		return src
	}
	scale := float64(maxW) / float64(w)
	nw := maxW
	nh := int(float64(h)*scale + 0.5)
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func (a *FontAtlas) packImage(key string, src *image.RGBA) ImageEntry {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	for {
		x, y, ok := a.colorShelf.alloc(a, w, h, false)
		if ok {
			blitRGBA(a.colorPix, a.colorW, x, y, src)
			a.markColorDirty(x, y, w, h)
			e := ImageEntry{
				UV: insetUV(x, y, w, h, a.colorW, a.colorH),
				W:  float32(w) / a.scale, H: float32(h) / a.scale,
				physW: w, physH: h, physX: x, physY: y,
			}
			a.images[key] = e
			return e
		}
		a.growColor(a.colorH * 2)
	}
}

// ImageUV returns the UV for an image already packed in the atlas.
func (a *FontAtlas) ImageUV(key string) (ImageEntry, bool) {
	e, ok := a.images[key]
	return e, ok
}

func (a *FontAtlas) markMonoDirty(x, y, w, h int) {
	a.dirty = append(a.dirty, DirtyRect{Page: PageMono, X: x, Y: y, W: w, H: h, Pix: extractMono(a.monoPix, a.monoW, x, y, w, h)})
}

func (a *FontAtlas) markColorDirty(x, y, w, h int) {
	a.dirty = append(a.dirty, DirtyRect{Page: PageColor, X: x, Y: y, W: w, H: h, Pix: extractRGBA(a.colorPix, a.colorW, x, y, w, h)})
}

func extractMono(pix []byte, stride, x, y, w, h int) []byte {
	out := make([]byte, w*h)
	for row := 0; row < h; row++ {
		copy(out[row*w:], pix[(y+row)*stride+x:(y+row)*stride+x+w])
	}
	return out
}

func extractRGBA(pix []byte, stride, x, y, w, h int) []byte {
	out := make([]byte, w*h*4)
	for row := 0; row < h; row++ {
		copy(out[row*w*4:], pix[((y+row)*stride+x)*4:((y+row)*stride+x+w)*4])
	}
	return out
}

func blitAlpha(dst []byte, stride, ox, oy int, src *image.Alpha) {
	for y := 0; y < src.Rect.Dy(); y++ {
		for x := 0; x < src.Rect.Dx(); x++ {
			dst[(oy+y)*stride+(ox+x)] = src.Pix[y*src.Stride+x]
		}
	}
}

func blitRGBA(dst []byte, stride, ox, oy int, src *image.RGBA) {
	for y := 0; y < src.Rect.Dy(); y++ {
		for x := 0; x < src.Rect.Dx(); x++ {
			i := ((oy+y)*stride + (ox + x)) * 4
			j := y*src.Stride + x*4
			dst[i], dst[i+1], dst[i+2], dst[i+3] = src.Pix[j], src.Pix[j+1], src.Pix[j+2], src.Pix[j+3]
		}
	}
}

func rasterizeOutline(face *font.Face, outline font.GlyphOutline) *image.Alpha {
	xPpem, _ := face.Ppem()
	scale := float32(xPpem) / float32(face.Upem())
	if scale <= 0 {
		scale = logicalFontPx / float32(face.Upem())
	}
	minX, minY, maxX, maxY := boundsOutline(outline, scale)
	pad := GlyphRasterPad
	w := int(maxX-minX) + pad*2
	h := int(maxY-minY) + pad*2
	if w < 2 {
		w = 2
	}
	if h < 2 {
		h = 2
	}
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	var rs vector.Rasterizer
	rs.Reset(w, h)
	addOutline(&rs, outline, scale, float32(pad)-minX, float32(pad)-minY)
	rs.ClosePath()
	rs.Draw(rgba, rgba.Bounds(), image.NewUniform(color.White), image.Point{})
	return toAlpha(rgba)
}

func boundsOutline(o font.GlyphOutline, scale float32) (minX, minY, maxX, maxY float32) {
	minX, minY = float32(1e9), float32(1e9)
	for _, seg := range o.Segments {
		for _, p := range seg.ArgsSlice() {
			x, y := p.X*scale, -p.Y*scale
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	return
}

func addOutline(rs *vector.Rasterizer, o font.GlyphOutline, scale, dx, dy float32) {
	for _, seg := range o.Segments {
		args := seg.ArgsSlice()
		switch seg.Op {
		case ot.SegmentOpMoveTo:
			p := args[0]
			rs.MoveTo(p.X*scale+dx, -p.Y*scale+dy)
		case ot.SegmentOpLineTo:
			p := args[0]
			rs.LineTo(p.X*scale+dx, -p.Y*scale+dy)
		case ot.SegmentOpQuadTo:
			p0, p1 := args[0], args[1]
			rs.QuadTo(p0.X*scale+dx, -p0.Y*scale+dy, p1.X*scale+dx, -p1.Y*scale+dy)
		case ot.SegmentOpCubeTo:
			p0, p1, p2 := args[0], args[1], args[2]
			rs.CubeTo(p0.X*scale+dx, -p0.Y*scale+dy, p1.X*scale+dx, -p1.Y*scale+dy, p2.X*scale+dx, -p2.Y*scale+dy)
		}
	}
}

func toAlpha(rgba *image.RGBA) *image.Alpha {
	out := image.NewAlpha(rgba.Bounds())
	for y := 0; y < rgba.Rect.Dy(); y++ {
		for x := 0; x < rgba.Rect.Dx(); x++ {
			_, _, _, a := rgba.At(x, y).RGBA()
			out.Pix[y*out.Stride+x] = uint8(a >> 8)
		}
	}
	return out
}

func decodeBitmap(b font.GlyphBitmap) *image.RGBA {
	switch b.Format {
	case font.PNG:
		img, err := png.Decode(bytes.NewReader(b.Data))
		if err != nil {
			return nil
		}
		out := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
		for y := 0; y < out.Rect.Dy(); y++ {
			for x := 0; x < out.Rect.Dx(); x++ {
				out.Set(x, y, img.At(x, y))
			}
		}
		return out
	case font.BlackAndWhite:
		out := image.NewRGBA(image.Rect(0, 0, b.Width, b.Height))
		for y := 0; y < b.Height; y++ {
			for x := 0; x < b.Width; x++ {
				bit := (b.Data[(y*b.Width+x)/8] >> (7 - (x % 8))) & 1
				if bit != 0 {
					out.SetRGBA(x, y, color.RGBA{A: 255})
				}
			}
		}
		return out
	}
	return nil
}

// IconUV returns UV for a named icon already packed in the atlas.
func (a *FontAtlas) IconUV(name string) (Rect, bool) {
	e, ok := a.icons[name]
	return e.UV, ok
}

// MonoPixels returns the mono page bytes.
func (a *FontAtlas) MonoPixels() []byte { return a.monoPix }

// ColorPixels returns the color page bytes.
func (a *FontAtlas) ColorPixels() []byte { return a.colorPix }

// MonoSize returns mono page dimensions.
func (a *FontAtlas) MonoSize() (int, int) { return a.monoW, a.monoH }

// ColorSize returns color page dimensions.
func (a *FontAtlas) ColorSize() (int, int) { return a.colorW, a.colorH }

// DirtyRects returns pending dirty regions.
func (a *FontAtlas) DirtyRects() []DirtyRect { return a.dirty }

// ClearDirty clears dirty tracking after upload.
func (a *FontAtlas) ClearDirty() { a.dirty = a.dirty[:0] }

// NeedsFullRebuild reports atlas growth requiring full re-upload.
func (a *FontAtlas) NeedsFullRebuild() bool { return a.fullRebuild }

// ClearFullRebuild clears the full rebuild flag.
func (a *FontAtlas) ClearFullRebuild() { a.fullRebuild = false }

// Pixels returns mono page (legacy).
func (a *FontAtlas) Pixels() []byte { return a.monoPix }

// GlyphUV is deprecated; returns zero rect.
func (a *FontAtlas) GlyphUV(r rune) Rect { _ = r; return Rect{} }

// Measure is a legacy monospace estimate.
func (a *FontAtlas) Measure(s string) (w, h float32) {
	return float32(len([]rune(s))) * a.CellW, a.CellH
}

// DrawText is deprecated; use shape.Engine.DrawString.
func (a *FontAtlas) DrawText(dl *DrawList, s string, x, y float32, c Color) float32 {
	_ = dl
	_ = s
	_ = x
	_ = y
	_ = c
	return 0
}
