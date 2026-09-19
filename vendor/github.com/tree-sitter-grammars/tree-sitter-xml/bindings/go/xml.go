package tree_sitter_xml

// #cgo CPPFLAGS: -I../../dtd/src
// #cgo CFLAGS: -std=c11 -fPIC
// #include "../../xml/src/parser.c"
// #include "../../xml/src/scanner.c"
import "C"

import "unsafe"

// Get the tree-sitter Language for XML.
func LanguageXML() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_xml())
}
