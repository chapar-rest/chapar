package poly2tri

import (
	"math"
)

func Triangulate(tcx *SweepContext) {
	tcx.InitTriangulation()
	tcx.CreateAdvancingFront()
	// log.Println("tcx.triangles", tcx.Triangles)
	// log.Println("tcx.map", tcx.Map)
	// log.Println("tcx.Points[0]", tcx.Points[0], tcx.Points[0].Edges)
	// log.Println("tcx.Points[1]", tcx.Points[1], tcx.Points[1].Edges)
	// log.Println("tcx.Points[2]", tcx.Points[2], tcx.Points[2].Edges)
	// log.Println("tcx.Points[3]", tcx.Points[3], tcx.Points[3].Edges)

	// panic("laaa")
	// Sweep points; build mesh
	SweepPoints(tcx)
	// Clean up
	FinalizationPolygon(tcx)
}

func SweepPoints(tcx *SweepContext) {
	length := tcx.PointCount()
	for i := 1; i < length; i++ {
		point := tcx.GetPoint(i)
		node := PointEvent(tcx, point)
		edges := point.Edges

		for j := 0; edges != nil && j < len(edges); j++ {
			EdgeEventByEdge(tcx, edges[j], node)
		}
	}
}

func FinalizationPolygon(tcx *SweepContext) {

	// Get an Internal triangle to start with
	t := tcx.GetFront().GetHead().Next.Triangle
	p := tcx.GetFront().GetHead().Next.Point
	for !t.GetConstrainedEdgeCW(p) {
		t = t.NeighborCCW(p)
	}

	// Collect interior triangles constrained by edges
	tcx.MeshClean(t)
}

func PointEvent(tcx *SweepContext, point *Point) *Node {
	node := tcx.LocateNode(point)
	new_node := NewFrontTriangle(tcx, point, node)

	// Only need to check +epsilon since point never have smaller
	// x value than node due to how we fetch nodes from the front
	if point.X <= node.Point.X+(EPSILON) {
		Fill(tcx, node)
	}

	FillAdvancingFront(tcx, new_node)
	return new_node
}

func EdgeEventByEdge(tcx *SweepContext, edge *Edge, node *Node) {
	tcx.EdgeEvent.ConstrainedEdge = edge
	tcx.EdgeEvent.Right = (edge.P.X > edge.Q.X)

	if IsEdgeSideOfTriangle(node.Triangle, edge.P, edge.Q) {
		return
	}

	// For now we will do all needed filling
	// TODO: integrate with flip process might give some better performance
	//       but for now this avoid the issue with cases that needs both flips and fills
	FillEdgeEvent(tcx, edge, node)
	EdgeEventByPoints(tcx, edge.P, edge.Q, node.Triangle, edge.Q)
}

func EdgeEventByPoints(tcx *SweepContext, ep *Point, eq *Point, triangle *Triangle, point *Point) {
	if IsEdgeSideOfTriangle(triangle, ep, eq) {
		return
	}

	p1 := triangle.PointCCW(point)
	o1 := Orient2d(eq, p1, ep)
	if o1 == Orientation["COLLINEAR"] {
		// TODO integrate here changes from C++ version
		// (C++ repo revision 09880a869095 dated March 8, 2011)
		panic("poly2tri EdgeEvent: Collinear not supported!" /*, [eq, p1, ep]*/)
	}

	p2 := triangle.PointCW(point)
	o2 := Orient2d(eq, p2, ep)
	if o2 == Orientation["COLLINEAR"] {
		// TODO integrate here changes from C++ version
		// (C++ repo revision 09880a869095 dated March 8, 2011)
		panic("poly2tri EdgeEvent: Collinear not supported!" /*, [eq, p2, ep]*/)
	}

	if o1 == o2 {
		// Need to decide if we are rotating CW or CCW to get to a triangle
		// that will cross edge
		if o1 == Orientation["CW"] {
			triangle = triangle.NeighborCCW(point)
		} else {
			triangle = triangle.NeighborCW(point)
		}
		EdgeEventByPoints(tcx, ep, eq, triangle, point)
	} else {
		// This triangle crosses constraint so lets flippin start!
		FlipEdgeEvent(tcx, ep, eq, triangle, point)
	}
}

func IsEdgeSideOfTriangle(triangle *Triangle, ep *Point, eq *Point) bool {
	index := triangle.EdgeIndex(ep, eq)

	if index != -1 {
		triangle.MarkConstrainedEdgeByIndex(index)
		t := triangle.GetNeighbor(index)
		if t != nil {
			t.MarkConstrainedEdgeByPoints(ep, eq)
		}

		return true
	}

	return false
}

