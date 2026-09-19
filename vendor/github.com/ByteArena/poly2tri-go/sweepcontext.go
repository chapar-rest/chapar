package poly2tri

import (
	"sort"
)

var kAlpha = 0.3

type Edge struct {
	P *Point
	Q *Point
}

func NewEdge(p1, p2 *Point) *Edge {

	actualP := p1
	actualQ := p2

	if p1.Y > p2.Y {
		actualP = p2
		actualQ = p1
	} else if p1.Y == p2.Y {

		if p1.X > p2.X {
			actualP = p2
			actualQ = p1
		} else if p1.X == p2.X {
			panic("poly2tri Invalid Edge constructor: repeated points!")
		}
	}

	res := &Edge{
		P: actualP,
		Q: actualQ,
	}

	res.Q.Edges = append(res.Q.Edges, res)
	return res
}

type Basin struct {
	LeftNode    *Node
	BottomNode  *Node
	RightNode   *Node
	Width       float64
	LeftHighest bool
}

func NewBasin() *Basin {
	return &Basin{
		LeftNode:    nil,
		BottomNode:  nil,
		RightNode:   nil,
		Width:       0.0,
		LeftHighest: false,
	}
}

type EdgeEvent struct {
	ConstrainedEdge *Edge
	Right           bool
}

func NewEdgeEvent() *EdgeEvent {
	return &EdgeEvent{
		ConstrainedEdge: nil,
		Right:           false,
	}
}

type SweepContext struct {
	Triangles []*Triangle
	Map       []*Triangle
	Points    []*Point
	EdgeList  []*Edge
	PMin      *Point
	PMax      *Point
	Front     *AdvancingFront
	Head      *Point
	Tail      *Point
	AfHead    *Node
	AfMiddle  *Node
	AfTail    *Node
	Basin     *Basin
	EdgeEvent *EdgeEvent
}

func NewSweepContext(contour []*Point, cloneArrays bool) *SweepContext {

	var actualPoints []*Point

	if cloneArrays {
		actualPoints = make([]*Point, len(contour))
		for i, point := range contour {
			actualPoints[i] = point.Clone()
		}
	} else {
		actualPoints = contour
	}

	res := &SweepContext{
		Triangles: make([]*Triangle, 0),
		Map:       make([]*Triangle, 0),
		Points:    actualPoints,
		EdgeList:  make([]*Edge, 0),
		PMin:      nil,
		PMax:      nil,
		Front:     nil,
		Head:      nil,
		Tail:      nil,
		AfHead:    nil,
		AfMiddle:  nil,
		AfTail:    nil,
		Basin:     NewBasin(),
		EdgeEvent: NewEdgeEvent(),
	}

	res.InitEdges(res.Points)
	// log.Println("tcx.Points[0]", res.Points[0], res.Points[0].Edges)
	// log.Println("tcx.Points[1]", res.Points[1], res.Points[1].Edges)
	// log.Println("tcx.Points[2]", res.Points[2], res.Points[2].Edges)
	// log.Println("tcx.Points[3]", res.Points[3], res.Points[3].Edges)
	// panic("laaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	return res
}

func (sc *SweepContext) AddHole(polyline []*Point) *SweepContext {
	sc.InitEdges(polyline)

	for _, point := range polyline {
		sc.Points = append(sc.Points, point)
	}

	return sc
}

func (sc *SweepContext) AddHoles(holes [][]*Point) *SweepContext {

	for _, hole := range holes {
		sc.AddHole(hole)
	}

	return sc
}

func (sc *SweepContext) AddPoint(point *Point) *SweepContext {
	sc.Points = append(sc.Points, point)
	return sc
}

func (sc *SweepContext) AddPoints(points []*Point) *SweepContext {
	for _, point := range points {
		sc.AddPoint(point)
	}
	return sc
}

func (sc *SweepContext) Triangulate() *SweepContext {
	Triangulate(sc)
	return sc
}

func (sc *SweepContext) GetBoundingBox() (min *Point, max *Point) {
	return sc.PMin, sc.PMax
}

func (sc *SweepContext) GetTriangles() []*Triangle {
	return sc.Triangles
}

func (sc *SweepContext) GetFront() *AdvancingFront {
	return sc.Front
}

