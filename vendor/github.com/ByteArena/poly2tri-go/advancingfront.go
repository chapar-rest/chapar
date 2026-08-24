package poly2tri

type Node struct {
	Point    *Point
	Triangle *Triangle
	Next     *Node
	Prev     *Node
	Value    float64
}

func NewNode(p *Point, t *Triangle) *Node {
	return &Node{
		Point:    p,
		Triangle: t,
		Next:     nil,
		Prev:     nil,
		Value:    p.X,
	}
}

type AdvancingFront struct {
	Head       *Node
	Tail       *Node
	SearchNode *Node
}

func NewAdvancingFront(head, tail *Node) *AdvancingFront {
	return &AdvancingFront{
		Head:       head,
		Tail:       tail,
		SearchNode: head,
	}
}

func (af *AdvancingFront) GetHead() *Node {
	return af.Head
}

func (af *AdvancingFront) SetHead(node *Node) {
	af.Head = node
}

func (af *AdvancingFront) GetTail() *Node {
	return af.Tail
}

func (af *AdvancingFront) SetTail(node *Node) {
	af.Tail = node
}

func (af *AdvancingFront) GetSearch() *Node {
	return af.SearchNode
}

func (af *AdvancingFront) SetSearch(node *Node) {
	af.SearchNode = node
}

func (af *AdvancingFront) FindSearchNode(x float64) *Node {
	return af.GetSearch()
}

func (af *AdvancingFront) LocateNode(x float64) *Node {

	node := af.SearchNode

	if x < node.Value {
		node = node.Prev
		for node != nil {
			if x >= node.Value {
				af.SearchNode = node
				return node
			}
			node = node.Prev
		}
	} else {
		node = node.Next
		for node != nil {
			if x < node.Value {
				af.SearchNode = node.Prev
				return node.Prev
			}

			node = node.Next
		}
	}
	return nil
}

func (af *AdvancingFront) LocatePoint(point *Point) *Node {

	px := point.X
	node := af.FindSearchNode(px)
	nx := node.Point.X

	if px == nx {
		// Here we are comparing point references, not values
		if point != node.Point {
			// We might have two nodes with same x value for a short time
			if point == node.Prev.Point {
				node = node.Prev
			} else if point == node.Next.Point {
				node = node.Next
			} else {
				panic("poly2tri Invalid AdvancingFront.locatePoint() call")
			}
		}
	} else if px < nx {
		node = node.Prev
		for node != nil {
			if point == node.Point {
				break
			}
			node = node.Prev
		}
	} else {
		node = node.Next
		for node != nil {
			if point == node.Point {
				break
			}
			node = node.Next
		}
	}

	if node != nil {
		af.SearchNode = node
	}

	return node
}
