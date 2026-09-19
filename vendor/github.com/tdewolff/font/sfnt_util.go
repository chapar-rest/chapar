package font

import "math"

type bboxPather struct {
	XMin, XMax, YMin, YMax float64
}

func (p *bboxPather) MoveTo(x float64, y float64) {
	p.XMin = math.Min(p.XMin, x)
	p.XMax = math.Max(p.XMax, x)
	p.YMin = math.Min(p.YMin, y)
	p.YMax = math.Max(p.YMax, y)
}

func (p *bboxPather) LineTo(x float64, y float64) {
	p.XMin = math.Min(p.XMin, x)
	p.XMax = math.Max(p.XMax, x)
	p.YMin = math.Min(p.YMin, y)
	p.YMax = math.Max(p.YMax, y)
}

func (p *bboxPather) QuadTo(cpx float64, cpy float64, x float64, y float64) {
	p.XMin = math.Min(math.Min(p.XMin, cpx), x)
	p.XMax = math.Max(math.Max(p.XMax, cpx), x)
	p.YMin = math.Min(math.Min(p.YMin, cpy), y)
	p.YMax = math.Max(math.Max(p.YMax, cpy), y)
}

func (p *bboxPather) CubeTo(cpx1 float64, cpy1 float64, cpx2 float64, cpy2 float64, x float64, y float64) {
	p.XMin = math.Min(math.Min(math.Min(p.XMin, cpx1), cpx2), x)
	p.XMax = math.Max(math.Max(math.Max(p.XMax, cpx1), cpx2), x)
	p.YMin = math.Min(math.Min(math.Min(p.YMin, cpy1), cpy2), y)
	p.YMax = math.Max(math.Max(math.Max(p.YMax, cpy1), cpy2), y)
}

func (p *bboxPather) Close() {
}
