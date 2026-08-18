package layout

import "math"

// customEngine is the default pure-Go layout solver supporting flex, grid, and stack.
type customEngine struct{}

func (customEngine) Compute(root *Element, width, height float32) {
	layoutNode(root, width, height, true)
	flatten(root, 0, 0)
}

func layoutNode(e *Element, constraintW, constraintH float32, isRoot bool) {
	s := &e.Style

	w := resolveWidth(s, constraintW)
	h := resolveHeight(s, constraintH)

	if isRoot {
		e.cx = 0
		e.cy = 0
		if isUnset(s.Width) {
			w = constraintW
		}
		if isUnset(s.Height) {
			h = constraintH
		}
		w = clampDim(w, s.MinWidth, s.MaxWidth)
		h = clampDim(h, s.MinHeight, s.MaxHeight)
	}

	e.cw = w
	e.ch = h

	contentW := maxf(0, w-s.Padding.Left-s.Padding.Right)
	contentH := maxf(0, h-s.Padding.Top-s.Padding.Bottom)

	switch s.Disp {
	case DisplayGrid:
		layoutGrid(e, contentW, contentH)
	case DisplayStack:
		layoutStack(e, contentW, contentH)
	default:
		layoutFlex(e, contentW, contentH)
	}

	for _, c := range e.Children {
		if c.Style.Pos == PositionAbsolute {
			layoutAbsoluteChild(c, contentW, contentH)
		}
	}
	for _, c := range e.Children {
		if c.Overlay && c.Style.Pos != PositionAbsolute {
			// Out of flex flow and zero-sized until the widget positions itself
			// (e.g. Menu.OpenAt, DialogHost.Position). Avoids reserving column
			// space or painting a full-width band at the origin.
			c.cx = 0
			c.cy = 0
			c.cw = 0
			c.ch = 0
		}
	}
}

func layoutAbsoluteChild(e *Element, parentW, parentH float32) {
	s := &e.Style

	left := edgeOrZero(s.Left)
	top := edgeOrZero(s.Top)
	right := edgeOrZero(s.Right)
	bottom := edgeOrZero(s.Bottom)

	w := s.Width
	h := s.Height

	pinnedW := !isUnset(s.Left) && !isUnset(s.Right)
	pinnedH := !isUnset(s.Top) && !isUnset(s.Bottom)

	if pinnedW {
		w = maxf(0, parentW-left-right)
	} else if isUnset(w) {
		w = intrinsicWidth(e, parentW)
	}
	if pinnedH {
		h = maxf(0, parentH-top-bottom)
	} else if isUnset(h) {
		h = intrinsicHeight(e, parentH)
	}

	w = clampDim(w, s.MinWidth, s.MaxWidth)
	h = clampDim(h, s.MinHeight, s.MaxHeight)

	x := left
	y := top
	if isUnset(s.Left) && !isUnset(s.Right) {
		x = parentW - right - w
	}
	if isUnset(s.Top) && !isUnset(s.Bottom) {
		y = parentH - bottom - h
	}

	e.cx = x
	e.cy = y
	e.cw = w
	e.ch = h

	childW := maxf(0, w-s.Padding.Left-s.Padding.Right)
	childH := maxf(0, h-s.Padding.Top-s.Padding.Bottom)
	layoutContainerChildren(e, childW, childH)
}

func layoutContainerChildren(e *Element, contentW, contentH float32) {
	switch e.Style.Disp {
	case DisplayGrid:
		layoutGrid(e, contentW, contentH)
	case DisplayStack:
		layoutStack(e, contentW, contentH)
	default:
		layoutFlex(e, contentW, contentH)
	}
	for _, c := range e.Children {
		if c.Style.Pos == PositionAbsolute {
			layoutAbsoluteChild(c, contentW, contentH)
		}
	}
}

func resolveWidth(s *Style, constraint float32) float32 {
	w := constraint
	if !isUnset(s.Width) {
		w = s.Width
	}
	return clampDim(w, s.MinWidth, s.MaxWidth)
}

func resolveHeight(s *Style, constraint float32) float32 {
	h := constraint
	if !isUnset(s.Height) {
		h = s.Height
	}
	return clampDim(h, s.MinHeight, s.MaxHeight)
}

func clampDim(v, minV, maxV float32) float32 {
	if !isUnset(minV) && v < minV {
		v = minV
	}
	if !isUnset(maxV) && v > maxV {
		v = maxV
	}
	return v
}

