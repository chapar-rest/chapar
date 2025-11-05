package svgparser

import (
	"github.com/inkeliz/giosvg/internal/svgparser/simplexml"
	"image/color"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"
	"golang.org/x/image/math/fixed"
)

// This file defines colors and gradients used in SVG

// Pattern groups a basic color and a gradient pattern
// A nil value may by used to indicated that the function (fill or stroke) is off
type Pattern interface {
	isPattern()
}

type CurrentColor struct {
}

type PlainColor struct {
	color.NRGBA
}

func NewPlainColor(r, g, b, a uint8) PlainColor {
	return PlainColor{NRGBA: color.NRGBA{r, g, b, a}}
}

func (PlainColor) isPattern()   {}
func (Gradient) isPattern()     {}
func (CurrentColor) isPattern() {}

// enables to differentiate between black and nil color
type optionnalColor struct {
	valid bool
	color PlainColor
}

func toOptColor(p PlainColor) optionnalColor {
	return optionnalColor{valid: true, color: p}
}

func (o optionnalColor) asColor() color.Color {
	if o.valid {
		return o.color
	}
	return nil
}

func (o optionnalColor) asPattern() Pattern {
	if o.valid {
		return o.color
	}
	return nil
}

// parseSVGColor parses an SVG color string in all forms
// including all SVG1.1 names, obtained from the colornames package
func parseSVGColor(colorStr string) (optionnalColor, error) {
	v := strings.ToLower(colorStr)
	if strings.HasPrefix(v, "url") { // We are not handling urls
		// and gradients and stuff at this point
		return toOptColor(NewPlainColor(0, 0, 0, 255)), nil
	}
	switch v {
	case "none":
		// nil signals that the function (fill or stroke) is off;
		// not the same as black
		return optionnalColor{}, nil
	default:
		cn, ok := colornames.Map[v]
		if ok {
			r, g, b, a := cn.RGBA()
			return toOptColor(NewPlainColor(uint8(r), uint8(g), uint8(b), uint8(a))), nil
		}
	}
	cStr := strings.TrimPrefix(colorStr, "rgb(")
	if cStr != colorStr {
		cStr := strings.TrimSuffix(cStr, ")")
		vals := strings.Split(cStr, ",")
		if len(vals) != 3 {
			return toOptColor(PlainColor{}), errParamMismatch
		}
		var cvals [3]uint8
		var err error
		for i := range cvals {
			cvals[i], err = parseColorValue(vals[i])
			if err != nil {
				return optionnalColor{}, err
			}
		}
		return toOptColor(NewPlainColor(cvals[0], cvals[1], cvals[2], 0xFF)), nil
	}
	if colorStr[0] == '#' {
		r, g, b, err := parseSVGColorNum(colorStr)
		if err != nil {
			return optionnalColor{}, err
		}
		return toOptColor(NewPlainColor(r, g, b, 0xFF)), nil
	}
	return optionnalColor{}, errParamMismatch
}

