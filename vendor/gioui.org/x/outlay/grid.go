// SPDX-License-Identifier: Unlicense OR MIT

package outlay

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// AxisPosition describes the position of a viewport on a given axis.
type AxisPosition struct {
	// First and last are the indicies of the first and last visible
	// cell on the axis.
	First, Last int
	// Offset is the pixel offset from the beginning of the first cell to
	// the first visible pixel.
	Offset int
	// OffsetAbs is the estimated absolute position of the first visible
	// pixel within the entire axis, mesaured in pixels.
	OffsetAbs int
	// Length is the estimated total size of the axis, measured in pixels.
	Length int
}

// normalize resolves the First and Offset fields so that Offset is valid within
// the cell dimensions of First.
func (a *AxisPosition) normalize(gtx layout.Context, axis layout.Axis, elements int, dimensioner Dimensioner) {
	if a.First < 0 {
		a.First = 0
	}
	if a.First > elements {
		a.First = elements - 1
	}

	constraint := axis.Convert(gtx.Constraints.Max).X
	for a.Offset < 0 && a.First > 0 {
		a.First--
		dim := dimensioner(axis, a.First, constraint)
		a.Offset += dim
		a.OffsetAbs += dim
	}
	if a.Offset < 0 {
		a.Offset = 0
	}
	for a.Offset > dimensioner(axis, a.First, constraint) && a.First < elements-1 {
		dim := dimensioner(axis, a.First, constraint)
		a.First++
		a.Offset -= dim
		a.OffsetAbs += dim
	}
}

// computePosition recomputes the Last, Length, and OffsetAbs fields using
// the current Offset and First fields.
func (a *AxisPosition) computePosition(gtx layout.Context, axis layout.Axis, elements, maxPx int, dimensioner Dimensioner) (pixelsUsed int) {
	constraint := axis.Convert(gtx.Constraints.Max).X
	firstWidth := dimensioner(axis, a.First, constraint)
	a.Last = a.First
	pixelsUsed = firstWidth - a.Offset
	pixelsTotal := firstWidth
	cellsTotal := 1
	for pixelsUsed < maxPx && a.Last < elements-1 {
		a.Last++
		dim := dimensioner(axis, a.Last, constraint)
		pixelsUsed += dim
		pixelsTotal += dim
		cellsTotal++
	}
	avgCell := float32(pixelsTotal) / float32(cellsTotal)
	a.Length = int(float32(pixelsTotal) / float32(cellsTotal) * float32(elements))
	a.OffsetAbs = int(float32(a.First)*avgCell) + a.Offset
	return pixelsUsed
}

// update recalculates all position fields along the given axis. The
// First and Offset fields are used as the source of truth for this,
// though they do not need to be pre-normalized.
func (a *AxisPosition) update(gtx layout.Context, axis layout.Axis, elements, maxPx int, dimensioner Dimensioner) {
	a.normalize(gtx, axis, elements, dimensioner)
	pixelsUsed := a.computePosition(gtx, axis, elements, maxPx, dimensioner)
	if pixelsUsed < maxPx {
		a.Offset -= (maxPx - pixelsUsed)
		a.normalize(gtx, axis, elements, dimensioner)
		_ = a.computePosition(gtx, axis, elements, maxPx, dimensioner)
	}
}

// Grid provides a scrollable two dimensional viewport that efficiently
// lays out only content visible or nearly visible within the current
// viewport.
type Grid struct {
	Vertical   AxisPosition
	Horizontal AxisPosition
	Vscroll    gesture.Scroll
	Hscroll    gesture.Scroll
	// LockedRows is a quantity of rows (starting from row 0) to lock to
	// the top of the grid's viewport. These rows will not be included in
	// the indicies provided in the Vertical AxisPosition field.
	LockedRows int
}

// Cell is the layout function for a grid cell, with row,col parameters.
type Cell func(gtx layout.Context, row, col int) layout.Dimensions

// Dimensioner is a function that provides the dimensions (in pixels) of an element
// on a given axis. The constraint parameter provides the size of the visible portion
// of the axis for applications that want it.
type Dimensioner func(axis layout.Axis, index, constraint int) int

