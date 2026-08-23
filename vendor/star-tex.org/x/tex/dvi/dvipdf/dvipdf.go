// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dvipdf prodivdes a DVI renderer that renders to a PDF document.
package dvipdf // import "star-tex.org/x/tex/dvi/dvipdf"

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	pdf "codeberg.org/go-pdf/fpdf"
	"star-tex.org/x/tex/dvi"
	"star-tex.org/x/tex/font/afm"
	"star-tex.org/x/tex/font/fixed"
	"star-tex.org/x/tex/kpath"
)

// codePageEncoding holds informations about the characters encoding of TrueType
// font files, needed by go-pdf/fpdf to embed fonts in a PDF document.
// We use cp1252 (code page 1252, Windows Western) to encode characters.
// See:
//   - https://en.wikipedia.org/wiki/Windows-1252
//
//go:embed cp1252.map
var codePageEncoding []byte

type fntkey struct {
	name string
	size fixed.Int12_20
}

var (
	_ dvi.Renderer = (*Renderer)(nil)
)

// Renderer renders a DVI document as a PDF document.
type Renderer struct {
	bkg color.Color

	pre   dvi.CmdPre
	post  dvi.CmdPost
	conv  float32 // converts DVI units to pixels
	tconv float32 // converts unmagnified DVI units to pixels
	dpi   float32

	ctx   kpath.Context
	fonts map[fntkey]struct{}

	// Switch to embed fonts in PDF file.
	// The default is to embed fonts.
	// This makes the PDF file more portable but also larger.
	embed bool

	cfg  pdf.InitType
	xoff float32
	yoff float32

	cur fntkey // current font.
	out io.Writer
	pdf *pdf.Fpdf
	err error
}

// Option customizes a dvipdf renderer
type Option func(r *Renderer)

// WithBackground configures the renderer to use the provided color c as
// a background for the resulting PDF.
func WithBackground(c color.Color) Option {
	return func(r *Renderer) {
		r.bkg = c
	}
}

// WithEmbedFonts configures the renderer to embed (or not) the fonts
// inside the resulting PDF.
func WithEmbedFonts(v bool) Option {
	return func(r *Renderer) {
		r.embed = v
	}
}

var (
	stdPaperSizes = map[string]pdf.SizeType{
		"a1":      {Wd: 1683.78, Ht: 2383.94},
		"a2":      {Wd: 1190.55, Ht: 1683.78},
		"a3":      {Wd: 841.89, Ht: 1190.55},
		"a4":      {Wd: 595.28, Ht: 841.89},
		"a5":      {Wd: 420.94, Ht: 595.28},
		"a6":      {Wd: 297.64, Ht: 420.94},
		"a7":      {Wd: 209.76, Ht: 297.64},
		"letter":  {Wd: 612, Ht: 792},
		"legal":   {Wd: 612, Ht: 1008},
		"tabloid": {Wd: 792, Ht: 1224},
	}
)

// WithPaper configures the renderer to use a standard paper size.
func WithPaper(name string) Option {
	return func(r *Renderer) {
		sz, ok := stdPaperSizes[strings.ToLower(name)]
		if !ok {
			panic(fmt.Errorf("unknown paper name %q", name))
		}
		r.cfg.Size = sz
	}
}

// WithPaperSize configures the renderer to use a specific paper size, in points.
func WithPaperSize(width, height float64) Option {
	return func(r *Renderer) {
		r.cfg.Size = pdf.SizeType{Wd: width, Ht: height}
	}
}

// New creates a new PDF renderer, with the provided kpathsea context and
// output writer.
func New(ctx kpath.Context, w io.Writer, opts ...Option) *Renderer {
	rnd := &Renderer{
		ctx:   ctx,
		fonts: make(map[fntkey]struct{}),
		embed: true,
		dpi:   72,
		xoff:  72,
		yoff:  72,
		cfg: pdf.InitType{
			UnitStr: "pt",
			Size:    stdPaperSizes["a4"],
		},
		out: w,
	}

	for _, opt := range opts {
		opt(rnd)
	}

	return rnd
}

// Err returns the current error status of the renderer, if any.
func (pr *Renderer) Err() error {
	return pr.err
}

func (pr *Renderer) setErr(err error) {
	if pr.err != nil {
		return
	}
	pr.err = err
}

// Close closes the PDF document.
func (pr *Renderer) Close() error {
	if pr.err != nil {
		return pr.err
	}

	pr.pdf.Close()

	err := pr.pdf.Output(pr.out)
	if err != nil {
		return fmt.Errorf("could not close output PDF document: %w", err)
	}

	return nil
}

