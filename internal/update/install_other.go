//go:build !darwin && !windows

package update

import "context"

// Detect: Linux installs put the binary under /usr/local/bin with sudo, so
// the app can't replace itself; the user downloads the new version.
func Detect() Method { return Manual }

func Install(context.Context, string) error { return ErrUnsupported }

func Restart() error { return ErrUnsupported }

func Cleanup() {}