func (g *Grid) drawRow(gtx layout.Context, row, rowHeight int, dimensioner Dimensioner, cellFunc Cell) layout.Dimensions {
	xPos := -g.Horizontal.Offset
	for col := g.Horizontal.First; col <= g.Horizontal.Last; col++ {
		trans := op.Offset(image.Pt(xPos, 0)).Push(gtx.Ops)
		c := gtx
		c.Constraints = layout.Exact(image.Pt(dimensioner(layout.Horizontal, col, gtx.Constraints.Max.X), rowHeight))
		dims := cellFunc(c, row, col)
		trans.Pop()
		xPos += dims.Size.X
	}
	return layout.Dimensions{
		Size: image.Point{
			X: xPos,
			Y: rowHeight,
		},
	}
}

func (g *Grid) Update(gtx layout.Context, rows, cols int, dimensioner Dimensioner) {
	rowHeight := dimensioner(layout.Vertical, 0, gtx.Constraints.Max.Y)

	// Update horizontal scroll position.
	hScrollDelta := g.Hscroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Horizontal,
		pointer.ScrollRange{Min: -gtx.Constraints.Max.X / 2, Max: gtx.Constraints.Max.X / 2},
		pointer.ScrollRange{},
	)
	g.Horizontal.Offset += hScrollDelta

	// Get vertical scroll info.
	vScrollDelta := g.Vscroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical,
		pointer.ScrollRange{},
		pointer.ScrollRange{Min: -gtx.Constraints.Max.Y / 2, Max: gtx.Constraints.Max.Y / 2},
	)
	g.Vertical.Offset += vScrollDelta

	g.Horizontal.update(gtx, layout.Horizontal, cols, gtx.Constraints.Max.X, dimensioner)

	lockedHeight := rowHeight * g.LockedRows
	g.Vertical.update(gtx, layout.Vertical, rows-g.LockedRows, gtx.Constraints.Max.Y-lockedHeight, dimensioner)
}

// Layout the Grid.
//
// BUG(whereswaldon): all rows are set to the height returned by dimensioner(layout.Vertical, 0, gtx.Constraints.Max.Y).
// Support for variable-height rows is welcome as a patch.
func (g *Grid) Layout(gtx layout.Context, rows, cols int, dimensioner Dimensioner, cellFunc Cell) layout.Dimensions {
	g.Update(gtx, rows, cols, dimensioner)
	if rows == 0 || cols == 0 {
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}
	rowHeight := dimensioner(layout.Vertical, 0, gtx.Constraints.Max.Y)
	lockedHeight := rowHeight * g.LockedRows

	contentMacro := op.Record(gtx.Ops)

	// Draw locked rows in a macro.
	macro := op.Record(gtx.Ops)
	clp := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	yOffset := 0
	listDims := image.Point{}
	for row := 0; row < g.LockedRows && row < rows; row++ {
		offset := op.Offset(image.Pt(0, yOffset)).Push(gtx.Ops)
		rowDims := g.drawRow(gtx, row, rowHeight, dimensioner, cellFunc)
		yOffset += rowDims.Size.Y
		offset.Pop()
		listDims.X = max(listDims.X, rowDims.Size.X)
		listDims.Y += rowDims.Size.Y
	}
	clp.Pop()
	lockedRows := macro.Stop()

	// Draw normal rows, then place locked rows on top.
	clp = clip.Rect{
		Min: image.Point{Y: lockedHeight},
		Max: gtx.Constraints.Max,
	}.Push(gtx.Ops)
	yOffset -= g.Vertical.Offset
	firstRow := g.Vertical.First + g.LockedRows
	lastRow := g.Vertical.Last + g.LockedRows
	for row := firstRow; row <= lastRow && row < rows; row++ {
		offset := op.Offset(image.Pt(0, yOffset)).Push(gtx.Ops)
		rowDims := g.drawRow(gtx, row, rowHeight, dimensioner, cellFunc)
		yOffset += rowDims.Size.Y
		offset.Pop()
		listDims.X = max(listDims.X, rowDims.Size.X)
		listDims.Y += rowDims.Size.Y
	}
	clp.Pop()
	lockedRows.Add(gtx.Ops)

	listDims = gtx.Constraints.Constrain(listDims)

	content := contentMacro.Stop()

	// Enable scroll wheel within the grid.
	cl := clip.Rect{Max: listDims}.Push(gtx.Ops)
	g.Vscroll.Add(gtx.Ops)
	g.Hscroll.Add(gtx.Ops)

	// We draw into a macro, and call the macro inside the clip set up for scrolling, so that cells in the grid are
	// children of the clip area, not siblings. This ensures cells can receive pointer events.
	content.Add(gtx.Ops)
	cl.Pop()

	return layout.Dimensions{Size: listDims, Baseline: 0}
}
