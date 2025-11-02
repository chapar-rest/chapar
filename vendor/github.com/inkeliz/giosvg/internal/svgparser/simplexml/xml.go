// Package simplexml is used to parse the SVG xml file,
// but it avoids `encoding/xml`for JS, and it's compatible
// with TinyGo.
package simplexml

import (
	"io"
)

// A Name represents an XML name (Local) annotated
// with a name space identifier (Space).
// In tokens returned by Decoder.Token, the Space identifier
// is given as a canonical URL, not the short prefix used
// in the document being parsed.
type Name struct {
	Space, Local string
}

// An Attr represents an attribute in an XML element (Name=Value).
type Attr struct {
	Name  Name
	Value string
}

// A StartElement represents an XML start element.
type StartElement struct {
	Name Name
	Attr []Attr
}

// An EndElement represents an XML end element.
type EndElement struct {
	Name Name
}

// A Token is an interface holding one of the token types:
// StartElement, EndElement.
type Token interface{}

type Decoder interface {
	Token() (Token, error)
}

func NewDecoder(r io.Reader) Decoder {
	return newDecoder(r)
}
