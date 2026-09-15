//go:build !nogpu && !js

package yoga

import (
	"fmt"
	"runtime"

	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

func init() {
	runtime.LockOSThread()
}

// Window owns the window, GPU renderer, text engine, and input state.
type Window struct {
	window   *glfw.Window
	text     *shape.Engine
	renderer *render.Renderer
	mouse    *input.Mouse
	keyboard *input.Keyboard
	clip     input.Clipboard
	cursors  map[input.Cursor]*glfw.Cursor
	closed   bool
	drawList render.DrawList
	winHost  *glfwWindowHost

	// ui app path: set by runApp.
	uiApp   App
	uiCtx   *ui.Ctx
	uiFocus *ui.FocusScope
}

// New creates the window, text engine, WebGPU renderer, and input wiring.
func New(cfg Config) (*Window, error) {
	cfg = cfg.applyDefaults()

	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("yoga: glfw init: %w", err)
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	prepareCustomTitleBarHints(cfg.CustomTitleBar)
	window, err := glfw.CreateWindow(cfg.Width, cfg.Height, cfg.Title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("yoga: create window: %w", err)
	}

	w, h := window.GetSize()
	fbW, fbH := window.GetFramebufferSize()
	scale := float32(1)
	if w > 0 {
		scale = float32(fbW) / float32(w)
	}

	text, err := shape.NewEngine(scale, true)
	if err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil, fmt.Errorf("yoga: text engine: %w", err)
	}

	renderer, err := render.NewRenderer(wgpuglfw.GetSurfaceDescriptor(window), fbW, fbH, w, h, text.Atlas)
	if err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil, fmt.Errorf("yoga: create renderer: %w", err)
	}
	renderer.ClearColor = theme.Current().Surface

	clip := input.Clipboard(&glfwClipboard{window: window})
	icons := render.NewSpriteSheet(text.Atlas)
	a := &Window{
		window:   window,
		text:     text,
		renderer: renderer,
		mouse:    &input.Mouse{},
		keyboard: &input.Keyboard{},
		clip:     clip,
	}
	a.winHost = newGLFWWindowHost(window, cfg)
	a.initCursors()
	a.wireCallbacks()
	SetResources(text, icons, clip)
	return a, nil
}

func (a *Window) initCursors() {
	shapes := map[input.Cursor]glfw.StandardCursor{
		input.CursorDefault:  glfw.ArrowCursor,
		input.CursorPointer:  glfw.HandCursor,
		input.CursorResizeEW: glfw.HResizeCursor,
		input.CursorResizeNS: glfw.VResizeCursor,
		input.CursorText:     glfw.IBeamCursor,
	}
	a.cursors = make(map[input.Cursor]*glfw.Cursor, len(shapes))
	for kind, shape := range shapes {
		a.cursors[kind] = glfw.CreateStandardCursor(shape)
	}
}

func (a *Window) applyCursor() {
	c := a.cursors[a.mouse.Cursor]
	if c == nil {
		c = a.cursors[input.CursorDefault]
	}
	a.window.SetCursor(c)
}

func (a *Window) wireCallbacks() {
	a.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		a.mouse.SetPos(float32(x), float32(y))
	})
	a.window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		a.mouse.Mods = mapMods(mods)
		switch button {
		case glfw.MouseButtonLeft:
			a.mouse.SetButton(action == glfw.Press)
		case glfw.MouseButtonRight:
			a.mouse.SetRightButton(action == glfw.Press)
		}
	})
	a.window.SetScrollCallback(func(_ *glfw.Window, xoff, yoff float64) {
		a.mouse.AddScrollX(float32(xoff))
		a.mouse.AddScroll(float32(yoff))
	})
	a.window.SetCharCallback(func(_ *glfw.Window, char rune) {
		a.keyboard.TypeRune(char)
	})
	a.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		if k, ok := mapKey(key); ok {
			a.keyboard.PressKey(k, mapMods(mods))
		}
	})
	a.window.SetFramebufferSizeCallback(func(win *glfw.Window, fbW, fbH int) {
		logicalW, logicalH := win.GetSize()
		if fbW <= 0 || fbH <= 0 {
			return
		}
		a.renderer.Resize(fbW, fbH, logicalW, logicalH)
		// Repaint synchronously during live resize so the window doesn't blank.
		if a.uiApp != nil {
			a.uiCtx.MarkNeedsPaint()
			a.paintAppFrame(a.uiApp, float32(logicalW), float32(logicalH))
			a.uiCtx.ClearNeedsPaint()
			if _, ok := a.uiCtx.AnimationWait(); ok {
				a.uiCtx.MarkNeedsPaint()
			}
		}
	})
}

func (a *Window) Window() *glfw.Window { return a.window }

