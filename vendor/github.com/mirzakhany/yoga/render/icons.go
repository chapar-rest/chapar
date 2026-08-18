package render

import (
	"bytes"
	"embed"
	"image"
	"sort"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// A curated Material-style icon set (Pictogrammers MDI, Apache-2.0) ships with
// the framework. Each is a single-path 24px SVG; they are rasterized into the
// font atlas at the device pixel scale so they stay crisp, and tinted at draw
// time by the vertex color (same path as glyphs) so they follow the theme.
//
//go:embed assets/icons/*.svg
var iconFS embed.FS

// iconRegistry maps an icon name to its raw SVG bytes. Built-ins are loaded in
// init; applications may add or override icons with RegisterIcon before baking
// the atlas (NewMonoAtlas*).
var iconRegistry = map[string][]byte{}

// RegisterIcon adds or replaces a named icon from raw SVG bytes. Call it before
// creating the atlas; to add icons after the atlas exists, re-bake and call
// Renderer.UpdateAtlas.
func RegisterIcon(name string, svg []byte) {
	iconRegistry[name] = append([]byte(nil), svg...)
}

func init() {
	entries, err := iconFS.ReadDir("assets/icons")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		data, err := iconFS.ReadFile("assets/icons/" + e.Name())
		if err != nil {
			continue
		}
		iconRegistry[strings.TrimSuffix(e.Name(), ".svg")] = data
	}
}

// iconNames returns the registered icon names in stable sorted order (so atlas
// packing is deterministic).
func iconNames() []string {
	names := make([]string, 0, len(iconRegistry))
	for n := range iconRegistry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// rasterizeIcon renders an SVG into a px-by-px 8-bit coverage mask. Material
// icons fill with opaque black, so the rasterized alpha channel is exactly the
// coverage we want for an R8 atlas cell.
func rasterizeIcon(svg []byte, px int) (*image.Alpha, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg))
	if err != nil {
		return nil, err
	}
	icon.SetTarget(0, 0, float64(px), float64(px))

	rgba := image.NewRGBA(image.Rect(0, 0, px, px))
	scanner := rasterx.NewScannerGV(px, px, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(px, px, scanner)
	icon.Draw(raster, 1.0)

	alpha := image.NewAlpha(image.Rect(0, 0, px, px))
	for i := 0; i < px*px; i++ {
		alpha.Pix[i] = rgba.Pix[i*4+3] // copy the A channel as coverage
	}
	return alpha, nil
}
