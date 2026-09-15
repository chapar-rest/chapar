//go:build js

package yoga

import (
	"fmt"
	"syscall/js"
	"time"

	"github.com/cogentcore/webgpu/wgpu"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// Window owns the canvas, GPU renderer, text engine, and input state (browser).
type Window struct {
	canvas   js.Value
	text     *shape.Engine
	renderer *render.Renderer
	mouse    *input.Mouse
	keyboard *input.Keyboard
	clip     input.Clipboard
	closed   bool
	drawList render.DrawList
	winHost  *jsWindowHost

	logicalW, logicalH int
	fbW, fbH           int
	dpr                float64

	wakeCh      chan struct{}
	eventFuncs  []js.Func
	rafCallback js.Func
	rafPending  bool
	running     bool

	uiApp   App
	uiCtx   *ui.Ctx
	uiFocus *ui.FocusScope
}

// New creates the canvas, text engine, WebGPU renderer, and input wiring.
func New(cfg Config) (*Window, error) {
	cfg = cfg.applyDefaults()

	if !js.Global().Get("navigator").Get("gpu").Truthy() {
		return nil, fmt.Errorf("yoga: WebGPU not available in this browser (navigator.gpu missing)")
	}

	doc := js.Global().Get("document")
	canvas := doc.Call("createElement", "canvas")
	canvas.Set("id", "yoga-canvas")
	style := canvas.Get("style")
	style.Set("display", "block")
	style.Set("width", "100vw")
	style.Set("height", "100vh")
	style.Set("touchAction", "none")
	doc.Get("body").Call("appendChild", canvas)
	doc.Get("body").Get("style").Set("margin", "0")
	doc.Set("title", cfg.Title)

	dpr := js.Global().Get("devicePixelRatio").Float()
	if dpr <= 0 {
		dpr = 1
	}
	logicalW, logicalH := viewportSize()
	if logicalW <= 0 {
		logicalW = cfg.Width
	}
	if logicalH <= 0 {
		logicalH = cfg.Height
	}
	fbW := int(float64(logicalW) * dpr)
	fbH := int(float64(logicalH) * dpr)
	canvas.Set("width", fbW)
	canvas.Set("height", fbH)

	text, err := shape.NewEngine(float32(dpr), false)
	if err != nil {
		return nil, fmt.Errorf("yoga: text engine: %w", err)
	}

	renderer, err := render.NewRenderer(
		&wgpu.SurfaceDescriptor{Canvas: canvas},
		fbW, fbH, logicalW, logicalH, text.Atlas,
	)
	if err != nil {
		return nil, fmt.Errorf("yoga: create renderer: %w", err)
	}
	renderer.ClearColor = theme.Current().Surface

	a := &Window{
		canvas:   canvas,
		text:     text,
		renderer: renderer,
		mouse:    &input.Mouse{},
		keyboard: &input.Keyboard{},
		clip:     &jsClipboard{},
		winHost:  &jsWindowHost{customTitleBar: cfg.CustomTitleBar},
		logicalW: logicalW,
		logicalH: logicalH,
		fbW:      fbW,
		fbH:      fbH,
		dpr:      dpr,
		wakeCh:   make(chan struct{}, 1),
	}
	a.wireCallbacks()
	SetResources(text, render.NewSpriteSheet(text.Atlas), a.clip)
	return a, nil
}

func viewportSize() (w, h int) {
	vv := js.Global().Get("visualViewport")
	if vv.Truthy() {
		return vv.Get("width").Int(), vv.Get("height").Int()
	}
	return js.Global().Get("innerWidth").Int(), js.Global().Get("innerHeight").Int()
}

func (a *Window) wake() {
	select {
	case a.wakeCh <- struct{}{}:
	default:
	}
	a.scheduleFrame()
}

func (a *Window) scheduleFrame() {
	if a.rafPending || a.closed {
		return
	}
	a.rafPending = true
	js.Global().Call("requestAnimationFrame", a.rafCallback)
}

func (a *Window) wireCallbacks() {
	a.rafCallback = js.FuncOf(func(this js.Value, args []js.Value) any {
		a.rafPending = false
		a.wake()
		return nil
	})

	add := func(target js.Value, event string, fn func(js.Value)) {
		wrapped := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				fn(args[0])
			}
			a.wake()
			return nil
		})
		a.eventFuncs = append(a.eventFuncs, wrapped)
		target.Call("addEventListener", event, wrapped)
	}

	add(a.canvas, "pointermove", func(e js.Value) {
		rect := a.canvas.Call("getBoundingClientRect")
		x := float32(e.Get("clientX").Float() - rect.Get("left").Float())
		y := float32(e.Get("clientY").Float() - rect.Get("top").Float())
		a.mouse.SetPos(x, y)
	})
	add(a.canvas, "pointerdown", func(e js.Value) {
		a.canvas.Call("setPointerCapture", e.Get("pointerId"))
		btn := e.Get("button").Int()
		a.mouse.Mods = mapDOMMods(e)
		switch btn {
		case 0:
			a.mouse.SetButton(true)
		case 2:
			a.mouse.SetRightButton(true)
		}
	})
	add(a.canvas, "pointerup", func(e js.Value) {
		btn := e.Get("button").Int()
		a.mouse.Mods = mapDOMMods(e)
		switch btn {
		case 0:
			a.mouse.SetButton(false)
		case 2:
			a.mouse.SetRightButton(false)
		}
	})
	add(a.canvas, "wheel", func(e js.Value) {
		e.Call("preventDefault")
		// DOM wheel deltas are pixels; Yoga scroll expects roughly line units (~1).
		a.mouse.AddScrollX(float32(-e.Get("deltaX").Float() / 40))
		a.mouse.AddScroll(float32(-e.Get("deltaY").Float() / 40))
	})
	add(a.canvas, "contextmenu", func(e js.Value) {
		e.Call("preventDefault")
	})

	win := js.Global().Get("window")
	add(win, "keydown", func(e js.Value) {
		if k, ok := mapDOMKey(e.Get("key").String(), e.Get("code").String()); ok {
			a.keyboard.PressKey(k, mapDOMMods(e))
			if k == input.KeyTab || k == input.KeyBackspace || k == input.KeyEnter {
				e.Call("preventDefault")
			}
		}
	})
	add(win, "keypress", func(e js.Value) {
		key := e.Get("key").String()
		if len(key) == 1 {
			a.keyboard.TypeRune([]rune(key)[0])
		}
	})
	add(win, "resize", func(e js.Value) {
		_ = e
		a.handleResize()
	})
}

