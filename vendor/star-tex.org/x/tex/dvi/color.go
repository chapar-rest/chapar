// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dvi

import (
	"bufio"
	"bytes"
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"

	"star-tex.org/x/tex/kpath"
)

type colorHandler interface {
	Color() color.Color
	Background() color.Color
}

type nilColorHandler struct{}

func (nilColorHandler) Color() color.Color      { return color.Black }
func (nilColorHandler) Background() color.Color { return color.White }

var _ colorHandler = (*nilColorHandler)(nil)

// ColorHandler handles special DVI commands related with colors.
type ColorHandler struct {
	stack []color.Color
	r     colorReader
	names map[string]color.Color
}

var (
	_ Handler      = (*ColorHandler)(nil)
	_ colorHandler = (*ColorHandler)(nil)
)

func NewColorHandler(ktx kpath.Context) *ColorHandler {
	ch := &ColorHandler{
		stack: []color.Color{
			color.White,
			color.Black,
		},
	}
	ch.names = newColorNames(ktx, &ch.r)

	return ch
}

// Color returns the current color from the ColorHandler.
func (ch *ColorHandler) Color() color.Color {
	return ch.stack[len(ch.stack)-1]
}

// Background returns the current color background from the ColorHandler.
func (ch *ColorHandler) Background() color.Color {
	return ch.stack[0]
}

var (
	colorPrefix = []byte("color ")
	bkgPrefix   = []byte("background ")

	colorPushPrefix = []byte("color push ")
	colorPopPrefix  = []byte("color pop")
)

// Handle handles a special DVI command.
func (ch *ColorHandler) Handle(p []byte) error {
	if bytes.HasPrefix(p, colorPrefix) {
		return ch.handleColor(p)
	}

	if bytes.HasPrefix(p, bkgPrefix) {
		return ch.handleBkg(p)
	}

	return ErrSkipHandler
}

func (ch *ColorHandler) handleColor(p []byte) error {
	switch {
	case bytes.HasPrefix(p, colorPushPrefix):
		ch.stack = append(ch.stack, ch.colorFrom(p[len(colorPushPrefix):]))
	case bytes.HasPrefix(p, colorPopPrefix):
		ch.stack = ch.stack[:len(ch.stack)-1]
	default:
		// reset stack.
		ch.stack[0] = ch.colorFrom(p[6:])
		ch.stack = ch.stack[:1]
	}
	return nil
}

func (ch *ColorHandler) handleBkg(p []byte) error {
	ch.stack[0] = ch.colorFrom(p[len(bkgPrefix):])
	return nil
}

func (ch *ColorHandler) colorFrom(p []byte) color.Color {
	s := string(p)
	switch {
	case s == "Black":
		return color.Black

	case s == "White":
		return color.White

	case strings.HasPrefix(s, "gray "):
		ch.r.reset(p[5:])
		v := ch.r.f64()
		return color.Gray{Y: v}

	case strings.HasPrefix(s, "rgb "):
		ch.r.reset(p[4:])
		r := ch.r.f64()
		g := ch.r.f64()
		b := ch.r.f64()
		return color.RGBA{R: r, G: g, B: b, A: 255}

	case strings.HasPrefix(s, "Gray "):
		ch.r.reset(p[5:])
		v := ch.r.u8()
		return color.Gray{Y: v}

	case strings.HasPrefix(s, "RGB "):
		ch.r.reset(p[4:])
		r := ch.r.u8()
		g := ch.r.u8()
		b := ch.r.u8()
		return color.RGBA{R: r, G: g, B: b, A: 255}

	case strings.HasPrefix(s, "HTML "):
		ch.r.reset(p[5:])
		r, g, b := ch.r.hex()
		return color.RGBA{R: r, G: g, B: b, A: 255}

	case strings.HasPrefix(s, "cmy "):
		ch.r.reset(p[4:])
		c := ch.r.f64()
		m := ch.r.f64()
		y := ch.r.f64()
		return color.CMYK{C: c, M: m, Y: y}

	case strings.HasPrefix(s, "cmyk "):
		ch.r.reset(p[5:])
		c := ch.r.f64()
		m := ch.r.f64()
		y := ch.r.f64()
		k := ch.r.f64()
		return color.CMYK{C: c, M: m, Y: y, K: k}

	case strings.HasPrefix(s, "hsb "):
		ch.r.reset(p[4:])
		return ch.r.hsb()

	case strings.HasPrefix(s, "HSB "):
		ch.r.reset(p[4:])
		return ch.r.hsbi()

	default:
		// color name from a "well-known" file.
		name := strings.TrimSpace(s)
		c, ok := ch.names[name]
		if !ok {
			panic(fmt.Errorf("could not find color %q", name))
		}
		return c
	}
}

