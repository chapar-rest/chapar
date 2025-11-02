package svgparser

import (
	"github.com/inkeliz/giosvg/internal/svgparser/simplexml"
	"io"
)

// PathStyle holds the state of the SVG style
type PathStyle struct {
	FillOpacity, LineOpacity float64
	LineWidth                float64
	UseNonZeroWinding        bool

	Join                    JoinOptions
	Dash                    DashOptions
	FillerColor, LinerColor Pattern // either PlainColor or Gradient

	Transform Matrix2D // current transform
}

// SvgPath binds a style to a path
type SvgPath struct {
	Path  Path
	Style PathStyle
}

// Bounds defines a bounding box, such as a viewport
// or a path extent.
type Bounds struct{ X, Y, W, H float64 }

// SVGRender holds data from parsed SVGs.
// See the `Draw` methods to use it.
type SVGRender struct {
	ViewBox      Bounds
	Titles       []string // Title elements collect here
	Descriptions []string // Description elements collect here
	SVGPaths     []SvgPath
	Transform    Matrix2D

	grads map[string]*Gradient
	defs  map[string][]definition
}

// ReadIcon reads the Icon from the given io.Reader
// This only supports a sub-set of SVG, but
// is enough to draw many icons. errMode determines if the icon ignores, errors out, or logs a warning
// if it does not handle an element found in the icon file.
func ReadIcon(stream io.Reader) (*SVGRender, error) {
	icon := &SVGRender{defs: make(map[string][]definition), grads: make(map[string]*Gradient), Transform: Identity}
	cursor := &iconCursor{styleStack: []PathStyle{DefaultStyle}, icon: icon}
	decoder := simplexml.NewDecoder(stream)
	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return icon, err
		}
		// Inspect the type of the XML token
		switch se := t.(type) {
		case simplexml.StartElement:
			// Reads all recognized style attributes from the start element
			// and places it on top of the styleStack
			err = cursor.pushStyle(se.Attr)
			if err != nil {
				return icon, err
			}
			err = cursor.readStartElement(se)
			if err != nil {
				return icon, err
			}
		case simplexml.EndElement:
			// pop style
			cursor.styleStack = cursor.styleStack[:len(cursor.styleStack)-1]
			switch se.Name.Local {
			case "g":
				if cursor.inDefs {
					cursor.currentDef = append(cursor.currentDef, definition{
						Tag: "endg",
					})
				}
			case "title":
				cursor.inTitleText = false
			case "desc":
				cursor.inDescText = false
			case "defs":
				if len(cursor.currentDef) > 0 {
					cursor.icon.defs[cursor.currentDef[0].ID] = cursor.currentDef
					cursor.currentDef = make([]definition, 0)
				}
				cursor.inDefs = false
			case "radialGradient", "linearGradient":
				cursor.inGrad = false
			}
		}
	}
	return icon, nil
}
