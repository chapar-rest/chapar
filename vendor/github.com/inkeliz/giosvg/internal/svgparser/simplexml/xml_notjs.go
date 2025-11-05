//go:build !js
// +build !js

package simplexml

import (
	"encoding/xml"
	"golang.org/x/net/html/charset"
	"io"
	"unsafe"
)

type decoder struct {
	*xml.Decoder
}

func newDecoder(r io.Reader) Decoder {
	d := xml.NewDecoder(r)
	d.CharsetReader = charset.NewReaderLabel

	return &decoder{Decoder: d}
}

func (d *decoder) Token() (Token, error) {
	t, err := d.Decoder.Token()
	if err != nil {
		return nil, err
	}
	switch t := t.(type) {
	case xml.StartElement:
		t.Name.Space = ""
		return *(*StartElement)(unsafe.Pointer(&t)), nil
	case xml.EndElement:
		t.Name.Space = ""
		return *(*EndElement)(unsafe.Pointer(&t)), nil
	default:
		// Unsupported operation.
		return nil, nil
	}
}