func NewFrontTriangle(tcx *SweepContext, point *Point, node *Node) *Node {
	triangle := NewTriangle(point, node.Point, node.Next.Point)

	triangle.MarkNeighbor(node.Triangle)
	tcx.AddToMap(triangle)

	new_node := NewNode(point, nil)
	new_node.Next = node.Next
	new_node.Prev = node
	node.Next.Prev = new_node
	node.Next = new_node

	if !Legalize(tcx, triangle) {
		tcx.MapTriangleToNodes(triangle)
	}

	return new_node
}

func Fill(tcx *SweepContext, node *Node) {
	triangle := NewTriangle(node.Prev.Point, node.Point, node.Next.Point)

	// TODO: should copy the constrained_edge value from neighbor triangles
	//       for now constrained_edge values are copied during the legalize
	triangle.MarkNeighbor(node.Prev.Triangle)
	triangle.MarkNeighbor(node.Triangle)

	tcx.AddToMap(triangle)

	// Update the advancing front
	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev

	// If it was legalized the triangle has already been mapped
	if !Legalize(tcx, triangle) {
		tcx.MapTriangleToNodes(triangle)
	}
}

func FillAdvancingFront(tcx *SweepContext, n *Node) {
	// Fill right holes

	node := n.Next
	for node.Next != nil {
		// TODO integrate here changes from C++ version
		// (C++ repo revision acf81f1f1764 dated April 7, 2012)
		if IsAngleObtuse(node.Point, node.Next.Point, node.Prev.Point) {
			break
		}
		Fill(tcx, node)
		node = node.Next
	}

	// Fill left holes
	node = n.Prev
	for node.Prev != nil {
		// TODO integrate here changes from C++ version
		// (C++ repo revision acf81f1f1764 dated April 7, 2012)
		if IsAngleObtuse(node.Point, node.Next.Point, node.Prev.Point) {
			break
		}
		Fill(tcx, node)
		node = node.Prev
	}

	// Fill right basins
	if n.Next != nil && n.Next.Next != nil {
		if IsBasinAngleRight(n) {
			FillBasin(tcx, n)
		}
	}
}

func IsBasinAngleRight(node *Node) bool {
	ax := node.Point.X - node.Next.Next.Point.X
	ay := node.Point.Y - node.Next.Next.Point.Y
	if ay < 0 {
		panic("unordered y")
	}
	return (ax >= 0 || math.Abs(ax) < ay)
}

func Legalize(tcx *SweepContext, t *Triangle) bool {
	// To legalize a triangle we start by finding if any of the three edges
	// violate the Delaunay condition
	for i := 0; i < 3; i++ {
		if t.DelaunayEdge[i] {
			continue
		}

		ot := t.GetNeighbor(i)
		if ot != nil {
			p := t.GetPoint(i)
			op := ot.OppositePoint(t, p)
			oi := ot.Index(op)

			// If this is a Constrained Edge or a Delaunay Edge(only during recursive legalization)
			// then we should not try to legalize
			if ot.ConstrainedEdge[oi] || ot.DelaunayEdge[oi] {
				t.ConstrainedEdge[i] = ot.ConstrainedEdge[oi]
				continue
			}

			inside := InCircle(p, t.PointCCW(p), t.PointCW(p), op)
			if inside {
				// Lets mark this shared edge as Delaunay
				t.DelaunayEdge[i] = true
				ot.DelaunayEdge[oi] = true

				// Lets rotate shared edge one vertex CW to legalize it
				RotateTrianglePair(t, p, ot, op)

				// We now got one valid Delaunay Edge shared by two triangles
				// This gives us 4 new edges to check for Delaunay

				// Make sure that triangle to node mapping is done only one time for a specific triangle
				not_legalized := !Legalize(tcx, t)
				if not_legalized {
					tcx.MapTriangleToNodes(t)
				}

				not_legalized = !Legalize(tcx, ot)
				if not_legalized {
					tcx.MapTriangleToNodes(ot)
				}
				// Reset the Delaunay edges, since they only are valid Delaunay edges
				// until we add a new triangle or point.
				// XXX: need to think about this. Can these edges be tried after we
				//      return to previous recursive level?
				t.DelaunayEdge[i] = false
				ot.DelaunayEdge[oi] = false

				// If triangle have been legalized no need to check the other edges since
				// the recursive legalization will handles those so we can end here.
				return true
			}
		}
	}
	return false
}

