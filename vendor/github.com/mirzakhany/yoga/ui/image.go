package ui

import (
	"hash/fnv"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ImageFit controls how a bitmap maps into its layout box when both dimensions are set.
type ImageFit int

const (
	FitContain ImageFit = iota // default: letterbox, preserve aspect
	FitCover                   // crop to fill
	FitFill                    // stretch to box
)

type imageSource struct {
	bytes    []byte
	filePath string
	fs       fs.FS
	fsName   string
	fit      ImageFit
	svg      bool
}

type imageState struct {
	sourceTag   string
	rawBytes    []byte
	fingerprint uint64
	rgba        *image.RGBA
	atlasKey    string
	intrinsicW  float32
	intrinsicH  float32
	loadErr     bool
	decodeErr   bool
	svgDoc      *render.SVGDoc
	rasterW     int
	rasterH     int
	tintHex     string
}

// Image displays a PNG or JPEG decoded from data. id keys the atlas slot and load cache.
func Image(id string, data []byte) *Node {
	return &Node{
		kind: kindImage,
		id:   id,
		extra: &imageSource{
			bytes: append([]byte(nil), data...),
			fit:   FitContain,
		},
	}
}

// ImageFile loads an image from the filesystem on first layout.
func ImageFile(id, path string) *Node {
	return &Node{
		kind: kindImage,
		id:   id,
		extra: &imageSource{
			filePath: path,
			fit:      FitContain,
		},
	}
}

// ImageFS loads an image from fs.FS (e.g. embed.FS) on first layout.
func ImageFS(id string, fsys fs.FS, name string) *Node {
	return &Node{
		kind: kindImage,
		id:   id,
		extra: &imageSource{
			fs:     fsys,
			fsName: name,
			fit:    FitContain,
		},
	}
}

// Fit sets how the bitmap maps when both Width and Height are specified.
func (n *Node) Fit(fit ImageFit) *Node {
	if d, ok := n.extra.(*imageSource); ok {
		d.fit = fit
	}
	return n
}

func (n *Node) layoutImage(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "image")
	}
	src, _ := n.extra.(*imageSource)
	if src == nil {
		src = &imageSource{fit: FitContain}
	}
	st := c.Widget(id, func() any { return &imageState{} }).(*imageState)
	th := c.Theme()

	n.resolveImageBytes(st, src)
	if src.svg {
		tint := n.svgTint(th)
		n.ensureSVGDoc(st, tint)
	} else {
		n.decodeImage(st, id)
	}

	w, h := n.imageLayoutSize(st, n.spec)
	if src.svg {
		n.rasterizeSVG(st, id, w, h, src.fit, n.svgTint(th))
	}
	stStyle := applyLayoutSpec(layout.Box().Size(w, h).FlexShrink(0), n.spec)
	el := layout.New(stStyle)
	n.paintImage(el, st, src, th)
	return el
}

func (n *Node) paintImage(el *layout.Element, st *imageState, src *imageSource, th *theme.Theme) {
	fit := src.fit
	atlasKey := st.atlasKey
	rgba := st.rgba
	bad := st.loadErr || st.decodeErr || rgba == nil
	placeholder := th.ChromeMuted
	iw, ih := st.intrinsicW, st.intrinsicH

	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		frame := el.Frame
		if bad {
			dl.AddRoundedRect(frame, th.Radius.Small, placeholder)
			if sheet := frameIcons(); sheet != nil {
				inset := f32min(frame.W, frame.H) * 0.25
				inner := render.Rect{
					X: frame.X + inset, Y: frame.Y + inset,
					W: frame.W - 2*inset, H: frame.H - 2*inset,
				}
				sheet.Draw(dl, icons.ImageOff, inner, th.ForegroundMuted)
			}
			return
		}
		dst := imageDestRect(frame, iw, ih, fit)
		if sheet := frameIcons(); sheet != nil {
			sheet.DrawImageEntry(dl, atlasKey, dst)
		}
	}
}

