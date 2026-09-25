package assets

import _ "embed"

// ChaparPNG is the product logo for empty-state and splash views.
//
//go:embed images/chapar.png
var ChaparPNG []byte

// ChaparConfusedPNG is shown above the error of a failed request.
//
//go:embed images/chapar-confused.png
var ChaparConfusedPNG []byte