type colorReader struct {
	p []byte
	c int
}

func (cr *colorReader) reset(p []byte) {
	cr.p = p
	cr.c = 0
}

func (cr *colorReader) token() []byte {
	var (
		beg = cr.c
		end = cr.c
	)
	i := bytes.Index(cr.p[cr.c:], []byte(" "))
	switch i {
	case -1:
		end = len(cr.p)
		cr.c = end
	default:
		end = cr.c + i
		cr.c = end + 1
	}
	return cr.p[beg:end]
}

func (cr *colorReader) f64() uint8 {
	tok := cr.token()
	v, err := strconv.ParseFloat(string(tok), 64)
	if err != nil {
		panic(err) // FIXME(sbinet): warn only?
	}
	return toU8(v)
}

func (cr *colorReader) u8() uint8 {
	tok := cr.token()
	v, err := strconv.ParseInt(string(tok), 10, 8)
	if err != nil {
		panic(err) // FIXME(sbinet): warn only?
	}
	return uint8(v)
}

func (cr *colorReader) hex() (uint8, uint8, uint8) {
	tok := cr.token()
	b, err := strconv.ParseInt(string(tok), 16, 64)
	if err != nil {
		panic(err) // FIXME(sbinet): warn only?
	}
	r := b / 65536
	g := b / 256
	b -= g * 256
	g -= r * 256
	return uint8(r), uint8(g), uint8(b)
}

func (cr *colorReader) hsb() color.Color {
	data := []string{
		string(cr.token()),
		string(cr.token()),
		string(cr.token()),
	}
	return colorFrom("hsb", data)
}

func (cr *colorReader) hsbi() color.Color {
	data := []string{
		string(cr.token()),
		string(cr.token()),
		string(cr.token()),
	}
	return colorFrom("HSB", data)
}

var colorFiles = []string{
	"latex/xcolor/xcolor.sty",
	"latex/graphics/dvipsnam.def",
	"latex/xcolor/svgnam.def",
	"latex/xcolor/x11nam.def",
}

func newColorNames(ktx kpath.Context, cr *colorReader) map[string]color.Color {
	names := make(map[string]color.Color)
	for _, name := range colorFiles {
		fname, err := ktx.Find(name)
		if err != nil {
			panic(err) // FIXME(sbinet): warn only ?
		}
		f, err := ktx.Open(fname)
		if err != nil {
			panic(err) // FIXME(sbinet): warn only ?
		}
		defer f.Close()

		colorNamesFrom(cr, f, fname, &names)
	}
	return names
}

