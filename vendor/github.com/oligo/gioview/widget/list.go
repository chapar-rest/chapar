package widget

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	gvlayout "github.com/oligo/gioview/layout"
	"github.com/oligo/gioview/theme"
)

// WrapList holds the persistent state for a wrappable layout.List that has a
// scrollbar attached.
type WrapList struct {
	widget.Scrollbar
	gvlayout.ListWrap
}

// WrapListStyle configures the presentation of a wrappable layout.List with a scrollbar.
type WrapListStyle struct {
	state *WrapList
	material.ScrollbarStyle
	material.AnchorStrategy
}

func List(th *theme.Theme, state *WrapList) *WrapListStyle {
	return &WrapListStyle{
		state:          state,
		ScrollbarStyle: material.Scrollbar(th.Theme, &state.Scrollbar),
	}
}

// Layout the list and its scrollbar.
func (l WrapListStyle) Layout(gtx layout.Context, length int, w layout.ListElement) layout.Dimensions {
	originalConstraints := gtx.Constraints

	// Determine how much space the scrollbar occupies.
	barWidth := gtx.Dp(l.Width())

	if l.AnchorStrategy == material.Occupy {

		// Reserve space for the scrollbar using the gtx constraints.
		max := l.state.Axis.Convert(gtx.Constraints.Max)
		min := l.state.Axis.Convert(gtx.Constraints.Min)
		max.Y -= barWidth
		if max.Y < 0 {
			max.Y = 0
		}
		min.Y -= barWidth
		if min.Y < 0 {
			min.Y = 0
		}
		gtx.Constraints.Max = l.state.Axis.Convert(max)
		gtx.Constraints.Min = l.state.Axis.Convert(min)
	}

	listDims := l.state.ListWrap.Layout(gtx, length, w, nil)
	gtx.Constraints = originalConstraints

	// Draw the scrollbar.
	anchoring := layout.E
	if l.state.Axis == layout.Horizontal {
		anchoring = layout.S
	}
	majorAxisSize := l.state.Axis.Convert(listDims.Size).X
	start, end := fromListPosition(l.state.Position(), length, majorAxisSize)
	// layout.Direction respects the minimum, so ensure that the
	// scrollbar will be drawn on the correct edge even if the provided
	// layout.Context had a zero minimum constraint.
	gtx.Constraints.Min = listDims.Size
	if l.AnchorStrategy == material.Occupy {
		min := l.state.Axis.Convert(gtx.Constraints.Min)
		min.Y += barWidth
		gtx.Constraints.Min = l.state.Axis.Convert(min)
	}
	anchoring.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return l.ScrollbarStyle.Layout(gtx, l.state.Axis, start, end)
	})

	if delta := l.state.ScrollDistance(); delta != 0 {
		// Handle any changes to the list position as a result of user interaction
		// with the scrollbar.
		l.state.ListWrap.ScrollBy(delta * float32(length))
	}

	if l.AnchorStrategy == material.Occupy {
		// Increase the width to account for the space occupied by the scrollbar.
		cross := l.state.Axis.Convert(listDims.Size)
		cross.Y += barWidth
		listDims.Size = l.state.Axis.Convert(cross)
	}

	return listDims
}

// fromListPosition converts a layout.Position into two floats representing
// the location of the viewport on the underlying content. It needs to know
// the number of elements in the list and the major-axis size of the list
// in order to do this. The returned values will be in the range [0,1], and
// start will be less than or equal to end.
func fromListPosition(lp layout.Position, elements int, majorAxisSize int) (start, end float32) {
	// Approximate the size of the scrollable content.
	lengthEstPx := float32(lp.Length)
	elementLenEstPx := lengthEstPx / float32(elements)

	// Determine how much of the content is visible.
	listOffsetF := float32(lp.Offset)
	listOffsetL := float32(lp.OffsetLast)

	// Compute the location of the beginning of the viewport using estimated element size and known
	// pixel offsets.
	viewportStart := clamp1((float32(lp.First)*elementLenEstPx + listOffsetF) / lengthEstPx)
	viewportEnd := clamp1((float32(lp.First+lp.Count)*elementLenEstPx + listOffsetL) / lengthEstPx)
	viewportFraction := viewportEnd - viewportStart

	// Compute the expected visible proportion of the list content based solely on the ratio
	// of the visible size and the estimated total size.
	visiblePx := float32(majorAxisSize)
	visibleFraction := visiblePx / lengthEstPx

	// Compute the error between the two methods of determining the viewport and diffuse the
	// error on either end of the viewport based on how close we are to each end.
	err := visibleFraction - viewportFraction
	adjStart := viewportStart
	adjEnd := viewportEnd
	if viewportFraction < 1 {
		startShare := viewportStart / (1 - viewportFraction)
		endShare := (1 - viewportEnd) / (1 - viewportFraction)
		startErr := startShare * err
		endErr := endShare * err

		adjStart -= startErr
		adjEnd += endErr
	}
	return adjStart, adjEnd
}

// clamp1 limits v to range [0..1].
func clamp1(v float32) float32 {
	if v >= 1 {
		return 1
	} else if v <= 0 {
		return 0
	} else {
		return v
	}
}
