// Package ui is the ergonomic view layer: a Swift-style View DSL (Column, Row,
// Text, Button, …) that materializes layout.Element trees each frame. Widgets
// keep micro-state (hover, caret, scroll) in a per-id store; app data lives in
// retained application structs.
package ui

import (
	"strings"

	"github.com/mirzakhany/yoga/shape"
)

// Small float helpers shared by widgets (kept here, not in the theme package,
// since they are layout math rather than palette concerns).

func f32max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func f32min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func clampf(v, lo, hi float32) float32 {
	return f32max(lo, f32min(v, hi))
}

// wrapText greedily word-wraps s into lines no wider than maxW at the given
// logical text size. Explicit newlines are honored; a single word wider than
// maxW gets its own line (it is clipped by the caller rather than split).
func wrapText(eng *shape.Engine, s string, size, maxW float32) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, w := range words[1:] {
			cand := line + " " + w
			if tw, _ := eng.MeasureAt(cand, size); tw > maxW {
				out = append(out, line)
				line = w
				continue
			}
			line = cand
		}
		out = append(out, line)
	}
	return out
}

// Overlay widgets (menus, dialogs) clamp themselves into this viewport so they
// never open off-screen. The app shell should call SetViewport from its Layout
// with the window's logical size. A zero viewport disables clamping.
var viewportW, viewportH float32

// SetViewport records the window's logical size for overlay positioning.
func SetViewport(w, h float32) { viewportW, viewportH = w, h }

// clampToViewport shifts the (x, y, w, h) rectangle so it lies inside the
// recorded viewport (preferring to keep the top-left visible). Returns the
// input unchanged when no viewport has been recorded.
func clampToViewport(x, y, w, h float32) (float32, float32) {
	if viewportW <= 0 || viewportH <= 0 {
		return x, y
	}
	x = f32max(0, f32min(x, viewportW-w))
	y = f32max(0, f32min(y, viewportH-h))
	return x, y
}
