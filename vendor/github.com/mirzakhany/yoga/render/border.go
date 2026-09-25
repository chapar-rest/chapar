package render

import "math"

// Corners holds independent corner radii (top-left clockwise).
type Corners struct {
	TopLeft, TopRight, BottomRight, BottomLeft Px
}

// UniformCorners returns the same radius on all four corners.
func UniformCorners(r Px) Corners {
	return Corners{TopLeft: r, TopRight: r, BottomRight: r, BottomLeft: r}
}

// BorderEdges holds per-side border widths.
type BorderEdges struct {
	Top, Right, Bottom, Left Px
}

// UniformBorder returns the same width on all four sides.
func UniformBorder(w Px) BorderEdges {
	return BorderEdges{Top: w, Right: w, Bottom: w, Left: w}
}

// BorderStyle selects how border strokes are painted.
type BorderStyle int

const (
	BorderSolid BorderStyle = iota
	BorderDotted
	BorderDashed
)

func clampCornerRadii(r Rect, c Corners) Corners {
	maxW := r.W / 2
	maxH := r.H / 2
	clamp := func(v Px) Px {
		if v < 0 {
			return 0
		}
		if v > maxW {
			v = maxW
		}
		if v > maxH {
			v = maxH
		}
		return v
	}
	return Corners{
		TopLeft: clamp(c.TopLeft), TopRight: clamp(c.TopRight),
		BottomRight: clamp(c.BottomRight), BottomLeft: clamp(c.BottomLeft),
	}
}

func insetCorners(c Corners, inset Px) Corners {
	insetCorner := func(v Px) Px {
		v -= inset
		if v < 0 {
			return 0
		}
		return v
	}
	return Corners{
		TopLeft: insetCorner(c.TopLeft), TopRight: insetCorner(c.TopRight),
		BottomRight: insetCorner(c.BottomRight), BottomLeft: insetCorner(c.BottomLeft),
	}
}

func anyCorner(c Corners) bool {
	return c.TopLeft > 0 || c.TopRight > 0 || c.BottomRight > 0 || c.BottomLeft > 0
}

func uniformBorder(w BorderEdges) (Px, bool) {
	if w.Top <= 0 && w.Right <= 0 && w.Bottom <= 0 && w.Left <= 0 {
		return 0, false
	}
	ref := w.Top
	if w.Right > 0 {
		if ref == 0 {
			ref = w.Right
		} else if w.Right != ref {
			return 0, false
		}
	}
	if w.Bottom > 0 {
		if ref == 0 {
			ref = w.Bottom
		} else if w.Bottom != ref {
			return 0, false
		}
	}
	if w.Left > 0 {
		if ref == 0 {
			ref = w.Left
		} else if w.Left != ref {
			return 0, false
		}
	}
	allSet := w.Top > 0 && w.Right > 0 && w.Bottom > 0 && w.Left > 0
	if !allSet {
		return 0, false
	}
	return ref, true
}

// AddRoundedRectCorners fills r using independent corner radii.
func (d *DrawList) AddRoundedRectCorners(r Rect, corners Corners, c Color) {
	corners = clampCornerRadii(r, corners)
	if !anyCorner(corners) {
		d.AddRect(r, c)
		return
	}
	x, y, w, h := r.X, r.Y, r.W, r.H
	tl, tr, br, bl := corners.TopLeft, corners.TopRight, corners.BottomRight, corners.BottomLeft
	const pi = float32(math.Pi)

	// Center cross.
	cx := x + tl
	cy := y + tl
	cw := w - tl - tr
	ch := h - tl - bl
	if cw > 0 && ch > 0 {
		d.quad(Rect{X: cx, Y: cy, W: cw, H: ch}, Rect{solidUV, solidUV, 0, 0}, c, 0)
	}
	// Top strip between corners.
	topH := tl
	if tr > topH {
		topH = tr
	}
	if cw > 0 && topH > 0 {
		d.quad(Rect{X: cx, Y: y, W: cw, H: topH}, Rect{solidUV, solidUV, 0, 0}, c, 0)
	}
	// Bottom strip between corners.
	botH := bl
	if br > botH {
		botH = br
	}
	if cw > 0 && botH > 0 {
		d.quad(Rect{X: cx, Y: y + h - botH, W: cw, H: botH}, Rect{solidUV, solidUV, 0, 0}, c, 0)
	}
	// Left strip.
	lh := h - tl - bl
	if lh > 0 && (tl > 0 || bl > 0) {
		lw := tl
		if bl > lw {
			lw = bl
		}
		d.quad(Rect{X: x, Y: y + tl, W: lw, H: lh}, Rect{solidUV, solidUV, 0, 0}, c, 0)
	}
	// Right strip.
	rh := h - tr - br
	if rh > 0 && (tr > 0 || br > 0) {
		rw := tr
		if br > rw {
			rw = br
		}
		d.quad(Rect{X: x + w - rw, Y: y + tr, W: rw, H: rh}, Rect{solidUV, solidUV, 0, 0}, c, 0)
	}
	d.addCornerFan(x+tl, y+tl, tl, pi, 3*pi/2, c)
	d.addCornerFan(x+w-tr, y+tr, tr, 3*pi/2, 2*pi, c)
	d.addCornerFan(x+w-br, y+h-br, br, 0, pi/2, c)
	d.addCornerFan(x+bl, y+h-bl, bl, pi/2, pi, c)
}