// runApp drives the ui.App frame loop: it rebuilds the body when the loop
// wakes (input, Invalidate, or Animate), hit-tests, then paints only when
// something visual changed. Body is rebuilt a second time before paint when
// input handlers mutated state so the same frame reflects that change; idle
// WaitEvents means none of this runs.
func (a *Window) runApp(app App) {
	a.uiApp = app
	a.uiFocus = ui.NewFocusScope()
	a.uiCtx = ui.New(a.text, a.uiFocus, glfw.PostEmptyEvent)
	a.uiCtx.SetIcons(Icons())
	a.uiCtx.SetClipboard(a.clip)
	a.uiCtx.SetWindow(a.winHost)

	var lastW, lastH float32
	var lastMX, lastMY float32
	for !a.window.ShouldClose() {
		if theme.SyncSystem() {
			a.renderer.ClearColor = theme.Current().Surface
			a.uiCtx.MarkNeedsPaint()
		}
		fw, fh := a.window.GetSize()
		if fw > 0 && fh > 0 {
			w, h := float32(fw), float32(fh)
			if w != lastW || h != lastH {
				a.uiCtx.MarkNeedsPaint()
				lastW, lastH = w, h
			}
			a.mouse.Cursor = input.CursorDefault

			// Build for input, dispatch, then route keys.
			inRoot := a.buildAppFrame(app, w, h)

			a.uiCtx.BeginInputPhase()
			layout.Dispatch(inRoot, a.mouse)
			a.uiFocus.HandleMouse(a.mouse)
			if a.uiCtx.Commands() != nil {
				a.uiCtx.Commands().Dispatch(a.keyboard)
			}
			if hook, ok := app.(KeyHook); ok {
				a.routeAppKeys(hook)
			}
			a.uiFocus.Route(a.keyboard)

			if a.winHost != nil {
				a.winHost.updateFrame(a.mouse, w, h)
			}

			// Coarse input that always implies a visual update (click, scroll,
			// typing, drag). Hover transitions mark via trackHover in widgets.
			if a.mouse.Pressed || a.mouse.Released ||
				a.mouse.RightPressed || a.mouse.RightReleased ||
				a.mouse.ScrollX != 0 || a.mouse.ScrollY != 0 ||
				len(a.keyboard.Chars) > 0 || len(a.keyboard.Keys) > 0 ||
				(a.mouse.Down && (a.mouse.X != lastMX || a.mouse.Y != lastMY)) {
				a.uiCtx.MarkNeedsPaint()
			}
			lastMX, lastMY = a.mouse.X, a.mouse.Y

			a.applyCursor()
			a.mouse.EndFrame()
			a.keyboard.EndFrame()
			a.uiCtx.EndInputPhase()

			// Animated widgets (caret, spinner, tooltip delay) must present even
			// when input did not dirty the frame.
			if _, ok := a.uiCtx.AnimationWait(); ok {
				a.uiCtx.MarkNeedsPaint()
			}

			paint, rebuild := ui.FramePaintPlan(a.uiCtx.NeedsPaint(), a.uiCtx.InputDirty())
			if paint {
				if rebuild {
					a.paintAppFrame(app, w, h)
				} else {
					a.presentFrame(inRoot)
				}
				a.uiCtx.ClearNeedsPaint()
			}
			// Re-arm so the next Animate wake presents even if this wake skipped paint.
			if _, ok := a.uiCtx.AnimationWait(); ok {
				a.uiCtx.MarkNeedsPaint()
			}
			a.uiCtx.EndFrame()
		}

		if d, ok := a.uiCtx.AnimationWait(); ok {
			if d < 0 {
				d = 0
			}
			glfw.WaitEventsTimeout(d.Seconds())
		} else {
			glfw.WaitEvents()
		}
	}

	if c, ok := app.(Closer); ok {
		c.Close()
	}
}

func (a *Window) Close() {
	if a.closed {
		return
	}
	a.closed = true
	for _, c := range a.cursors {
		if c != nil {
			c.Destroy()
		}
	}
	if a.renderer != nil {
		a.renderer.Destroy()
	}
	if a.window != nil {
		a.window.Destroy()
	}
	glfw.Terminate()
}

