//go:build !nogpu && !darwin

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