// AddRoundedRectBorderCorners draws a fill with a uniform border using independent corners.
func (d *DrawList) AddRoundedRectBorderCorners(r Rect, corners Corners, borderWidth float32, fill, border Color) {
	if borderWidth <= 0 {
		d.AddRoundedRectCorners(r, corners, fill)
		return
	}
	corners = clampCornerRadii(r, corners)
	d.AddRoundedRectCorners(r, corners, border)
	inner := Rect{
		X: r.X + borderWidth, Y: r.Y + borderWidth,
		W: r.W - 2*borderWidth, H: r.H - 2*borderWidth,
	}
	if inner.W <= 0 || inner.H <= 0 {
		return
	}
	d.AddRoundedRectCorners(inner, insetCorners(corners, borderWidth), fill)
}

// PaintBox fills and/or strokes a box with per-corner radii and per-side widths.
func (d *DrawList) PaintBox(r Rect, corners Corners, widths BorderEdges, fill, border Color, style BorderStyle) {
	corners = clampCornerRadii(r, corners)
	hasFill := fill.A > 0
	hasBorder := border.A > 0 && (widths.Top > 0 || widths.Right > 0 || widths.Bottom > 0 || widths.Left > 0)

	if !hasBorder {
		if hasFill {
			d.AddRoundedRectCorners(r, corners, fill)
		}
		return
	}

	if bw, ok := uniformBorder(widths); ok && style == BorderSolid {
		if hasFill {
			d.AddRoundedRectBorderCorners(r, corners, bw, fill, border)
		} else {
			d.AddRoundedRectStroke(r, corners, widths, border, BorderSolid)
		}
		return
	}

	if hasFill {
		d.AddRoundedRectCorners(r, corners, fill)
	}
	d.AddRoundedRectStroke(r, corners, widths, border, style)
}

// AddRoundedRectStroke strokes selected sides of a rounded rectangle.
func (d *DrawList) AddRoundedRectStroke(r Rect, corners Corners, widths BorderEdges, c Color, style BorderStyle) {
	corners = clampCornerRadii(r, corners)
	x, y, w, h := r.X, r.Y, r.W, r.H
	tl, tr, br, bl := corners.TopLeft, corners.TopRight, corners.BottomRight, corners.BottomLeft
	const pi = float32(math.Pi)

	if widths.Top > 0 {
		d.strokeSegment(x+tl, y+widths.Top/2, x+w-tr, y+widths.Top/2, widths.Top, c, style)
	}
	if widths.Bottom > 0 {
		d.strokeSegment(x+bl, y+h-widths.Bottom/2, x+w-br, y+h-widths.Bottom/2, widths.Bottom, c, style)
	}
	if widths.Left > 0 {
		d.strokeSegment(x+widths.Left/2, y+tl, x+widths.Left/2, y+h-bl, widths.Left, c, style)
	}
	if widths.Right > 0 {
		d.strokeSegment(x+w-widths.Right/2, y+tr, x+w-widths.Right/2, y+h-br, widths.Right, c, style)
	}

	if widths.Top > 0 && widths.Left > 0 && tl > 0 {
		d.strokeArc(x+tl, y+tl, tl-widths.Top/2, pi, 3*pi/2, widths.Top, c, style)
	}
	if widths.Top > 0 && widths.Right > 0 && tr > 0 {
		d.strokeArc(x+w-tr, y+tr, tr-widths.Top/2, 3*pi/2, 2*pi, widths.Top, c, style)
	}
	if widths.Bottom > 0 && widths.Right > 0 && br > 0 {
		d.strokeArc(x+w-br, y+h-br, br-widths.Bottom/2, 0, pi/2, widths.Bottom, c, style)
	}
	if widths.Bottom > 0 && widths.Left > 0 && bl > 0 {
		d.strokeArc(x+bl, y+h-bl, bl-widths.Bottom/2, pi/2, pi, widths.Bottom, c, style)
	}
}

