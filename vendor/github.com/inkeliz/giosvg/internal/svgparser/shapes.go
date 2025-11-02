package svgparser

import (
	"gioui.org/f32"
	"math"
)

// This file implements the transformation from
// high level shapes to their path equivalent

const (
	cubicsPerHalfCircle = 8 // Number of cubic beziers to approx half a circle

	// fixed point t parameterization shift factor;
	// (2^this)/64 is the max length of t for float32
	tStrokeShift = 14

	// maxDx is the maximum radians a cubic splice is allowed to span
	// in ellipse parametric when approximating an off-axis ellipse.
	maxDx float64 = math.Pi / 8
)

// toFixedP converts two floats to a fixed point.
func toFixedP(x, y float64) (p f32.Point) {
	p.X = float32(x)
	p.Y = float32(y)
	return
}

// addRect adds a rectangle of the indicated size, rotated
// around the center by rot degrees.
func (p *Path) addRect(minX, minY, maxX, maxY, rot float64) {
	rot *= math.Pi / 180
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	m := Identity.Translate(cx, cy).Rotate(rot).Translate(-cx, -cy)
	q := &matrixAdder{M: m, path: p}
	q.Start(toFixedP(minX, minY))
	q.Line(toFixedP(maxX, minY))
	q.Line(toFixedP(maxX, maxY))
	q.Line(toFixedP(minX, maxY))
	q.path.Stop(true)
}

// length is the distance from the origin of the point
func length(v f32.Point) float32 {
	vx, vy := float64(v.X), float64(v.Y)
	return float32(math.Sqrt(vx*vx + vy*vy))
}

// addArc strokes a circular arc by approximation with bezier curves
func addArc(p *matrixAdder, a, s1, s2 f32.Point, clockwise bool, trimStart,
	trimEnd float32, firstPoint func(p f32.Point)) (ps1, ds1, ps2, ds2 f32.Point) {
	// Approximate the circular arc using a set of cubic bezier curves by the method of
	// L. Maisonobe, "Drawing an elliptical arc using polylines, quadratic
	// or cubic Bezier curves", 2003
	// https://www.spaceroots.org/documents/elllipse/elliptical-arc.pdf
	// The method was simplified for circles.
	theta1 := math.Atan2(float64(s1.Y-a.Y), float64(s1.X-a.X))
	theta2 := math.Atan2(float64(s2.Y-a.Y), float64(s2.X-a.X))
	if !clockwise {
		for theta1 < theta2 {
			theta1 += math.Pi * 2
		}
	} else {
		for theta2 < theta1 {
			theta2 += math.Pi * 2
		}
	}
	deltaTheta := theta2 - theta1
	if trimStart > 0 {
		ds := (deltaTheta * float64(trimStart)) / float64(1<<tStrokeShift)
		deltaTheta -= ds
		theta1 += ds
	}
	if trimEnd > 0 {
		ds := (deltaTheta * float64(trimEnd)) / float64(1<<tStrokeShift)
		deltaTheta -= ds
	}

	segs := int(math.Abs(deltaTheta)/(math.Pi/cubicsPerHalfCircle)) + 1
	dTheta := deltaTheta / float64(segs)
	tde := math.Tan(dTheta / 2)
	alpha := float32(math.Sin(dTheta) * (math.Sqrt(4+3*tde*tde) - 1) * (64.0 / 3.0)) // Math is fun!
	r := float64(length(s1.Sub(a)))                                                  // Note r is *64
	ldp := f32.Point{X: -float32(r * math.Sin(theta1)), Y: float32(r * math.Cos(theta1))}
	ds1 = ldp
	ps1 = f32.Point{X: a.X + ldp.Y, Y: a.Y - ldp.X}
	firstPoint(ps1)
	s1 = ps1
	for i := 1; i <= segs; i++ {
		eta := theta1 + dTheta*float64(i)
		ds2 = f32.Point{X: -float32(r * math.Sin(eta)), Y: float32(r * math.Cos(eta))}
		ps2 = f32.Point{X: a.X + ds2.Y, Y: a.Y - ds2.X} // Using deriviative to calc new pt, because circle
		p1 := s1.Add(ldp.Mul(alpha))
		p2 := ps2.Sub(ds2.Mul(alpha))
		p.CubeBezier(p1, p2, ps2)
		s1, ldp = ps2, ds2
	}
	return
}