func InCircle(pa, pb, pc, pd *Point) bool {
	adx := pa.X - pd.X
	ady := pa.Y - pd.Y
	bdx := pb.X - pd.X
	bdy := pb.Y - pd.Y

	adxbdy := adx * bdy
	bdxady := bdx * ady
	oabd := adxbdy - bdxady
	if oabd <= 0 {
		return false
	}

	cdx := pc.X - pd.X
	cdy := pc.Y - pd.Y

	cdxady := cdx * ady
	adxcdy := adx * cdy
	ocad := cdxady - adxcdy
	if ocad <= 0 {
		return false
	}

	bdxcdy := bdx * cdy
	cdxbdy := cdx * bdy

	alift := adx*adx + ady*ady
	blift := bdx*bdx + bdy*bdy
	clift := cdx*cdx + cdy*cdy

	det := alift*(bdxcdy-cdxbdy) + blift*ocad + clift*oabd
	return det > 0
}

func RotateTrianglePair(t *Triangle, p *Point, ot *Triangle, op *Point) {
	n1 := t.NeighborCCW(p)
	n2 := t.NeighborCW(p)
	n3 := ot.NeighborCCW(op)
	n4 := ot.NeighborCW(op)

	ce1 := t.GetConstrainedEdgeCCW(p)
	ce2 := t.GetConstrainedEdgeCW(p)
	ce3 := ot.GetConstrainedEdgeCCW(op)
	ce4 := ot.GetConstrainedEdgeCW(op)

	de1 := t.GetDelaunayEdgeCCW(p)
	de2 := t.GetDelaunayEdgeCW(p)
	de3 := ot.GetDelaunayEdgeCCW(op)
	de4 := ot.GetDelaunayEdgeCW(op)

	t.Legalize(p, op)
	ot.Legalize(op, p)

	// Remap delaunay_edge
	ot.SetDelaunayEdgeCCW(p, de1)
	t.SetDelaunayEdgeCW(p, de2)
	t.SetDelaunayEdgeCCW(op, de3)
	ot.SetDelaunayEdgeCW(op, de4)

	// Remap constrained_edge
	ot.SetConstrainedEdgeCCW(p, ce1)
	t.SetConstrainedEdgeCW(p, ce2)
	t.SetConstrainedEdgeCCW(op, ce3)
	ot.SetConstrainedEdgeCW(op, ce4)

	// Remap neighbors
	// XXX: might optimize the markNeighbor by keeping track of
	//      what side should be assigned to what neighbor after the
	//      rotation. Now mark neighbor does lots of testing to find
	//      the right side.
	t.ClearNeighbors()
	ot.ClearNeighbors()
	if n1 != nil {
		ot.MarkNeighbor(n1)
	}
	if n2 != nil {
		t.MarkNeighbor(n2)
	}
	if n3 != nil {
		t.MarkNeighbor(n3)
	}
	if n4 != nil {
		ot.MarkNeighbor(n4)
	}
	t.MarkNeighbor(ot)
}

func FillBasin(tcx *SweepContext, node *Node) {
	if Orient2d(node.Point, node.Next.Point, node.Next.Next.Point) == Orientation["CCW"] {
		tcx.Basin.LeftNode = node.Next.Next
	} else {
		tcx.Basin.LeftNode = node.Next
	}

	// Find the bottom and right node
	tcx.Basin.BottomNode = tcx.Basin.LeftNode
	for tcx.Basin.BottomNode.Next != nil && tcx.Basin.BottomNode.Point.Y >= tcx.Basin.BottomNode.Next.Point.Y {
		tcx.Basin.BottomNode = tcx.Basin.BottomNode.Next
	}
	if tcx.Basin.BottomNode == tcx.Basin.LeftNode {
		// No valid basin
		return
	}

	tcx.Basin.RightNode = tcx.Basin.BottomNode
	for tcx.Basin.RightNode.Next != nil && tcx.Basin.RightNode.Point.Y < tcx.Basin.RightNode.Next.Point.Y {
		tcx.Basin.RightNode = tcx.Basin.RightNode.Next
	}
	if tcx.Basin.RightNode == tcx.Basin.BottomNode {
		// No valid basins
		return
	}

	tcx.Basin.Width = tcx.Basin.RightNode.Point.X - tcx.Basin.LeftNode.Point.X
	tcx.Basin.LeftHighest = tcx.Basin.LeftNode.Point.Y > tcx.Basin.RightNode.Point.Y

	FillBasinReq(tcx, tcx.Basin.BottomNode)
}

