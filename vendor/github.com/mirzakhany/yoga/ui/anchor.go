package ui

import "github.com/mirzakhany/yoga/render"

// Placement selects which side of an anchor an overlay prefers.
type Placement int

const (
	PlacementBottom Placement = iota
	PlacementTop
	PlacementLeft
	PlacementRight
)

const anchorGap float32 = 6

// placeAnchor positions a content box relative to anchor, centered on the
// preferred side (tooltips). Flips when that side lacks room, then clamps.
func placeAnchor(anchor render.Rect, contentW, contentH float32, preferred Placement) (x, y float32) {
	return placeAnchorMode(anchor, contentW, contentH, preferred, true)
}

// placeAnchorStart positions like placeAnchor but aligns to the trigger's
// start edge (menus / popovers), matching Menu.OpenAt.
func placeAnchorStart(anchor render.Rect, contentW, contentH float32, preferred Placement) (x, y float32) {
	return placeAnchorMode(anchor, contentW, contentH, preferred, false)
}

func placeAnchorMode(anchor render.Rect, contentW, contentH float32, preferred Placement, center bool) (x, y float32) {
	candidates := []Placement{preferred}
	switch preferred {
	case PlacementTop:
		candidates = append(candidates, PlacementBottom, PlacementRight, PlacementLeft)
	case PlacementLeft:
		candidates = append(candidates, PlacementRight, PlacementBottom, PlacementTop)
	case PlacementRight:
		candidates = append(candidates, PlacementLeft, PlacementBottom, PlacementTop)
	default: // Bottom
		candidates = append(candidates, PlacementTop, PlacementRight, PlacementLeft)
	}

	for _, p := range candidates {
		if sideHasRoom(anchor, contentW, contentH, p) {
			cx, cy := anchorPoint(anchor, contentW, contentH, p, center)
			return clampToViewport(cx, cy, contentW, contentH)
		}
	}
	cx, cy := anchorPoint(anchor, contentW, contentH, preferred, center)
	return clampToViewport(cx, cy, contentW, contentH)
}

func sideHasRoom(anchor render.Rect, w, h float32, p Placement) bool {
	if viewportW <= 0 || viewportH <= 0 {
		return true
	}
	switch p {
	case PlacementTop:
		return anchor.Y-h-anchorGap >= 0
	case PlacementLeft:
		return anchor.X-w-anchorGap >= 0
	case PlacementRight:
		return anchor.X+anchor.W+anchorGap+w <= viewportW
	default: // Bottom
		return anchor.Y+anchor.H+anchorGap+h <= viewportH
	}
}

func anchorPoint(anchor render.Rect, w, h float32, p Placement, center bool) (x, y float32) {
	switch p {
	case PlacementTop:
		x = anchor.X
		if center {
			x = anchor.X + (anchor.W-w)/2
		}
		return x, anchor.Y - h - anchorGap
	case PlacementLeft:
		y = anchor.Y
		if center {
			y = anchor.Y + (anchor.H-h)/2
		}
		return anchor.X - w - anchorGap, y
	case PlacementRight:
		y = anchor.Y
		if center {
			y = anchor.Y + (anchor.H-h)/2
		}
		return anchor.X + anchor.W + anchorGap, y
	default: // Bottom
		x = anchor.X
		if center {
			x = anchor.X + (anchor.W-w)/2
		}
		return x, anchor.Y + anchor.H + anchorGap
	}
}
