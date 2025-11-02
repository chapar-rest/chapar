//go:build js
// +build js

package simplexml

import (
	"bytes"
	"io"
	"syscall/js"
	"unsafe"
)

type decoder struct {
	document js.Value

	parsed bool
	tokens []Token
	next   int
}

var (
	_Parser = js.Global().Get("DOMParser").New()

	_Reflect    = js.Global().Get("Reflect")
	_ReflectGet = _Reflect.Get("get").Call("bind").Invoke

	_length     = js.ValueOf("length")
	_tagName    = js.ValueOf("tagName")
	_attributes = js.ValueOf("attributes")
	_children   = js.ValueOf("children")
	_value      = js.ValueOf("value")
	_name       = js.ValueOf("name")
)

func newDecoder(r io.Reader) Decoder {
	var b []byte
	switch r := r.(type) {
	case *bytes.Reader:
		s := (*[1]uintptr)(unsafe.Pointer(&r))
		b = *(*[]byte)(unsafe.Pointer(s[0]))
	case io.Reader:
		b, _ = io.ReadAll(r)
	}

	d := &decoder{document: _Parser.Call("parseFromString", *(*string)(unsafe.Pointer(&b)), "image/svg+xml")}
	d.decode(_ReflectGet(d.document, _children))
	return d
}

func (d *decoder) decode(children js.Value) {
	if !children.Truthy() {
		return
	}

	for i := 0; i < _ReflectGet(children, _length).Int(); i++ {
		elem := children.Index(i)

		tag := Name{Local: _ReflectGet(elem, _tagName).String()}
		start, end := StartElement{Name: tag}, EndElement{Name: tag}

		attributes := _ReflectGet(elem, _attributes)
		start.Attr = make([]Attr, _ReflectGet(attributes, _length).Int())
		for i := 0; i < len(start.Attr); i++ {
			a := attributes.Index(i)
			start.Attr[i] = Attr{Name: Name{Local: _ReflectGet(a, _name).String()}, Value: _ReflectGet(a, _value).String()}
		}

		d.tokens = append(d.tokens, start)

		if c := _ReflectGet(elem, _children); c.Truthy() && _ReflectGet(c, _length).Int() > 0 {
			d.decode(c)
		}

		d.tokens = append(d.tokens, end)
	}
}

func (d *decoder) Token() (Token, error) {
	if len(d.tokens) <= d.next {
		return nil, io.EOF
	}

	t := d.tokens[d.next]
	d.next++
	return t, nil
}
