// Package ui is the ergonomic view layer: a Swift-style View DSL (Column, Row,
// Text, Button, …) that materializes layout.Element trees each frame. Widgets
// keep micro-state (hover, caret, scroll) in a per-id store; app data lives in
// retained application structs.
package ui

import (
	"strings"
	"unicode/utf8"

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
// maxW is split between characters so nothing is clipped.
func wrapText(eng *shape.Engine, s string, size, maxW float32) []string {
	return wrapTextWeight(eng, s, size, shape.WeightRegular, maxW)
}

// wrapTextWeight is wrapText measured at a font weight.
func wrapTextWeight(eng *shape.Engine, s string, size float32, weight int, maxW float32) []string {
	if eng == nil {
		return strings.Split(s, "\n")
	}
	fits := func(t string) bool {
		w, _ := eng.MeasureAtWeight(t, size, weight)
		return w <= maxW
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			cand := word
			if line != "" {
				cand = line + " " + word
			}
			if fits(cand) {
				line = cand
				continue
			}
			if line != "" {
				out = append(out, line)
			}
			for !fits(word) {
				cut := splitToFit(word, fits)
				out = append(out, word[:cut])
				word = word[cut:]
			}
			line = word
		}
		out = append(out, line)
	}
	return out
}

// splitToFit returns the byte length of the longest rune prefix of word that
// fits, and at least one rune so the caller always makes progress.
func splitToFit(word string, fits func(string) bool) int {
	_, first := utf8.DecodeRuneInString(word)
	cut := first
	for i, r := range word {
		if i < first {
			continue
		}
		end := i + utf8.RuneLen(r)
		if !fits(word[:end]) {
			break
		}
		cut = end
	}
	return cut
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

// removeAt deletes index i from s and clears the slot the tail vacated.
//
// The backing array outlives the shortened slice, so the idiomatic
// append(s[:i], s[i+1:]...) leaves whatever the last element referenced
// reachable for as long as the slice's owner lives. For a slice of widgets,
// nodes, or documents that is a live subtree the caller believes it dropped.
func removeAt[T any](s []T, i int) []T {
	if i < 0 || i >= len(s) {
		return s
	}
	var zero T
	copy(s[i:], s[i+1:])
	s[len(s)-1] = zero
	return s[:len(s)-1]
}
