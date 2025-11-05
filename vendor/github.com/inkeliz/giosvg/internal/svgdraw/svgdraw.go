package svgdraw

import (
	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/inkeliz/giosvg/internal/svgparser"
	"math"
)

type Driver struct {
	Ops   *op.Ops
	Scale float32
	index int
}

func (d *Driver) SetupDrawers(willFill, willStroke bool) (f svgparser.Filler, s svgparser.Stroker) {
	if willFill {
		f = &filler{pathOp: d.Ops}
	}
	if willStroke {
		s = &stroker{pathOp: d.Ops, scale: d.Scale}
	}
	return f, s
}

type filler struct {
	path   *clip.Path
	pathOp *op.Ops
}

func (f *filler) Clear() {}

func (f *filler) Start(a f32.Point) {
	if f.path == nil {
		f.path = new(clip.Path)
		f.path.Begin(f.pathOp)
	}

	f.path.MoveTo(a)
}

func (f *filler) Line(b f32.Point) {
	f.path.LineTo(b)
}

func (f *filler) QuadBezier(b, c f32.Point) {
	f.path.QuadTo(b, c)
}

func (f *filler) CubeBezier(b, c, d f32.Point) {
	f.path.CubeTo(b, c, d)
}

func (f *filler) Stop(closeLoop bool) {
	if f.path != nil {
		f.path.Close()
	}
}

func (f *filler) Draw(color svgparser.Pattern, opacity float64) {
	defer clip.Outline{Path: f.path.End()}.Op().Push(f.pathOp).Pop()

	switch c := color.(type) {
	case svgparser.CurrentColor:
		paint.PaintOp{}.Add(f.pathOp)
	case svgparser.PlainColor:
		if opacity < 1 {
			c.NRGBA.A = uint8(math.Round(256 * opacity))
		}
		paint.ColorOp{Color: c.NRGBA}.Add(f.pathOp)
		paint.PaintOp{}.Add(f.pathOp)
	}
}

func (f *filler) SetWinding(useNonZeroWinding bool) {}

type stroker struct {
	path   *clip.Path
	pathOp *op.Ops
	Width  float32
	scale  float32
}

func (s *stroker) Clear() {}

func (s *stroker) Start(a f32.Point) {
	if s.path == nil {
		s.path = new(clip.Path)
		s.path.Begin(s.pathOp)
	}

	s.path.MoveTo(a)
}

func (s *stroker) Line(b f32.Point) {
	s.path.LineTo(b)
}

func (s *stroker) QuadBezier(b, c f32.Point) {
	s.path.QuadTo(b, c)
}

func (s *stroker) CubeBezier(b, c, d f32.Point) {
	s.path.CubeTo(b, c, d)
}

func (s *stroker) Stop(closeLoop bool) {
	if s.path != nil && closeLoop {
		s.path.Close()
	}
}

func (s *stroker) Draw(color svgparser.Pattern, opacity float64) {
	defer clip.Stroke{Path: s.path.End(), Width: s.Width}.Op().Push(s.pathOp).Pop()

	switch c := color.(type) {
	case svgparser.CurrentColor:
		paint.PaintOp{}.Add(s.pathOp)
	case svgparser.PlainColor:
		if opacity < 1 {
			c.NRGBA.A = uint8(math.Round(256 * opacity))
		}
		paint.ColorOp{Color: c.NRGBA}.Add(s.pathOp)
		paint.PaintOp{}.Add(s.pathOp)
	}
}

func (s *stroker) SetStrokeOptions(options svgparser.StrokeOptions) {
	s.Width = options.LineWidth * s.scale
}
