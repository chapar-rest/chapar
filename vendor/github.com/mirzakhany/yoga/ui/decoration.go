package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// paintChromeBox paints a control chrome box with explicit fill/border overrides.
func paintChromeBox(dl *render.DrawList, frame render.Rect, r resolvedSpec, fill, border render.Color, bw, radius float32) {
	patched := r
	patched.hasBg = true
	patched.bg = fill
	patched.hasBorder = true
	patched.border = border
	patched.borderW = layout.Edges{Top: bw, Right: bw, Bottom: bw, Left: bw}
	patched.hasRadii = true
	patched.radii = layout.Corners{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}
	paintResolvedBox(dl, frame, patched, radius)
}

// paintResolvedBox fills and/or strokes frame using a resolved visual spec.
func paintResolvedBox(dl *render.DrawList, frame render.Rect, r resolvedSpec, defaultRadius float32) {
	corners := layout.Corners{TopLeft: defaultRadius, TopRight: defaultRadius, BottomRight: defaultRadius, BottomLeft: defaultRadius}
	if r.hasRadii {
		corners = r.radii
	}
	widths := r.borderW
	style := BorderSolid
	if r.hasBorderStyle {
		style = r.borderStyle
	}
	fill := r.bg
	if !r.hasBg {
		fill = render.Color{}
	}
	border := r.border
	hasBorder := r.hasBorder && border.A > 0 && widths.AnyPositive()
	if !hasBorder {
		border = render.Color{}
		widths = layout.Edges{}
	}
	rc := render.Corners{
		TopLeft: corners.TopLeft, TopRight: corners.TopRight,
		BottomRight: corners.BottomRight, BottomLeft: corners.BottomLeft,
	}
	bw := render.BorderEdges{Top: widths.Top, Right: widths.Right, Bottom: widths.Bottom, Left: widths.Left}
	dl.PaintBox(frame, rc, bw, fill, border, style)
}

func uniformBorderWidth(w layout.Edges, fallback float32) float32 {
	if w.Top > 0 {
		return w.Top
	}
	if w.Right > 0 {
		return w.Right
	}
	if w.Bottom > 0 {
		return w.Bottom
	}
	if w.Left > 0 {
		return w.Left
	}
	return fallback
}

func uniformRadius(c layout.Corners, fallback float32) float32 {
	if c.TopLeft > 0 {
		return c.TopLeft
	}
	if c.TopRight > 0 {
		return c.TopRight
	}
	if c.BottomRight > 0 {
		return c.BottomRight
	}
	if c.BottomLeft > 0 {
		return c.BottomLeft
	}
	return fallback
}