func (n *Node) resolveImageBytes(st *imageState, src *imageSource) {
	switch {
	case len(src.bytes) > 0:
		tag := "bytes"
		if st.sourceTag != tag || !bytesEqual(st.rawBytes, src.bytes) {
			st.rawBytes = append([]byte(nil), src.bytes...)
			st.fingerprint = hashBytes(st.rawBytes)
			st.sourceTag = tag
			st.rgba = nil
			st.atlasKey = ""
			st.svgDoc = nil
			st.rasterW, st.rasterH = 0, 0
			st.tintHex = ""
			st.loadErr = false
			st.decodeErr = false
		}
	case src.filePath != "":
		tag := "file:" + src.filePath
		if st.sourceTag != tag {
			data, err := os.ReadFile(src.filePath)
			if err != nil {
				st.loadErr = true
				st.decodeErr = true
				st.rgba = nil
				st.sourceTag = tag
				return
			}
			st.rawBytes = data
			st.fingerprint = hashBytes(data)
			st.sourceTag = tag
			st.rgba = nil
			st.atlasKey = ""
			st.svgDoc = nil
			st.rasterW, st.rasterH = 0, 0
			st.tintHex = ""
			st.loadErr = false
			st.decodeErr = false
		}
	case src.fs != nil && src.fsName != "":
		tag := "fs:" + src.fsName
		if st.sourceTag != tag {
			data, err := fs.ReadFile(src.fs, src.fsName)
			if err != nil {
				st.loadErr = true
				st.decodeErr = true
				st.rgba = nil
				st.sourceTag = tag
				return
			}
			st.rawBytes = data
			st.fingerprint = hashBytes(data)
			st.sourceTag = tag
			st.rgba = nil
			st.atlasKey = ""
			st.svgDoc = nil
			st.rasterW, st.rasterH = 0, 0
			st.tintHex = ""
			st.loadErr = false
			st.decodeErr = false
		}
	default:
		if st.sourceTag != "empty" {
			st.sourceTag = "empty"
			st.loadErr = true
			st.decodeErr = true
			st.rgba = nil
		}
	}
}

func (n *Node) decodeImage(st *imageState, id string) {
	if st.loadErr || len(st.rawBytes) == 0 {
		return
	}
	if st.rgba != nil && st.atlasKey != "" {
		return
	}
	img, _, err := image.Decode(bytesReader(st.rawBytes))
	if err != nil {
		st.decodeErr = true
		st.rgba = nil
		return
	}
	rgba := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	st.rgba = rgba
	st.intrinsicW = float32(rgba.Bounds().Dx())
	st.intrinsicH = float32(rgba.Bounds().Dy())
	st.atlasKey = imageAtlasKey(id, st.fingerprint)
	if sheet := frameIcons(); sheet != nil {
		if _, ok := sheet.Atlas().EnsureImage(st.atlasKey, rgba); !ok {
			st.decodeErr = true
		}
	}
}

func (n *Node) imageLayoutSize(st *imageState, spec Spec) (w, h float32) {
	const fallback = 48
	if st.loadErr || st.decodeErr || st.intrinsicW <= 0 || st.intrinsicH <= 0 {
		if spec.hasW && spec.hasH {
			return spec.width, spec.height
		}
		if spec.hasW {
			return spec.width, spec.width
		}
		if spec.hasH {
			return spec.height, spec.height
		}
		return fallback, fallback
	}
	iw, ih := st.intrinsicW, st.intrinsicH
	aspect := iw / ih
	if spec.hasW && spec.hasH {
		return spec.width, spec.height
	}
	if spec.hasW {
		return spec.width, spec.width / aspect
	}
	if spec.hasH {
		return spec.height * aspect, spec.height
	}
	return iw, ih
}

func imageDestRect(frame render.Rect, iw, ih float32, fit ImageFit) render.Rect {
	if iw <= 0 || ih <= 0 {
		return frame
	}
	boxW, boxH := frame.W, frame.H
	if boxW <= 0 || boxH <= 0 {
		return frame
	}
	switch fit {
	case FitFill:
		return frame
	case FitCover:
		scale := f32max(boxW/iw, boxH/ih)
		dw, dh := iw*scale, ih*scale
		return render.Rect{
			X: frame.X + (boxW-dw)/2,
			Y: frame.Y + (boxH-dh)/2,
			W: dw, H: dh,
		}
	default: // FitContain
		scale := f32min(boxW/iw, boxH/ih)
		dw, dh := iw*scale, ih*scale
		return render.Rect{
			X: frame.X + (boxW-dw)/2,
			Y: frame.Y + (boxH-dh)/2,
			W: dw, H: dh,
		}
	}
}

func imageAtlasKey(id string, fp uint64) string {
	return id + ":" + uintHex(fp)
}

func hashBytes(b []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}

func uintHex(v uint64) string {
	const hexdigits = "0123456789abcdef"
	var buf [16]byte
	for i := 7; i >= 0; i-- {
		buf[i*2+1] = hexdigits[v&0xf]
		buf[i*2] = hexdigits[(v>>4)&0xf]
		v >>= 8
	}
	return string(buf[:])
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type byteReader struct {
	b []byte
	i int
}

func bytesReader(b []byte) *byteReader { return &byteReader{b: b} }

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
