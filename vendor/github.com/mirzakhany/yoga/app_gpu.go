//go:build !nogpu

package yoga

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
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
	renderer.ClearColor = cfg.ClearColor

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
			a.paintAppFrame(a.uiApp, float32(logicalW), float32(logicalH))
		}
	})
}

// Text returns the shaped text engine for constructing widgets.
func (a *Window) Text() *shape.Engine { return a.text }

// Atlas returns the glyph atlas (legacy accessor).
func (a *Window) Atlas() *render.FontAtlas { return a.text.Atlas }

func (a *Window) Clipboard() input.Clipboard { return a.clip }
func (a *Window) Window() *glfw.Window       { return a.window }

// runApp drives the ui.App frame loop: it rebuilds the body every frame from
// the per-frame ui.Ctx, so overlay and animation registration are collected
// fresh each frame. The body is built twice per drawn frame: once to hit-test
// input against fresh geometry, then again so paint reflects any state changed
// by that input the same frame. The event loop only iterates on input or an
// Animate request, so this double-build never runs while idle.
func (a *Window) runApp(app App) {
	a.uiApp = app
	a.uiFocus = ui.NewFocusScope()
	a.uiCtx = ui.New(a.text, a.uiFocus, glfw.PostEmptyEvent)
	a.uiCtx.SetIcons(Icons())
	a.uiCtx.SetClipboard(a.clip)

	if cc, ok := app.(interface{ ClearColor() render.Color }); ok {
		a.renderer.ClearColor = cc.ClearColor()
	}

	for !a.window.ShouldClose() {
		fw, fh := a.window.GetSize()
		if fw > 0 && fh > 0 {
			w, h := float32(fw), float32(fh)
			a.mouse.Cursor = input.CursorDefault

			// Build for input, dispatch, then route keys.
			inRoot := a.buildAppFrame(app, w, h)
			layout.Dispatch(inRoot, a.mouse)
			a.uiFocus.HandleMouse(a.mouse)
			if hook, ok := app.(KeyHook); ok {
				a.routeAppKeys(hook)
			}
			a.uiFocus.Route(a.keyboard)

			a.applyCursor()
			a.mouse.EndFrame()
			a.keyboard.EndFrame()

			if cc, ok := app.(interface{ ClearColor() render.Color }); ok {
				a.renderer.ClearColor = cc.ClearColor()
			}

			// Rebuild for paint so it reflects post-input state.
			a.paintAppFrame(app, w, h)
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

// buildAppFrame builds and solves one frame via the shared ui driver.
func (a *Window) buildAppFrame(app App, w, h float32) *layout.Element {
	return ui.BuildFrame(a.uiCtx, app.Body, w, h, a.mouse, a.keyboard)
}

// paintAppFrame rebuilds the body and submits a GPU frame.
func (a *Window) paintAppFrame(app App, w, h float32) {
	root := a.buildAppFrame(app, w, h)
	a.drawList.Reset()
	layout.Paint(root, &a.drawList, a.text)
	_ = a.text.FlushAtlas(a.renderer)
	if err := a.renderer.Render(&a.drawList); err != nil && !transientSurfaceError(err) {
		fmt.Println("render error:", err)
	}
}

// routeAppKeys lets the app consume key events before focus routing. Consumed
// events are removed from this frame's keyboard so the focused widget skips them.
func (a *Window) routeAppKeys(hook KeyHook) {
	keys := a.keyboard.Keys
	kept := keys[:0]
	for _, k := range keys {
		if !hook.OnKey(a.uiCtx, k) {
			kept = append(kept, k)
		}
	}
	a.keyboard.Keys = kept
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

func transientSurfaceError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Surface timed out") ||
		strings.Contains(s, "Surface is outdated") ||
		strings.Contains(s, "Surface was lost")
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
	case glfw.KeyA:
		return input.KeyA, true
	case glfw.KeyC:
		return input.KeyC, true
	case glfw.KeyV:
		return input.KeyV, true
	case glfw.KeyX:
		return input.KeyX, true
	case glfw.KeyZ:
		return input.KeyZ, true
	case glfw.KeyS:
		return input.KeyS, true
	case glfw.KeyF:
		return input.KeyF, true
	case glfw.KeyH:
		return input.KeyH, true
	case glfw.KeyEscape:
		return input.KeyEscape, true
	case glfw.KeySpace:
		return input.KeySpace, true
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
