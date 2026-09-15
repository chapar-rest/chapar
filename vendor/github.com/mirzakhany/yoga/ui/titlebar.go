package ui

import (
	"time"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

const titleBarWidgetID = "yoga-titlebar"

type titleBarState struct {
	lastPress time.Time
}

type titleBarView struct {
	children []View
}

// TitleBar is a VS Code-style custom title bar row. It reserves space for native
// window controls on macOS and appends framework min/max/close buttons on
// Windows and Linux. Empty areas drag the window; double-click toggles maximize.
func TitleBar(children ...View) *titleBarView {
	return &titleBarView{children: children}
}

func (t *titleBarView) Layout(c *Ctx) *layout.Element {
	th := c.Theme()
	height := th.Metrics.TitleBarHeight
	kids := make([]View, 0, len(t.children)+2)

	win := c.Window()
	if win != nil && win.NativeControls() {
		inset := win.ControlsInset()
		if inset > 0 {
			// Fixed width only — Spacer() defaults to FlexGrow(1) and would absorb
			// leftover row space, pushing menus away from the traffic lights.
			kids = append(kids, Spacer().Width(inset).Grow(0).Shrink(0))
		}
	}
	kids = append(kids, t.children...)
	if win != nil && !win.NativeControls() && win.CustomTitleBar() {
		kids = append(kids, WindowControls())
	}

	row := Row(kids...).
		Gap(th.Spacing.S).
		PaddingXY(th.Spacing.M, 0).
		Height(height).
		Background(TokenChrome).
		Shrink(0).
		Align(AlignCenter)

	old := c.pushEnv(env{
		controlHeight:    th.Metrics.TitleBarControlHeight,
		hasControlHeight: true,
	})
	el := row.Layout(c)
	c.popEnv(old)
	st := c.Widget(titleBarWidgetID, func() any { return &titleBarState{} }).(*titleBarState)

	prevOnMouse := el.OnMouse
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if prevOnMouse != nil {
			prevOnMouse(e, m)
		}
		if m.Consumed {
			return
		}
		inside := e.Frame.Contains(m.X, m.Y)
		if !inside {
			return
		}
		if m.Pressed {
			now := c.now
			if !st.lastPress.IsZero() && now.Sub(st.lastPress) < 400*time.Millisecond {
				if win != nil {
					win.ToggleMaximize()
				}
				st.lastPress = time.Time{}
				m.Consumed = true
				return
			}
			st.lastPress = now
			if win != nil {
				win.BeginMove()
			}
			m.Consumed = true
		}
	}
	return el
}

type windowControlsView struct{}

// WindowControls renders minimize, maximize/restore, and close buttons for
// undecorated windows. TitleBar includes these automatically on Windows/Linux.
func WindowControls() *windowControlsView {
	return &windowControlsView{}
}

func (w *windowControlsView) Layout(c *Ctx) *layout.Element {
	th := c.Theme()
	h := th.Metrics.TitleBarHeight
	btnW := h * 1.35
	if btnW < 40 {
		btnW = 40
	}
	win := c.Window()
	maxIcon := icons.Maximize
	if win != nil && win.IsMaximized() {
		maxIcon = icons.Minimize2
	}
	return Row(
		windowControlBtn("yoga-win-min", icons.Minus, btnW, h, false, func() {
			if win != nil {
				win.Minimize()
			}
		}),
		windowControlBtn("yoga-win-max", maxIcon, btnW, h, false, func() {
			if win != nil {
				win.ToggleMaximize()
			}
		}),
		windowControlBtn("yoga-win-close", icons.X, btnW, h, true, func() {
			if win != nil {
				win.Close()
			}
		}),
	).Shrink(0).Layout(c)
}

type winCtrlState struct {
	hovered, pressed bool
}

func windowControlBtn(id string, icon icons.Icon, w, h float32, closeBtn bool, onClick func()) View {
	return &winCtrlButton{id: id, icon: icon, w: w, h: h, closeBtn: closeBtn, onClick: onClick}
}

type winCtrlButton struct {
	id       string
	icon     icons.Icon
	w, h     float32
	closeBtn bool
	onClick  func()
}

func (b *winCtrlButton) Layout(c *Ctx) *layout.Element {
	st := c.Widget(b.id, func() any { return &winCtrlState{} }).(*winCtrlState)
	th := c.Theme()
	el := layout.New(layout.Box().Size(b.w, b.h).FlexShrink(0))
	icon := b.icon
	closeBtn := b.closeBtn
	onClick := b.onClick
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		frame := el.Frame
		bg := render.Color{}
		if st.pressed {
			if closeBtn {
				bg = th.Error
			} else {
				bg = th.ListActive
			}
		} else if st.hovered {
			if closeBtn {
				bg = th.Error
				bg.A = 0.85
			} else {
				bg = th.ListHover
			}
		}
		if bg.A > 0 {
			dl.AddRect(frame, bg)
		}
		col := th.Foreground
		if closeBtn && (st.hovered || st.pressed) {
			col = th.AccentForeground
		}
		inset := b.h * 0.32
		inner := render.Rect{X: frame.X + (frame.W-inset)/2, Y: frame.Y + (frame.H-inset)/2, W: inset, H: inset}
		if sheet := frameIcons(); sheet != nil && !icon.Empty() {
			sheet.Draw(dl, icon, inner, col)
		}
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		inside := e.Frame.Contains(m.X, m.Y)
		trackHover(c, &st.hovered, inside)
		if inside {
			m.SetCursor(CursorPointer)
		}
		if inside && m.Pressed {
			trackBool(c, &st.pressed, true)
			m.Consumed = true
		}
		if m.Released {
			if st.pressed && inside && onClick != nil {
				onClick()
				c.MarkNeedsPaint()
			}
			trackBool(c, &st.pressed, false)
		}
	}
	return el
}
