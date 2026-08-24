package layout

type gridPlacement struct {
	child                      *Element
	col, row, colSpan, rowSpan int
}

func layoutGrid(e *Element, contentW, contentH float32) {
	s := &e.Style
	placements, cols, numRows := gridPlacements(e)
	if len(placements) == 0 && cols == 0 && len(s.Rows) == 0 {
		return
	}
	if cols == 0 {
		cols = 1
	}
	if numRows == 0 {
		numRows = 1
	}

	colTracks := make([]Track, cols)
	copy(colTracks, s.Cols)
	for i := range colTracks {
		if colTracks[i].Kind == 0 && colTracks[i].Value == 0 {
			colTracks[i] = Auto()
		}
	}

	rowTracks := make([]Track, numRows)
	if len(s.Rows) > 0 {
		copy(rowTracks, s.Rows)
	}
	autoRowTrack := s.AutoRows
	if autoRowTrack.Kind == 0 && autoRowTrack.Value == 0 {
		autoRowTrack = Fr(1)
	}
	for i := range rowTracks {
		if rowTracks[i].Kind == 0 && rowTracks[i].Value == 0 {
			rowTracks[i] = autoRowTrack
		}
	}

	colSizes := resolveTracks(colTracks, contentW, s.ColGap, func(trackIdx int) float32 {
		return maxItemCross(placements, trackIdx, true, cols)
	})
	rowSizes := resolveTracks(rowTracks, contentH, s.RowGap, func(trackIdx int) float32 {
		return maxItemCross(placements, trackIdx, false, numRows)
	})

	colPos := trackPositions(colSizes, s.ColGap)
	rowPos := trackPositions(rowSizes, s.RowGap)

	for _, p := range placements {
		c := p.child
		col0 := p.col - 1
		row0 := p.row - 1
		if col0 < 0 {
			col0 = 0
		}
		if row0 < 0 {
			row0 = 0
		}

		x := colPos[col0]
		y := rowPos[row0]
		w := sumSpan(colSizes, col0, p.colSpan, s.ColGap)
		h := sumSpan(rowSizes, row0, p.rowSpan, s.RowGap)

		alignCell(c, x, y, w, h)
	}
}

// gridPlacements auto-places flow children and returns column/row counts.
func gridPlacements(e *Element) (placements []gridPlacement, cols, numRows int) {
	s := &e.Style
	children := flowChildren(e)
	cols = len(s.Cols)
	if cols == 0 {
		cols = 1
	}
	if len(children) == 0 && len(s.Cols) == 0 && len(s.Rows) == 0 {
		return nil, 0, 0
	}

	placements = make([]gridPlacement, 0, len(children))
	maxRow := 0
	autoCol, autoRow := 1, 1

	for _, c := range children {
		cs := &c.Style
		colSpan := cs.ColSpan
		if colSpan <= 0 {
			colSpan = 1
		}
		rowSpan := cs.RowSpan
		if rowSpan <= 0 {
			rowSpan = 1
		}
		col := cs.ColStart
		row := cs.RowStart
		if col <= 0 || row <= 0 {
			if col <= 0 {
				col = autoCol
			}
			if row <= 0 {
				row = autoRow
			}
			autoCol += colSpan
			if autoCol > cols {
				autoCol = 1
				autoRow++
			}
		}
		placements = append(placements, gridPlacement{c, col, row, colSpan, rowSpan})
		endRow := row + rowSpan - 1
		if endRow > maxRow {
			maxRow = endRow
		}
	}

	numRows = len(s.Rows)
	if numRows == 0 {
		numRows = maxRow
		if numRows == 0 {
			numRows = 1
		}
	} else if maxRow > numRows {
		numRows = maxRow
	}
	return placements, cols, numRows
}

func maxItemCross(placements []gridPlacement, trackIdx int, isCol bool, trackCount int) float32 {
	maxV := float32(0)
	for _, p := range placements {
		start := p.col - 1
		span := p.colSpan
		if !isCol {
			start = p.row - 1
			span = p.rowSpan
		}
		if start <= trackIdx && trackIdx < start+span {
			c := p.child
			cs := &c.Style
			// Intrinsic sizes are border-box (they include the child's own
			// padding), matching how explicit Width/Height are interpreted.
			if isCol {
				if !isUnset(cs.Width) {
					maxV = maxf(maxV, cs.Width)
				} else {
					maxV = maxf(maxV, intrinsicWidth(c, mathMax))
				}
			} else {
				if !isUnset(cs.Height) {
					maxV = maxf(maxV, cs.Height)
				} else {
					maxV = maxf(maxV, intrinsicHeight(c, mathMax))
				}
			}
		}
	}
	return maxV
}

const mathMax = float32(1e9)

