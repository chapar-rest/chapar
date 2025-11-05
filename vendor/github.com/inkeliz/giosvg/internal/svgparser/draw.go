package svgparser

import (
	"gioui.org/f32"
)

// Given a parsed SVG document, implements how to
// draw it on screen.
// This requires a driver implementing the actual draw operations,
// such as a rasterizer to output .png images or a pdf writer.

// Drawer knows how to do the actual draw operations
// but doesn't need any SVG kwowledge
// In particular, tranformations matrix are already applied to the points
// before sending them to the Drawer.
type Drawer interface {
	// Start starts a new path at the given point.
	Start(a f32.Point)

	// Line Adds a line for the current point to `b`
	Line(b f32.Point)

	// QuadBezier adds a quadratic bezier curve to the path
	QuadBezier(b, c f32.Point)

	// CubeBezier adds a cubic bezier curve to the path
	CubeBezier(b, c, d f32.Point)

	// Stop closes the path to the start point if `closeLoop` is true
	Stop(closeLoop bool)

	// Draw fills or strokes the accumulated path using the given color
	Draw(color Pattern, opacity float64)
}

type Filler interface {
	Drawer

	// SetWinding Decide to use or not the "non-zero winding" rule for the current path
	SetWinding(useNonZeroWinding bool)
}

type Stroker interface {
	Drawer

	// SetStrokeOptions Parametrize the stroking style for the current path
	SetStrokeOptions(options StrokeOptions)
}

type Driver interface {
	// SetupDrawers returns the backend painters, and
	// will be called at the beginning of every path.
	SetupDrawers(willFill, willStroke bool) (Filler, Stroker)
}

type DashOptions struct {
	Dash       []float64 // values for the dash pattern (nil or an empty slice for no dashes)
	DashOffset float64   // starting offset into the dash array
}

// JoinMode type to specify how segments join.
type JoinMode uint8

// JoinMode constants determine how stroke segments bridge the gap at a join
// ArcClip mode is like MiterClip applied to arcs, and is not part of the SVG2.0
// standard.
const (
	Arc JoinMode = iota // New in SVG2
	Round
	Bevel
	Miter
	MiterClip // New in SVG2
	ArcClip   // Like MiterClip applied to arcs, and is not part of the SVG2.0 standard.
)

func (s JoinMode) String() string {
	switch s {
	case Round:
		return "Round"
	case Bevel:
		return "Bevel"
	case Miter:
		return "Miter"
	case MiterClip:
		return "MiterClip"
	case Arc:
		return "Arc"
	case ArcClip:
		return "ArcClip"
	default:
		return "<unknown JoinMode>"
	}
}

// CapMode defines how to draw caps on the ends of lines
type CapMode uint8

const (
	NilCap CapMode = iota // default value
	ButtCap
	SquareCap
	RoundCap
	CubicCap     // Not part of the SVG2.0 standard.
	QuadraticCap // Not part of the SVG2.0 standard.
)

func (c CapMode) String() string {
	switch c {
	case NilCap:
		return "NilCap"
	case ButtCap:
		return "ButtCap"
	case SquareCap:
		return "SquareCap"
	case RoundCap:
		return "RoundCap"
	case CubicCap:
		return "CubicCap"
	case QuadraticCap:
		return "QuadraticCap"
	default:
		return "<unknown CapMode>"
	}
}

// GapMode defines how to bridge gaps when the miter limit is exceeded,
// and is not part of the SVG2.0 standard.
type GapMode uint8

const (
	NilGap GapMode = iota
	FlatGap
	RoundGap
	CubicGap
	QuadraticGap
)

func (g GapMode) String() string {
	switch g {
	case NilGap:
		return "NilGap"
	case FlatGap:
		return "FlatGap"
	case RoundGap:
		return "RoundGap"
	case CubicGap:
		return "CubicGap"
	case QuadraticGap:
		return "QuadraticGap"
	default:
		return "<unknown GapMode>"
	}
}

