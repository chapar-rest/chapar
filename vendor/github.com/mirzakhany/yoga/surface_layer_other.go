//go:build !nogpu && !js && !darwin

package yoga

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/mirzakhany/yoga/render"
)

func syncSurfaceLayer(*glfw.Window, render.Color) {}
