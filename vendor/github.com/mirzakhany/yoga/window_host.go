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

	dragging                   bool
	dragMouseX, dragMouseY     float64
	resizing                   bool
	resizeEdge                 resizeEdge
	resizeWinX, resizeWinY     int
	resizeWinW, resizeWinH     int
	resizeMouseX, resizeMouseY float64
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
	if beginNativeWindowMove(h.window) {
		return
	}
	mx, my := h.window.GetCursorPos()
	if h.IsMaximized() {
		h.restoreForMove(mx, my)
	} else {
		h.dragMouseX, h.dragMouseY = mx, my
	}
	h.dragging = true
}

// restoreForMove unmaximizes and places the restored window so the grab point
// stays under the cursor (standard Windows/Linux title-bar tear-off).
func (h *glfwWindowHost) restoreForMove(mx, my float64) {
	maxX, maxY := h.window.GetPos()
	maxW, _ := h.window.GetSize()
	h.window.Restore()
	newW, _ := h.window.GetSize()
	screenX := maxX + int(mx)
	screenY := maxY + int(my)
	grabX := int(mx)
	if maxW > 0 && newW > 0 {
		grabX = int(mx * float64(newW) / float64(maxW))
	}
	if grabX < 0 {
		grabX = 0
	} else if newW > 0 && grabX > newW {
		grabX = newW
	}
	h.window.SetPos(screenX-grabX, screenY-int(my))
	h.dragMouseX = float64(grabX)
	h.dragMouseY = my
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
			curX, curY := h.window.GetPos()
			mx, my := h.window.GetCursorPos()
			// Move by remaining window-relative error so the grab point stays
			// under the cursor. A frozen origin plus window-relative cursor
			// oscillates after each SetPos.
			h.window.SetPos(curX+int(mx-h.dragMouseX), curY+int(my-h.dragMouseY))
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