func FillBasinReq(tcx *SweepContext, node *Node) {
	// if shallow stop filling
	if IsShallow(tcx, node) {
		return
	}

	Fill(tcx, node)

	if node.Prev == tcx.Basin.LeftNode && node.Next == tcx.Basin.RightNode {
		return
	} else if node.Prev == tcx.Basin.LeftNode {
		o := Orient2d(node.Point, node.Next.Point, node.Next.Next.Point)
		if o == Orientation["CW"] {
			return
		}
		node = node.Next
	} else if node.Next == tcx.Basin.RightNode {
		o := Orient2d(node.Point, node.Prev.Point, node.Prev.Prev.Point)
		if o == Orientation["CCW"] {
			return
		}
		node = node.Prev
	} else {
		// Continue with the neighbor node with lowest Y value
		if node.Prev.Point.Y < node.Next.Point.Y {
			node = node.Prev
		} else {
			node = node.Next
		}
	}

	FillBasinReq(tcx, node)
}

func IsShallow(tcx *SweepContext, node *Node) bool {
	var height float64
	if tcx.Basin.LeftHighest {
		height = tcx.Basin.LeftNode.Point.Y - node.Point.Y
	} else {
		height = tcx.Basin.RightNode.Point.Y - node.Point.Y
	}

	// if shallow stop filling
	if tcx.Basin.Width > height {
		return true
	}

	return false
}

func FillEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	if tcx.EdgeEvent.Right {
		FillRightAboveEdgeEvent(tcx, edge, node)
	} else {
		FillLeftAboveEdgeEvent(tcx, edge, node)
	}
}

func FillRightAboveEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	for node.Next.Point.X < edge.P.X {
		// Check if next node is below the edge
		if Orient2d(edge.Q, node.Next.Point, edge.P) == Orientation["CCW"] {
			FillRightBelowEdgeEvent(tcx, edge, node)
		} else {
			node = node.Next
		}
	}
}

func FillRightBelowEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	if node.Point.X < edge.P.X {
		if Orient2d(node.Point, node.Next.Point, node.Next.Next.Point) == Orientation["CCW"] {
			// Concave
			FillRightConcaveEdgeEvent(tcx, edge, node)
		} else {
			// Convex
			FillRightConvexEdgeEvent(tcx, edge, node)
			// Retry this one
			FillRightBelowEdgeEvent(tcx, edge, node)
		}
	}
}

func FillRightConcaveEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	Fill(tcx, node.Next)

	if node.Next.Point != edge.P {
		// Next above or below edge?
		if Orient2d(edge.Q, node.Next.Point, edge.P) == Orientation["CCW"] {
			// Below
			if Orient2d(node.Point, node.Next.Point, node.Next.Next.Point) == Orientation["CCW"] {
				// Next is concave
				FillRightConcaveEdgeEvent(tcx, edge, node)
			} else {
				// Next is convex
			}
		}
	}
}

func FillRightConvexEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	// Next concave or convex?
	if Orient2d(node.Next.Point, node.Next.Next.Point, node.Next.Next.Next.Point) == Orientation["CCW"] {
		// Concave
		FillRightConcaveEdgeEvent(tcx, edge, node.Next)
	} else {
		// Convex
		// Next above or below edge?
		if Orient2d(edge.Q, node.Next.Next.Point, edge.P) == Orientation["CCW"] {
			// Below
			FillRightConvexEdgeEvent(tcx, edge, node.Next)
		} else {
			// Above
		}
	}
}

func FillLeftAboveEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	for node.Prev.Point.X > edge.P.X {
		// Check if next node is below the edge
		if Orient2d(edge.Q, node.Prev.Point, edge.P) == Orientation["CW"] {
			FillLeftBelowEdgeEvent(tcx, edge, node)
		} else {
			node = node.Prev
		}
	}
}

func FillLeftBelowEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	if node.Point.X > edge.P.X {
		if Orient2d(node.Point, node.Prev.Point, node.Prev.Prev.Point) == Orientation["CW"] {
			// Concave
			FillLeftConcaveEdgeEvent(tcx, edge, node)
		} else {
			// Convex
			FillLeftConvexEdgeEvent(tcx, edge, node)
			// Retry this one
			FillLeftBelowEdgeEvent(tcx, edge, node)
		}
	}
}

func FillLeftConvexEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	// Next concave or convex?
	if Orient2d(node.Prev.Point, node.Prev.Prev.Point, node.Prev.Prev.Prev.Point) == Orientation["CW"] {
		// Concave
		FillLeftConcaveEdgeEvent(tcx, edge, node.Prev)
	} else {
		// Convex
		// Next above or below edge?
		if Orient2d(edge.Q, node.Prev.Prev.Point, edge.P) == Orientation["CW"] {
			// Below
			FillLeftConvexEdgeEvent(tcx, edge, node.Prev)
		} else {
			// Above
		}
	}
}