func (a *Window) handleResize() {
	dpr := js.Global().Get("devicePixelRatio").Float()
	if dpr <= 0 {
		dpr = 1
	}
	logicalW, logicalH := viewportSize()
	if logicalW <= 0 || logicalH <= 0 {
		return
	}
	fbW := int(float64(logicalW) * dpr)
	fbH := int(float64(logicalH) * dpr)
	a.canvas.Set("width", fbW)
	a.canvas.Set("height", fbH)
	a.dpr = dpr
	a.logicalW, a.logicalH = logicalW, logicalH
	a.fbW, a.fbH = fbW, fbH
	a.renderer.Resize(fbW, fbH, logicalW, logicalH)
	if a.uiApp != nil {
		a.uiCtx.MarkNeedsPaint()
	}
}

func (a *Window) applyCursor() {
	css := "default"
	switch a.mouse.Cursor {
	case input.CursorPointer:
		css = "pointer"
	case input.CursorResizeEW:
		css = "ew-resize"
	case input.CursorResizeNS:
		css = "ns-resize"
	case input.CursorText:
		css = "text"
	}
	a.canvas.Get("style").Set("cursor", css)
}

// runApp drives the UI frame loop using requestAnimationFrame + a wake channel.
func (a *Window) runApp(app App) {
	a.uiApp = app
	a.uiFocus = ui.NewFocusScope()
	a.uiCtx = ui.New(a.text, a.uiFocus, a.wake)
	a.uiCtx.SetIcons(Icons())
	a.uiCtx.SetClipboard(a.clip)
	a.uiCtx.SetWindow(a.winHost)
	a.running = true

	var lastW, lastH float32
	var lastMX, lastMY float32

	a.uiCtx.MarkNeedsPaint()
	a.scheduleFrame()

	for a.running && !a.closed {
		if theme.SyncSystem() {
			a.renderer.ClearColor = theme.Current().Surface
			a.uiCtx.MarkNeedsPaint()
		}

		w, h := float32(a.logicalW), float32(a.logicalH)
		if w > 0 && h > 0 {
			if w != lastW || h != lastH {
				a.uiCtx.MarkNeedsPaint()
				lastW, lastH = w, h
			}
			a.mouse.Cursor = input.CursorDefault

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
			if _, ok := a.uiCtx.AnimationWait(); ok {
				a.uiCtx.MarkNeedsPaint()
			}
			a.uiCtx.EndFrame()
		}

		if d, ok := a.uiCtx.AnimationWait(); ok {
			if d < 0 {
				d = 0
			}
			timer := time.NewTimer(d)
			select {
			case <-a.wakeCh:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
				a.scheduleFrame()
			}
		} else {
			<-a.wakeCh
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
	a.running = false
	a.wake()
	for _, f := range a.eventFuncs {
		f.Release()
	}
	a.rafCallback.Release()
	if a.renderer != nil {
		a.renderer.Destroy()
	}
}

type jsWindowHost struct {
	customTitleBar bool
}

func (h *jsWindowHost) CustomTitleBar() bool { return h.customTitleBar }
func (h *jsWindowHost) NativeControls() bool { return false }
func (h *jsWindowHost) ControlsInset() float32 {
	return 0
}
func (h *jsWindowHost) Close()            {}
func (h *jsWindowHost) Minimize()         {}
func (h *jsWindowHost) ToggleMaximize()   {}
func (h *jsWindowHost) IsMaximized() bool { return false }
func (h *jsWindowHost) BeginMove()        {}

type jsClipboard struct{}

func (c *jsClipboard) Get() string {
	// Synchronous clipboard read is restricted; return empty and rely on paste events later.
	return ""
}

func (c *jsClipboard) Set(s string) {
	js.Global().Get("navigator").Get("clipboard").Call("writeText", s)
}

func mapDOMMods(e js.Value) input.Mod {
	var mods input.Mod
	if e.Get("shiftKey").Bool() {
		mods |= input.ModShift
	}
	if e.Get("ctrlKey").Bool() {
		mods |= input.ModCtrl
	}
	if e.Get("altKey").Bool() {
		mods |= input.ModAlt
	}
	if e.Get("metaKey").Bool() {
		mods |= input.ModSuper
	}
	return mods
}

func mapDOMKey(key, code string) (input.Key, bool) {
	switch key {
	case "ArrowLeft":
		return input.KeyLeft, true
	case "ArrowRight":
		return input.KeyRight, true
	case "ArrowUp":
		return input.KeyUp, true
	case "ArrowDown":
		return input.KeyDown, true
	case "Home":
		return input.KeyHome, true
	case "End":
		return input.KeyEnd, true
	case "Backspace":
		return input.KeyBackspace, true
	case "Delete":
		return input.KeyDelete, true
	case "Enter":
		return input.KeyEnter, true
	case "Tab":
		return input.KeyTab, true
	case "Escape":
		return input.KeyEscape, true
	case " ":
		return input.KeySpace, true
	case "F1":
		return input.KeyF1, true
	case "F2":
		return input.KeyF2, true
	case "F3":
		return input.KeyF3, true
	case "F4":
		return input.KeyF4, true
	case "F5":
		return input.KeyF5, true
	case "F6":
		return input.KeyF6, true
	case "F7":
		return input.KeyF7, true
	case "F8":
		return input.KeyF8, true
	case "F9":
		return input.KeyF9, true
	case "F10":
		return input.KeyF10, true
	case "F11":
		return input.KeyF11, true
	case "F12":
		return input.KeyF12, true
	case "a", "A":
		return input.KeyA, true
	case "b", "B":
		return input.KeyB, true
	case "c", "C":
		return input.KeyC, true
	case "d", "D":
		return input.KeyD, true
	case "e", "E":
		return input.KeyE, true
	case "f", "F":
		return input.KeyF, true
	case "g", "G":
		return input.KeyG, true
	case "h", "H":
		return input.KeyH, true
	case "i", "I":
		return input.KeyI, true
	case "j", "J":
		return input.KeyJ, true
	case "k", "K":
		return input.KeyK, true
	case "l", "L":
		return input.KeyL, true
	case "m", "M":
		return input.KeyM, true
	case "n", "N":
		return input.KeyN, true
	case "o", "O":
		return input.KeyO, true
	case "p", "P":
		return input.KeyP, true
	case "q", "Q":
		return input.KeyQ, true
	case "r", "R":
		return input.KeyR, true
	case "s", "S":
		return input.KeyS, true
	case "t", "T":
		return input.KeyT, true
	case "u", "U":
		return input.KeyU, true
	case "v", "V":
		return input.KeyV, true
	case "w", "W":
		return input.KeyW, true
	case "x", "X":
		return input.KeyX, true
	case "y", "Y":
		return input.KeyY, true
	case "z", "Z":
		return input.KeyZ, true
	case "0":
		return input.Key0, true
	case "1":
		return input.Key1, true
	case "2":
		return input.Key2, true
	case "3":
		return input.Key3, true
	case "4":
		return input.Key4, true
	case "5":
		return input.Key5, true
	case "6":
		return input.Key6, true
	case "7":
		return input.Key7, true
	case "8":
		return input.Key8, true
	case "9":
		return input.Key9, true
	case ",":
		return input.KeyComma, true
	case ".":
		return input.KeyPeriod, true
	case "/":
		return input.KeySlash, true
	case "-":
		return input.KeyMinus, true
	case "=":
		return input.KeyEqual, true
	case "[":
		return input.KeyLeftBracket, true
	case "]":
		return input.KeyRightBracket, true
	case "`":
		return input.KeyBacktick, true
	case ";":
		return input.KeySemicolon, true
	case "'":
		return input.KeyApostrophe, true
	case "\\":
		return input.KeyBackslash, true
	}
	_ = code
	return input.KeyNone, false
}
