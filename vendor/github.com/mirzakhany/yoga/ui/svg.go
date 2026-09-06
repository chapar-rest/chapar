package ui

import (
	"fmt"
	"io/fs"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// SVG displays a custom SVG document rasterized into the color atlas.
// id keys the atlas slot and load cache. Original fills and strokes are kept;
// currentColor follows the node's text color (theme foreground by default).
func SVG(id string, data []byte) *Node {
	return &Node{
		kind: kindImage,
		id:   id,
		extra: &imageSource{
			bytes: append([]byte(nil), data...),
			fit:   FitContain,
			svg:   true,
		},
	}
}

// SVGFile loads an SVG from the filesystem on first layout.
func SVGFile(id, path string) *Node {
	return &Node{
		kind: kindImage,
		id:   id,
		extra: &imageSource{
			filePath: path,
			fit:      FitContain,
			svg:      true,
		},
	}
}

// SVGFS loads an SVG from fs.FS (e.g. embed.FS) on first layout.
func SVGFS(id string, fsys fs.FS, name string) *Node {
	return &Node{
		kind: kindImage,
		id:   id,
		extra: &imageSource{
			fs:     fsys,
			fsName: name,
			fit:    FitContain,
			svg:    true,
		},
	}
}

func (n *Node) svgTint(th *theme.Theme) render.Color {
	if col, ok := n.spec.fg.resolve(th); ok {
		return col
	}
	return th.Foreground
}

func (n *Node) ensureSVGDoc(st *imageState, tint render.Color) {
	if st.loadErr || len(st.rawBytes) == 0 {
		return
	}
	hex := renderColorHex(tint)
	if st.svgDoc != nil && st.tintHex == hex && st.intrinsicW > 0 {
		return
	}
	doc, err := render.ParseSVGDoc(st.rawBytes, tint)
	if err != nil {
		st.decodeErr = true
		st.svgDoc = nil
		st.rgba = nil
		return
	}
	st.svgDoc = doc
	st.tintHex = hex
	st.intrinsicW, st.intrinsicH = doc.Size()
	st.decodeErr = false
	st.rgba = nil
	st.atlasKey = ""
	st.rasterW, st.rasterH = 0, 0
}

func (n *Node) rasterizeSVG(st *imageState, id string, layoutW, layoutH float32, fit ImageFit, tint render.Color) {
	if st.loadErr || st.decodeErr || st.svgDoc == nil {
		return
	}
	dst := imageDestRect(render.Rect{W: layoutW, H: layoutH}, st.intrinsicW, st.intrinsicH, rasterFit(fit))
	scale := float32(1)
	if sheet := frameIcons(); sheet != nil {
		scale = sheet.Atlas().Scale()
	}
	pxW := int(dst.W*scale + 0.5)
	pxH := int(dst.H*scale + 0.5)
	hex := renderColorHex(tint)
	if st.rgba != nil && st.atlasKey != "" && st.rasterW == pxW && st.rasterH == pxH && st.tintHex == hex {
		return
	}
	rgba, err := st.svgDoc.Rasterize(pxW, pxH)
	if err != nil || rgba == nil {
		st.decodeErr = true
		st.rgba = nil
		return
	}
	st.rgba = rgba
	st.rasterW, st.rasterH = pxW, pxH
	st.atlasKey = fmt.Sprintf("%s:%dx%d:%s", imageAtlasKey(id, st.fingerprint), pxW, pxH, hex)
	if sheet := frameIcons(); sheet != nil {
		if _, ok := sheet.Atlas().EnsureImage(st.atlasKey, rgba); !ok {
			st.decodeErr = true
		}
	}
}

func rasterFit(fit ImageFit) ImageFit {
	if fit == FitFill {
		return FitContain
	}
	return fit
}

func renderColorHex(c render.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", hexByteUI(c.R), hexByteUI(c.G), hexByteUI(c.B))
}

func hexByteUI(v float32) byte {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 255
	}
	return byte(v*255 + 0.5)
}
