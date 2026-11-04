package tree_sitter_php

// #cgo CFLAGS: -I../../php/src -std=c11 -fPIC
// #include "../../php/src/parser.c"
// #include "../../php/src/scanner.c"
import "C"

import "unsafe"

// Get the tree-sitter Language for the PHP grammar.
func LanguagePHP() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_php())
}
