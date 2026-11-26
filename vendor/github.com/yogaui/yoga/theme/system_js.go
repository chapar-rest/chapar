//go:build js

package theme

import "syscall/js"

func osPrefersDark() bool {
	media := js.Global().Call("matchMedia", "(prefers-color-scheme: dark)")
	if media.Truthy() {
		return media.Get("matches").Bool()
	}
	return false
}

func osPrefersCoarsePointer() bool {
	media := js.Global().Call("matchMedia", "(pointer: coarse)")
	return media.Truthy() && media.Get("matches").Bool()
}

func osPrefersReducedMotion() bool {
	media := js.Global().Call("matchMedia", "(prefers-reduced-motion: reduce)")
	return media.Truthy() && media.Get("matches").Bool()
}
