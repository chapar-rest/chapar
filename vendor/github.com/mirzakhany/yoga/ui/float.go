package ui

import "github.com/mirzakhany/yoga/layout"

// Corner names a corner of a box.
type Corner int

const (
	CornerTopLeft Corner = iota
	CornerTopRight
	CornerBottomLeft
	CornerBottomRight
)

type floatSpec struct {
	corner Corner
	dx, dy float32
}

// Float takes the node out of its parent's flow and pins it to corner c of the
// parent's content box, dx and dy in from the corner's two edges. The node
// keeps its own size and takes no room from its siblings.
//
// A floating node is drawn over the siblings before it and receives pointer
// events ahead of them, so it suits a control laid over content — a Format
// button in the corner of an editor, for example:
//
//	Column(
//		ViewOf(editor).Grow(1),
//		Button("format", Text("Format")).Float(CornerBottomRight, 20, 20),
//	)
func (n *Node) Float(c Corner, dx, dy float32) *Node {
	n.float = &floatSpec{corner: c, dx: dx, dy: dy}
	return n
}

func (f *floatSpec) apply(st *layout.Style) {
	st.Pos = layout.PositionAbsolute
	switch f.corner {
	case CornerTopRight, CornerBottomRight:
		st.Right = f.dx
	default:
		st.Left = f.dx
	}
	switch f.corner {
	case CornerBottomLeft, CornerBottomRight:
		st.Bottom = f.dy
	default:
		st.Top = f.dy
	}
}
