package svgparser

// this file implements the translation of an SVG2.0 path into a Path.

import (
	"errors"
	"gioui.org/f32"
	"log"
	"math"
	"unicode"
)

// ErrorMode is the for setting how the parser reacts to unparsed elements
type ErrorMode uint8

const (
	// IgnoreErrorMode skips unparsed SVG elements
	IgnoreErrorMode ErrorMode = iota
	// WarnErrorMode outputs a warning when an unparsed SVG element is found
	WarnErrorMode
	// StrictErrorMode causes a error when an unparsed SVG element is found
	StrictErrorMode
)

var (
	errParamMismatch  = errors.New("param mismatch")
	errCommandUnknown = errors.New("unknown command")
	errZeroLengthID   = errors.New("zero length id")
)

// pathCursor is used to parse SVG format path strings into a Path
type pathCursor struct {
	path                   Path
	placeX, placeY         float64
	curX, curY             float64
	cntlPtX, cntlPtY       float64
	pathStartX, pathStartY float64
	points                 []float64
	lastKey                uint8
	errorMode              ErrorMode
	inPath                 bool
}

func (c *pathCursor) init() {
	c.placeX = 0.0
	c.placeY = 0.0
	c.points = c.points[0:0]
	c.lastKey = ' '
	c.path.Clear()
	c.inPath = false
}

// compilePath translates the svgPath description string into a path.
// The resulting path element is stored in the pathCursor.
func (c *pathCursor) compilePath(svgPath string) error {
	c.init()
	lastIndex := -1
	for i, v := range svgPath {
		if unicode.IsLetter(v) && v != 'e' {
			if lastIndex != -1 {
				if err := c.addSeg(svgPath[lastIndex:i]); err != nil {
					return err
				}
			}
			lastIndex = i
		}
	}
	if lastIndex != -1 {
		if err := c.addSeg(svgPath[lastIndex:]); err != nil {
			return err
		}
	}
	return nil
}

func reflect(px, py, rx, ry float64) (x, y float64) {
	return px*2 - rx, py*2 - ry
}

func (c *pathCursor) valsToAbs(last float64) {
	for i := 0; i < len(c.points); i++ {
		last += c.points[i]
		c.points[i] = last
	}
}

func (c *pathCursor) pointsToAbs(sz int) {
	lastX := c.placeX
	lastY := c.placeY
	for j := 0; j < len(c.points); j += sz {
		for i := 0; i < sz; i += 2 {
			c.points[i+j] += lastX
			c.points[i+1+j] += lastY
		}
		lastX = c.points[(j+sz)-2]
		lastY = c.points[(j+sz)-1]
	}
}

func (c *pathCursor) hasSetsOrMore(sz int, rel bool) bool {
	if !(len(c.points) >= sz && len(c.points)%sz == 0) {
		return false
	}
	if rel {
		c.pointsToAbs(sz)
	}
	return true
}

// readFloat reads a floating point value and adds it to the cursor's points slice.
func (c *pathCursor) readFloat(numStr string) error {
	last := 0
	isFirst := true
	for i, n := range numStr {
		if n == '.' {
			if isFirst {
				isFirst = false
				continue
			}
			f, err := parseBasicFloat(numStr[last:i])
			if err != nil {
				return err
			}
			c.points = append(c.points, f)
			last = i
		}
	}
	f, err := parseBasicFloat(numStr[last:])
	if err != nil {
		return err
	}
	c.points = append(c.points, f)
	return nil
}

// getPoints reads a set of floating point values from the SVG format number string,
// and add them to the cursor's points slice.
func (c *pathCursor) getPoints(dataPoints string) error {
	lastIndex := -1
	c.points = c.points[0:0]
	lr := ' '
	for i, r := range dataPoints {
		if !unicode.IsNumber(r) && r != '.' && !(r == '-' && lr == 'e') && r != 'e' {
			if lastIndex != -1 {
				if err := c.readFloat(dataPoints[lastIndex:i]); err != nil {
					return err
				}
			}
			if r == '-' {
				lastIndex = i
			} else {
				lastIndex = -1
			}
		} else if lastIndex == -1 {
			lastIndex = i
		}
		lr = r
	}
	if lastIndex != -1 && lastIndex != len(dataPoints) {
		if err := c.readFloat(dataPoints[lastIndex:]); err != nil {
			return err
		}
	}
	return nil
}