func resolveTracks(tracks []Track, totalSpace, gap float32, autoMeasure func(int) float32) []float32 {
	n := len(tracks)
	if n == 0 {
		return nil
	}
	sizes := make([]float32, n)
	fixedTotal := float32(0)
	frTotal := float32(0)
	autoCount := 0

	for i, t := range tracks {
		switch t.Kind {
		case TrackFixed:
			sizes[i] = t.Value
			fixedTotal += t.Value
		case TrackFr:
			frTotal += t.Value
			if frTotal == 0 {
				frTotal = 1
			}
		case TrackAuto:
			autoCount++
		}
	}

	gapTotal := float32(0)
	if n > 1 {
		gapTotal = gap * float32(n-1)
	}
	remaining := totalSpace - fixedTotal - gapTotal
	if remaining < 0 {
		remaining = 0
	}

	autoSizes := make([]float32, n)
	autoSum := float32(0)
	for i, t := range tracks {
		if t.Kind == TrackAuto {
			s := autoMeasure(i)
			autoSizes[i] = s
			autoSum += s
		}
	}
	if autoSum > remaining && autoSum > 0 {
		scale := remaining / autoSum
		for i := range autoSizes {
			if tracks[i].Kind == TrackAuto {
				autoSizes[i] *= scale
			}
		}
		autoSum = remaining
	}
	remaining -= autoSum
	for i, t := range tracks {
		if t.Kind == TrackAuto {
			sizes[i] = autoSizes[i]
		}
	}

	if frTotal > 0 {
		frUnit := remaining / frTotal
		for i, t := range tracks {
			if t.Kind == TrackFr {
				sizes[i] = frUnit * t.Value
			}
		}
	}
	return sizes
}

func trackPositions(sizes []float32, gap float32) []float32 {
	pos := make([]float32, len(sizes))
	cursor := float32(0)
	for i, sz := range sizes {
		pos[i] = cursor
		cursor += sz
		if i < len(sizes)-1 {
			cursor += gap
		}
	}
	return pos
}

func sumSpan(sizes []float32, start, span int, gap float32) float32 {
	if span <= 0 {
		span = 1
	}
	total := float32(0)
	for i := 0; i < span; i++ {
		idx := start + i
		if idx >= len(sizes) {
			break
		}
		total += sizes[idx]
		if i < span-1 {
			total += gap
		}
	}
	return total
}

func alignCell(c *Element, cellX, cellY, cellW, cellH float32) {
	cs := &c.Style
	w := cs.Width
	h := cs.Height
	if isUnset(w) {
		w = cellW - cs.Margin.Left - cs.Margin.Right
	}
	if isUnset(h) {
		h = cellH - cs.Margin.Top - cs.Margin.Bottom
	}
	w = clampDim(w, cs.MinWidth, cs.MaxWidth)
	h = clampDim(h, cs.MinHeight, cs.MaxHeight)

	freeX := cellW - w - cs.Margin.Left - cs.Margin.Right
	freeY := cellH - h - cs.Margin.Top - cs.Margin.Bottom

	jx := cs.Justify
	if jx == JustifyStart && cs.Align == AlignAuto {
		jx = JustifyStart
	}
	ay := alignItem(AlignStretch, cs.SelfAlign)
	if ay == AlignStretch && isUnset(cs.Height) {
		h = cellH - cs.Margin.Top - cs.Margin.Bottom
		freeY = 0
	}
	if alignItem(AlignStretch, cs.SelfAlign) == AlignStretch && isUnset(cs.Width) {
		w = cellW - cs.Margin.Left - cs.Margin.Right
		freeX = 0
	}

	x := cellX + cs.Margin.Left + stackMainPos(jx, freeX)
	y := cellY + cs.Margin.Top + alignCrossPos(ay, freeY)

	c.cx = x
	c.cy = y
	layoutNode(c, w, h, false)
}

func gridIntrinsicWidth(e *Element) float32 {
	s := &e.Style
	placements, cols, _ := gridPlacements(e)
	if cols == 0 {
		cols = 1
	}
	tracks := make([]Track, cols)
	copy(tracks, s.Cols)
	for i := range tracks {
		if tracks[i].Kind == 0 && tracks[i].Value == 0 {
			tracks[i] = Auto()
		}
	}
	// fr against an indefinite size is treated as auto (CSS Grid).
	tracks = frAsAuto(tracks)
	sizes := resolveTracks(tracks, mathMax, s.ColGap, func(trackIdx int) float32 {
		return maxItemCross(placements, trackIdx, true, cols)
	})
	return sumTracks(sizes, s.ColGap)
}

func gridIntrinsicHeight(e *Element) float32 {
	s := &e.Style
	placements, _, numRows := gridPlacements(e)
	if numRows == 0 {
		numRows = 1
	}
	tracks := make([]Track, numRows)
	if len(s.Rows) > 0 {
		copy(tracks, s.Rows)
	}
	auto := s.AutoRows
	if auto.Kind == 0 && auto.Value == 0 {
		auto = Fr(1)
	}
	for i := range tracks {
		if tracks[i].Kind == 0 && tracks[i].Value == 0 {
			tracks[i] = auto
		}
	}
	tracks = frAsAuto(tracks)
	sizes := resolveTracks(tracks, mathMax, s.RowGap, func(trackIdx int) float32 {
		return maxItemCross(placements, trackIdx, false, numRows)
	})
	return sumTracks(sizes, s.RowGap)
}

// frAsAuto rewrites fr tracks to auto for intrinsic/indefinite measurement.
func frAsAuto(tracks []Track) []Track {
	out := append([]Track(nil), tracks...)
	for i, t := range out {
		if t.Kind == TrackFr {
			out[i] = Auto()
		}
	}
	return out
}

func sumTracks(sizes []float32, gap float32) float32 {
	total := float32(0)
	for i, sz := range sizes {
		total += sz
		if i < len(sizes)-1 {
			total += gap
		}
	}
	return total
}