func mapKey(k glfw.Key) (input.Key, bool) {
	switch k {
	case glfw.KeyLeft:
		return input.KeyLeft, true
	case glfw.KeyRight:
		return input.KeyRight, true
	case glfw.KeyUp:
		return input.KeyUp, true
	case glfw.KeyDown:
		return input.KeyDown, true
	case glfw.KeyHome:
		return input.KeyHome, true
	case glfw.KeyEnd:
		return input.KeyEnd, true
	case glfw.KeyBackspace:
		return input.KeyBackspace, true
	case glfw.KeyDelete:
		return input.KeyDelete, true
	case glfw.KeyEnter, glfw.KeyKPEnter:
		return input.KeyEnter, true
	case glfw.KeyTab:
		return input.KeyTab, true
	case glfw.KeyEscape:
		return input.KeyEscape, true
	case glfw.KeySpace:
		return input.KeySpace, true
	case glfw.KeyA:
		return input.KeyA, true
	case glfw.KeyB:
		return input.KeyB, true
	case glfw.KeyC:
		return input.KeyC, true
	case glfw.KeyD:
		return input.KeyD, true
	case glfw.KeyE:
		return input.KeyE, true
	case glfw.KeyF:
		return input.KeyF, true
	case glfw.KeyG:
		return input.KeyG, true
	case glfw.KeyH:
		return input.KeyH, true
	case glfw.KeyI:
		return input.KeyI, true
	case glfw.KeyJ:
		return input.KeyJ, true
	case glfw.KeyK:
		return input.KeyK, true
	case glfw.KeyL:
		return input.KeyL, true
	case glfw.KeyM:
		return input.KeyM, true
	case glfw.KeyN:
		return input.KeyN, true
	case glfw.KeyO:
		return input.KeyO, true
	case glfw.KeyP:
		return input.KeyP, true
	case glfw.KeyQ:
		return input.KeyQ, true
	case glfw.KeyR:
		return input.KeyR, true
	case glfw.KeyS:
		return input.KeyS, true
	case glfw.KeyT:
		return input.KeyT, true
	case glfw.KeyU:
		return input.KeyU, true
	case glfw.KeyV:
		return input.KeyV, true
	case glfw.KeyW:
		return input.KeyW, true
	case glfw.KeyX:
		return input.KeyX, true
	case glfw.KeyY:
		return input.KeyY, true
	case glfw.KeyZ:
		return input.KeyZ, true
	case glfw.Key0:
		return input.Key0, true
	case glfw.Key1:
		return input.Key1, true
	case glfw.Key2:
		return input.Key2, true
	case glfw.Key3:
		return input.Key3, true
	case glfw.Key4:
		return input.Key4, true
	case glfw.Key5:
		return input.Key5, true
	case glfw.Key6:
		return input.Key6, true
	case glfw.Key7:
		return input.Key7, true
	case glfw.Key8:
		return input.Key8, true
	case glfw.Key9:
		return input.Key9, true
	case glfw.KeyF1:
		return input.KeyF1, true
	case glfw.KeyF2:
		return input.KeyF2, true
	case glfw.KeyF3:
		return input.KeyF3, true
	case glfw.KeyF4:
		return input.KeyF4, true
	case glfw.KeyF5:
		return input.KeyF5, true
	case glfw.KeyF6:
		return input.KeyF6, true
	case glfw.KeyF7:
		return input.KeyF7, true
	case glfw.KeyF8:
		return input.KeyF8, true
	case glfw.KeyF9:
		return input.KeyF9, true
	case glfw.KeyF10:
		return input.KeyF10, true
	case glfw.KeyF11:
		return input.KeyF11, true
	case glfw.KeyF12:
		return input.KeyF12, true
	case glfw.KeyComma:
		return input.KeyComma, true
	case glfw.KeyPeriod:
		return input.KeyPeriod, true
	case glfw.KeySlash:
		return input.KeySlash, true
	case glfw.KeyMinus:
		return input.KeyMinus, true
	case glfw.KeyEqual:
		return input.KeyEqual, true
	case glfw.KeyLeftBracket:
		return input.KeyLeftBracket, true
	case glfw.KeyRightBracket:
		return input.KeyRightBracket, true
	case glfw.KeyGraveAccent:
		return input.KeyBacktick, true
	case glfw.KeySemicolon:
		return input.KeySemicolon, true
	case glfw.KeyApostrophe:
		return input.KeyApostrophe, true
	case glfw.KeyBackslash:
		return input.KeyBackslash, true
	}
	return input.KeyNone, false
}

func mapMods(m glfw.ModifierKey) input.Mod {
	var mods input.Mod
	if m&glfw.ModShift != 0 {
		mods |= input.ModShift
	}
	if m&glfw.ModControl != 0 {
		mods |= input.ModCtrl
	}
	if m&glfw.ModAlt != 0 {
		mods |= input.ModAlt
	}
	if m&glfw.ModSuper != 0 {
		mods |= input.ModSuper
	}
	return mods
}

type glfwClipboard struct{ window *glfw.Window }

func (c *glfwClipboard) Get() string  { return c.window.GetClipboardString() }
func (c *glfwClipboard) Set(s string) { c.window.SetClipboardString(s) }
