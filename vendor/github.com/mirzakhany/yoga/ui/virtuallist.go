package ui

import "github.com/mirzakhany/yoga/layout"

// VirtualList is a Scroll over count rows, each rowH tall, that builds only
// the rows in view (plus a few either side). Use it for result lists with
// thousands of rows, where building every row each frame would stall the UI.
// Rows must fit rowH; they are laid out at exactly that height.
func VirtualList(id string, count int, rowH float32, row func(i int) View) *Node {
	return Scroll(id, &virtualRows{id: id, count: count, rowH: rowH, row: row})
}

// virtualOverscan is how many rows are built beyond each edge of the view.
const virtualOverscan = 8

type virtualRows struct {
	id    string
	count int
	rowH  float32
	row   func(i int) View
}

// virtualRange is the rows to build for a view of height h scrolled to off.
func virtualRange(count int, rowH, off, h float32) (lo, hi int) {
	if count <= 0 || rowH <= 0 {
		return 0, 0
	}
	if h <= 0 {
		h = 1200 // first frame: the view's height is not known yet
	}
	lo = max(0, int(off/rowH)-virtualOverscan)
	hi = min(count, int((off+h)/rowH)+1+virtualOverscan)
	return min(lo, hi), hi
}

func (v *virtualRows) Layout(c *Ctx) *layout.Element {
	var off, h float32
	if sv, ok := c.Widget(v.id, func() any { return NewScrollView(nil) }).(*ScrollView); ok {
		off, h = sv.scrollY, sv.hostHeight()
	}
	lo, hi := virtualRange(v.count, v.rowH, off, h)
	gap := func(rows int) View {
		return Raw(layout.New(layout.Box().H(float32(rows) * v.rowH).FlexShrink(0)))
	}
	kids := make([]View, 0, hi-lo+2)
	kids = append(kids, gap(lo))
	for i := lo; i < hi; i++ {
		kids = append(kids, ViewOf(v.row(i)).Height(v.rowH).Shrink(0))
	}
	kids = append(kids, gap(v.count-hi))
	return Column(kids...).Layout(c)
}