type JoinOptions struct {
	MiterLimit   float32  // he miter cutoff value for miter, arc, miterclip and arcClip joinModes
	LineJoin     JoinMode // JoinMode for curve segments
	TrailLineCap CapMode  // capping functions for leading and trailing line ends. If one is nil, the other function is used at both ends.

	LeadLineCap CapMode // not part of the standard specification
	LineGap     GapMode // not part of the standard specification. determines how a gap on the convex side of two lines joining is filled
}

type StrokeOptions struct {
	LineWidth float32 // width of the line
	Join      JoinOptions
	Dash      DashOptions
}

// DefaultStyle sets the default PathStyle to fill black, winding rule,
// full opacity, no stroke, ButtCap line end and Bevel line connect.
var DefaultStyle = PathStyle{
	FillOpacity:       1.0,
	LineOpacity:       1.0,
	LineWidth:         1.0,
	UseNonZeroWinding: true,
	Join: JoinOptions{
		MiterLimit:   4,
		LineJoin:     Bevel,
		TrailLineCap: ButtCap,
	},
	FillerColor: NewPlainColor(0x00, 0x00, 0x00, 0xff),
	Transform:   Identity,
}

// SetTarget sets the Transform matrix to draw within the bounds of the rectangle arguments
func (s *SVGRender) SetTarget(x, y, w, h float64) {
	scaleW := w / s.ViewBox.W
	scaleH := h / s.ViewBox.H
	s.Transform = Identity.Translate(x-s.ViewBox.X, y-s.ViewBox.Y).Scale(scaleW, scaleH)
}

// Draw the compiled SVG icon into the driver `d`.
// `opacity` is composed (mutliplied) with the eventual
// <stroke-opacity> and <fill-opacity> style attributes.
// All elements should be contained by the Bounds rectangle of the SVGRender:
// see `SetTarget` method.
func (s *SVGRender) Draw(d Driver, opacity float64) {
	for _, svgp := range s.SVGPaths {
		svgp.drawTransformed(d, opacity, s.Transform)
	}
}

// drawTransformed draws the compiled SvgPath into the driver while applying transform t.
func (svgp *SvgPath) drawTransformed(d Driver, opacity float64, t Matrix2D) {
	m := svgp.Style.Transform
	svgp.Style.Transform = t.Mult(m)
	defer func() { svgp.Style.Transform = m }() // Restore untransformed matrix

	filler, stroker := d.SetupDrawers(svgp.Style.FillerColor != nil, svgp.Style.LinerColor != nil)
	if filler != nil { // nil color disable filling
		filler.SetWinding(svgp.Style.UseNonZeroWinding)

		for _, op := range svgp.Path {
			op.drawTo(filler, svgp.Style.Transform)
		}
		filler.Stop(false)

		filler.Draw(svgp.Style.FillerColor, svgp.Style.FillOpacity*opacity)
		filler.SetWinding(true) // default is true
	}

	if stroker != nil { // nil color disable lining
		lineGap := svgp.Style.Join.LineGap
		if lineGap == NilGap {
			lineGap = DefaultStyle.Join.LineGap
		}
		lineCap := svgp.Style.Join.TrailLineCap
		if lineCap == NilCap {
			lineCap = DefaultStyle.Join.TrailLineCap
		}
		leadLineCap := lineCap
		if svgp.Style.Join.LeadLineCap != NilCap {
			leadLineCap = svgp.Style.Join.LeadLineCap
		}
		stroker.SetStrokeOptions(StrokeOptions{
			LineWidth: float32(svgp.Style.LineWidth),
			Join: JoinOptions{
				MiterLimit:   svgp.Style.Join.MiterLimit,
				LineJoin:     svgp.Style.Join.LineJoin,
				LeadLineCap:  leadLineCap,
				TrailLineCap: lineCap,
				LineGap:      lineGap,
			},
			Dash: svgp.Style.Dash,
		})

		for _, op := range svgp.Path {
			op.drawTo(stroker, svgp.Style.Transform)
		}
		stroker.Stop(false)

		stroker.Draw(svgp.Style.LinerColor, svgp.Style.LineOpacity*opacity)
	}
}
