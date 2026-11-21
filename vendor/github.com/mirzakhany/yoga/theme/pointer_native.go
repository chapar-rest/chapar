//go:build !js

package theme

// osPrefersCoarsePointer reports whether the primary pointer is a finger.
// Native desktop builds drive a mouse or trackpad.
func osPrefersCoarsePointer() bool { return false }