func (d *DrawList) strokeSegment(x0, y0, x1, y1, width float32, c Color, style BorderStyle) {
	if width <= 0 {
		return
	}
	dx, dy := x1-x0, y1-y0
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length <= 0 {
		return
	}
	switch style {
	case BorderDotted:
		step := width * 2
		if step < 1 {
			step = 1
		}
		for dist := float32(0); dist <= length; dist += step {
			t := dist / length
			px := x0 + dx*t
			py := y0 + dy*t
			d.AddRoundedRect(Rect{X: px - width/2, Y: py - width/2, W: width, H: width}, width/2, c)
		}
	case BorderDashed:
		dash := width * 3
		gap := width * 2
		if dash < 1 {
			dash = 1
		}
		for dist := float32(0); dist < length; {
			end := dist + dash
			if end > length {
				end = length
			}
			t0, t1 := dist/length, end/length
			d.strokeSolidSegment(x0+dx*t0, y0+dy*t0, x0+dx*t1, y0+dy*t1, width, c)
			dist += dash + gap
		}
	default:
		d.strokeSolidSegment(x0, y0, x1, y1, width, c)
	}
}

func (d *DrawList) strokeSolidSegment(x0, y0, x1, y1, width float32, c Color) {
	dx, dy := x1-x0, y1-y0
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length <= 0 {
		return
	}
	nx, ny := -dy/length, dx/length
	hw := width / 2
	d.addSolidTriangle(x0+nx*hw, y0+ny*hw, x1+nx*hw, y1+ny*hw, x1-nx*hw, y1-ny*hw, c)
	d.addSolidTriangle(x0+nx*hw, y0+ny*hw, x1-nx*hw, y1-ny*hw, x0-nx*hw, y0-ny*hw, c)
}

func (d *DrawList) strokeArc(cx, cy, radius, start, end, width float32, c Color, style BorderStyle) {
	if radius <= 0 || width <= 0 {
		return
	}
	arcLen := radius * (end - start)
	if arcLen <= 0 {
		return
	}
	switch style {
	case BorderDotted:
		step := width * 2
		if step < 1 {
			step = 1
		}
		for dist := float32(0); dist <= arcLen; dist += step {
			t := start + (end-start)*(dist/arcLen)
			px := cx + radius*float32(math.Cos(float64(t)))
			py := cy + radius*float32(math.Sin(float64(t)))
			d.AddRoundedRect(Rect{X: px - width/2, Y: py - width/2, W: width, H: width}, width/2, c)
		}
	default:
		// Solid/dashed arcs: approximate with short segments.
		segments := int(arcLen / 2)
		if segments < 4 {
			segments = 4
		}
		if segments > 32 {
			segments = 32
		}
		prevX := cx + radius*float32(math.Cos(float64(start)))
		prevY := cy + radius*float32(math.Sin(float64(start)))
		for i := 1; i <= segments; i++ {
			t := start + (end-start)*float32(i)/float32(segments)
			x := cx + radius*float32(math.Cos(float64(t)))
			y := cy + radius*float32(math.Sin(float64(t)))
			if style == BorderDashed && i%3 == 0 {
				prevX, prevY = x, y
				continue
			}
			d.strokeSolidSegment(prevX, prevY, x, y, width, c)
			prevX, prevY = x, y
		}
	}
}
