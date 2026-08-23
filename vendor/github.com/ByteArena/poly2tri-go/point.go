package poly2tri

import (
	"math"
)

type Point struct {
	X     float64
	Y     float64
	Edges []*Edge
}

func NewPoint(x, y float64) *Point {
	return &Point{
		X:     x,
		Y:     y,
		Edges: make([]*Edge, 0),
	}
}

func (p *Point) GetX() float64 {
	return p.X
}

func (p *Point) GetY() float64 {
	return p.Y
}

func (p *Point) String() string {
	return XYString(p)
}

func (p *Point) ToJSON() string {
	panic("poly2tri:Point.ToJSON not implemented")
}

func (p *Point) Clone() *Point {
	return NewPoint(p.X, p.Y)
}

func (p *Point) SetZero() *Point {
	return p.Set(0, 0)
}

func (p *Point) Set(x, y float64) *Point {
	p.X = x
	p.Y = y
	return p
}

func (p *Point) Negate() *Point {
	p.X *= -1
	p.Y *= -1
	return p
}

func (p *Point) Add(n XYInterface) *Point {
	p.X += n.GetX()
	p.Y += n.GetY()
	return p
}

func (p *Point) Sub(n XYInterface) *Point {
	p.X -= n.GetX()
	p.Y -= n.GetY()
	return p
}

func (p *Point) Mul(n XYInterface) *Point {
	p.X *= n.GetX()
	p.Y *= n.GetY()
	return p
}

func (p *Point) Length() float64 {
	return math.Sqrt(p.X*p.X + p.Y*p.Y)
}

func (p *Point) Normalize() float64 {
	len := p.Length()
	p.X /= len
	p.Y /= len
	return len
}

func (p *Point) Equals(p2 XYInterface) bool {
	return XYEquals(p, p2)
}

func PointNegate(p *Point) *Point {
	return p.Clone().Negate()
}

func PointAdd(a, b *Point) *Point {
	return a.Clone().Add(b)
}

func PointSub(a, b *Point) *Point {
	return a.Clone().Sub(b)
}

func PointMul(a, b *Point) *Point {
	return a.Clone().Mul(b)
}

func PointCross(a, b *Point) {
	panic("poly2tri:Point.PointCross not implemented")
}

func PointString(a *Point) string {
	return a.String()
}

func PointCompare(a, b *Point) float64 {
	return XYCompare(a, b)
}

func PointEquals(a, b *Point) bool {
	return XYEquals(a, b)
}

func PointDot(a, b XYInterface) float64 {
	return a.GetX()*b.GetX() + a.GetY()*b.GetY()
}

type SortablePointsCollection []*Point

func (c SortablePointsCollection) Len() int           { return len(c) }
func (c SortablePointsCollection) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c SortablePointsCollection) Less(i, j int) bool { return XYCompare(c[i], c[j]) < 0 }
