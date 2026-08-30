package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// BorderStyle selects how border strokes are painted.
type BorderStyle = layout.BorderStyle

const (
	BorderSolid  = layout.BorderSolid
	BorderDotted = layout.BorderDotted
	BorderDashed = layout.BorderDashed
)

const (
	borderSideTop = 1 << iota
	borderSideRight
	borderSideBottom
	borderSideLeft
	borderSideAll = borderSideTop | borderSideRight | borderSideBottom | borderSideLeft
)

const (
	radiusCornerTL = 1 << iota
	radiusCornerTR
	radiusCornerBR
	radiusCornerBL
	radiusCornerAll = radiusCornerTL | radiusCornerTR | radiusCornerBR | radiusCornerBL
)

func (s *Spec) ensureBorder() {
	if s.borderSet == 0 {
		s.borderSet = borderSideAll
	}
}

func (s *Spec) ensureRadius() {
	if s.radiusSet == 0 {
		s.radiusSet = radiusCornerAll
	}
}

// Border sets a uniform token stroke on all sides.
func (s Spec) Border(t Token, width float32) Spec {
	s.border = colorRef{on: true, token: t}
	s.borderW = layout.Edges{Top: width, Right: width, Bottom: width, Left: width}
	s.borderSet = borderSideAll
	return s
}

// BorderColor sets a uniform literal stroke on all sides.
func (s Spec) BorderColor(c render.Color, width float32) Spec {
	s.border = colorRef{on: true, useLit: true, lit: c}
	s.borderW = layout.Edges{Top: width, Right: width, Bottom: width, Left: width}
	s.borderSet = borderSideAll
	return s
}

// BorderTop sets the top border width (token color from Border/BorderColor).
func (s Spec) BorderTop(t Token, width float32) Spec {
	s.ensureBorder()
	s.border = colorRef{on: true, token: t}
	s.borderW.Top = width
	s.borderSet |= borderSideTop
	return s
}

// BorderRight sets the right border width.
func (s Spec) BorderRight(t Token, width float32) Spec {
	s.ensureBorder()
	s.border = colorRef{on: true, token: t}
	s.borderW.Right = width
	s.borderSet |= borderSideRight
	return s
}

// BorderBottom sets the bottom border width.
func (s Spec) BorderBottom(t Token, width float32) Spec {
	s.ensureBorder()
	s.border = colorRef{on: true, token: t}
	s.borderW.Bottom = width
	s.borderSet |= borderSideBottom
	return s
}

// BorderLeft sets the left border width.
func (s Spec) BorderLeft(t Token, width float32) Spec {
	s.ensureBorder()
	s.border = colorRef{on: true, token: t}
	s.borderW.Left = width
	s.borderSet |= borderSideLeft
	return s
}

// BorderTopColor sets the top border with a literal color.
func (s Spec) BorderTopColor(c render.Color, width float32) Spec {
	s.ensureBorder()
	s.border = colorRef{on: true, useLit: true, lit: c}
	s.borderW.Top = width
	s.borderSet |= borderSideTop
	return s
}

// BorderRightColor sets the right border with a literal color.
func (s Spec) BorderRightColor(c render.Color, width float32) Spec {
	s.ensureBorder()
	s.border = colorRef{on: true, useLit: true, lit: c}
	s.borderW.Right = width
	s.borderSet |= borderSideRight
	return s
}

// BorderBottomColor sets the bottom border with a literal color.
func (s Spec) BorderBottomColor(c render.Color, width float32) Spec {
	s.ensureBorder()
	s.border = colorRef{on: true, useLit: true, lit: c}
	s.borderW.Bottom = width
	s.borderSet |= borderSideBottom
	return s
}

// BorderLeftColor sets the left border with a literal color.
func (s Spec) BorderLeftColor(c render.Color, width float32) Spec {
	s.ensureBorder()
	s.border = colorRef{on: true, useLit: true, lit: c}
	s.borderW.Left = width
	s.borderSet |= borderSideLeft
	return s
}

// BorderStyle sets dotted/dashed/solid stroke style.
func (s Spec) BorderStyle(st BorderStyle) Spec {
	s.borderStyle = st
	s.hasBorderStyle = true
	return s
}

// Radius sets a uniform corner radius on all corners.
func (s Spec) Radius(r float32) Spec {
	s.radii = layout.Corners{TopLeft: r, TopRight: r, BottomRight: r, BottomLeft: r}
	s.radiusSet = radiusCornerAll
	return s
}

// RadiusTopLeft sets the top-left corner radius.
func (s Spec) RadiusTopLeft(r float32) Spec {
	s.ensureRadius()
	s.radii.TopLeft = r
	s.radiusSet |= radiusCornerTL
	return s
}

// RadiusTopRight sets the top-right corner radius.
func (s Spec) RadiusTopRight(r float32) Spec {
	s.ensureRadius()
	s.radii.TopRight = r
	s.radiusSet |= radiusCornerTR
	return s
}

// RadiusBottomRight sets the bottom-right corner radius.
func (s Spec) RadiusBottomRight(r float32) Spec {
	s.ensureRadius()
	s.radii.BottomRight = r
	s.radiusSet |= radiusCornerBR
	return s
}

// RadiusBottomLeft sets the bottom-left corner radius.
func (s Spec) RadiusBottomLeft(r float32) Spec {
	s.ensureRadius()
	s.radii.BottomLeft = r
	s.radiusSet |= radiusCornerBL
	return s
}

func mergeBorderSides(dst, src layout.Edges, mask, set uint8) layout.Edges {
	if set&borderSideTop != 0 {
		dst.Top = src.Top
	}
	if set&borderSideRight != 0 {
		dst.Right = src.Right
	}
	if set&borderSideBottom != 0 {
		dst.Bottom = src.Bottom
	}
	if set&borderSideLeft != 0 {
		dst.Left = src.Left
	}
	return dst
}

func mergeRadiusCorners(dst, src layout.Corners, set uint8) layout.Corners {
	if set&radiusCornerTL != 0 {
		dst.TopLeft = src.TopLeft
	}
	if set&radiusCornerTR != 0 {
		dst.TopRight = src.TopRight
	}
	if set&radiusCornerBR != 0 {
		dst.BottomRight = src.BottomRight
	}
	if set&radiusCornerBL != 0 {
		dst.BottomLeft = src.BottomLeft
	}
	return dst
}
