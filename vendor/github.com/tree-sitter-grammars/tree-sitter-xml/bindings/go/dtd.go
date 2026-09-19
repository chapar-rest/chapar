package tree_sitter_xml

// #cgo CPPFLAGS: -I../../xml/src
// #cgo CFLAGS: -std=c11 -fPIC
// #include "../../dtd/src/parser.c"
// #include "../../dtd/src/scanner.c"
import "C"

import "unsafe"

// Get the tree-sitter Language for DTD.
func LanguageDTD() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_dtd())
}