func (c *pathCursor) reflectControlQuad() {
	switch c.lastKey {
	case 'q', 'Q', 'T', 't':
		c.cntlPtX, c.cntlPtY = reflect(c.placeX, c.placeY, c.cntlPtX, c.cntlPtY)
	default:
		c.cntlPtX, c.cntlPtY = c.placeX, c.placeY
	}
}

func (c *pathCursor) reflectControlCube() {
	switch c.lastKey {
	case 'c', 'C', 's', 'S':
		c.cntlPtX, c.cntlPtY = reflect(c.placeX, c.placeY, c.cntlPtX, c.cntlPtY)
	default:
		c.cntlPtX, c.cntlPtY = c.placeX, c.placeY
	}
}

// addSeg decodes an SVG seqment string into equivalent raster path commands saved
// in the cursor's Path
func (c *pathCursor) addSeg(segString string) error {
	// Parse the string describing the numeric points in SVG format
	if err := c.getPoints(segString[1:]); err != nil {
		return err
	}
	l := len(c.points)
	k := segString[0]
	rel := false
	switch k {
	case 'z':
		fallthrough
	case 'Z':
		if len(c.points) != 0 {
			return errParamMismatch
		}
		if c.inPath {
			c.path.Stop(true)
			c.placeX = c.pathStartX
			c.placeY = c.pathStartY
			c.inPath = false
		}
	case 'm':
		rel = true
		fallthrough
	case 'M':
		if !c.hasSetsOrMore(2, rel) {
			return errParamMismatch
		}
		c.pathStartX, c.pathStartY = c.points[0], c.points[1]
		c.inPath = true
		c.path.Start(f32.Point{X: float32(c.pathStartX + c.curX), Y: float32(c.pathStartY + c.curY)})
		for i := 2; i < l-1; i += 2 {
			c.path.Line(f32.Point{
				X: float32(c.points[i] + c.curX),
				Y: float32(c.points[i+1] + c.curY),
			})
		}
		c.placeX = c.points[l-2]
		c.placeY = c.points[l-1]
	case 'l':
		rel = true
		fallthrough
	case 'L':
		if !c.hasSetsOrMore(2, rel) {
			return errParamMismatch
		}
		for i := 0; i < l-1; i += 2 {
			c.path.Line(f32.Point{
				X: float32(c.points[i] + c.curX),
				Y: float32(c.points[i+1] + c.curY),
			})
		}
		c.placeX = c.points[l-2]
		c.placeY = c.points[l-1]
	case 'v':
		c.valsToAbs(c.placeY)
		fallthrough
	case 'V':
		if !c.hasSetsOrMore(1, false) {
			return errParamMismatch
		}
		for _, p := range c.points {
			c.path.Line(f32.Point{
				X: float32(c.placeX + c.curX),
				Y: float32(p + c.curY),
			})
		}
		c.placeY = c.points[l-1]
	case 'h':
		c.valsToAbs(c.placeX)
		fallthrough
	case 'H':
		if !c.hasSetsOrMore(1, false) {
			return errParamMismatch
		}
		for _, p := range c.points {
			c.path.Line(f32.Point{
				X: float32(p + c.curX),
				Y: float32(c.placeY + c.curY),
			})
		}
		c.placeX = c.points[l-1]
	case 'q':
		rel = true
		fallthrough
	case 'Q':
		if !c.hasSetsOrMore(4, rel) {
			return errParamMismatch
		}
		for i := 0; i < l-3; i += 4 {
			c.path.QuadBezier(
				f32.Point{
					X: float32(c.points[i] + c.curX),
					Y: float32(c.points[i+1] + c.curY),
				},
				f32.Point{
					X: float32(c.points[i+2] + c.curX),
					Y: float32(c.points[i+3] + c.curY),
				})
		}
		c.cntlPtX, c.cntlPtY = c.points[l-4], c.points[l-3]
		c.placeX = c.points[l-2]
		c.placeY = c.points[l-1]
	case 't':
		rel = true
		fallthrough
	case 'T':
		if !c.hasSetsOrMore(2, rel) {
			return errParamMismatch
		}
		for i := 0; i < l-1; i += 2 {
			c.reflectControlQuad()
			c.path.QuadBezier(
				f32.Point{
					X: float32(c.cntlPtX + c.curX),
					Y: float32(c.cntlPtY + c.curY),
				},
				f32.Point{
					X: float32(c.points[i] + c.curX),
					Y: float32(c.points[i+1] + c.curY),
				})
			c.lastKey = k
			c.placeX = c.points[i]
			c.placeY = c.points[i+1]
		}
	case 'c':
		rel = true
		fallthrough
	case 'C':
		if !c.hasSetsOrMore(6, rel) {
			return errParamMismatch
		}
		for i := 0; i < l-5; i += 6 {
			c.path.CubeBezier(
				f32.Point{
					X: float32(c.points[i] + c.curX),
					Y: float32(c.points[i+1] + c.curY),
				},
				f32.Point{
					X: float32(c.points[i+2] + c.curX),
					Y: float32(c.points[i+3] + c.curY),
				},
				f32.Point{
					X: float32(c.points[i+4] + c.curX),
					Y: float32(c.points[i+5] + c.curY),
				})
		}
		c.cntlPtX, c.cntlPtY = c.points[l-4], c.points[l-3]
		c.placeX = c.points[l-2]
		c.placeY = c.points[l-1]
	case 's':
		rel = true
		fallthrough
	case 'S':
		if !c.hasSetsOrMore(4, rel) {
			return errParamMismatch
		}
		for i := 0; i < l-3; i += 4 {
			c.reflectControlCube()
			c.path.CubeBezier(
				f32.Point{
					X: float32(c.cntlPtX + c.curX), Y: float32(c.cntlPtY + c.curY),
				},
				f32.Point{
					X: float32(c.points[i] + c.curX), Y: float32(c.points[i+1] + c.curY),
				},
				f32.Point{
					X: float32(c.points[i+2] + c.curX), Y: float32(c.points[i+3] + c.curY),
				},
			)
			c.lastKey = k
			c.cntlPtX, c.cntlPtY = c.points[i], c.points[i+1]
			c.placeX = c.points[i+2]
			c.placeY = c.points[i+3]
		}
	case 'a', 'A':
		if !c.hasSetsOrMore(7, false) {
			return errParamMismatch
		}
		for i := 0; i < l-6; i += 7 {
			if k == 'a' {
				c.points[i+5] += c.placeX
				c.points[i+6] += c.placeY
			}
			c.addArcFromA(c.points[i:])
		}
	default:
		if c.errorMode == StrictErrorMode {
			return errCommandUnknown
		}
		if c.errorMode == WarnErrorMode {
			log.Println("Ignoring svg command " + string(k))
		}
	}
	// So we know how to extend some segment types
	c.lastKey = k
	return nil
}

// ellipseAt adds a path of an elipse centered at cx, cy of radius rx and ry
// to the pathCursor
func (c *pathCursor) ellipseAt(cx, cy, rx, ry float64) {
	c.placeX, c.placeY = cx+rx, cy
	c.points = c.points[0:0]
	c.points = append(c.points, rx, ry, 0.0, 1.0, 0.0, c.placeX, c.placeY)
	c.path.Start(f32.Point{
		X: float32(c.placeX),
		Y: float32(c.placeY),
	})
	c.placeX, c.placeY = c.path.addArc(c.points, cx, cy, c.placeX, c.placeY)
	c.path.Stop(true)
}

// addArcFromA adds a path of an arc element to the cursor path to the pathCursor
func (c *pathCursor) addArcFromA(points []float64) {
	cx, cy := findEllipseCenter(&points[0], &points[1], points[2]*math.Pi/180, c.placeX,
		c.placeY, points[5], points[6], points[4] == 0, points[3] == 0)
	c.placeX, c.placeY = c.path.addArc(c.points, cx+c.curX, cy+c.curY, c.placeX+c.curX, c.placeY+c.curY)
}
