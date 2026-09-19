package layout

func layoutStack(e *Element, contentW, contentH float32) {
	s := &e.Style
	children := flowChildren(e)
	if len(children) == 0 {
		return
	}

	jx := s.StackJustify
	ay := s.StackAlign

	for _, c := range children {
		cs := &c.Style
		w := cs.Width
		h := cs.Height
		if isUnset(w) {
			if ay == AlignStretch {
				w = contentW - cs.Margin.Left - cs.Margin.Right
			} else {
				// intrinsicWidth already includes the child's own padding
				// (border-box), so use it directly.
				w = intrinsicWidth(c, contentW)
			}
		}
		if isUnset(h) {
			if ay == AlignStretch {
				h = contentH - cs.Margin.Top - cs.Margin.Bottom
			} else {
				h = intrinsicHeight(c, contentH)
			}
		}
		w = clampDim(w, cs.MinWidth, cs.MaxWidth)
		h = clampDim(h, cs.MinHeight, cs.MaxHeight)

		freeX := contentW - w - cs.Margin.Left - cs.Margin.Right
		freeY := contentH - h - cs.Margin.Top - cs.Margin.Bottom

		x := cs.Margin.Left + stackMainPos(jx, freeX)
		y := cs.Margin.Top + alignCrossPos(ay, freeY)

		c.cx = x
		c.cy = y
		layoutNode(c, w, h, false)
	}
}

func stackMainPos(j Justify, free float32) float32 {
	switch j {
	case JustifyCenter:
		return free / 2
	case JustifyEnd:
		return free
	default:
		return 0
	}
}
