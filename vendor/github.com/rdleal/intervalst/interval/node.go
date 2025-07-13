package interval

import "math"

type color bool

const (
	red   color = true
	black color = false
)

type node[V, T any] struct {
	Interval interval[V, T]
	MaxEnd   T
	Right    *node[V, T]
	Left     *node[V, T]
	Color    color
	Size     int
}

func newNode[V, T any](intervl interval[V, T], c color) *node[V, T] {
	return &node[V, T]{
		Interval: intervl,
		MaxEnd:   intervl.End,
		Color:    c,
		Size:     1,
	}
}

func flipColors[T, V any](n *node[V, T]) {
	n.Color = !n.Color
	if n.Left != nil {
		n.Left.Color = !n.Left.Color
	}
	if n.Right != nil {
		n.Right.Color = !n.Right.Color
	}
}

func isRed[V, T any](n *node[V, T]) bool {
	if n == nil {
		return false
	}
	return n.Color == red
}

func min[V, T any](n *node[V, T]) *node[V, T] {
	for n != nil && n.Left != nil {
		n = n.Left
	}
	return n
}

func max[V, T any](n *node[V, T]) *node[V, T] {
	for n != nil && n.Right != nil {
		n = n.Right
	}
	return n
}

func updateSize[V, T any](n *node[V, T]) {
	n.Size = 1 + size(n.Left) + size(n.Right)
}

func height[V, T any](n *node[V, T]) float64 {
	if n == nil {
		return 0
	}

	return 1 + math.Max(height(n.Left), height(n.Right))
}

func size[V, T any](n *node[V, T]) int {
	if n == nil {
		return 0
	}
	return n.Size
}

func updateMaxEnd[V, T any](n *node[V, T], cmp CmpFunc[T]) {
	n.MaxEnd = n.Interval.End
	if n.Left != nil && cmp.gt(n.Left.MaxEnd, n.MaxEnd) {
		n.MaxEnd = n.Left.MaxEnd
	}

	if n.Right != nil && cmp.gt(n.Right.MaxEnd, n.MaxEnd) {
		n.MaxEnd = n.Right.MaxEnd
	}
}

func rotateLeft[V, T any](n *node[V, T], cmp CmpFunc[T]) *node[V, T] {
	x := n.Right
	n.Right = x.Left
	x.Left = n
	x.Color = n.Color
	x.MaxEnd = n.MaxEnd
	n.Color = red
	x.Size = n.Size

	updateSize(n)
	updateMaxEnd(n, cmp)
	return x
}

func rotateRight[V, T any](n *node[V, T], cmp CmpFunc[T]) *node[V, T] {
	x := n.Left
	n.Left = x.Right
	x.Right = n
	x.Color = n.Color
	x.MaxEnd = n.MaxEnd
	n.Color = red
	x.Size = n.Size

	updateSize(n)
	updateMaxEnd(n, cmp)
	return x
}

func balanceNode[V, T any](n *node[V, T], cmp CmpFunc[T]) *node[V, T] {
	if isRed(n.Right) && !isRed(n.Left) {
		n = rotateLeft(n, cmp)
	}

	if isRed(n.Left) && isRed(n.Left.Left) {
		n = rotateRight(n, cmp)
	}

	if isRed(n.Left) && isRed(n.Right) {
		flipColors(n)
	}

	return n
}

func moveRedLeft[V, T any](n *node[V, T], cmp CmpFunc[T]) *node[V, T] {
	flipColors(n)
	if n.Right != nil && isRed(n.Right.Left) {
		n.Right = rotateRight(n.Right, cmp)
		n = rotateLeft(n, cmp)
		flipColors(n)
	}
	return n
}

func moveRedRight[V, T any](n *node[V, T], cmp CmpFunc[T]) *node[V, T] {
	flipColors(n)
	if n.Left != nil && isRed(n.Left.Left) {
		n = rotateRight(n, cmp)
		flipColors(n)
	}
	return n
}

func fixUp[V, T any](n *node[V, T], cmp CmpFunc[T]) *node[V, T] {
	updateMaxEnd(n, cmp)

	return balanceNode(n, cmp)
}
