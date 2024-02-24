package outlay

import (
	"image"

	"golang.org/x/exp/slices"

	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
)

// mainConstraint returns the min and max main constraints for axis a.
func mainConstraint(a layout.Axis, cs layout.Constraints) (int, int) {
	if a == layout.Horizontal {
		return cs.Min.X, cs.Max.X
	}
	return cs.Min.Y, cs.Max.Y
}

// crossConstraint returns the min and max cross constraints for axis a.
func crossConstraint(a layout.Axis, cs layout.Constraints) (int, int) {
	if a == layout.Horizontal {
		return cs.Min.Y, cs.Max.Y
	}
	return cs.Min.X, cs.Max.X
}

// constraints returns the constraints for axis a.
func constraints(a layout.Axis, mainMin, mainMax, crossMin, crossMax int) layout.Constraints {
	if a == layout.Horizontal {
		return layout.Constraints{Min: image.Pt(mainMin, crossMin), Max: image.Pt(mainMax, crossMax)}
	}
	return layout.Constraints{Min: image.Pt(crossMin, mainMin), Max: image.Pt(crossMax, mainMax)}
}

// Flex lays out child elements along an axis, according to alignment, weights, and
// the configured system locale. It differs from gioui.org/layout.Flex by flipping
// the visual order of its children in RTL locales.
type Flex struct {
	// Axis is the main axis, either Horizontal or Vertical.
	Axis layout.Axis
	// Spacing controls the distribution of space left after
	// layout.
	Spacing layout.Spacing
	// Alignment is the alignment in the cross axis.
	Alignment layout.Alignment
	// WeightSum is the sum of weights used for the weighted
	// size of Flexed children. If WeightSum is zero, the sum
	// of all Flexed weights is used.
	WeightSum float32
}

// FlexChild is the descriptor for a Flex child.
type FlexChild struct {
	flex   bool
	weight float32

	widget layout.Widget

	// Scratch space.
	call op.CallOp
	dims layout.Dimensions
}

// Rigid returns a Flex child with a maximal constraint of the
// remaining space.
func Rigid(widget layout.Widget) FlexChild {
	return FlexChild{
		widget: widget,
	}
}

// Flexed returns a Flex child forced to take up weight fraction of the
// space left over from Rigid children. The fraction is weight
// divided by either the weight sum of all Flexed children or the Flex
// WeightSum if non zero.
func Flexed(weight float32, widget layout.Widget) FlexChild {
	return FlexChild{
		flex:   true,
		weight: weight,
		widget: widget,
	}
}

// Layout a list of children. The position of the children are
// determined by the specified order, but Rigid children are laid out
// before Flexed children. If the locale of the gtx specifies a horizontal,
// RTL language, the children will be allocated space in the order that they
// are provided, but will be displayed in the inverse order to match RTL
// conventions.
func (f Flex) Layout(gtx layout.Context, children ...FlexChild) layout.Dimensions {
	size := 0
	cs := gtx.Constraints
	mainMin, mainMax := mainConstraint(f.Axis, cs)
	crossMin, crossMax := crossConstraint(f.Axis, cs)
	remaining := mainMax
	var totalWeight float32
	cgtx := gtx
	// Lay out Rigid children.
	for i, child := range children {
		if child.flex {
			totalWeight += child.weight
			continue
		}
		macro := op.Record(gtx.Ops)
		cgtx.Constraints = constraints(f.Axis, 0, remaining, crossMin, crossMax)
		dims := child.widget(cgtx)
		c := macro.Stop()
		sz := f.Axis.Convert(dims.Size).X
		size += sz
		remaining -= sz
		if remaining < 0 {
			remaining = 0
		}
		children[i].call = c
		children[i].dims = dims
	}
	if w := f.WeightSum; w != 0 {
		totalWeight = w
	}
	// fraction is the rounding error from a Flex weighting.
	var fraction float32
	flexTotal := remaining
	// Lay out Flexed children.
	for i, child := range children {
		if !child.flex {
			continue
		}
		var flexSize int
		if remaining > 0 && totalWeight > 0 {
			// Apply weight and add any leftover fraction from a
			// previous Flexed.
			childSize := float32(flexTotal) * child.weight / totalWeight
			flexSize = int(childSize + fraction + .5)
			fraction = childSize - float32(flexSize)
			if flexSize > remaining {
				flexSize = remaining
			}
		}
		macro := op.Record(gtx.Ops)
		cgtx.Constraints = constraints(f.Axis, flexSize, flexSize, crossMin, crossMax)
		dims := child.widget(cgtx)
		c := macro.Stop()
		sz := f.Axis.Convert(dims.Size).X
		size += sz
		remaining -= sz
		if remaining < 0 {
			remaining = 0
		}
		children[i].call = c
		children[i].dims = dims
	}
	maxCross := crossMin
	var maxBaseline int
	for _, child := range children {
		if c := f.Axis.Convert(child.dims.Size).Y; c > maxCross {
			maxCross = c
		}
		if b := child.dims.Size.Y - child.dims.Baseline; b > maxBaseline {
			maxBaseline = b
		}
	}
	var space int
	if mainMin > size {
		space = mainMin - size
	}
	var mainSize int
	switch f.Spacing {
	case layout.SpaceSides:
		mainSize += space / 2
	case layout.SpaceStart:
		mainSize += space
	case layout.SpaceEvenly:
		mainSize += space / (1 + len(children))
	case layout.SpaceAround:
		if len(children) > 0 {
			mainSize += space / (len(children) * 2)
		}
	}
	if (f.Axis == layout.Horizontal && gtx.Locale.Direction.Axis() == system.Horizontal) ||
		(f.Axis == layout.Vertical && gtx.Locale.Direction.Axis() == system.Vertical) {
		if gtx.Locale.Direction.Progression() == system.TowardOrigin {
			slices.Reverse(children)
		}
	}
	for i, child := range children {
		dims := child.dims
		b := dims.Size.Y - dims.Baseline
		var cross int
		switch f.Alignment {
		case layout.End:
			cross = maxCross - f.Axis.Convert(dims.Size).Y
		case layout.Middle:
			cross = (maxCross - f.Axis.Convert(dims.Size).Y) / 2
		case layout.Baseline:
			if f.Axis == layout.Horizontal {
				cross = maxBaseline - b
			}
		}
		pt := f.Axis.Convert(image.Pt(mainSize, cross))
		trans := op.Offset(pt).Push(gtx.Ops)
		child.call.Add(gtx.Ops)
		trans.Pop()
		mainSize += f.Axis.Convert(dims.Size).X
		if i < len(children)-1 {
			switch f.Spacing {
			case layout.SpaceEvenly:
				mainSize += space / (1 + len(children))
			case layout.SpaceAround:
				if len(children) > 0 {
					mainSize += space / len(children)
				}
			case layout.SpaceBetween:
				if len(children) > 1 {
					mainSize += space / (len(children) - 1)
				}
			}
		}
	}
	switch f.Spacing {
	case layout.SpaceSides:
		mainSize += space / 2
	case layout.SpaceEnd:
		mainSize += space
	case layout.SpaceEvenly:
		mainSize += space / (1 + len(children))
	case layout.SpaceAround:
		if len(children) > 0 {
			mainSize += space / (len(children) * 2)
		}
	}
	sz := f.Axis.Convert(image.Pt(mainSize, maxCross))
	sz = cs.Constrain(sz)
	return layout.Dimensions{Size: sz, Baseline: sz.Y - maxBaseline}
}