func colorNamesFrom(cr *colorReader, f io.Reader, fname string, db *map[string]color.Color) {
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		switch {
		case bytes.HasPrefix(line, []byte(`\definecolorset`)):
			toks := brackets(string(line))
			models := strings.Split(toks[1], "/")
		definecolorset:
			for sc.Scan() {
				text := sc.Text()
				line := strings.Trim(text, ";%{ }")
				beg := strings.Index(line, ",")
				end := strings.Index(line, "/")
				name := strings.TrimSpace(line[:beg])
				data := strings.TrimSpace(line[beg+1 : end])
				(*db)[name] = colorFrom(models[0], strings.Split(data, ","))
				if strings.Contains(text, "}") {
					break definecolorset
				}
			}

		case bytes.HasPrefix(line, []byte(`\preparecolorset`)):
			// FIXME(sbinet): handle different models rgb/cmy/ ?
		preparecolorset:
			for sc.Scan() {
				line := strings.Trim(sc.Text(), ";%}")
				if strings.HasPrefix(line, `\endinput`) {
					break preparecolorset
				}
				toks := strings.Split(line, ",")
				r := atof(toks[1])
				g := atof(toks[2])
				b := atof(toks[2])
				(*db)[toks[0]] = color.RGBA{toU8(r), toU8(g), toU8(b), 255}
			}

		case bytes.HasPrefix(line, []byte(`\DefineNamedColor`)):
			// line should be of the form:
			// \DefineNamedColor{named}{GreenYellow}   {cmyk}{0.15,0,0.69,0}
			toks := brackets(string(line[len(`\DefineNamedColor`):]))
			name := toks[1]
			model := toks[2]
			toks = strings.Split(toks[3], ",")
			switch model {
			case "cmyk":
				c := atof(strings.TrimSpace(toks[0]))
				m := atof(strings.TrimSpace(toks[1]))
				y := atof(strings.TrimSpace(toks[2]))
				k := atof(strings.TrimSpace(toks[3]))
				(*db)[name] = color.CMYK{toU8(c), toU8(m), toU8(y), toU8(k)}
			case "rgb":
				r := atof(strings.TrimSpace(toks[1]))
				g := atof(strings.TrimSpace(toks[2]))
				b := atof(strings.TrimSpace(toks[3]))
				(*db)[name] = color.RGBA{toU8(r), toU8(g), toU8(b), 255}
			default:
				panic(fmt.Errorf("unknown color model %q in %q", model, line))
			}
		}
	}

	if err := sc.Err(); err != nil {
		panic(fmt.Errorf("could not scan %q: %+v", fname, err))
	}
}

func colorFrom(model string, data []string) color.Color {
	switch model {
	case "rgb":
		r := atof(data[0])
		g := atof(data[1])
		b := atof(data[2])
		return color.RGBA{R: toU8(r), G: toU8(g), B: toU8(b), A: 255}

	case "hsb", "HSB":
		hu := atof(data[0])
		sa := atof(data[1])
		br := atof(data[2])
		if model == "HSB" {
			hu /= 255
			sa /= 255
			br /= 255
		}

		var (
			r float64
			g float64
			b float64

			i = int(6 * hu)
			f = 6*hu - float64(i)
		)
		switch i {
		case 0:
			r = br * (1 - sa*0)
			g = br * (1 - sa*(1-f))
			b = br * (1 - sa*1)
		case 1:
			r = br * (1 - sa*f)
			g = br * (1 - sa*0)
			b = br * (1 - sa*1)
		case 2:
			r = br * (1 - sa*1)
			g = br * (1 - sa*0)
			b = br * (1 - sa*(1-f))
		case 3:
			r = br * (1 - sa*1)
			g = br * (1 - sa*f)
			b = br * (1 - sa*0)
		case 4:
			r = br * (1 - sa*(1-f))
			g = br * (1 - sa*1)
			b = br * (1 - sa*0)
		case 5:
			r = br * (1 - sa*0)
			g = br * (1 - sa*1)
			b = br * (1 - sa*f)
		default:
			r = br * (1 - sa*0)
			g = br * (1 - sa*1)
			b = br * (1 - sa*1)
		}
		return color.RGBA{R: toU8(r), G: toU8(g), B: toU8(b), A: 255}

	case "cmyk":
		c := atof(data[0])
		m := atof(data[1])
		y := atof(data[2])
		k := atof(data[3])
		return color.CMYK{C: toU8(c), M: toU8(m), Y: toU8(y), K: toU8(k)}

	case "gray":
		y := atof(data[0])
		return color.Gray{Y: toU8(y)}

	default:
		panic(fmt.Errorf("unknown color model=%q with data=%q", model, data))
	}
}

func toU8(v float64) uint8 {
	return uint8(255*v + 0.5)
}

func atof(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err) // FIXME(sbinet): warn only?
	}
	return v
}

func brackets(s string) []string {
	var (
		vs []string
		o  strings.Builder
	)
	for _, v := range s {
		switch v {
		case '{', '}':
			if o.Len() > 0 {
				vs = append(vs, o.String())
			}
			o.Reset()
		case ' ':
			// ignore.
		default:
			o.WriteString(string(v))
		}
	}
	return vs
}