// roundGap bridges miter-limit gaps with a circular arc
func roundGap(p *matrixAdder, a, tNorm, lNorm f32.Point) {
	addArc(p, a, a.Add(tNorm), a.Add(lNorm), true, 0, 0, p.Line)
	p.Line(a.Add(lNorm)) // just to be sure line joins cleanly,
	// last pt in stoke arc may not be precisely s2
}

// addRoundRect adds a rectangle of the indicated size, rotated
// around the center by rot degrees with rounded corners of radius
// rx in the x axis and ry in the y axis. gf specifes the shape of the
// filleting function.
func (p *Path) addRoundRect(minX, minY, maxX, maxY, rx, ry, rot float64) {
	if rx <= 0 || ry <= 0 {
		p.addRect(minX, minY, maxX, maxY, rot)
		return
	}
	rot *= math.Pi / 180

	w := maxX - minX
	if w < rx*2 {
		rx = w / 2
	}
	h := maxY - minY
	if h < ry*2 {
		ry = h / 2
	}
	stretch := rx / ry
	midY := minY + h/2
	m := Identity.Translate(minX+w/2, midY).Rotate(rot).Scale(1, 1/stretch).Translate(-minX-w/2, -minY-h/2)
	maxY = midY + h/2*stretch
	minY = midY - h/2*stretch

	q := &matrixAdder{M: m, path: p}

	q.Start(toFixedP(minX+rx, minY))
	q.Line(toFixedP(maxX-rx, minY))
	roundGap(q, toFixedP(maxX-rx, minY+rx), toFixedP(0, -rx), toFixedP(rx, 0))
	q.Line(toFixedP(maxX, maxY-rx))
	roundGap(q, toFixedP(maxX-rx, maxY-rx), toFixedP(rx, 0), toFixedP(0, rx))
	q.Line(toFixedP(minX+rx, maxY))
	roundGap(q, toFixedP(minX+rx, maxY-rx), toFixedP(0, rx), toFixedP(-rx, 0))
	q.Line(toFixedP(minX, minY+rx))
	roundGap(q, toFixedP(minX+rx, minY+rx), toFixedP(-rx, 0), toFixedP(0, -rx))
	q.path.Stop(true)
}

// addArc adds an arc to the adder p
func (p *Path) addArc(points []float64, cx, cy, px, py float64) (lx, ly float64) {
	rotX := points[2] * math.Pi / 180 // Convert degress to radians
	largeArc := points[3] != 0
	sweep := points[4] != 0
	startAngle := math.Atan2(py-cy, px-cx) - rotX
	endAngle := math.Atan2(points[6]-cy, points[5]-cx) - rotX
	deltaTheta := endAngle - startAngle
	arcBig := math.Abs(deltaTheta) > math.Pi

	// Approximate ellipse using cubic bezeir splines
	etaStart := math.Atan2(math.Sin(startAngle)/points[1], math.Cos(startAngle)/points[0])
	etaEnd := math.Atan2(math.Sin(endAngle)/points[1], math.Cos(endAngle)/points[0])
	deltaEta := etaEnd - etaStart
	if (arcBig && !largeArc) || (!arcBig && largeArc) { // Go has no boolean XOR
		if deltaEta < 0 {
			deltaEta += math.Pi * 2
		} else {
			deltaEta -= math.Pi * 2
		}
	}
	// This check might be needed if the center point of the elipse is
	// at the midpoint of the start and end lines.
	if deltaEta < 0 && sweep {
		deltaEta += math.Pi * 2
	} else if deltaEta >= 0 && !sweep {
		deltaEta -= math.Pi * 2
	}

	// Round up to determine number of cubic splines to approximate bezier curve
	segs := int(math.Abs(deltaEta)/maxDx) + 1
	dEta := deltaEta / float64(segs) // span of each segment
	// Approximate the ellipse using a set of cubic bezier curves by the method of
	// L. Maisonobe, "Drawing an elliptical arc using polylines, quadratic
	// or cubic Bezier curves", 2003
	// https://www.spaceroots.org/documents/elllipse/elliptical-arc.pdf
	tde := math.Tan(dEta / 2)
	alpha := math.Sin(dEta) * (math.Sqrt(4+3*tde*tde) - 1) / 3 // Math is fun!
	lx, ly = px, py
	sinTheta, cosTheta := math.Sin(rotX), math.Cos(rotX)
	ldx, ldy := ellipsePrime(points[0], points[1], sinTheta, cosTheta, etaStart, cx, cy)
	for i := 1; i <= segs; i++ {
		eta := etaStart + dEta*float64(i)
		var px, py float64
		if i == segs {
			px, py = points[5], points[6] // Just makes the end point exact; no roundoff error
		} else {
			px, py = ellipsePointAt(points[0], points[1], sinTheta, cosTheta, eta, cx, cy)
		}
		dx, dy := ellipsePrime(points[0], points[1], sinTheta, cosTheta, eta, cx, cy)
		p.CubeBezier(toFixedP(lx+alpha*ldx, ly+alpha*ldy),
			toFixedP(px-alpha*dx, py-alpha*dy), toFixedP(px, py))
		lx, ly, ldx, ldy = px, py, dx, dy
	}
	return lx, ly
}

