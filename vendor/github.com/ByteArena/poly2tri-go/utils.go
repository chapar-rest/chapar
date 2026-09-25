package poly2tri

import (
	"math"
)

var EPSILON = 0.00001

var Orientation = map[string]int{
	"CW":       1,
	"CCW":      -1,
	"COLINEAR": 0,
}

func Orient2d(pa, pb, pc *Point) int {

	detleft := (pa.X - pc.X) * (pb.Y - pc.Y)
	detright := (pa.Y - pc.Y) * (pb.X - pc.X)

	val := detleft - detright

	if math.Abs(val) < EPSILON {
		return Orientation["COLLINEAR"]
	} else if val > 0 {
		return Orientation["CCW"]
	}

	return Orientation["CW"]
}

func InScanArea(pa, pb, pc, pd *Point) bool {
	oadb := (pa.X-pb.X)*(pd.Y-pb.Y) - (pd.X-pb.X)*(pa.Y-pb.Y)
	if oadb >= -EPSILON {
		return false
	}

	oadc := (pa.X-pc.X)*(pd.Y-pc.Y) - (pd.X-pc.X)*(pa.Y-pc.Y)
	if oadc <= EPSILON {
		return false
	}

	return true
}

func IsAngleObtuse(pa, pb, pc *Point) bool {
	ax := pb.X - pa.X
	ay := pb.Y - pa.Y
	bx := pc.X - pa.X
	by := pc.Y - pa.Y

	return (ax*bx + ay*by) < 0
}