func (pr *Renderer) Init(pre *dvi.CmdPre, post *dvi.CmdPost) {
	pr.pre = *pre
	pr.post = *post
	if pr.dpi == 0 {
		pr.dpi = 72 // FIXME(sbinet): infer from fonts?
	}
	res := pr.dpi
	conv := float32(pr.pre.Num) / 254000.0 * (res / float32(pr.pre.Den))
	pr.tconv = conv
	pr.conv = conv * float32(pr.pre.Mag) / 1000.0

	//	conv = 1/(float32(pre.Num)/float32(pre.Den)*
	//		(float32(pre.Mag)/1000.0)*
	//		(pr.dpi*shrink/254000.0)) + 0.5

	if pr.bkg == nil {
		pr.bkg = color.White
	}

	pr.pdf = pdf.NewCustom(&pr.cfg)
	pr.pdf.SetProducer("Star-Tex", true)
}

func (pr *Renderer) BOP(bop *dvi.CmdBOP) {
	pr.pdf.SetMargins(0, 0, 0)
	pr.pdf.AddPage()

	r, g, b, a := pdfRGBA(color.Black)
	pr.pdf.SetFillColor(r, g, b)
	pr.pdf.SetDrawColor(r, g, b)
	pr.pdf.SetTextColor(r, g, b)
	pr.pdf.SetAlpha(a, "Normal")

	pr.pdf.TransformBegin()
	pr.pdf.TransformTranslate(float64(pr.xoff), float64(pr.yoff))
}

func (pr *Renderer) EOP() {
	pr.pdf.TransformEnd()
}

func (pr *Renderer) DrawGlyph(x, y int32, font dvi.Font, glyph rune, c color.Color) {
	if pr.err != nil {
		return
	}

	if pr.cur != fntkeyFrom(font) {
		pr.font(font)
		size := float64(pr.points(int32(font.Size())))

		style := ""
		pr.pdf.SetFont(font.Name(), style, size)
	}

	r, g, b, _ := pdfRGBA(c)
	pr.pdf.SetTextColor(r, g, b)

	var (
		xx  = float64(pr.points(x))
		yy  = float64(pr.points(y))
		str = string(glyph)
	)
	pr.pdf.Text(xx, yy, str)
}

func (pr *Renderer) DrawRule(x, y, w, h int32, c color.Color) {
	if pr.err != nil {
		return
	}

	xx := float64(pr.points(x))
	yy := float64(pr.points(y))
	ww := float64(pr.points(w))
	hh := float64(pr.points(h))

	r, g, b, _ := pdfRGBA(c)
	pr.pdf.SetFillColor(r, g, b)

	// as per DVI specs dvistd0.pdf, DrawRule is called with:
	//  (x, y): lower left coordinates of rule
	//  h: height of the rule
	//  w: width of the rule
	//
	// fpdf expects (x,y) to be the upper left of the rectangle.

	var (
		x0 = xx
		y0 = yy - hh
	)
	pr.pdf.Rect(x0, y0, ww, hh, "F")
}

func (pr *Renderer) points(v int32) float32 {
	x := pr.conv * float32(v)
	//return roundF32(x / shrink)
	return x
}

// func (pr *Renderer) rulepoints(v int32) int32 {
// 	x := int32(pr.conv * float32(v))
// 	if float32(x) < pr.conv*float32(v) {
// 		return x + 1
// 	}
// 	return x
// }

func fntkeyFrom(fnt dvi.Font) fntkey {
	return fntkey{
		name: fnt.Name(),
		size: fnt.Size(),
	}
}

func (pr *Renderer) font(fnt dvi.Font) {
	key := fntkeyFrom(fnt)
	if _, ok := pr.fonts[key]; ok {
		pr.cur = key
		return
	}

	fnames, err := pr.ctx.FindAll(fnt.Name() + ".pfb")
	if err != nil {
		pr.setErr(err)
		return
	}
	fname := fnames[0]
	f, err := pr.ctx.Open(fname)
	if err != nil {
		pr.setErr(err)
		return
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		pr.setErr(err)
		return
	}

	var (
		encoding []byte
		metrics  *afm.Font
	)
	{
		fnames, err := pr.ctx.FindAll(fnt.Name() + ".afm")
		if err != nil {
			pr.setErr(err)
			return
		}
		fname := fnames[0]
		f, err := pr.ctx.Open(fname)
		if err != nil {
			pr.setErr(err)
			return
		}
		defer f.Close()

		fnt, err := afm.Parse(f)
		if err != nil {
			pr.setErr(err)
			return
		}
		metrics = fnt
	}

	switch metrics.EncodingScheme() {
	case "FontSpecific":
		buf := new(bytes.Buffer)
		for _, ch := range metrics.CharMetrics() {
			fmt.Fprintf(buf, "!%02X U+%04X %s\n", ch.Code(), ch.Code(), ch.Name())
		}
		encoding = buf.Bytes()
	default:
		encoding = codePageEncoding
	}

	pdfKey := pdfFontKey{font: fnt.Name(), embed: pr.embed}
	zdata, jdata, err := getFont(pr.ctx, pdfKey, filepath.Base(fname), raw, encoding)
	if err != nil {
		pr.setErr(err)
		return
	}

	const style = "" // FIXME(sbinet): handle italic/bold ?
	pr.pdf.AddFontFromBytes(key.name, style, jdata, zdata)
	if pr.pdf.Err() {
		pr.setErr(fmt.Errorf("could not add font %q: %w", fnt.Name(), pr.pdf.Error()))
		return
	}

	pr.fonts[key] = struct{}{}
	pr.cur = key
}

