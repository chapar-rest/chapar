//go:build !nogpu && !js && !darwin

package yoga

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

func applyCustomTitleBarChrome(window *glfw.Window, h *glfwWindowHost) {
	h.nativeControls = false
	h.undecorated = true
	h.controlsInset = 0
	_ = window
}

func beginNativeWindowMove(window *glfw.Window) bool {
	_ = window
	return false
}
