package component

import (
	"math"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/outlay"
)

// Grid holds the persistent state for a layout.List that has a
// scrollbar attached.
type GridState struct {
	VScrollbar widget.Scrollbar
	HScrollbar widget.Scrollbar
	outlay.Grid
}

// TableStyle is the persistent state for a table with heading and grid.
type TableStyle GridStyle

// Table makes a grid with its persistent state.
func Table(th *material.Theme, state *GridState) TableStyle {
	return TableStyle{
		State:           state,
		VScrollbarStyle: material.Scrollbar(th, &state.VScrollbar),
		HScrollbarStyle: material.Scrollbar(th, &state.HScrollbar),
	}
}

// GridStyle is the persistent state for the grid.
type GridStyle struct {
	State           *GridState
	VScrollbarStyle material.ScrollbarStyle
	HScrollbarStyle material.ScrollbarStyle
	material.AnchorStrategy
}

// Grid makes a grid with its persistent state.
func Grid(th *material.Theme, state *GridState) GridStyle {
	return GridStyle{
		State:           state,
		VScrollbarStyle: material.Scrollbar(th, &state.VScrollbar),
		HScrollbarStyle: material.Scrollbar(th, &state.HScrollbar),
	}
}

// constrain a value to be between min and max (inclusive).
func constrain(value *int, min int, max int) {
	if *value < min {
		*value = min
	}
	if *value > max {
		*value = max
	}
}

// Layout will draw a table with a heading, using fixed column widths and row height.
func (t TableStyle) Layout(gtx layout.Context, rows, cols int, dimensioner outlay.Dimensioner, headingFunc layout.ListElement, cellFunc outlay.Cell) layout.Dimensions {
	t.State.Grid.LockedRows = 1
	return GridStyle(t).Layout(gtx, rows+1, cols, dimensioner, func(gtx layout.Context, row, col int) layout.Dimensions {
		if row == 0 {
			return headingFunc(gtx, col)
		}
		return cellFunc(gtx, row-1, col)
	})
}

// Layout will draw a grid, using fixed column widths and row height.
func (g GridStyle) Layout(gtx layout.Context, rows, cols int, dimensioner outlay.Dimensioner, cellFunc outlay.Cell) layout.Dimensions {
	// Determine how much space the scrollbars occupy when present.
	hBarWidth := gtx.Dp(g.HScrollbarStyle.Width())
	vBarWidth := gtx.Dp(g.VScrollbarStyle.Width())

	// Reserve space for the scrollbars using the gtx constraints.
	if g.AnchorStrategy == material.Occupy {
		gtx.Constraints.Max.X -= vBarWidth
		gtx.Constraints.Max.Y -= hBarWidth
	}

	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	// Draw grid.
	dim := g.State.Grid.Layout(gtx, rows, cols, dimensioner, cellFunc)

	// Calculate column widths in pixels. Width is sum of widths.
	totalWidth := g.State.Horizontal.Length
	totalHeight := g.State.Vertical.Length

	// Make the scroll bar stick to the grid.
	if gtx.Constraints.Max.X > dim.Size.X {
		gtx.Constraints.Max.X = dim.Size.X
		if g.AnchorStrategy == material.Occupy {
			gtx.Constraints.Max.X += vBarWidth
		}
	}

	// Get horizontal scroll info.
	delta := g.HScrollbarStyle.Scrollbar.ScrollDistance()
	if delta != 0 {
		g.State.Horizontal.Offset += int(float32(totalWidth-vBarWidth) * delta)
	}

	// Get vertical scroll info.
	delta = g.VScrollbarStyle.Scrollbar.ScrollDistance()
	if delta != 0 {
		g.State.Vertical.Offset += int(math.Round(float64(float32(totalHeight-hBarWidth) * delta)))
	}

	var start float32
	var end float32

	// Draw vertical scroll-bar.
	if vBarWidth > 0 {
		c := gtx
		start = float32(g.State.Vertical.OffsetAbs) / float32(totalHeight)
		end = start + float32(c.Constraints.Max.Y)/float32(totalHeight)
		if g.AnchorStrategy == material.Overlay {
			c.Constraints.Max.Y -= hBarWidth
		} else {
			c.Constraints.Max.X += vBarWidth
		}
		c.Constraints.Min = c.Constraints.Max
		layout.E.Layout(c, func(gtx layout.Context) layout.Dimensions {
			return g.VScrollbarStyle.Layout(gtx, layout.Vertical, start, end)
		})
	}

	// Draw horizontal scroll-bar if it is visible.
	if hBarWidth > 0 {
		c := gtx
		start = float32(g.State.Horizontal.OffsetAbs) / float32(totalWidth)
		end = start + float32(c.Constraints.Max.X)/float32(totalWidth)
		if g.AnchorStrategy == material.Overlay {
			c.Constraints.Max.X -= vBarWidth
		} else {
			c.Constraints.Max.Y += hBarWidth
		}
		c.Constraints.Min = c.Constraints.Max
		layout.S.Layout(c, func(gtx layout.Context) layout.Dimensions {
			return g.HScrollbarStyle.Layout(gtx, layout.Horizontal, start, end)
		})
	}
	if g.AnchorStrategy == material.Occupy {
		dim.Size.Y += hBarWidth
	}

	return dim
}