type fontsCache struct {
	sync.RWMutex
	cache map[pdfFontKey]pdfFontVal
}

// pdfFontKey represents a PDF font request.
// pdfFontKey needs to know whether the font will be embedded or not,
// as gofpdf.MakeFont will generate different informations.
type pdfFontKey struct {
	font  string
	embed bool
}

type pdfFontVal struct {
	z, j []byte
}

func (c *fontsCache) get(key pdfFontKey) (pdfFontVal, bool) {
	c.RLock()
	defer c.RUnlock()
	v, ok := c.cache[key]
	return v, ok
}

func (c *fontsCache) add(k pdfFontKey, v pdfFontVal) {
	c.Lock()
	defer c.Unlock()
	c.cache[k] = v
}

var pdfFonts = &fontsCache{
	cache: make(map[pdfFontKey]pdfFontVal),
}

func getFont(ctx kpath.Context, key pdfFontKey, name string, font, encoding []byte) (z, j []byte, err error) {
	if v, ok := pdfFonts.get(key); ok {
		return v.z, v.j, nil
	}

	v, err := makeFont(ctx, key, name, font, encoding)
	if err != nil {
		return nil, nil, err
	}
	return v.z, v.j, nil
}

func makeFont(ctx kpath.Context, key pdfFontKey, name string, font, encoding []byte) (val pdfFontVal, err error) {
	tmpdir, err := os.MkdirTemp("", "go-fpdf-makefont-")
	if err != nil {
		return val, err
	}
	defer os.RemoveAll(tmpdir)

	indir := filepath.Join(tmpdir, "input")
	err = os.Mkdir(indir, 0755)
	if err != nil {
		return val, err
	}

	outdir := filepath.Join(tmpdir, "output")
	err = os.Mkdir(outdir, 0755)
	if err != nil {
		return val, err
	}

	var (
		ext   = filepath.Ext(name)
		base  = key.font
		fname = filepath.Join(indir, base+ext)
		cname = filepath.Join(indir, base+".map")
	)

	err = os.WriteFile(fname, font, 0644)
	if err != nil {
		return val, err
	}

	err = os.WriteFile(cname, encoding, 0644)
	if err != nil {
		return val, err
	}

	err = os.WriteFile(filepath.Join(indir, "cp1252.map"), codePageEncoding, 0644)
	if err != nil {
		return val, err
	}

	if ext := filepath.Ext(name); ext == ".pfb" {
		fnames, err := ctx.FindAll(base + ".afm")
		if err != nil {
			return val, err
		}
		f, err := ctx.Open(fnames[0])
		if err != nil {
			return val, err
		}
		defer f.Close()

		raw, err := io.ReadAll(f)
		if err != nil {
			return val, err
		}

		err = os.WriteFile(filepath.Join(indir, base+".afm"), raw, 0644)
		if err != nil {
			return val, err
		}
	}

	err = pdf.MakeFont(fname, cname, outdir, io.Discard, key.embed)
	if err != nil {
		return val, err
	}

	if key.embed {
		z, err := os.ReadFile(filepath.Join(outdir, base+".z"))
		if err != nil {
			return val, err
		}
		val.z = z
	}

	j, err := os.ReadFile(filepath.Join(outdir, base+".json"))
	if err != nil {
		return val, err
	}
	val.j = j

	pdfFonts.add(key, val)

	return val, nil
}

// rgba converts a Go color into a gofpdf 3-tuple int + 1 float64
func pdfRGBA(c color.Color) (int, int, int, float64) {
	r, g, b, a := c.RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8), float64(a) / math.MaxUint16
}
