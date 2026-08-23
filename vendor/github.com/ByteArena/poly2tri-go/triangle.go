package poly2tri

import (
	"fmt"
)

type Triangle struct {
	Points          []*Point
	Neighbors       []*Triangle
	Interior        bool
	ConstrainedEdge []bool
	DelaunayEdge    []bool
}

func NewTriangle(a, b, c *Point) *Triangle {
	return &Triangle{
		Points:          []*Point{a, b, c},
		Neighbors:       []*Triangle{nil, nil, nil},
		Interior:        false,
		ConstrainedEdge: []bool{false, false, false},
		DelaunayEdge:    []bool{false, false, false},
	}
}

func (t *Triangle) String() string {
	return fmt.Sprintf("[%s%s%s]", t.Points[0], t.Points[1], t.Points[2])
}

func (t *Triangle) GetPoint(index int) *Point {
	return t.Points[index]
}

func (t *Triangle) GetPoints(index int) []*Point {
	return t.Points
}

func (t *Triangle) GetNeighbor(index int) *Triangle {
	if index < len(t.Neighbors) {
		return t.Neighbors[index]
	}

	return nil
}

func (t *Triangle) ContainsPoint(point *Point) bool {
	// comparing references, not values
	return (point == t.Points[0]) || (point == t.Points[1]) || (point == t.Points[2])
}

func (t *Triangle) ContainsEdge(edge *Edge) bool {
	return t.ContainsPoint(edge.P) && t.ContainsPoint(edge.Q)
}

func (t *Triangle) ContainsPoints(p1, p2 *Point) bool {
	return t.ContainsPoint(p1) && t.ContainsPoint(p2)
}

func (t *Triangle) IsInterior() bool {
	return t.Interior
}

func (t *Triangle) SetInterior(interior bool) *Triangle {
	t.Interior = interior
	return t
}

func (t *Triangle) MarkNeighborPointers(p1, p2 *Point, tri *Triangle) {
	points := t.Points

	// Here we are comparing point references, not values

	if (p1 == points[2] && p2 == points[1]) || (p1 == points[1] && p2 == points[2]) {
		t.Neighbors[0] = tri
	} else if (p1 == points[0] && p2 == points[2]) || (p1 == points[2] && p2 == points[0]) {
		t.Neighbors[1] = tri
	} else if (p1 == points[0] && p2 == points[1]) || (p1 == points[1] && p2 == points[0]) {
		t.Neighbors[2] = tri
	} else {
		panic("poly2tri Invalid Triangle.markNeighborPointers() call")
	}
}

func (t *Triangle) MarkNeighbor(tri *Triangle) {
	points := t.Points
	if tri.ContainsPoints(points[1], points[2]) {
		t.Neighbors[0] = tri
		tri.MarkNeighborPointers(points[1], points[2], t)
	} else if tri.ContainsPoints(points[0], points[2]) {
		t.Neighbors[1] = tri
		tri.MarkNeighborPointers(points[0], points[2], t)
	} else if tri.ContainsPoints(points[0], points[1]) {
		t.Neighbors[2] = tri
		tri.MarkNeighborPointers(points[0], points[1], t)
	}
}

func (t *Triangle) ClearNeighbors() {
	t.Neighbors[0] = nil
	t.Neighbors[1] = nil
	t.Neighbors[2] = nil
}

func (t *Triangle) ClearDelaunayEdges() {
	t.DelaunayEdge[0] = false
	t.DelaunayEdge[1] = false
	t.DelaunayEdge[2] = false
}

func (t *Triangle) PointCW(p *Point) *Point {
	// Here we are comparing point references, not values

	if p == t.Points[0] {
		return t.Points[2]
	}

	if p == t.Points[1] {
		return t.Points[0]
	}

	if p == t.Points[2] {
		return t.Points[1]
	}

	return nil
}

func (t *Triangle) PointCCW(p *Point) *Point {

	// Here we are comparing point references, not values

	if p == t.Points[0] {
		return t.Points[1]
	}

	if p == t.Points[1] {
		return t.Points[2]
	}

	if p == t.Points[2] {
		return t.Points[0]
	}

	return nil
}

func (t *Triangle) NeighborCW(p *Point) *Triangle {

	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return t.Neighbors[1]
	}

	if p == t.Points[1] {
		return t.Neighbors[2]
	}

	return t.Neighbors[0]
}

func (t *Triangle) NeighborCCW(p *Point) *Triangle {

	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return t.Neighbors[2]
	}

	if p == t.Points[1] {
		return t.Neighbors[0]
	}

	return t.Neighbors[1]
}

func (t *Triangle) GetConstrainedEdgeCW(p *Point) bool {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return t.ConstrainedEdge[1]
	} else if p == t.Points[1] {
		return t.ConstrainedEdge[2]
	}

	return t.ConstrainedEdge[0]
}