func edgeOrZero(v float32) float32 {
	if isUnset(v) {
		return 0
	}
	return v
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func intrinsicWidth(e *Element, constraint float32) float32 {
	s := &e.Style
	if !isUnset(s.Width) {
		return clampDim(s.Width, s.MinWidth, s.MaxWidth)
	}
	if len(e.Children) == 0 {
		return s.Padding.Left + s.Padding.Right
	}
	w := float32(0)
	switch s.Disp {
	case DisplayGrid:
		w = gridIntrinsicWidth(e)
	case DisplayStack:
		for _, c := range flowChildren(e) {
			w = maxf(w, childOuterWidth(c, constraint))
		}
	default:
		if isRowDirection(s.Dir) {
			if s.Wrap == DoWrap {
				w = flexIntrinsicWrappedMain(e, true)
			} else {
				w = flexIntrinsicMain(e, constraint, true)
			}
		} else if s.Wrap == DoWrap {
			w = flexIntrinsicWrappedCross(e, constraint, false)
		} else {
			w = flexIntrinsicCross(e, constraint, false)
		}
	}
	w += s.Padding.Left + s.Padding.Right
	return clampDim(w, s.MinWidth, s.MaxWidth)
}

func intrinsicHeight(e *Element, constraint float32) float32 {
	s := &e.Style
	if !isUnset(s.Height) {
		return clampDim(s.Height, s.MinHeight, s.MaxHeight)
	}
	if len(e.Children) == 0 {
		return s.Padding.Top + s.Padding.Bottom
	}
	h := float32(0)
	switch s.Disp {
	case DisplayGrid:
		h = gridIntrinsicHeight(e)
	case DisplayStack:
		for _, c := range flowChildren(e) {
			h = maxf(h, childOuterHeight(c, constraint))
		}
	default:
		if isRowDirection(s.Dir) {
			if s.Wrap == DoWrap {
				h = flexIntrinsicWrappedCross(e, constraint, true)
			} else {
				h = flexIntrinsicCross(e, constraint, true)
			}
		} else if s.Wrap == DoWrap {
			h = flexIntrinsicWrappedMain(e, false)
		} else {
			h = flexIntrinsicMain(e, constraint, false)
		}
	}
	h += s.Padding.Top + s.Padding.Bottom
	return clampDim(h, s.MinHeight, s.MaxHeight)
}

func flowChildren(e *Element) []*Element {
	var out []*Element
	for _, c := range e.Children {
		if c.Style.Pos != PositionAbsolute && !c.Overlay {
			out = append(out, c)
		}
	}
	return out
}

func childOuterWidth(c *Element, crossConstraint float32) float32 {
	s := &c.Style
	w := resolveCrossSize(c, crossConstraint, true)
	return w + s.Margin.Left + s.Margin.Right
}

func childOuterHeight(c *Element, crossConstraint float32) float32 {
	s := &c.Style
	h := resolveCrossSize(c, crossConstraint, false)
	return h + s.Margin.Top + s.Margin.Bottom
}

// resolveCrossSize returns the child's border-box size on the given axis: the
// explicit style size when set, otherwise the intrinsic size (which already
// includes the child's own padding).
func resolveCrossSize(c *Element, crossConstraint float32, horizontal bool) float32 {
	s := &c.Style
	if horizontal {
		if !isUnset(s.Width) {
			return clampDim(s.Width, s.MinWidth, s.MaxWidth)
		}
		return intrinsicWidth(c, crossConstraint)
	}
	if !isUnset(s.Height) {
		return clampDim(s.Height, s.MinHeight, s.MaxHeight)
	}
	return intrinsicHeight(c, crossConstraint)
}

// flexOuterMain is the border-box size along the flex main axis.
func flexOuterMain(c *Element, mainConstraint, crossSize float32, horizontalMain bool) float32 {
	s := &c.Style
	if horizontalMain {
		if !isUnset(s.Width) {
			return clampDim(s.Width, s.MinWidth, s.MaxWidth)
		}
	} else if !isUnset(s.Height) {
		return clampDim(s.Height, s.MinHeight, s.MaxHeight)
	}
	b := flexBasis(c, mainConstraint, crossSize, horizontalMain)
	if horizontalMain {
		b += s.Padding.Left + s.Padding.Right
	} else {
		b += s.Padding.Top + s.Padding.Bottom
	}
	return b
}

func flexBasis(c *Element, mainConstraint, crossSize float32, horizontalMain bool) float32 {
	s := &c.Style
	if !isUnset(s.Basis) {
		return s.Basis
	}
	if horizontalMain {
		if !isUnset(s.Width) {
			return clampDim(s.Width, s.MinWidth, s.MaxWidth)
		}
		return intrinsicWidth(c, crossSize) - s.Padding.Left - s.Padding.Right
	}
	if !isUnset(s.Height) {
		return clampDim(s.Height, s.MinHeight, s.MaxHeight)
	}
	return intrinsicHeight(c, crossSize) - s.Padding.Top - s.Padding.Bottom
}

func alignItem(containerAlign, itemAlign Align) Align {
	if itemAlign == AlignAuto {
		return containerAlign
	}
	return itemAlign
}

func alignCrossPos(a Align, freeSpace float32) float32 {
	switch a {
	case AlignCenter:
		return freeSpace / 2
	case AlignEnd:
		return freeSpace
	default:
		return 0
	}
}

func isRowDirection(d FlexDirection) bool {
	return d == Row || d == RowReverse
}

func mainAxisHorizontal(d FlexDirection) bool {
	return isRowDirection(d)
}

func mainGap(s *Style, horizontalMain bool) float32 {
	if horizontalMain {
		return s.ColGap
	}
	return s.RowGap
}

func crossGap(s *Style, horizontalMain bool) float32 {
	if horizontalMain {
		return s.RowGap
	}
	return s.ColGap
}

func distributeFlex(flexGrow, flexShrink, basis, size []float32, freeSpace float32) {
	if freeSpace > 0 {
		totalGrow := float32(0)
		for _, g := range flexGrow {
			totalGrow += g
		}
		if totalGrow <= 0 {
			return
		}
		for i := range size {
			if flexGrow[i] > 0 {
				size[i] = basis[i] + freeSpace*(flexGrow[i]/totalGrow)
			}
		}
		return
	}
	if freeSpace < 0 {
		totalShrink := float32(0)
		for i, sh := range flexShrink {
			if sh > 0 {
				totalShrink += basis[i] * sh
			}
		}
		if totalShrink <= 0 {
			return
		}
		remaining := -freeSpace
		for i := range size {
			if flexShrink[i] > 0 {
				delta := remaining * (basis[i] * flexShrink[i] / totalShrink)
				size[i] = maxf(basis[i]-delta, 0)
			}
		}
	}
}

func layoutFlex(e *Element, contentW, contentH float32) {
	s := &e.Style
	if s.Wrap == DoWrap {
		layoutFlexWrap(e, contentW, contentH)
		return
	}
	children := flowChildren(e)
	if len(children) == 0 {
		return
	}

	horizontalMain := mainAxisHorizontal(s.Dir)
	reverse := s.Dir == RowReverse || s.Dir == ColumnReverse

	var mainSize, crossSize float32
	if horizontalMain {
		mainSize, crossSize = contentW, contentH
	} else {
		mainSize, crossSize = contentH, contentW
	}

	type item struct {
		child *Element
		basis float32
		main  float32
		cross float32
		mMain float32 // margin on main axis (start+end)
		mCrossStart, mCrossEnd float32
	}

	items := make([]item, len(children))
	grow := make([]float32, len(children))
	shrink := make([]float32, len(children))
	basis := make([]float32, len(children))
	mains := make([]float32, len(children))

	totalMain := float32(0)
	gap := mainGap(s, horizontalMain)

	for i, c := range children {
		cs := &c.Style
		var cross float32
		if horizontalMain {
			cross = resolveCrossSize(c, crossSize, false)
			if alignItem(s.Align, cs.SelfAlign) == AlignStretch && isUnset(cs.Height) {
				cross = crossSize - cs.Margin.Top - cs.Margin.Bottom
			}
		} else {
			cross = resolveCrossSize(c, crossSize, true)
			if alignItem(s.Align, cs.SelfAlign) == AlignStretch && isUnset(cs.Width) {
				cross = crossSize - cs.Margin.Left - cs.Margin.Right
			}
		}
		cross = clampDim(cross, 0, math.MaxFloat32)

		b := flexOuterMain(c, mainSize, cross, horizontalMain)
		if horizontalMain {
			b = clampDim(b, cs.MinWidth, cs.MaxWidth)
		} else {
			b = clampDim(b, cs.MinHeight, cs.MaxHeight)
		}

		var mMain, mCrossStart, mCrossEnd float32
		if horizontalMain {
			mMain = cs.Margin.Left + cs.Margin.Right
			mCrossStart, mCrossEnd = cs.Margin.Top, cs.Margin.Bottom
		} else {
			mMain = cs.Margin.Top + cs.Margin.Bottom
			mCrossStart, mCrossEnd = cs.Margin.Left, cs.Margin.Right
		}

		items[i] = item{child: c, basis: b, main: b, cross: cross, mMain: mMain, mCrossStart: mCrossStart, mCrossEnd: mCrossEnd}
		grow[i] = cs.Grow
		shrink[i] = cs.Shrink
		basis[i] = b
		mains[i] = b
		totalMain += b + mMain
	}

	if len(children) > 1 {
		totalMain += gap * float32(len(children)-1)
	}

	freeMain := mainSize - totalMain
	distributeFlex(grow, shrink, basis, mains, freeMain)

	for i := range items {
		items[i].main = mains[i]
	}

	outerMain := make([]float32, len(items))
	marginStarts := make([]float32, len(items))
	for i := range items {
		outerMain[i] = mains[i] + items[i].mMain
		if horizontalMain {
			marginStarts[i] = items[i].child.Style.Margin.Left
		} else {
			marginStarts[i] = items[i].child.Style.Margin.Top
		}
	}

	starts := flexMainStarts(s.Justify, outerMain, marginStarts, gap, mainSize)

	for i, it := range items {
		c := it.child
		main := it.main
		cross := it.cross

		crossFree := crossSize - cross - it.mCrossStart - it.mCrossEnd
		crossOff := alignCrossPos(alignItem(s.Align, c.Style.SelfAlign), crossFree)

		mainStart := starts[i]
		if reverse {
			mainStart = mainSize - mainStart - main
			if horizontalMain {
				mainStart -= c.Style.Margin.Right
			} else {
				mainStart -= c.Style.Margin.Bottom
			}
		}

		var x, y, w, h float32
		if horizontalMain {
			w, h = main, cross
			x = mainStart
			y = crossOff + c.Style.Margin.Top
		} else {
			w, h = cross, main
			x = crossOff + c.Style.Margin.Left
			y = mainStart
		}

		c.cx = x
		c.cy = y
		layoutNode(c, w, h, false)
	}
}

func flexMainStarts(j Justify, outer, marginStarts []float32, gap, mainSize float32) []float32 {
	n := len(outer)
	starts := make([]float32, n)
	if n == 0 {
		return starts
	}

	used := float32(0)
	for i := range outer {
		used += outer[i]
		if i > 0 {
			used += gap
		}
	}
	slack := mainSize - used

	switch j {
	case JustifyEnd:
		cursor := slack
		for i := range outer {
			if i > 0 {
				cursor += gap
			}
			starts[i] = cursor + marginStarts[i]
			cursor += outer[i]
		}
	case JustifyCenter:
		cursor := slack / 2
		for i := range outer {
			if i > 0 {
				cursor += gap
			}
			starts[i] = cursor + marginStarts[i]
			cursor += outer[i]
		}
	case JustifyBetween:
		if n == 1 {
			starts[0] = marginStarts[0]
			break
		}
		starts[0] = marginStarts[0]
		starts[n-1] = mainSize - outer[n-1] + marginStarts[n-1]
		inner := mainSize - outer[0] - outer[n-1]
		step := inner / float32(n-1)
		cursor := outer[0]
		for i := 1; i < n-1; i++ {
			cursor += step
			starts[i] = cursor + marginStarts[i]
			cursor += outer[i]
		}
	case JustifyAround:
		step := slack / float32(n)
		cursor := step / 2
		for i := range outer {
			if i > 0 {
				cursor += gap
			}
			starts[i] = cursor + marginStarts[i]
			cursor += outer[i] + step
		}
	case JustifyEvenly:
		step := slack / float32(n+1)
		cursor := step
		for i := range outer {
			if i > 0 {
				cursor += gap
			}
			starts[i] = cursor + marginStarts[i]
			cursor += outer[i] + step
		}
	default: // Start
		cursor := float32(0)
		for i := range outer {
			if i > 0 {
				cursor += gap
			}
			starts[i] = cursor + marginStarts[i]
			cursor += outer[i]
		}
	}
	return starts
}

func flexIntrinsicMain(e *Element, constraint float32, horizontal bool) float32 {
	children := flowChildren(e)
	gap := float32(0)
	if horizontal {
		gap = e.Style.ColGap
	} else {
		gap = e.Style.RowGap
	}
	total := float32(0)
	for i, c := range children {
		s := &c.Style
		var b float32
		if horizontal {
			b = flexOuterMain(c, constraint, constraint, true) + s.Margin.Left + s.Margin.Right
		} else {
			b = flexOuterMain(c, constraint, constraint, false) + s.Margin.Top + s.Margin.Bottom
		}
		total += b
		if i > 0 {
			total += gap
		}
	}
	return total
}

func flexIntrinsicCross(e *Element, constraint float32, horizontal bool) float32 {
	children := flowChildren(e)
	maxCross := float32(0)
	for _, c := range children {
		s := &c.Style
		var cross float32
		if horizontal {
			cross = resolveCrossSize(c, constraint, false) + s.Margin.Top + s.Margin.Bottom
		} else {
			cross = resolveCrossSize(c, constraint, true) + s.Margin.Left + s.Margin.Right
		}
		maxCross = maxf(maxCross, cross)
	}
	return maxCross
}
