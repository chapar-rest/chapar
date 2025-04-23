// Some extra layouts for Gioui.
//
// ListWrap is copied from https://git.sr.ht/~pierrec/giox. 
// All rights are reversed to the author.
//
package layout

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
)

// inf is an infinite axis constraint.
const inf = 1e6

// ListWrap lays out as many items as possible along the main axis
// before wrapping to the cross axis.
//
// The number of items along the main axis must be the same for
// all rows or columns on the cross axis.
type ListWrap struct {
	Axis      layout.Axis
	Alignment layout.Alignment
	list      layout.List
	num       int        // number of items per row/column
	els       []listData // cache used when displaying items
}

type listData struct {
	main  int
	cross int
	base  int
	call  op.CallOp
}

func (l *ListWrap) init() {
	if l.list.Axis == l.Axis {
		l.list.Axis = Swap(l.Axis)
	}
}

// ListElement is called for each row/column with the CallOp containing all the elements
// to be displayed for that row/column and their combined dimensions.
type WrappedListElement func(layout.Context, int, layout.Dimensions, op.CallOp) layout.Dimensions

func listWrapBlock(gtx layout.Context, idx int, dims layout.Dimensions, c op.CallOp) layout.Dimensions {
	c.Add(gtx.Ops)
	return dims
}

func (l *ListWrap) Layout(gtx layout.Context, num int, el layout.ListElement, w WrappedListElement) layout.Dimensions {
	l.init()
	left := l.Axis.Convert(gtx.Constraints.Max).X
	if num == 0 || left == 0 {
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}

	// Consider that the same number of items are displayed for all columns/rows.
	cs := gtx.Constraints
	gtx.Constraints.Min = image.Point{}
	elgtx := gtx
	// To help keeping a constant number per column/row, give infinite space
	// on the cross axis.
	if l.Axis == layout.Horizontal {
		elgtx.Constraints.Max.Y = inf
	} else {
		elgtx.Constraints.Max.X = inf
	}
	// Calculate the number of items that can be displayed on the main axis.
	els := l.els[:0]
	first := l.list.Position.First
	var mainSize, crossSize, base int
	for i := first * l.num; left > 0 && i < num; i++ {
		m := op.Record(gtx.Ops)
		dims := el(elgtx, i)
		c := m.Stop()
		main := l.Axis.Convert(dims.Size).X
		if main == 0 {
			// Ignore elements not contributing to the main axis.
			continue
		}
		left -= main
		if left >= 0 || len(els) == 0 {
			// Add the item as there is room left or no room but
			// we still want one item displayed even if truncated
			cross := l.Axis.Convert(dims.Size).Y
			els = append(els, listData{main, cross, dims.Baseline, c})
			mainSize += main
			crossSize = max(crossSize, cross)
			base = max(base, dims.Baseline)
		}
	}
	if l.num != len(els) {
		// The number of items per column/row has changed.
		// Unless the last column/row is displayed with a number
		// less than l.num, update l.num.
		// This avoids the case where the tail of the list does not
		// use all of the column/row and skews l.num.
		if l.num == 0 || num%l.num == 0 || first != num/l.num {
			l.num = len(els)
		}
		l.els = els
	}
	if l.num == 0 {
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}

	if w == nil {
		w = listWrapBlock
	}

	dims := l.list.Layout(gtx, (num+l.num-1)/l.num, func(gtx layout.Context, idx int) layout.Dimensions {
		if idx == first {
			// Reuse the items processed while calculating the number of elements per column/row.
			first = -1
		} else {
			mainSize, crossSize, base = 0, 0, 0
			els = els[:0]
			for i, n := idx*l.num, min(num, (idx+1)*l.num); i < n; i++ {
				m := op.Record(gtx.Ops)
				dims := el(elgtx, i)
				c := m.Stop()

				main := l.Axis.Convert(dims.Size).X
				if main == 0 {
					// Ignore elements not contributing to the main axis.
					n--
					continue
				}
				cross := l.Axis.Convert(dims.Size).Y
				els = append(els, listData{main, cross, dims.Baseline, c})

				mainSize += main
				crossSize = max(crossSize, cross)
				base = max(base, dims.Baseline)
			}
		}

		m := op.Record(gtx.Ops)
		for _, el := range els {
			var cross int
			switch l.Alignment {
			case layout.Start:
			case layout.End:
				cross = crossSize - el.cross
			case layout.Middle:
				cross = (crossSize - el.cross) / 2
			case layout.Baseline:
				if l.Axis == layout.Horizontal {
					cross = base - el.base
				}
			}
			if cross == 0 {
				el.call.Add(gtx.Ops)
			} else {
				pt := l.Axis.Convert(image.Pt(0, cross))
				op.Offset(pt).Add(gtx.Ops)
				el.call.Add(gtx.Ops)
				op.Offset(pt.Mul(-1)).Add(gtx.Ops)
			}
			pt := l.Axis.Convert(image.Pt(el.main, 0))
			op.Offset(pt).Add(gtx.Ops)
		}
		c := m.Stop()
		size := l.Axis.Convert(image.Pt(mainSize, crossSize))
		dims := layout.Dimensions{Size: size, Baseline: base}
		gtx.Constraints.Max.X = max(gtx.Constraints.Max.X, size.X)
		gtx.Constraints.Max.Y = max(gtx.Constraints.Max.Y, size.Y)
		return w(gtx, idx, dims, c)
	})
	dims.Size.X = max(dims.Size.X, cs.Min.X)
	dims.Size.Y = max(dims.Size.Y, cs.Min.Y)
	return dims
}

func (l *ListWrap) Position() layout.Position {
	return l.list.Position
}

func (l *ListWrap) ScrollBy(num float32) {
	l.list.ScrollBy(num)
}

func Swap(a layout.Axis) layout.Axis {
	return (a + 1) % 2
}
