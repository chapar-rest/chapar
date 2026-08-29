// Package yoga is the application entry layer of the framework. It hides the
// platform boilerplate (GLFW window creation, HiDPI scaling, the WebGPU
// renderer, font-atlas baking, input wiring, and the per-frame loop) behind a
// tiny API:
//
//	yoga.Run(myApp)   // myApp implements yoga.App (a single Body(c) method)
//
// Run creates the window, then drives the per-frame loop until the window
// closes. An app's Body(c) builds its element tree each frame from the per-frame
// ui.Ctx (invalidation, animation, overlays, focus, and window dialog/toast/file
// hosts).
//
// Layering: yoga sits above render/input/layout/ui. The runtime reads the active
// theme for the GPU framebuffer clear color.
//
// Threading: the OS event/render loop must run on the thread that created the
// window (a hard requirement of GLFW on macOS). yoga.New locks the OS thread and
// Run blocks the calling (main) goroutine for the lifetime of the window.
// Application/background work may still run on other goroutines, but all window,
// input, and rendering calls happen on the main goroutine inside Run.
package yoga

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/ui"
)

// Config describes the window to create. Zero fields fall back to the defaults
// applied by New (see applyDefaults).
type Config struct {
	Title         string
	Width, Height int
	// CustomTitleBar opts into a VS Code-style title bar: the app draws chrome
	// with ui.TitleBar and the framework supplies platform window controls
	// (native traffic lights on macOS, drawn min/max/close on Windows/Linux).
	CustomTitleBar bool
}

// applyDefaults fills any unset Config fields with sensible defaults.
func (c Config) applyDefaults() Config {
	if c.Title == "" {
		c.Title = "Yoga"
	}
	if c.Width <= 0 {
		c.Width = 1100
	}
	if c.Height <= 0 {
		c.Height = 720
	}
	return c
}

// App is the application contract. An app implements a single method — Body
// builds the element tree for the current state, given the per-frame ui.Ctx
// that carries invalidation, animation scheduling, overlay registration, and
// focus. The runtime owns the window, per-frame rebuild, input dispatch, key
// routing, portal/animation plumbing, and the window dialog/toast/file hosts.
//
// Run an App with yoga.Run(app).
type App interface {
	Body(c *ui.Ctx) ui.View
}

// Closer is an optional App capability: Close releases app-owned resources
// (worker goroutines, files) when the window closes.
type Closer interface {
	Close()
}

// KeyHook is an optional App capability for app-global shortcuts. OnKey runs for
// each key event after command-palette Dispatch and before focus routing;
// returning true consumes the event so the focused widget does not also receive it.
// Prefer c.Commands().Register for actions that should appear in the palette.
type KeyHook interface {
	OnKey(c *ui.Ctx, k input.KeyEvent) bool
}

// Run creates the window (and the text engine / resources), then builds the App
// and drives its frame loop until the window closes.
//
// build is called *after* the window and resources exist, because components
// measure text at construction time and need yoga.Text() to be live. So an app
// constructs its retained widgets inside build, not before Run:
//
//	yoga.Run(cfg, BuildMyApp)   // BuildMyApp is func() *MyApp
func Run[T App](cfg Config, build func() T) error {
	a, err := New(cfg)
	if err != nil {
		return err
	}
	defer a.Close()
	a.runApp(build())
	return nil
}