func parseColorValue(v string) (uint8, error) {
	if v[len(v)-1] == '%' {
		n, err := strconv.Atoi(strings.TrimSpace(v[:len(v)-1]))
		if err != nil {
			return 0, err
		}
		return uint8(n * 0xFF / 100), nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if n > 255 {
		n = 255
	}
	return uint8(n), err
}

// parseSVGColorNum reads the SFG color string e.g. #FBD9BD
func parseSVGColorNum(colorStr string) (r, g, b uint8, err error) {
	colorStr = strings.TrimPrefix(colorStr, "#")
	var t uint64
	if len(colorStr) != 6 {
		// SVG specs say duplicate characters in case of 3 digit hex number
		colorStr = string([]byte{colorStr[0], colorStr[0],
			colorStr[1], colorStr[1], colorStr[2], colorStr[2]})
	}
	for _, v := range []struct {
		c *uint8
		s string
	}{
		{&r, colorStr[0:2]},
		{&g, colorStr[2:4]},
		{&b, colorStr[4:6]}} {
		t, err = strconv.ParseUint(v.s, 16, 8)
		if err != nil {
			return
		}
		*v.c = uint8(t)
	}
	return
}

// GradientUnits is the type for gradient units
type GradientUnits byte

// SVG bounds paremater constants
const (
	ObjectBoundingBox GradientUnits = iota
	UserSpaceOnUse
)

// SpreadMethod is the type for spread parameters
type SpreadMethod byte

// SVG spread parameter constants
const (
	PadSpread SpreadMethod = iota
	ReflectSpread
	RepeatSpread
)

// GradStop represents a stop in the SVG 2.0 gradient specification
type GradStop struct {
	StopColor color.Color
	Offset    float64
	Opacity   float64
}

// Gradient holds a description of an SVG 2.0 gradient
type Gradient struct {
	Direction gradientDirecter
	Stops     []GradStop
	Bounds    Bounds
	Matrix    Matrix2D
	Spread    SpreadMethod
	Units     GradientUnits
}

// ApplyPathExtent use the given path extent to adjust the bounding box,
// if required by `Units`.
// The `Direction` field is not modified, but a matrix accounting for both the bouding box and
// the gradient matrix is returned
func (g *Gradient) ApplyPathExtent(extent fixed.Rectangle26_6) Matrix2D {
	if g.Units == ObjectBoundingBox {
		mnx, mny := float64(extent.Min.X/64), float64(extent.Min.Y/64)
		mxx, mxy := float64(extent.Max.X/64), float64(extent.Max.Y/64)
		g.Bounds.X, g.Bounds.Y = mnx, mny
		g.Bounds.W, g.Bounds.H = mxx-mnx, mxy-mny

		// units in Direction are fraction, so
		// we apply bounds
		return Identity.Scale(g.Bounds.W, g.Bounds.H).Mult(g.Matrix)
	}
	// units in Direction are already scaled to the view box
	// just return the gradient matrix
	return g.Matrix
}

// radial or linear
type gradientDirecter interface {
	isRadial() bool
}

// x1, y1, x2, y2
type Linear [4]float64

func (Linear) isRadial() bool { return false }

// cx, cy, fx, fy, r, fr
type Radial [6]float64

func (Radial) isRadial() bool { return true }

// getColor is a helper function to get the background color
// if ReadGradUrl needs it.
func getColor(clr Pattern) color.Color {
	switch c := clr.(type) {
	case Gradient: // This is a bit lazy but oh well
		for _, s := range c.Stops {
			if s.StopColor != nil {
				return s.StopColor
			}
		}
	case PlainColor:
		return c
	}
	return colornames.Black
}

func localizeGradIfStopClrNil(g *Gradient, defaultColor Pattern) Gradient {
	grad := *g
	for _, s := range grad.Stops {
		if s.StopColor == nil { // This means we need copy the gradient's Stop slice
			// and fill in the default color

			// Copy the stops
			stops := append([]GradStop{}, grad.Stops...)
			grad.Stops = stops
			// Use the background color when a stop color is nil
			clr := getColor(defaultColor)
			for i, s := range stops {
				if s.StopColor == nil {
					grad.Stops[i].StopColor = clr
				}
			}
			break // Only need to do this once
		}
	}
	return grad
}

// readGradURL reads an SVG format gradient url
// Since the context of the gradient can affect the colors
// the current fill or line color is passed in and used in
// the case of a nil stopClor value
func (c *iconCursor) readGradURL(v string, defaultColor Pattern) (grad Gradient, ok bool) {
	if strings.HasPrefix(v, "url(") && strings.HasSuffix(v, ")") {
		urlStr := strings.TrimSpace(v[4 : len(v)-1])
		if strings.HasPrefix(urlStr, "#") {
			var g *Gradient
			g, ok = c.icon.grads[urlStr[1:]]
			if ok {
				grad = localizeGradIfStopClrNil(g, defaultColor)
			}
		}
	}
	return
}

// readGradAttr reads an SVG gradient attribute
func (c *iconCursor) readGradAttr(attr simplexml.Attr) (err error) {
	switch attr.Name.Local {
	case "gradientTransform":
		c.grad.Matrix, err = c.parseTransform(attr.Value)
	case "gradientUnits":
		switch strings.TrimSpace(attr.Value) {
		case "userSpaceOnUse":
			c.grad.Units = UserSpaceOnUse
		case "objectBoundingBox":
			c.grad.Units = ObjectBoundingBox
		}
	case "spreadMethod":
		switch strings.TrimSpace(attr.Value) {
		case "pad":
			c.grad.Spread = PadSpread
		case "reflect":
			c.grad.Spread = ReflectSpread
		case "repeat":
			c.grad.Spread = RepeatSpread
		}
	}
	return
}
