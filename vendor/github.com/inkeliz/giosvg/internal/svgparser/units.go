package svgparser

import (
	"math"
	"strconv"
	"strings"
)

var root2 = math.Sqrt(2)

type unite uint8

// Absolute units supported.
const (
	Px unite = iota
	Cm
	Mm
	Pt
	In
	Q
	Pc
	Perc // Special case : percentage (%) relative to the viewbox
)

var absoluteUnits = [...]string{Px: "px", Cm: "cm", Mm: "mm", Pt: "pt", In: "in", Q: "Q", Pc: "pc", Perc: "%"}

var toPx = [...]float64{Px: 1, Cm: 96. / 2.54, Mm: 9.6 / 2.54, Pt: 96. / 72., In: 96., Q: 96. / 40. / 2.54, Pc: 96. / 6., Perc: 1}

// look for an absolute unit, or nothing (considered as pixels)
// % is also supported
func findUnit(s string) (u unite, value string) {
	s = strings.TrimSpace(s)
	for u, suffix := range absoluteUnits {
		if strings.HasSuffix(s, suffix) {
			valueS := strings.TrimSpace(strings.TrimSuffix(s, suffix))
			return unite(u), valueS
		}
	}
	return Px, s
}

// convert the unite to pixels. Return true if it is a %
func parseUnit(s string) (float64, bool, error) {
	unite, value := findUnit(s)
	out, err := strconv.ParseFloat(value, 64)
	return out * toPx[unite], unite == Perc, err
}

type percentageReference uint8

const (
	widthPercentage percentageReference = iota
	heightPercentage
	diagPercentage
)

// resolveUnit converts a length with a unit into its value in 'px'
// percentage are supported, and refer to the viewBox
// `asPerc` is only applied when `s` contains a percentage.
func (viewBox Bounds) resolveUnit(s string, asPerc percentageReference) (float64, error) {
	value, isPercentage, err := parseUnit(s)
	if err != nil {
		return 0, err
	}
	if isPercentage {
		w, h := viewBox.W, viewBox.H
		switch asPerc {
		case widthPercentage:
			return value / 100 * w, nil
		case heightPercentage:
			return value / 100 * h, nil
		case diagPercentage:
			normalizedDiag := math.Sqrt(w*w+h*h) / root2
			return value / 100 * normalizedDiag, nil
		}
	}
	return value, nil
}

// parseUnit converts a length with a unit into its value in 'px'
// percentage are supported, and refer to the current ViewBox
func (c *iconCursor) parseUnit(s string, asPerc percentageReference) (float64, error) {
	return c.icon.ViewBox.resolveUnit(s, asPerc)
}

func parseBasicFloat(s string) (float64, error) {
	value, _, err := parseUnit(s)
	return value, err
}

func readFraction(v string) (f float64, err error) {
	v = strings.TrimSpace(v)
	d := 1.0
	if strings.HasSuffix(v, "%") {
		d = 100
		v = strings.TrimSuffix(v, "%")
	}
	f, err = parseBasicFloat(v)
	f /= d
	// Is this is an unnecessary restriction? For now fractions can be all values not just in the range [0,1]
	// if f > 1 {
	// 	f = 1
	// } else if f < 0 {
	// 	f = 0
	// }
	return
}