// ellipsePrime gives tangent vectors for parameterized elipse; a, b, radii, eta parameter, center cx, cy
func ellipsePrime(a, b, sinTheta, cosTheta, eta, cx, cy float64) (px, py float64) {
	bCosEta := b * math.Cos(eta)
	aSinEta := a * math.Sin(eta)
	px = -aSinEta*cosTheta - bCosEta*sinTheta
	py = -aSinEta*sinTheta + bCosEta*cosTheta
	return
}

// ellipsePointAt gives points for parameterized elipse; a, b, radii, eta parameter, center cx, cy
func ellipsePointAt(a, b, sinTheta, cosTheta, eta, cx, cy float64) (px, py float64) {
	aCosEta := a * math.Cos(eta)
	bSinEta := b * math.Sin(eta)
	px = cx + aCosEta*cosTheta - bSinEta*sinTheta
	py = cy + aCosEta*sinTheta + bSinEta*cosTheta
	return
}

// findEllipseCenter locates the center of the Ellipse if it exists. If it does not exist,
// the radius values will be increased minimally for a solution to be possible
// while preserving the ra to rb ratio.  ra and rb arguments are pointers that can be
// checked after the call to see if the values changed. This method uses coordinate transformations
// to reduce the problem to finding the center of a circle that includes the origin
// and an arbitrary point. The center of the circle is then transformed
// back to the original coordinates and returned.
func findEllipseCenter(ra, rb *float64, rotX, startX, startY, endX, endY float64, sweep, smallArc bool) (cx, cy float64) {
	cos, sin := math.Cos(rotX), math.Sin(rotX)

	// Move origin to start point
	nx, ny := endX-startX, endY-startY

	// Rotate ellipse x-axis to coordinate x-axis
	nx, ny = nx*cos+ny*sin, -nx*sin+ny*cos
	// Scale X dimension so that ra = rb
	nx *= *rb / *ra // Now the ellipse is a circle radius rb; therefore foci and center coincide

	midX, midY := nx/2, ny/2
	midlenSq := midX*midX + midY*midY

	var hr float64
	if *rb**rb < midlenSq {
		// Requested ellipse does not exist; scale ra, rb to fit. Length of
		// span is greater than max width of ellipse, must scale *ra, *rb
		nrb := math.Sqrt(midlenSq)
		if *ra == *rb {
			*ra = nrb // prevents roundoff
		} else {
			*ra = *ra * nrb / *rb
		}
		*rb = nrb
	} else {
		hr = math.Sqrt(*rb**rb-midlenSq) / math.Sqrt(midlenSq)
	}
	// Notice that if hr is zero, both answers are the same.
	if (sweep && smallArc) || (!sweep && !smallArc) {
		cx = midX + midY*hr
		cy = midY - midX*hr
	} else {
		cx = midX - midY*hr
		cy = midY + midX*hr
	}

	// reverse scale
	cx *= *ra / *rb
	//Reverse rotate and translate back to original coordinates
	return cx*cos - cy*sin + startX, cx*sin + cy*cos + startY
}
