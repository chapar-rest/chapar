package layout

import "math"

type flexItem struct {
	child                      *Element
	basis, main, cross         float32
	mMain                      float32
	mCrossStart, mCrossEnd     float32
}

type flexLineRange struct {
	start, end int
}

func buildFlexItems(e *Element, mainSize, crossSize float32, horizontalMain bool) ([]flexItem, []float32, []float32, []float32) {
	s := &e.Style
	children := flowChildren(e)
	items := make([]flexItem, len(children))
	grow := make([]float32, len(children))
	shrink := make([]float32, len(children))
	basis := make([]float32, len(children))

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

		items[i] = flexItem{
			child: c, basis: b, main: b, cross: cross,
			mMain: mMain, mCrossStart: mCrossStart, mCrossEnd: mCrossEnd,
		}
		grow[i] = cs.Grow
		shrink[i] = cs.Shrink
		basis[i] = b
	}
	return items, grow, shrink, basis
}

func packingOuterMain(it flexItem, c *Element, horizontalMain bool) float32 {
	s := &c.Style
	if s.Grow > 0 {
		if horizontalMain && !isUnset(s.MinWidth) {
			return s.MinWidth + it.mMain
		}
		if !horizontalMain && !isUnset(s.MinHeight) {
			return s.MinHeight + it.mMain
		}
	}
	return it.main + it.mMain
}

func packFlexLines(packing []float32, mainSize, gap float32) []flexLineRange {
	var lines []flexLineRange
	if len(packing) == 0 {
		return lines
	}
	start := 0
	lineUsed := packing[0]
	for i := 1; i < len(packing); i++ {
		need := gap + packing[i]
		if lineUsed+need > mainSize+0.001 {
			lines = append(lines, flexLineRange{start: start, end: i})
			start = i
			lineUsed = packing[i]
			continue
		}
		lineUsed += need
	}
	lines = append(lines, flexLineRange{start: start, end: len(packing)})
	return lines
}

func lineCrossMax(items []flexItem, ln flexLineRange) float32 {
	maxCross := float32(0)
	for i := ln.start; i < ln.end; i++ {
		it := items[i]
		outer := it.cross + it.mCrossStart + it.mCrossEnd
		if outer > maxCross {
			maxCross = outer
		}
	}
	return maxCross
}

func layoutFlexWrap(e *Element, contentW, contentH float32) {
	s := &e.Style
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

	gap := mainGap(s, horizontalMain)
	cGap := crossGap(s, horizontalMain)

	items, grow, shrink, basis := buildFlexItems(e, mainSize, crossSize, horizontalMain)

	packing := make([]float32, len(items))
	for i := range items {
		packing[i] = packingOuterMain(items[i], children[i], horizontalMain)
	}
	lines := packFlexLines(packing, mainSize, gap)

	crossCursor := float32(0)
	for li, ln := range lines {
		if li > 0 {
			crossCursor += cGap
		}
		n := ln.end - ln.start
		lineMains := make([]float32, n)
		lineGrow := make([]float32, n)
		lineShrink := make([]float32, n)
		lineBasis := make([]float32, n)
		outerMain := make([]float32, n)
		marginStarts := make([]float32, n)

		used := float32(0)
		for j := 0; j < n; j++ {
			idx := ln.start + j
			it := items[idx]
			lineMains[j] = it.main
			lineGrow[j] = grow[idx]
			lineShrink[j] = shrink[idx]
			lineBasis[j] = basis[idx]
			outerMain[j] = it.main + it.mMain
			if horizontalMain {
				marginStarts[j] = it.child.Style.Margin.Left
			} else {
				marginStarts[j] = it.child.Style.Margin.Top
			}
			used += outerMain[j]
			if j > 0 {
				used += gap
			}
		}

		distributeFlex(lineGrow, lineShrink, lineBasis, lineMains, mainSize-used)

		for j := 0; j < n; j++ {
			outerMain[j] = lineMains[j] + items[ln.start+j].mMain
		}
		starts := flexMainStarts(s.Justify, outerMain, marginStarts, gap, mainSize)
		lineCross := lineCrossMax(items, ln)

		for j := 0; j < n; j++ {
			idx := ln.start + j
			it := items[idx]
			c := it.child
			main := lineMains[j]
			cross := it.cross

			crossFree := lineCross - cross - it.mCrossStart - it.mCrossEnd
			crossOff := alignCrossPos(alignItem(s.Align, c.Style.SelfAlign), crossFree)

			mainStart := starts[j]
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
				y = crossCursor + crossOff + c.Style.Margin.Top
			} else {
				w, h = cross, main
				x = crossCursor + crossOff + c.Style.Margin.Left
				y = mainStart
			}

			c.cx = x
			c.cy = y
			layoutNode(c, w, h, false)
		}

		crossCursor += lineCross
	}
}

func flexIntrinsicWrappedCross(e *Element, mainConstraint float32, horizontalMain bool) float32 {
	children := flowChildren(e)
	if len(children) == 0 {
		return 0
	}
	s := &e.Style
	mainSize := mainConstraint
	if isUnset(mainSize) || mainSize <= 0 {
		mainSize = math.MaxFloat32
	}
	cGap := crossGap(s, horizontalMain)
	items, _, _, _ := buildFlexItems(e, mainSize, mainConstraint, horizontalMain)
	packing := make([]float32, len(items))
	for i := range items {
		packing[i] = packingOuterMain(items[i], children[i], horizontalMain)
	}
	lines := packFlexLines(packing, mainSize, mainGap(s, horizontalMain))
	total := float32(0)
	for i, ln := range lines {
		if i > 0 {
			total += cGap
		}
		total += lineCrossMax(items, ln)
	}
	return total
}

func flexIntrinsicWrappedMain(e *Element, horizontalMain bool) float32 {
	children := flowChildren(e)
	if len(children) == 0 {
		return 0
	}
	s := &e.Style
	gap := mainGap(s, horizontalMain)
	items, _, _, _ := buildFlexItems(e, math.MaxFloat32, math.MaxFloat32, horizontalMain)
	packing := make([]float32, len(items))
	for i := range items {
		packing[i] = packingOuterMain(items[i], children[i], horizontalMain)
	}
	lines := packFlexLines(packing, math.MaxFloat32, gap)
	maxLine := float32(0)
	for _, ln := range lines {
		used := float32(0)
		for i := ln.start; i < ln.end; i++ {
			if i > ln.start {
				used += gap
			}
			used += packing[i]
		}
		maxLine = maxf(maxLine, used)
	}
	return maxLine
}