func FillLeftConcaveEdgeEvent(tcx *SweepContext, edge *Edge, node *Node) {
	Fill(tcx, node.Prev)
	if node.Prev.Point != edge.P {
		// Next above or below edge?
		if Orient2d(edge.Q, node.Prev.Point, edge.P) == Orientation["CW"] {
			// Below
			if Orient2d(node.Point, node.Prev.Point, node.Prev.Prev.Point) == Orientation["CW"] {
				// Next is concave
				FillLeftConcaveEdgeEvent(tcx, edge, node)
			} else {
				// Next is convex
			}
		}
	}
}

func FlipEdgeEvent(tcx *SweepContext, ep *Point, eq *Point, t *Triangle, p *Point) {
	ot := t.NeighborAcross(p)
	if ot == nil {
		panic("FLIP failed due to missing triangle!")
	}

	op := ot.OppositePoint(t, p)

	// Additional check from Java version (see issue #88)
	if t.GetConstrainedEdgeAcross(p) {
		//index := t.Index(p)
		panic("poly2tri Intersecting Constraints" /*, [p, op, t.getPoint((index + 1) % 3), t.getPoint((index + 2) % 3)]*/)
	}

	if InScanArea(p, t.PointCCW(p), t.PointCW(p), op) {

		// Lets rotate shared edge one vertex CW
		RotateTrianglePair(t, p, ot, op)
		tcx.MapTriangleToNodes(t)
		tcx.MapTriangleToNodes(ot)

		// XXX: in the original C++ code for the next 2 lines, we are
		// comparing point values (and not pointers). In this JavaScript
		// code, we are comparing point references (pointers). This works
		// because we can't have 2 different points with the same values.
		// But to be really equivalent, we should use "Point.equals" here.
		if p == eq && op == ep {
			if eq == tcx.EdgeEvent.ConstrainedEdge.Q && ep == tcx.EdgeEvent.ConstrainedEdge.P {
				t.MarkConstrainedEdgeByPoints(ep, eq)
				ot.MarkConstrainedEdgeByPoints(ep, eq)
				Legalize(tcx, t)
				Legalize(tcx, ot)
			} else {
				// XXX: I think one of the triangles should be legalized here?
			}
		} else {
			o := Orient2d(eq, op, ep)
			t := NextFlipTriangle(tcx, o, t, ot, p, op)
			FlipEdgeEvent(tcx, ep, eq, t, p)
		}
	} else {
		newP := NextFlipPoint(ep, eq, ot, op)
		FlipScanEdgeEvent(tcx, ep, eq, t, ot, newP)
		EdgeEventByPoints(tcx, ep, eq, t, p)
	}
}

func NextFlipTriangle(tcx *SweepContext, o int, t *Triangle, ot *Triangle, p *Point, op *Point) *Triangle {
	var edge_index int

	if o == Orientation["CCW"] {
		// ot is not crossing edge after flip
		edge_index = ot.EdgeIndex(p, op)
		ot.DelaunayEdge[edge_index] = true
		Legalize(tcx, ot)
		ot.ClearDelaunayEdges()
		return t
	}

	// t is not crossing edge after flip
	edge_index = t.EdgeIndex(p, op)

	t.DelaunayEdge[edge_index] = true
	Legalize(tcx, t)
	t.ClearDelaunayEdges()
	return ot
}

func NextFlipPoint(ep *Point, eq *Point, ot *Triangle, op *Point) *Point {
	o2d := Orient2d(eq, op, ep)
	if o2d == Orientation["CW"] {
		// Right
		return ot.PointCCW(op)
	} else if o2d == Orientation["CCW"] {
		// Left
		return ot.PointCW(op)
	} else {
		panic("poly2tri [Unsupported] nextFlipPoint: opposing point on constrained edge!" /*, [eq, op, ep]*/)
	}
}

func FlipScanEdgeEvent(tcx *SweepContext, ep *Point, eq *Point, flip_triangle *Triangle, t *Triangle, p *Point) {
	// TODO
	ot := t.NeighborAcross(p)
	if ot == nil {
		panic("FLIP failed due to missing triangle")
	}

	op := ot.OppositePoint(t, p)

	if InScanArea(eq, flip_triangle.PointCCW(eq), flip_triangle.PointCW(eq), op) {
		// flip with new edge op.eq
		FlipEdgeEvent(tcx, eq, op, ot, op)
	} else {
		newP := NextFlipPoint(ep, eq, ot, op)
		FlipScanEdgeEvent(tcx, ep, eq, flip_triangle, ot, newP)
	}
}
