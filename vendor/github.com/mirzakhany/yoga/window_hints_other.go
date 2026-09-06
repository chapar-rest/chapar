//go:build !nogpu && !darwin

package yoga

import "github.com/go-gl/glfw/v3.3/glfw"

func prepareCustomTitleBarHints(custom bool) {
	if custom {
		glfw.WindowHint(glfw.Decorated, glfw.False)
	}
}
