package theme

import (
	"math"

	"github.com/mirzakhany/yoga/render"
)

// Contrast targets from WCAG 2.1. Themes are expected to meet them; the
// contrast test in this package enforces them for every builtin palette.
const (
	// ContrastText is the 4.5:1 minimum for body-sized text (SC 1.4.3 AA).
	ContrastText = 4.5
	// ContrastLargeText is the 3:1 minimum for 18px+ or bold 14px+ text.
	ContrastLargeText = 3.0
	// ContrastNonText is the 3:1 minimum for control boundaries, focus
	// indicators, and meaningful graphics (SC 1.4.11 AA).
	ContrastNonText = 3.0
	// ContrastDisabled is the ratio disabled text is allowed to sit at. WCAG
	// exempts disabled controls; this keeps them legible without implying
	// they are actionable.
	ContrastDisabled = 2.0
)

// channelLuminance linearizes one sRGB channel for the luminance formula.
func channelLuminance(v float32) float64 {
	c := float64(v)
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// Luminance returns the WCAG relative luminance of c. Alpha is ignored; pass an
// already-composited color (see Flatten) when c is translucent.
func Luminance(c render.Color) float64 {
	return 0.2126*channelLuminance(c.R) +
		0.7152*channelLuminance(c.G) +
		0.0722*channelLuminance(c.B)
}

// Flatten composites a straight-alpha src over an opaque bg, which is what the
// renderer does when a translucent token is painted on a surface.
func Flatten(src, bg render.Color) render.Color {
	a := src.A
	return render.Color{
		R: src.R*a + bg.R*(1-a),
		G: src.G*a + bg.G*(1-a),
		B: src.B*a + bg.B*(1-a),
		A: 1,
	}
}

// ContrastRatio returns the WCAG 2.1 contrast ratio between fg and an opaque
// bg, compositing fg over bg first so translucent tokens report the ratio a
// reader actually sees. The result is in [1, 21].
func ContrastRatio(fg, bg render.Color) float64 {
	hi, lo := Luminance(Flatten(fg, bg)), Luminance(bg)
	if hi < lo {
		hi, lo = lo, hi
	}
	return (hi + 0.05) / (lo + 0.05)
}

// mix blends a toward b by t in [0,1], keeping a's alpha.
func mix(a, b render.Color, t float32) render.Color {
	return render.Color{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A,
	}
}

var (
	white = render.Color{R: 1, G: 1, B: 1, A: 1}
	black = render.Color{R: 0, G: 0, B: 0, A: 1}
)

// awayFrom returns the extreme (white or black) that moves c away from bg.
func awayFrom(bg render.Color) render.Color {
	if Luminance(bg) < 0.18 {
		return white
	}
	return black
}

// fit lightens or darkens c until it reaches want against bg, mixing toward
// whichever extreme moves away from bg. Hue survives because the mix is a
// straight interpolation toward an achromatic endpoint. When the target is
// unreachable (bg is mid-gray) it returns the closest attempt.
func fit(c, bg render.Color, want float64) render.Color {
	if ContrastRatio(c, bg) >= want {
		return c
	}
	target := awayFrom(bg)
	best, bestRatio := c, ContrastRatio(c, bg)
	for i := 1; i <= 100; i++ {
		cand := mix(c, target, float32(i)/100)
		r := ContrastRatio(cand, bg)
		if r >= want {
			return cand
		}
		if r > bestRatio {
			best, bestRatio = cand, r
		}
	}
	return best
}

// fitAll adjusts c until it meets want against every background, trying both
// directions and keeping the candidate with the best worst-case ratio. Used for
// tokens painted over more than one surface, such as a focus ring that must
// read against the workspace, the chrome, and the accent fill it outlines.
func fitAll(c render.Color, want float64, bgs ...render.Color) render.Color {
	worst := func(x render.Color) float64 {
		w := math.Inf(1)
		for _, bg := range bgs {
			w = math.Min(w, ContrastRatio(x, bg))
		}
		return w
	}
	if worst(c) >= want {
		return c
	}
	best, bestRatio := c, worst(c)
	for _, target := range []render.Color{white, black} {
		for i := 1; i <= 100; i++ {
			cand := mix(c, target, float32(i)/100)
			r := worst(cand)
			if r >= want {
				return cand
			}
			if r > bestRatio {
				best, bestRatio = cand, r
			}
		}
	}
	return best
}

// Distance returns how far apart two colors look, in 0..1. Contrast ratio only
// measures lightness, so two fills that differ in hue alone — an amber search
// hit and an orange active hit — score ~1.0:1 while being easy to tell apart.
// This is the metric for "these two states must look different at a glance"
// when the states are not meant to differ in lightness.
func Distance(a, b render.Color) float64 {
	dr := float64(a.R - b.R)
	dg := float64(a.G - b.G)
	db := float64(a.B - b.B)
	return math.Sqrt(dr*dr+dg*dg+db*db) / math.Sqrt(3)
}

// tint mixes a status or accent hue into a surface to build a subtle fill that
// still reads as that status. Amount is higher on dark surfaces, where a light
// hue needs more presence to be visible.
func tint(hue, surface render.Color, amount float32) render.Color {
	c := mix(surface, hue, amount)
	c.A = 1
	return c
}

// complementRing picks the extreme — white or black — that best covers the
// control fills the normal focus ring cannot clear. Between the two rings,
// every fill a control can have is then reachable by one of them.
func complementRing(ring render.Color, fills []render.Color) render.Color {
	score := func(cand render.Color) float64 {
		w := math.Inf(1)
		for _, fill := range fills {
			if fill.A == 0 || ContrastRatio(ring, fill) >= ContrastNonText {
				continue // the normal ring already handles this one
			}
			w = math.Min(w, ContrastRatio(cand, fill))
		}
		return w
	}
	if score(white) >= score(black) {
		return white
	}
	return black
}

// separate builds a wash from hue that stays readable under fg and is still
// far enough from other to be told apart from it. It walks the mix down from
// amount, preferring the strongest wash that clears both tests.
func separate(hue, other, surface, fg render.Color, amount float32) render.Color {
	const minDistance = 0.06
	var fallback render.Color
	for a := amount; a > 0.04; a -= 0.02 {
		c := tint(hue, surface, a)
		if ContrastRatio(fg, c) < ContrastText {
			continue
		}
		if fallback.A == 0 {
			fallback = c
		}
		if Distance(c, other) >= minDistance {
			return c
		}
	}
	if fallback.A != 0 {
		return fallback
	}
	return tint(hue, surface, 0.04)
}

// highlightFill tints surface with hue as strongly as it can — up to amount —
// while keeping fg readable on the result. Editor washes (search hits, bracket
// pairs) are painted under live text, so the text wins any tie.
func highlightFill(hue, surface, fg render.Color, amount float32) render.Color {
	for a := amount; a > 0.04; a -= 0.02 {
		c := tint(hue, surface, a)
		if ContrastRatio(fg, c) >= ContrastText {
			return c
		}
	}
	return tint(hue, surface, 0.04)
}

// alpha returns c at the given alpha.
func alpha(c render.Color, a float32) render.Color {
	c.A = a
	return c
}
