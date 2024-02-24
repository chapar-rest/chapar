package component

import (
	"image/color"
)

// WithAlpha returns the input color with the new alpha value.
func WithAlpha(c color.NRGBA, a uint8) color.NRGBA {
	return color.NRGBA{
		R: c.R,
		G: c.G,
		B: c.B,
		A: a,
	}
}

// AlphaPalette is the set of alpha values to be applied for certain
// material design states like hover, selected, etc...
type AlphaPalette struct {
	Hover, Selected uint8
}