func (t *Triangle) GetConstrainedEdgeCCW(p *Point) bool {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return t.ConstrainedEdge[2]
	} else if p == t.Points[1] {
		return t.ConstrainedEdge[0]
	}

	return t.ConstrainedEdge[1]
}

func (t *Triangle) GetConstrainedEdgeAcross(p *Point) bool {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return t.ConstrainedEdge[0]
	} else if p == t.Points[1] {
		return t.ConstrainedEdge[1]
	}

	return t.ConstrainedEdge[2]
}

func (t *Triangle) SetConstrainedEdgeCW(p *Point, ce bool) {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		t.ConstrainedEdge[1] = ce
	} else if p == t.Points[1] {
		t.ConstrainedEdge[2] = ce
	} else {
		t.ConstrainedEdge[0] = ce
	}
}

func (t *Triangle) SetConstrainedEdgeCCW(p *Point, ce bool) {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		t.ConstrainedEdge[2] = ce
	} else if p == t.Points[1] {
		t.ConstrainedEdge[0] = ce
	} else {
		t.ConstrainedEdge[1] = ce
	}
}

func (t *Triangle) GetDelaunayEdgeCW(p *Point) bool {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return t.DelaunayEdge[1]
	} else if p == t.Points[1] {
		return t.DelaunayEdge[2]
	} else {
		return t.DelaunayEdge[0]
	}
}

func (t *Triangle) GetDelaunayEdgeCCW(p *Point) bool {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return t.DelaunayEdge[2]
	} else if p == t.Points[1] {
		return t.DelaunayEdge[0]
	}

	return t.DelaunayEdge[1]

}

func (t *Triangle) SetDelaunayEdgeCW(p *Point, e bool) {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		t.DelaunayEdge[1] = e
	} else if p == t.Points[1] {
		t.DelaunayEdge[2] = e
	} else {
		t.DelaunayEdge[0] = e
	}
}

func (t *Triangle) SetDelaunayEdgeCCW(p *Point, e bool) {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		t.DelaunayEdge[2] = e
	} else if p == t.Points[1] {
		t.DelaunayEdge[0] = e
	} else {
		t.DelaunayEdge[1] = e
	}
}

func (t *Triangle) NeighborAcross(p *Point) *Triangle {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return t.Neighbors[0]
	} else if p == t.Points[1] {
		return t.Neighbors[1]
	}

	return t.Neighbors[2]
}

func (t *Triangle) OppositePoint(tri *Triangle, p *Point) *Point {
	cw := tri.PointCW(p)
	return t.PointCW(cw)
}

func (t *Triangle) Legalize(opoint *Point, npoint *Point) {

	// Here we are comparing point references, not values
	if opoint == t.Points[0] {
		t.Points[1] = t.Points[0]
		t.Points[0] = t.Points[2]
		t.Points[2] = npoint
	} else if opoint == t.Points[1] {
		t.Points[2] = t.Points[1]
		t.Points[1] = t.Points[0]
		t.Points[0] = npoint
	} else if opoint == t.Points[2] {
		t.Points[0] = t.Points[2]
		t.Points[2] = t.Points[1]
		t.Points[1] = npoint
	} else {
		panic("poly2tri Invalid Triangle.legalize() call")
	}
}

func (t *Triangle) Index(p *Point) int {
	// Here we are comparing point references, not values
	if p == t.Points[0] {
		return 0
	} else if p == t.Points[1] {
		return 1
	} else if p == t.Points[2] {
		return 2
	}

	panic("poly2tri Invalid Triangle.index() call")
}

func (t *Triangle) EdgeIndex(p1 *Point, p2 *Point) int {
	// Here we are comparing point references, not values
	if p1 == t.Points[0] {
		if p2 == t.Points[1] {
			return 2
		} else if p2 == t.Points[2] {
			return 1
		}
	} else if p1 == t.Points[1] {
		if p2 == t.Points[2] {
			return 0
		} else if p2 == t.Points[0] {
			return 2
		}
	} else if p1 == t.Points[2] {
		if p2 == t.Points[0] {
			return 1
		} else if p2 == t.Points[1] {
			return 0
		}
	}
	return -1
}

func (t *Triangle) MarkConstrainedEdgeByIndex(index int) {
	t.ConstrainedEdge[index] = true
}

func (t *Triangle) MarkConstrainedEdgeByEdge(edge *Edge) {
	t.MarkConstrainedEdgeByPoints(edge.P, edge.Q)
}

func (t *Triangle) MarkConstrainedEdgeByPoints(p, q *Point) {

	// Here we are comparing point references, not values
	if (q == t.Points[0] && p == t.Points[1]) || (q == t.Points[1] && p == t.Points[0]) {
		t.ConstrainedEdge[2] = true
	} else if (q == t.Points[0] && p == t.Points[2]) || (q == t.Points[2] && p == t.Points[0]) {
		t.ConstrainedEdge[1] = true
	} else if (q == t.Points[1] && p == t.Points[2]) || (q == t.Points[2] && p == t.Points[1]) {
		t.ConstrainedEdge[0] = true
	}
}
