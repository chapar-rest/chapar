package poly2tri

import (
	"fmt"
	"math"
)

type XYInterface interface {
	String() string
	GetX() float64
	GetY() float64
}

func XYCompare(a, b XYInterface) float64 {

	xeq := XYEqualsFloat(a.GetX(), b.GetX())
	yeq := XYEqualsFloat(a.GetY(), b.GetY())

	if yeq {
		if xeq {
			return 0
		}

		return a.GetX() - b.GetX()
	}

	return a.GetY() - b.GetY()
}

func XYEquals(a, b XYInterface) bool {
	return XYEqualsFloat(a.GetX(), b.GetX()) &&
		XYEqualsFloat(a.GetY(), b.GetY())
}

func XYString(a XYInterface) string {
	return fmt.Sprintf("(%f;%f)", a.GetX(), a.GetY())
}

func XYEqualsFloat(a, b float64) bool {
	return math.Abs(a-b) <= EPSILON
}

func XYCompareFloat(a, b float64) float64 {
	if XYEqualsFloat(a, b) {
		return 0
	}

	return a - b
}
