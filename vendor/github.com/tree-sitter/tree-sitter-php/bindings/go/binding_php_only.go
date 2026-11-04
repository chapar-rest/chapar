package tree_sitter_php

// #cgo CFLAGS: -I../../php_only/src -std=c11 -fPIC
// #include "../../php_only/src/parser.c"
// #include "../../php_only/src/scanner.c"
import "C"

import "unsafe"

// Get the tree-sitter Language for PHP-Only.
func LanguagePHPOnly() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_php_only())
}