func (sc *SweepContext) PointCount() int {
	return len(sc.Points)
}

func (sc *SweepContext) GetHead() *Point {
	return sc.Head
}

func (sc *SweepContext) SetHead(p1 *Point) {
	sc.Head = p1
}

func (sc *SweepContext) GetTail() *Point {
	return sc.Tail
}

func (sc *SweepContext) SetTail(p1 *Point) {
	sc.Tail = p1
}

func (sc *SweepContext) GetMap() []*Triangle {
	return sc.Map
}

func (sc *SweepContext) InitTriangulation() {

	xmax := sc.Points[0].X
	xmin := sc.Points[0].X
	ymax := sc.Points[0].Y
	ymin := sc.Points[0].Y

	// Calculate bounds
	//var i, len = this.points_.length;
	//for (i = 1; i < len; i++) {
	for _, p := range sc.Points {

		if p.X > xmax {
			xmax = p.X
		}

		if p.X < xmin {
			xmin = p.X
		}

		if p.Y > ymax {
			ymax = p.Y
		}

		if p.Y < ymin {
			ymin = p.Y
		}
	}

	sc.PMin = NewPoint(xmin, ymin)
	sc.PMax = NewPoint(xmax, ymax)

	dx := kAlpha * (xmax - xmin)
	dy := kAlpha * (ymax - ymin)
	sc.Head = NewPoint(xmax+dx, ymin-dy)
	sc.Tail = NewPoint(xmin-dx, ymin-dy)

	// Sort points along y-axis
	sort.Sort(SortablePointsCollection(sc.Points))
}

func (sc *SweepContext) InitEdges(polyline []*Point) {
	length := len(polyline)
	for i := 0; i < length; i++ {
		sc.EdgeList = append(
			sc.EdgeList,
			NewEdge(polyline[i], polyline[(i+1)%length]),
		)
	}
}

func (sc *SweepContext) GetPoint(index int) *Point {
	return sc.Points[index]
}

func (sc *SweepContext) AddToMap(triangle *Triangle) {
	sc.Map = append(sc.Map, triangle)
}

func (sc *SweepContext) LocateNode(point *Point) *Node {
	return sc.Front.LocateNode(point.X)
}

func (sc *SweepContext) CreateAdvancingFront() {

	// Initial triangle
	triangle := NewTriangle(sc.Points[0], sc.Tail, sc.Head)

	sc.Map = append(sc.Map, triangle)

	head := NewNode(triangle.GetPoint(1), triangle)
	middle := NewNode(triangle.GetPoint(0), triangle)
	tail := NewNode(triangle.GetPoint(2), nil)

	sc.Front = NewAdvancingFront(head, tail)

	head.Next = middle
	middle.Next = tail
	middle.Prev = head
	tail.Prev = middle
}

func (sc *SweepContext) RemoveNode(node *Node) {
	// do nothing
}

func (sc *SweepContext) MapTriangleToNodes(t *Triangle) {
	for i := 0; i < 3; i++ {
		if t.GetNeighbor(i) == nil {
			n := sc.Front.LocatePoint(t.PointCW(t.GetPoint(i)))
			if n != nil {
				n.Triangle = t
			}
		}
	}
}

func (sc *SweepContext) RemoveFromMap(triangle *Triangle) {

	for i, t := range sc.Map {
		if t == triangle {
			copy(sc.Map[i:], sc.Map[i+1:])
			sc.Map[len(sc.Map)-1] = nil
			sc.Map = sc.Map[:len(sc.Map)-1]
			break
		}
	}
}

func (sc *SweepContext) MeshClean(triangle *Triangle) {
	// New implementation avoids recursive calls and use a loop instead.
	// Cf. issues # 57, 65 and 69.

	triangles := []*Triangle{triangle}

	for len(triangles) > 0 {

		var t *Triangle
		t, triangles = triangles[len(triangles)-1], triangles[:len(triangles)-1] // pop

		if !t.IsInterior() {
			t.SetInterior(true)
			sc.Triangles = append(sc.Triangles, t)

			for i := 0; i < 3; i++ {
				if !t.ConstrainedEdge[i] {
					triangles = append(triangles, t.GetNeighbor(i))
				}
			}
		}
	}
}
