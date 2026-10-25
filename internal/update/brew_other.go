//go:build !darwin

package update

import (
	"context"
	"io"
)

// InstalledVersion is only known for macOS app bundles.
func InstalledVersion() string { return "" }

func BrewUpgrade(context.Context, io.Writer) error { return ErrUnsupported }
