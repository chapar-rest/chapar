//go:build !nogpu

package yoga

import (
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/mirzakhany/yoga/input"
)

const resizeEdgeSize = 6

type resizeEdge int

const (
	resizeNone resizeEdge = iota
	resizeTop
	resizeBottom
	resizeLeft
	resizeRight
	resizeTopLeft
	resizeTopRight
	resizeBottomLeft
	resizeBottomRight
)

// glfwWindowHost implements ui.WindowHost for GLFW windows.
type glfwWindowHost struct {
	window         *glfw.Window
	customTitleBar bool
	nativeControls bool
	controlsInset  float32
	undecorated    bool

	dragging                         bool
	dragWinX, dragWinY               int
	dragMouseX, dragMouseY           float64
	resizing                         bool
	resizeEdge                       resizeEdge
	resizeWinX, resizeWinY           int
	resizeWinW, resizeWinH           int
	resizeMouseX, resizeMouseY       float64
}

func newGLFWWindowHost(window *glfw.Window, cfg Config) *glfwWindowHost {
	h := &glfwWindowHost{
		window:         window,
		customTitleBar: cfg.CustomTitleBar,
	}
	if cfg.CustomTitleBar {
		applyCustomTitleBarChrome(window, h)
	}
	return h
}

func (h *glfwWindowHost) CustomTitleBar() bool { return h.customTitleBar }
func (h *glfwWindowHost) NativeControls() bool { return h.nativeControls }
func (h *glfwWindowHost) ControlsInset() float32 {
	if h.nativeControls {
		return h.controlsInset
	}
	return 0
}

func (h *glfwWindowHost) Close() { h.window.SetShouldClose(true) }

func (h *glfwWindowHost) Minimize() { h.window.Iconify() }

func (h *glfwWindowHost) ToggleMaximize() {
	if h.window.GetAttrib(glfw.Maximized) == glfw.True {
		h.window.Restore()
	} else {
		h.window.Maximize()
	}
}

func (h *glfwWindowHost) IsMaximized() bool {
	return h.window.GetAttrib(glfw.Maximized) == glfw.True
}

func (h *glfwWindowHost) BeginMove() {
	if h.IsMaximized() {
		return
	}
	x, y := h.window.GetPos()
	mx, my := h.window.GetCursorPos()
	h.dragging = true
	h.dragWinX, h.dragWinY = x, y
	h.dragMouseX, h.dragMouseY = mx, my
}

// updateFrame handles ongoing window drag and edge resize for undecorated windows.
func (h *glfwWindowHost) updateFrame(m *input.Mouse, w, hgt float32) {
	if m == nil {
		return
	}
	if h.dragging {
		if !m.Down {
			h.dragging = false
		} else {
			mx, my := h.window.GetCursorPos()
			nx := h.dragWinX + int(mx-h.dragMouseX)
			ny := h.dragWinY + int(my-h.dragMouseY)
			h.window.SetPos(nx, ny)
		}
	}
	if h.undecorated {
		h.updateResize(m, w, hgt)
	}
}

func (h *glfwWindowHost) updateResize(m *input.Mouse, w, hgt float32) {
	if h.IsMaximized() {
		h.resizing = false
		h.resizeEdge = resizeNone
		return
	}
	mx, my := m.X, m.Y
	edge := detectResizeEdge(mx, my, w, hgt)

	if h.resizing {
		if !m.Down {
			h.resizing = false
			h.resizeEdge = resizeNone
			return
		}
		curMX, curMY := h.window.GetCursorPos()
		dx := curMX - h.resizeMouseX
		dy := curMY - h.resizeMouseY
		x, y := h.resizeWinX, h.resizeWinY
		width, height := h.resizeWinW, h.resizeWinH
		minW, minH := 200, 120

		switch h.resizeEdge {
		case resizeLeft:
			nx := x + int(dx)
			nw := width - int(dx)
			if nw >= minW {
				x, width = nx, nw
			}
		case resizeRight:
			width = max(minW, width+int(dx))
		case resizeTop:
			ny := y + int(dy)
			nh := height - int(dy)
			if nh >= minH {
				y, height = ny, nh
			}
		case resizeBottom:
			height = max(minH, height+int(dy))
		case resizeTopLeft:
			nx := x + int(dx)
			nw := width - int(dx)
			ny := y + int(dy)
			nh := height - int(dy)
			if nw >= minW {
				x, width = nx, nw
			}
			if nh >= minH {
				y, height = ny, nh
			}
		case resizeTopRight:
			width = max(minW, width+int(dx))
			ny := y + int(dy)
			nh := height - int(dy)
			if nh >= minH {
				y, height = ny, nh
			}
		case resizeBottomLeft:
			nx := x + int(dx)
			nw := width - int(dx)
			if nw >= minW {
				x, width = nx, nw
			}
			height = max(minH, height+int(dy))
		case resizeBottomRight:
			width = max(minW, width+int(dx))
			height = max(minH, height+int(dy))
		}
		h.window.SetPos(x, y)
		h.window.SetSize(width, height)
		return
	}

	if edge != resizeNone {
		setResizeCursor(m, edge)
		if m.Pressed {
			wx, wy := h.window.GetPos()
			ww, wh := h.window.GetSize()
			curMX, curMY := h.window.GetCursorPos()
			h.resizing = true
			h.resizeEdge = edge
			h.resizeWinX, h.resizeWinY = wx, wy
			h.resizeWinW, h.resizeWinH = ww, wh
			h.resizeMouseX, h.resizeMouseY = curMX, curMY
			m.Consumed = true
		}
	}
}

func detectResizeEdge(mx, my, w, hgt float32) resizeEdge {
	e := float32(resizeEdgeSize)
	onLeft := mx <= e
	onRight := mx >= w-e
	onTop := my <= e
	onBottom := my >= hgt-e
	switch {
	case onTop && onLeft:
		return resizeTopLeft
	case onTop && onRight:
		return resizeTopRight
	case onBottom && onLeft:
		return resizeBottomLeft
	case onBottom && onRight:
		return resizeBottomRight
	case onTop:
		return resizeTop
	case onBottom:
		return resizeBottom
	case onLeft:
		return resizeLeft
	case onRight:
		return resizeRight
	default:
		return resizeNone
	}
}

func setResizeCursor(m *input.Mouse, edge resizeEdge) {
	switch edge {
	case resizeLeft, resizeRight:
		m.SetCursor(input.CursorResizeEW)
	case resizeTop, resizeBottom:
		m.SetCursor(input.CursorResizeNS)
	default:
		// GLFW has no diagonal resize cursors in our set; EW/NS is acceptable for v1.
		m.SetCursor(input.CursorResizeEW)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
